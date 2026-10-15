package siyi

import (
	"context"
	"log/slog"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/commands"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

var fireAndForget = map[uint8]bool{
	protocol.CmdCapturePhotoRecordVideo: true,
	protocol.CmdSendAircraftAttitude:    true,
	protocol.CmdSendRCChannels:          true,
	protocol.CmdSendRawGPS:              true,
}

var idempotentReads = map[uint8]bool{
	protocol.CmdRequestFirmwareVersion:    true,
	protocol.CmdRequestHardwareID:         true,
	protocol.CmdOneKeyCentering:           true,
	protocol.CmdRequestCameraSystemInfo:   true,
	protocol.CmdRequestGimbalAttitude:     true,
	protocol.CmdSetGimbalAttitude:         true,
	protocol.CmdRequestVideoStitchingMode: true,
	protocol.CmdRequestLaserDistance:      true,
	protocol.CmdRequestZoomRange:          true,
	protocol.CmdRequestLaserLatLon:        true,
	protocol.CmdRequestZoomMagnification:  true,
	protocol.CmdRequestGimbalMode:         true,
	protocol.CmdRequestPseudoColor:        true,
	protocol.CmdRequestEncodingParams:     true,
	protocol.CmdRequestMagneticEncoder:    true,
	protocol.CmdRequestControlMode:        true,
	protocol.CmdRequestWeakThreshold:      true,
	protocol.CmdRequestMotorVoltage:       true,
	protocol.CmdRequestGimbalSystemInfo:   true,
	protocol.CmdRequestThermalOutputMode:  true,
	protocol.CmdRequestThermalGain:        true,
	protocol.CmdRequestEnvCorrectionParams: true,
	protocol.CmdRequestEnvCorrectionSwitch: true,
	protocol.CmdRequestSystemTime:         true,
	protocol.CmdSingleAxisAttitude:        true,
	protocol.CmdGetIRThreshMapSta:         true,
	protocol.CmdGetIRThreshParam:          true,
	protocol.CmdGetIRThreshPrecision:      true,
	protocol.CmdGetPicNameType:            true,
	protocol.CmdGetMavlinkOSDFlag:         true,
	protocol.CmdGetAIModeSta:              true,
	protocol.CmdGetAITrackStreamSta:       true,
	protocol.CmdRequestWeakControlMode:    true,
	protocol.CmdGetIP:                     true,
}

var streamPushCmds = map[uint8]bool{
	protocol.CmdRequestGimbalAttitude:  true,
	protocol.CmdFunctionFeedback:       true,
	protocol.CmdRequestLaserDistance:   true,
	protocol.CmdRequestMagneticEncoder: true,
	protocol.CmdRequestMotorVoltage:    true,
	protocol.CmdAITrackStream:          true,
}

// Client provides high-level control and telemetry monitoring for SIYI gimbal cameras.
type Client struct {
	transport transport.Transport
	cfg       ClientConfig
	parser    *protocol.FrameParser

	mu         sync.Mutex
	seq        uint16
	closed     bool
	cancelFunc context.CancelFunc

	pendingMu sync.Mutex
	pending   map[uint8]chan protocol.Frame

	locksMu  sync.Mutex
	cmdLocks map[uint8]*sync.Mutex

	cbMu             sync.RWMutex
	attitudeCbs      []func(models.GimbalAttitude)
	feedbackCbs      []func(models.FunctionFeedback)
	laserCbs         []func(models.LaserDistance)
	aiCbs            []func(models.AITrackingTarget)
	connStatusCbs    []func(bool)
}

// NewClient instantiates a new Client wrapping the provided transport.
func NewClient(t transport.Transport, opts ...ClientOption) *Client {
	cfg := ClientConfig{
		DefaultTimeout: 2 * time.Second,
		MaxRetries:     2,
		RetryBaseDelay: 100 * time.Millisecond,
		AutoReconnect:  false,
		Logger:         slog.Default(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Client{
		transport:     t,
		cfg:           cfg,
		parser:        protocol.NewFrameParser(),
		pending:       make(map[uint8]chan protocol.Frame),
		cmdLocks:      make(map[uint8]*sync.Mutex),
		attitudeCbs:   make([]func(models.GimbalAttitude), 0),
		feedbackCbs:   make([]func(models.FunctionFeedback), 0),
		laserCbs:      make([]func(models.LaserDistance), 0),
		aiCbs:         make([]func(models.AITrackingTarget), 0),
		connStatusCbs: make([]func(bool), 0),
	}
}

// Connect establishes the connection and initiates background processing.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	if err := c.transport.Connect(ctx); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	go c.readerLoop(runCtx)

	if c.transport.SupportsHeartbeat() {
		go c.heartbeatLoop(runCtx)
	}

	c.notifyConnStatus(true)
	return nil
}

// Close gracefully shuts down the client and releases resources.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.mu.Unlock()

	// Drain any pending response channels
	c.pendingMu.Lock()
	for _, ch := range c.pending {
		close(ch)
	}
	c.pending = make(map[uint8]chan protocol.Frame)
	c.pendingMu.Unlock()

	err := c.transport.Close()
	c.notifyConnStatus(false)
	return err
}

// IsConnected reports whether the transport is active.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.closed && c.transport.IsConnected()
}

func (c *Client) notifyConnStatus(status bool) {
	c.cbMu.RLock()
	cbs := c.connStatusCbs
	c.cbMu.RUnlock()
	for _, cb := range cbs {
		cb(status)
	}
}

func (c *Client) getCmdLock(cmdID uint8) *sync.Mutex {
	c.locksMu.Lock()
	defer c.locksMu.Unlock()
	l, ok := c.cmdLocks[cmdID]
	if !ok {
		l = &sync.Mutex{}
		c.cmdLocks[cmdID] = l
	}
	return l
}

func (c *Client) nextSeq() uint16 {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.seq
	c.seq = (c.seq + 1) & protocol.SeqMax
	return s
}

func (c *Client) readerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-c.transport.Stream():
			if !ok {
				return
			}
			frames, err := c.parser.Feed(chunk)
			if err != nil {
				c.cfg.Logger.Warn("parser error skipped", "err", err)
			}
			for _, frame := range frames {
				c.dispatchFrame(frame)
			}
		}
	}
}

func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.IsConnected() {
				_ = c.Heartbeat(ctx)
			}
		}
	}
}

func (c *Client) dispatchFrame(frame protocol.Frame) {
	cmdID := frame.CmdID

	// Priority 1: explicitly pending requests
	c.pendingMu.Lock()
	ch, ok := c.pending[cmdID]
	c.pendingMu.Unlock()

	if ok && ch != nil {
		select {
		case ch <- frame:
			return
		default:
		}
	}

	// Priority 2: unsolicited telemetry/stream pushes
	if streamPushCmds[cmdID] {
		c.dispatchStream(frame)
	}
}

func (c *Client) dispatchStream(frame protocol.Frame) {
	c.cbMu.RLock()
	defer c.cbMu.RUnlock()

	switch frame.CmdID {
	case protocol.CmdRequestGimbalAttitude:
		if att, err := commands.DecodeGimbalAttitude(frame.Data); err == nil {
			for _, cb := range c.attitudeCbs {
				cb(att)
			}
		}
	case protocol.CmdFunctionFeedback:
		if fb, err := commands.DecodeFunctionFeedback(frame.Data); err == nil {
			for _, cb := range c.feedbackCbs {
				cb(fb)
			}
		}
	case protocol.CmdRequestLaserDistance:
		if ld, err := commands.DecodeLaserDistance(frame.Data); err == nil {
			for _, cb := range c.laserCbs {
				cb(ld)
			}
		}
	case protocol.CmdAITrackStream:
		if ai, err := commands.DecodeAITracking(frame.Data); err == nil {
			for _, cb := range c.aiCbs {
				cb(ai)
			}
		}
	}
}

// Unsubscribe is a function that cancels a telemetry subscription.
type Unsubscribe func()

// OnAttitude registers a listener for real-time attitude pushes and returns an unsubscribe func.
func (c *Client) OnAttitude(cb func(models.GimbalAttitude)) Unsubscribe {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.attitudeCbs = append(c.attitudeCbs, cb)
	return func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()
		for i, fn := range c.attitudeCbs {
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(cb).Pointer() {
				c.attitudeCbs = append(c.attitudeCbs[:i], c.attitudeCbs[i+1:]...)
				break
			}
		}
	}
}

// OnFunctionFeedback registers a listener for camera function feedback pushes and returns an unsubscribe func.
func (c *Client) OnFunctionFeedback(cb func(models.FunctionFeedback)) Unsubscribe {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.feedbackCbs = append(c.feedbackCbs, cb)
	return func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()
		for i, fn := range c.feedbackCbs {
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(cb).Pointer() {
				c.feedbackCbs = append(c.feedbackCbs[:i], c.feedbackCbs[i+1:]...)
				break
			}
		}
	}
}

// OnLaser registers a listener for laser distance telemetry pushes and returns an unsubscribe func.
func (c *Client) OnLaser(cb func(models.LaserDistance)) Unsubscribe {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.laserCbs = append(c.laserCbs, cb)
	return func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()
		for i, fn := range c.laserCbs {
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(cb).Pointer() {
				c.laserCbs = append(c.laserCbs[:i], c.laserCbs[i+1:]...)
				break
			}
		}
	}
}

// OnAITracking registers a listener for AI bounding-box tracking pushes and returns an unsubscribe func.
func (c *Client) OnAITracking(cb func(models.AITrackingTarget)) Unsubscribe {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.aiCbs = append(c.aiCbs, cb)
	return func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()
		for i, fn := range c.aiCbs {
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(cb).Pointer() {
				c.aiCbs = append(c.aiCbs[:i], c.aiCbs[i+1:]...)
				break
			}
		}
	}
}

// OnConnectionStatus registers a listener for connection changes and returns an unsubscribe func.
func (c *Client) OnConnectionStatus(cb func(bool)) Unsubscribe {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.connStatusCbs = append(c.connStatusCbs, cb)
	return func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()
		for i, fn := range c.connStatusCbs {
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(cb).Pointer() {
				c.connStatusCbs = append(c.connStatusCbs[:i], c.connStatusCbs[i+1:]...)
				break
			}
		}
	}
}

func (c *Client) sendCommandNoWait(ctx context.Context, cmdID uint8, payload []byte) error {
	if !c.IsConnected() {
		return transport.ErrNotConnected
	}
	seq := c.nextSeq()
	frame := protocol.Build(cmdID, payload, seq, false)
	return c.transport.Send(ctx, frame.ToBytes())
}

func (c *Client) sendCommand(ctx context.Context, cmdID uint8, payload []byte) ([]byte, error) {
	if !c.IsConnected() {
		return nil, transport.ErrNotConnected
	}

	seq := c.nextSeq()

	if fireAndForget[cmdID] {
		frame := protocol.Build(cmdID, payload, seq, false)
		return nil, c.transport.Send(ctx, frame.ToBytes())
	}

	cmdLock := c.getCmdLock(cmdID)
	cmdLock.Lock()
	defer cmdLock.Unlock()

	timeout := c.cfg.DefaultTimeout
	if d, ok := ctx.Deadline(); ok {
		t := time.Until(d)
		if t > 0 {
			timeout = t
		}
	}

	maxAttempts := 1
	if idempotentReads[cmdID] {
		maxAttempts = c.cfg.MaxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		respCh := make(chan protocol.Frame, 1)

		c.pendingMu.Lock()
		c.pending[cmdID] = respCh
		c.pendingMu.Unlock()

		frame := protocol.Build(cmdID, payload, seq, true)
		if err := c.transport.Send(ctx, frame.ToBytes()); err != nil {
			c.pendingMu.Lock()
			delete(c.pending, cmdID)
			c.pendingMu.Unlock()
			return nil, err
		}

		select {
		case respFrame := <-respCh:
			c.pendingMu.Lock()
			delete(c.pending, cmdID)
			c.pendingMu.Unlock()
			return respFrame.Data, nil

		case <-time.After(timeout):
			c.pendingMu.Lock()
			delete(c.pending, cmdID)
			c.pendingMu.Unlock()

			if attempt < maxAttempts-1 {
				backoff := c.cfg.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt)))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
			return nil, &TimeoutError{CmdID: cmdID, Duration: timeout.String()}

		case <-ctx.Done():
			c.pendingMu.Lock()
			delete(c.pending, cmdID)
			c.pendingMu.Unlock()
			return nil, ctx.Err()
		}
	}

	return nil, &TimeoutError{CmdID: cmdID, Duration: timeout.String()}
}

// -----------------------------------------------------------------------------
// System Commands
// -----------------------------------------------------------------------------

// Heartbeat sends a TCP keepalive frame (0x00).
func (c *Client) Heartbeat(ctx context.Context) error {
	_, err := c.sendCommand(ctx, protocol.CmdTCPHeartbeat, commands.EncodeHeartbeat())
	return err
}

// GetFirmwareVersion queries camera, gimbal, and zoom firmware words (0x01).
func (c *Client) GetFirmwareVersion(ctx context.Context) (models.FirmwareVersion, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestFirmwareVersion, commands.EncodeFirmwareVersion())
	if err != nil {
		return models.FirmwareVersion{}, err
	}
	return commands.DecodeFirmwareVersion(data)
}

// GetHardwareID queries the 12-byte hardware identifier string (0x02).
func (c *Client) GetHardwareID(ctx context.Context) (models.HardwareID, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestHardwareID, commands.EncodeHardwareID())
	if err != nil {
		return models.HardwareID{}, err
	}
	return commands.DecodeHardwareID(data)
}

// SetUTCTime synchronizes camera real-time clock using microsecond timestamp (0x30).
func (c *Client) SetUTCTime(ctx context.Context, unixUsec uint64) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetUTCTime, commands.EncodeSetUTCTime(unixUsec))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetUTCTimeAck(data)
}

// GetGimbalSystemInfo queries gimbal system state such as laser status (0x31).
func (c *Client) GetGimbalSystemInfo(ctx context.Context) (models.GimbalSystemInfo, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestGimbalSystemInfo, commands.EncodeGimbalSystemInfo())
	if err != nil {
		return models.GimbalSystemInfo{}, err
	}
	return commands.DecodeGimbalSystemInfo(data)
}

// GetSystemTime queries camera timestamp and uptime (0x40).
func (c *Client) GetSystemTime(ctx context.Context) (models.SystemTime, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestSystemTime, commands.EncodeSystemTime())
	if err != nil {
		return models.SystemTime{}, err
	}
	return commands.DecodeSystemTime(data)
}

// SoftReboot reboots camera and/or gimbal hardware (0x80).
func (c *Client) SoftReboot(ctx context.Context, camera, gimbal bool) (cameraOK, gimbalOK bool, err error) {
	data, err := c.sendCommand(ctx, protocol.CmdSoftReboot, commands.EncodeSoftReboot(camera, gimbal))
	if err != nil {
		return false, false, err
	}
	return commands.DecodeSoftRebootAck(data)
}

// GetIP queries camera network configuration (0x81).
func (c *Client) GetIP(ctx context.Context) (models.IPConfig, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetIP, commands.EncodeGetIP())
	if err != nil {
		return models.IPConfig{}, err
	}
	return commands.DecodeGetIP(data)
}

// SetIP configures camera IP, subnet mask, and default gateway (0x82).
func (c *Client) SetIP(ctx context.Context, cfg models.IPConfig) (bool, error) {
	payload, err := commands.EncodeSetIP(cfg)
	if err != nil {
		return false, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdSetIP, payload)
	if err != nil {
		return false, err
	}
	return commands.DecodeSetIPAck(data)
}

// -----------------------------------------------------------------------------
// Attitude & Motion Commands
// -----------------------------------------------------------------------------

// GetGimbalAttitude queries current Euler angles and angular rates (0x0D).
func (c *Client) GetGimbalAttitude(ctx context.Context) (models.GimbalAttitude, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestGimbalAttitude, commands.EncodeGimbalAttitude())
	if err != nil {
		return models.GimbalAttitude{}, err
	}
	return commands.DecodeGimbalAttitude(data)
}

// SendAircraftAttitude feeds vehicle flight attitude into gimbal stabilization loop (0x22, fire-and-forget).
func (c *Client) SendAircraftAttitude(ctx context.Context, att models.AircraftAttitude) error {
	_, err := c.sendCommand(ctx, protocol.CmdSendAircraftAttitude, commands.EncodeAircraftAttitude(att))
	return err
}

// RequestFCStream starts/stops telemetry streaming from flight controller (0x24).
func (c *Client) RequestFCStream(ctx context.Context, dataType models.FCDataType, freq models.DataStreamFreq) (models.FCDataType, error) {
	payload, err := commands.EncodeFCStream(dataType, freq)
	if err != nil {
		return 0, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdRequestFCDataStream, payload)
	if err != nil {
		return 0, err
	}
	return commands.DecodeFCStreamAck(data)
}

// RequestGimbalStream configures continuous streaming of gimbal attitude/laser/encoder data (0x25).
func (c *Client) RequestGimbalStream(ctx context.Context, dataType models.GimbalDataType, freq models.DataStreamFreq) (models.GimbalDataType, error) {
	payload, err := commands.EncodeGimbalStream(dataType, freq)
	if err != nil {
		return 0, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdRequestGimbalDataStream, payload)
	if err != nil {
		return 0, err
	}
	return commands.DecodeGimbalStreamAck(data)
}

// GetMagneticEncoder queries raw magnetic encoder values in degrees (0x26).
func (c *Client) GetMagneticEncoder(ctx context.Context) (models.MagneticEncoderAngles, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestMagneticEncoder, commands.EncodeMagneticEncoder())
	if err != nil {
		return models.MagneticEncoderAngles{}, err
	}
	return commands.DecodeMagneticEncoder(data)
}

// SendRawGPS sends raw GPS coordinates to camera for photo EXIF geotagging (0x3E, fire-and-forget).
func (c *Client) SendRawGPS(ctx context.Context, gps models.RawGPS) error {
	_, err := c.sendCommand(ctx, protocol.CmdSendRawGPS, commands.EncodeRawGPS(gps))
	return err
}

// -----------------------------------------------------------------------------
// Gimbal Control Commands
// -----------------------------------------------------------------------------

// Rotate commands rotation rates along yaw and pitch axes [-100, 100] (0x07).
func (c *Client) Rotate(ctx context.Context, yaw, pitch int) error {
	payload, err := commands.EncodeRotation(yaw, pitch)
	if err != nil {
		return err
	}
	data, err := c.sendCommand(ctx, protocol.CmdGimbalRotation, payload)
	if err != nil {
		return err
	}
	return commands.DecodeRotationAck(data)
}

// RotateNoWait sends a gimbal velocity command without waiting for an ACK (CMD 0x07).
// Intended for high-rate visual servoing control loops (50-100Hz).
func (c *Client) RotateNoWait(ctx context.Context, yaw, pitch int) error {
	payload, err := commands.EncodeRotation(yaw, pitch)
	if err != nil {
		return err
	}
	return c.sendCommandNoWait(ctx, protocol.CmdGimbalRotation, payload)
}

// OneKeyCentering triggers automatic recentering or downward looking position (0x08).
func (c *Client) OneKeyCentering(ctx context.Context, action models.CenteringAction) error {
	payload, err := commands.EncodeOneKeyCentering(action)
	if err != nil {
		return err
	}
	data, err := c.sendCommand(ctx, protocol.CmdOneKeyCentering, payload)
	if err != nil {
		return err
	}
	return commands.DecodeOneKeyCenteringAck(data)
}

// SetGimbalAttitude moves the gimbal to target yaw and pitch angles in degrees (0x0E).
func (c *Client) SetGimbalAttitude(ctx context.Context, yawDeg, pitchDeg float64) (models.SetAttitudeAck, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetGimbalAttitude, commands.EncodeSetAttitude(yawDeg, pitchDeg))
	if err != nil {
		return models.SetAttitudeAck{}, err
	}
	return commands.DecodeSetAttitudeAck(data)
}

// SetGimbalAttitudeNoWait sends a gimbal position setpoint without waiting for an ACK (CMD 0x0E).
// Fire-and-forget variant of SetGimbalAttitude for high-rate control loops.
func (c *Client) SetGimbalAttitudeNoWait(ctx context.Context, yawDeg, pitchDeg float64) error {
	payload := commands.EncodeSetAttitude(yawDeg, pitchDeg)
	return c.sendCommandNoWait(ctx, protocol.CmdSetGimbalAttitude, payload)
}

// GetGimbalMode queries gimbal lock/follow/FPV mode (0x19).
func (c *Client) GetGimbalMode(ctx context.Context) (models.GimbalMotionMode, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestGimbalMode, commands.EncodeGimbalMode())
	if err != nil {
		return 0, err
	}
	return commands.DecodeGimbalMode(data)
}

// SetSingleAxis targets a specific single axis angle in degrees (0x41). axis: 0=yaw, 1=pitch.
func (c *Client) SetSingleAxis(ctx context.Context, angleDeg float64, axis int) (models.SetAttitudeAck, error) {
	payload, err := commands.EncodeSingleAxis(angleDeg, axis)
	if err != nil {
		return models.SetAttitudeAck{}, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdSingleAxisAttitude, payload)
	if err != nil {
		return models.SetAttitudeAck{}, err
	}
	return commands.DecodeSingleAxisAck(data)
}

// SetSingleAxisNoWait sends a single-axis position setpoint without waiting for an ACK (CMD 0x41).
func (c *Client) SetSingleAxisNoWait(ctx context.Context, angleDeg float64, axis int) error {
	payload, err := commands.EncodeSingleAxis(angleDeg, axis)
	if err != nil {
		return err
	}
	return c.sendCommandNoWait(ctx, protocol.CmdSingleAxisAttitude, payload)
}

// -----------------------------------------------------------------------------
// Camera Control Commands
// -----------------------------------------------------------------------------

// GetCameraSystemInfo queries operating status, recording status, and HDMI output (0x0A).
func (c *Client) GetCameraSystemInfo(ctx context.Context) (models.CameraSystemInfo, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestCameraSystemInfo, commands.EncodeCameraSystemInfo())
	if err != nil {
		return models.CameraSystemInfo{}, err
	}
	return commands.DecodeCameraSystemInfo(data)
}

// Capture triggers photo or video recording actions (0x0C, fire-and-forget).
func (c *Client) Capture(ctx context.Context, funcType models.CaptureFuncType) error {
	_, err := c.sendCommand(ctx, protocol.CmdCapturePhotoRecordVideo, commands.EncodeCapture(funcType))
	return err
}

// TakePhoto captures a single still photo.
func (c *Client) TakePhoto(ctx context.Context) error {
	return c.Capture(ctx, models.CaptureFuncPhoto)
}

// StartRecording begins video recording to SD card.
func (c *Client) StartRecording(ctx context.Context) error {
	return c.Capture(ctx, models.CaptureFuncStartRecord)
}

// StopRecording halts video recording.
func (c *Client) StopRecording(ctx context.Context) error {
	return c.Capture(ctx, models.CaptureFuncStartRecord)
}

// GetEncodingParams queries video resolution, bitrate, and framerate (0x20).
func (c *Client) GetEncodingParams(ctx context.Context, stream models.StreamType) (models.EncodingParams, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestEncodingParams, commands.EncodeGetEncodingParams(stream))
	if err != nil {
		return models.EncodingParams{}, err
	}
	return commands.DecodeGetEncodingParams(data)
}

// SetEncodingParams configures resolution and bitrate for stream slot (0x21).
func (c *Client) SetEncodingParams(ctx context.Context, params models.EncodingParams) (bool, error) {
	payload, err := commands.EncodeSetEncodingParams(params)
	if err != nil {
		return false, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdSetEncodingParams, payload)
	if err != nil {
		return false, err
	}
	return commands.DecodeSetEncodingParamsAck(data)
}

// FormatSDCard formats the camera TF storage card (0x48).
func (c *Client) FormatSDCard(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSDFormat, commands.EncodeFormatSD())
	if err != nil {
		return false, err
	}
	return commands.DecodeFormatSDAck(data)
}

// GetPicNameType queries picture naming convention (0x49).
func (c *Client) GetPicNameType(ctx context.Context, ft models.FileType) (models.FileNameType, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetPicNameType, commands.EncodeGetPicNameType(ft))
	if err != nil {
		return 0, err
	}
	return commands.DecodeGetPicNameType(data)
}

// SetPicNameType sets picture naming convention (0x4A).
func (c *Client) SetPicNameType(ctx context.Context, ft models.FileType, nt models.FileNameType) error {
	_, err := c.sendCommand(ctx, protocol.CmdSetPicNameType, commands.EncodeSetPicNameType(ft, nt))
	return err
}

// GetMavlinkOSDFlag queries HDMI OSD overlay state (0x4B).
func (c *Client) GetMavlinkOSDFlag(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetMavlinkOSDFlag, commands.EncodeGetOSDFlag())
	if err != nil {
		return false, err
	}
	return commands.DecodeGetOSDFlag(data)
}

// SetMavlinkOSDFlag enables or disables HDMI OSD overlay (0x4C).
func (c *Client) SetMavlinkOSDFlag(ctx context.Context, on bool) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetMavlinkOSDFlag, commands.EncodeSetOSDFlag(on))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetOSDFlagAck(data)
}

// -----------------------------------------------------------------------------
// Zoom & Focus Commands
// -----------------------------------------------------------------------------

// ManualZoom controls zoom velocity (-1=zoom out, 0=stop, 1=zoom in) (0x05).
func (c *Client) ManualZoom(ctx context.Context, direction int) (float64, error) {
	payload, err := commands.EncodeManualZoom(direction)
	if err != nil {
		return 0, err
	}
	data, err := c.sendCommand(ctx, protocol.CmdManualZoomAutoFocus, payload)
	if err != nil {
		return 0, err
	}
	return commands.DecodeManualZoomAck(data)
}

// AbsoluteZoom commands optical zoom to an exact target magnification [1.0, 30.0] (0x0F).
func (c *Client) AbsoluteZoom(ctx context.Context, zoom float64) error {
	payload, err := commands.EncodeAbsoluteZoom(zoom)
	if err != nil {
		return err
	}
	data, err := c.sendCommand(ctx, protocol.CmdAbsoluteZoomAutoFocus, payload)
	if err != nil {
		return err
	}
	return commands.DecodeAbsoluteZoomAck(data)
}

// GetZoomRange queries maximum optical zoom level (0x16).
func (c *Client) GetZoomRange(ctx context.Context) (models.ZoomRange, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestZoomRange, commands.EncodeZoomRange())
	if err != nil {
		return models.ZoomRange{}, err
	}
	return commands.DecodeZoomRange(data)
}

// GetCurrentZoom queries current zoom magnification (0x18).
func (c *Client) GetCurrentZoom(ctx context.Context) (models.CurrentZoom, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestZoomMagnification, commands.EncodeCurrentZoom())
	if err != nil {
		return models.CurrentZoom{}, err
	}
	return commands.DecodeCurrentZoom(data)
}

// AutoFocus triggers focus at screen coordinates (0x04).
func (c *Client) AutoFocus(ctx context.Context, autoFocus uint8, touchX, touchY uint16) error {
	data, err := c.sendCommand(ctx, protocol.CmdAutoFocus, commands.EncodeAutoFocus(autoFocus, touchX, touchY))
	if err != nil {
		return err
	}
	return commands.DecodeAutoFocusAck(data)
}

// ManualFocus controls focal distance step (-1=near, 0=stop, 1=far) (0x06).
func (c *Client) ManualFocus(ctx context.Context, direction int) error {
	payload, err := commands.EncodeManualFocus(direction)
	if err != nil {
		return err
	}
	data, err := c.sendCommand(ctx, protocol.CmdManualFocus, payload)
	if err != nil {
		return err
	}
	return commands.DecodeManualFocusAck(data)
}

// -----------------------------------------------------------------------------
// Laser Ranging Commands
// -----------------------------------------------------------------------------

// GetLaserDistance queries target distance measurement in meters (0x15).
func (c *Client) GetLaserDistance(ctx context.Context) (models.LaserDistance, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestLaserDistance, commands.EncodeLaserDistance())
	if err != nil {
		return models.LaserDistance{}, err
	}
	return commands.DecodeLaserDistance(data)
}

// GetLaserTargetLatLon queries geographic coordinates of laser target point (0x17).
func (c *Client) GetLaserTargetLatLon(ctx context.Context) (models.LaserTargetLatLon, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestLaserLatLon, commands.EncodeLaserTargetLatLon())
	if err != nil {
		return models.LaserTargetLatLon{}, err
	}
	return commands.DecodeLaserTargetLatLon(data)
}

// SetLaserRangingState turns laser rangefinder beam on/off (0x32).
func (c *Client) SetLaserRangingState(ctx context.Context, on bool) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetLaserRangingState, commands.EncodeSetLaserRangingState(on))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetLaserRangingStateAck(data)
}

// -----------------------------------------------------------------------------
// RC & Video Commands
// -----------------------------------------------------------------------------

// SendRCChannels transmits 18-channel RC joystick values (0x23, fire-and-forget).
func (c *Client) SendRCChannels(ctx context.Context, ch models.RCChannels) error {
	_, err := c.sendCommand(ctx, protocol.CmdSendRCChannels, commands.EncodeRCChannels(ch))
	return err
}

// GetVideoStitchingMode queries multi-sensor display stitching layout (0x10).
func (c *Client) GetVideoStitchingMode(ctx context.Context) (models.VideoStitchingMode, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestVideoStitchingMode, commands.EncodeGetVideoStitchingMode())
	if err != nil {
		return 0, err
	}
	return commands.DecodeVideoStitchingMode(data)
}

// SetVideoStitchingMode configures multi-sensor display stitching layout (0x11).
func (c *Client) SetVideoStitchingMode(ctx context.Context, mode models.VideoStitchingMode) (models.VideoStitchingMode, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetVideoStitchingMode, commands.EncodeSetVideoStitchingMode(mode))
	if err != nil {
		return 0, err
	}
	return commands.DecodeSetVideoStitchingModeAck(data)
}

// -----------------------------------------------------------------------------
// AI Tracking Commands
// -----------------------------------------------------------------------------

// GetAIMode queries whether AI recognition and tracking module is active (0x4D).
func (c *Client) GetAIMode(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetAIModeSta, commands.EncodeGetAIMode())
	if err != nil {
		return false, err
	}
	return commands.DecodeAIMode(data)
}

// GetAIStreamStatus queries AI coordinate stream output status (0x4E).
func (c *Client) GetAIStreamStatus(ctx context.Context) (models.AIStreamStatus, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetAITrackStreamSta, commands.EncodeGetAIStreamStatus())
	if err != nil {
		return 0, err
	}
	return commands.DecodeAIStreamStatus(data)
}

// SetAIStreamOutput starts or stops continuous output of AI bounding boxes (0x51).
func (c *Client) SetAIStreamOutput(ctx context.Context, on bool) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetAITrackStreamSta, commands.EncodeSetAIStreamOutput(on))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetAIStreamOutputAck(data)
}

// -----------------------------------------------------------------------------
// Debug & ArduPilot Commands
// -----------------------------------------------------------------------------

// GetControlMode queries ArduPilot gimbal control mode (0x27).
func (c *Client) GetControlMode(ctx context.Context) (models.ControlMode, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestControlMode, commands.EncodeGetControlMode())
	if err != nil {
		return 0, err
	}
	return commands.DecodeControlMode(data)
}

// GetWeakThreshold queries weak control threshold limits (0x28).
func (c *Client) GetWeakThreshold(ctx context.Context) (models.WeakControlThreshold, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestWeakThreshold, commands.EncodeGetWeakThreshold())
	if err != nil {
		return models.WeakControlThreshold{}, err
	}
	return commands.DecodeWeakThreshold(data)
}

// SetWeakThreshold sets weak control threshold limits (0x29).
func (c *Client) SetWeakThreshold(ctx context.Context, t models.WeakControlThreshold) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetWeakThreshold, commands.EncodeSetWeakThreshold(t))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetWeakThresholdAck(data)
}

// GetMotorVoltage queries electrical voltages on yaw, pitch, roll motor drives (0x2A).
func (c *Client) GetMotorVoltage(ctx context.Context) (models.MotorVoltage, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestMotorVoltage, commands.EncodeGetMotorVoltage())
	if err != nil {
		return models.MotorVoltage{}, err
	}
	return commands.DecodeMotorVoltage(data)
}

// GetWeakControlMode queries weak control mode activation status (0x70).
func (c *Client) GetWeakControlMode(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestWeakControlMode, commands.EncodeGetWeakControlMode())
	if err != nil {
		return false, err
	}
	return commands.DecodeWeakControlMode(data)
}

// SetWeakControlMode enables or disables weak control mode (0x71).
func (c *Client) SetWeakControlMode(ctx context.Context, on bool) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdSetWeakControlMode, commands.EncodeSetWeakControlMode(on))
	if err != nil {
		return false, err
	}
	return commands.DecodeSetWeakControlModeAck(data)
}

// -----------------------------------------------------------------------------
// Thermal Imaging Commands
// -----------------------------------------------------------------------------

// GetTempAtPoint queries spot temperature at coordinate (x, y) (0x12).
func (c *Client) GetTempAtPoint(ctx context.Context, x, y uint16, flag models.TempMeasureFlag) (models.TempPoint, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetTempAtPoint, commands.EncodeTempAtPoint(x, y, flag))
	if err != nil {
		return models.TempPoint{}, err
	}
	return commands.DecodeTempAtPoint(data)
}

// GetLocalTemp queries regional temperature box (0x13).
func (c *Client) GetLocalTemp(ctx context.Context, startX, startY, endX, endY uint16, flag models.TempMeasureFlag) (models.TempRegion, error) {
	data, err := c.sendCommand(ctx, protocol.CmdLocalTempMeasurement, commands.EncodeLocalTemp(startX, startY, endX, endY, flag))
	if err != nil {
		return models.TempRegion{}, err
	}
	return commands.DecodeLocalTemp(data)
}

// GetGlobalTemp queries maximum and minimum temperatures in full thermal view (0x14).
func (c *Client) GetGlobalTemp(ctx context.Context, flag models.TempMeasureFlag) (models.TempGlobal, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGlobalTempMeasurement, commands.EncodeGlobalTemp(flag))
	if err != nil {
		return models.TempGlobal{}, err
	}
	return commands.DecodeGlobalTemp(data)
}

// GetPseudoColor queries thermal pseudo-color palette (0x1A).
func (c *Client) GetPseudoColor(ctx context.Context) (models.PseudoColor, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestPseudoColor, commands.EncodeGetPseudoColor())
	if err != nil {
		return 0, err
	}
	return commands.DecodePseudoColor(data)
}

// SetPseudoColor selects thermal pseudo-color palette (0x1B).
func (c *Client) SetPseudoColor(ctx context.Context, color models.PseudoColor) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetPseudoColor, commands.EncodeSetPseudoColor(color))
	if err != nil {
		return err
	}
	return commands.DecodeSetPseudoColorAck(data)
}

// GetThermalOutputMode queries thermal sensor framerate and mode (0x33).
func (c *Client) GetThermalOutputMode(ctx context.Context) (models.ThermalOutputMode, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestThermalOutputMode, commands.EncodeGetThermalOutputMode())
	if err != nil {
		return 0, err
	}
	return commands.DecodeThermalOutputMode(data)
}

// SetThermalOutputMode sets thermal output mode (0x34).
func (c *Client) SetThermalOutputMode(ctx context.Context, mode models.ThermalOutputMode) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetThermalOutputMode, commands.EncodeSetThermalOutputMode(mode))
	if err != nil {
		return err
	}
	return commands.DecodeSetThermalOutputModeAck(data)
}

// GetSingleTempFrame requests a single raw temperature frame (0x35).
func (c *Client) GetSingleTempFrame(ctx context.Context) error {
	data, err := c.sendCommand(ctx, protocol.CmdGetSingleTempFrame, commands.EncodeGetSingleTempFrame())
	if err != nil {
		return err
	}
	return commands.DecodeSingleTempFrameAck(data)
}

// GetThermalGain queries thermal gain mode (low/high) (0x37).
func (c *Client) GetThermalGain(ctx context.Context) (models.ThermalGain, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestThermalGain, commands.EncodeGetThermalGain())
	if err != nil {
		return 0, err
	}
	return commands.DecodeThermalGain(data)
}

// SetThermalGain sets thermal gain sensitivity (0x38).
func (c *Client) SetThermalGain(ctx context.Context, gain models.ThermalGain) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetThermalGain, commands.EncodeSetThermalGain(gain))
	if err != nil {
		return err
	}
	return commands.DecodeSetThermalGainAck(data)
}

// GetEnvCorrectionParams queries environmental thermal compensation parameters (0x39).
func (c *Client) GetEnvCorrectionParams(ctx context.Context) (models.EnvCorrectionParams, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestEnvCorrectionParams, commands.EncodeGetEnvCorrectionParams())
	if err != nil {
		return models.EnvCorrectionParams{}, err
	}
	return commands.DecodeEnvCorrectionParams(data)
}

// SetEnvCorrectionParams configures environmental thermal compensation parameters (0x3A).
func (c *Client) SetEnvCorrectionParams(ctx context.Context, p models.EnvCorrectionParams) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetEnvCorrectionParams, commands.EncodeSetEnvCorrectionParams(p))
	if err != nil {
		return err
	}
	return commands.DecodeSetEnvCorrectionParamsAck(data)
}

// GetEnvCorrectionSwitch queries environmental correction enabled status (0x3B).
func (c *Client) GetEnvCorrectionSwitch(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdRequestEnvCorrectionSwitch, commands.EncodeGetEnvCorrectionSwitch())
	if err != nil {
		return false, err
	}
	return commands.DecodeEnvCorrectionSwitch(data)
}

// SetEnvCorrectionSwitch enables or disables environmental thermal compensation (0x3C).
func (c *Client) SetEnvCorrectionSwitch(ctx context.Context, on bool) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetEnvCorrectionSwitch, commands.EncodeSetEnvCorrectionSwitch(on))
	if err != nil {
		return err
	}
	return commands.DecodeSetEnvCorrectionSwitchAck(data)
}

// GetIRThreshMapState queries IR isotherm threshold map display state (0x42).
func (c *Client) GetIRThreshMapState(ctx context.Context) (bool, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetIRThreshMapSta, commands.EncodeGetIRThreshMapState())
	if err != nil {
		return false, err
	}
	return commands.DecodeIRThreshMapState(data)
}

// SetIRThreshMapState enables or disables IR isotherm threshold map (0x43).
func (c *Client) SetIRThreshMapState(ctx context.Context, on bool) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetIRThreshMapSta, commands.EncodeSetIRThreshMapState(on))
	if err != nil {
		return err
	}
	return commands.DecodeSetIRThreshMapStateAck(data)
}

// GetIRThreshParams queries IR isotherm temperature threshold regions (0x44).
func (c *Client) GetIRThreshParams(ctx context.Context) (models.IRThreshParams, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetIRThreshParam, commands.EncodeGetIRThreshParams())
	if err != nil {
		return models.IRThreshParams{}, err
	}
	return commands.DecodeIRThreshParams(data)
}

// SetIRThreshParams configures IR isotherm temperature threshold regions (0x45).
func (c *Client) SetIRThreshParams(ctx context.Context, p models.IRThreshParams) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetIRThreshParam, commands.EncodeSetIRThreshParams(p))
	if err != nil {
		return err
	}
	return commands.DecodeSetIRThreshParamsAck(data)
}

// GetIRThreshPrecision queries IR threshold precision level (0x46).
func (c *Client) GetIRThreshPrecision(ctx context.Context) (models.IRThreshPrecision, error) {
	data, err := c.sendCommand(ctx, protocol.CmdGetIRThreshPrecision, commands.EncodeGetIRThreshPrecision())
	if err != nil {
		return 0, err
	}
	return commands.DecodeIRThreshPrecision(data)
}

// SetIRThreshPrecision sets IR threshold precision level (0x47).
func (c *Client) SetIRThreshPrecision(ctx context.Context, precision models.IRThreshPrecision) error {
	data, err := c.sendCommand(ctx, protocol.CmdSetIRThreshPrecision, commands.EncodeSetIRThreshPrecision(precision))
	if err != nil {
		return err
	}
	return commands.DecodeSetIRThreshPrecisionAck(data)
}

// ManualThermalShutter triggers manual calibration shutter for thermal core (0x4F).
func (c *Client) ManualThermalShutter(ctx context.Context) error {
	data, err := c.sendCommand(ctx, protocol.CmdManualThermalShutter, commands.EncodeManualThermalShutter())
	if err != nil {
		return err
	}
	return commands.DecodeManualThermalShutterAck(data)
}

// -----------------------------------------------------------------------------
// RTSP Streaming Convenience
// -----------------------------------------------------------------------------

// CreateStream constructs a stream.Stream receiver configured for this camera's RTSP endpoint.
func (c *Client) CreateStream(streamType models.StreamType, gen models.CameraGeneration) *stream.Stream {
	host := protocol.DefaultIP
	if ipGetter, ok := c.transport.(interface{ IP() string }); ok {
		if ip := ipGetter.IP(); ip != "" {
			host = ip
		}
	}
	rtspURL := stream.BuildRTSPURL(host, streamType, gen)
	return stream.New(stream.Config{
		RTSPURL: rtspURL,
	})
}

