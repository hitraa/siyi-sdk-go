package mavlink

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

const (
	// MAVLink v2 start of frame marker.
	MagicV2 = 0xFD

	// Default MAVLink UDP port used by SIYI and ArduPilot/PX4.
	DefaultPort = 14550

	// Common Message IDs
	MsgHeartbeat                  uint32 = 0
	MsgCommandLong                uint32 = 76
	MsgCommandAck                 uint32 = 77
	MsgCameraInformation          uint32 = 259
	MsgCameraCaptureStatus        uint32 = 262
	MsgGimbalDeviceSetAttitude    uint32 = 284
	MsgGimbalDeviceAttitudeStatus uint32 = 285

	// Common MAV_CMD IDs
	MavCmdDoMountControl     uint16 = 205
	MavCmdDoGimbalManagerPitchYaw uint16 = 1000
	MavCmdImageStartCapture  uint16 = 2000
	MavCmdImageStopCapture   uint16 = 2001
	MavCmdVideoStartCapture  uint16 = 2500
	MavCmdVideoStopCapture   uint16 = 2501
	MavCmdSetCameraZoom      uint16 = 531
	MavCmdSetCameraFocus     uint16 = 532
)

// CRCExtra lookup for standard messages
var crcExtraMap = map[uint32]byte{
	MsgHeartbeat:                  50,
	MsgCommandLong:                152,
	MsgCommandAck:                 143,
	MsgCameraInformation:          92,
	MsgCameraCaptureStatus:        12,
	MsgGimbalDeviceSetAttitude:    99,
	MsgGimbalDeviceAttitudeStatus: 137,
}

// Frame represents a MAVLink v2 packet.
type Frame struct {
	IncompatFlags byte
	CompatFlags   byte
	Sequence      byte
	SystemID      byte
	ComponentID   byte
	MessageID     uint32
	Payload       []byte
	Checksum      uint16
}

// Encode serializes a MAVLink v2 frame into wire bytes.
func (f *Frame) Encode() []byte {
	payLen := len(f.Payload)
	buf := make([]byte, 10+payLen+2)
	buf[0] = MagicV2
	buf[1] = byte(payLen)
	buf[2] = f.IncompatFlags
	buf[3] = f.CompatFlags
	buf[4] = f.Sequence
	buf[5] = f.SystemID
	buf[6] = f.ComponentID
	buf[7] = byte(f.MessageID & 0xFF)
	buf[8] = byte((f.MessageID >> 8) & 0xFF)
	buf[9] = byte((f.MessageID >> 16) & 0xFF)

	copy(buf[10:], f.Payload)

	crcExtra := crcExtraMap[f.MessageID]
	crc := CalculateCRC(buf[1:10+payLen], crcExtra)
	f.Checksum = crc

	binary.LittleEndian.PutUint16(buf[10+payLen:], crc)
	return buf
}

// DecodeFrame attempts to parse a MAVLink v2 frame from wire bytes.
func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < 12 { // Min header (10) + checksum (2)
		return nil, errors.New("packet too short for MAVLink v2")
	}
	if data[0] != MagicV2 {
		return nil, fmt.Errorf("invalid MAVLink magic 0x%02X (expected 0x%02X)", data[0], MagicV2)
	}

	payLen := int(data[1])
	totalExpected := 10 + payLen + 2
	if len(data) < totalExpected {
		return nil, fmt.Errorf("truncated MAVLink packet: have %d bytes, need %d", len(data), totalExpected)
	}

	msgID := uint32(data[7]) | (uint32(data[8]) << 8) | (uint32(data[9]) << 16)
	crcExtra := crcExtraMap[msgID]
	calcCRC := CalculateCRC(data[1:10+payLen], crcExtra)
	rxCRC := binary.LittleEndian.Uint16(data[10+payLen : 10+payLen+2])

	if calcCRC != rxCRC {
		return nil, fmt.Errorf("MAVLink CRC mismatch: got 0x%04X, calculated 0x%04X", rxCRC, calcCRC)
	}

	payload := make([]byte, payLen)
	copy(payload, data[10:10+payLen])

	return &Frame{
		IncompatFlags: data[2],
		CompatFlags:   data[3],
		Sequence:      data[4],
		SystemID:      data[5],
		ComponentID:   data[6],
		MessageID:     msgID,
		Payload:       payload,
		Checksum:      rxCRC,
	}, nil
}

// GimbalAttitudeStatus represents decoded MAVLink MSG #285 (GIMBAL_DEVICE_ATTITUDE_STATUS).
type GimbalAttitudeStatus struct {
	TimeBootMS   uint32
	Flags        uint16
	Q            [4]float32 // Quaternion [w, x, y, z]
	AngularVelX  float32    // Roll angular velocity [rad/s]
	AngularVelY  float32    // Pitch angular velocity [rad/s]
	AngularVelZ  float32    // Yaw angular velocity [rad/s]
	FailureFlags uint32
	YawDeg       float64
	PitchDeg     float64
	RollDeg      float64
}

// DecodeGimbalAttitudeStatus decodes payload for MSG #285.
func DecodeGimbalAttitudeStatus(payload []byte) (*GimbalAttitudeStatus, error) {
	if len(payload) < 40 {
		return nil, fmt.Errorf("payload too short for GIMBAL_DEVICE_ATTITUDE_STATUS: %d bytes", len(payload))
	}

	timeBoot := binary.LittleEndian.Uint32(payload[0:4])
	flags := binary.LittleEndian.Uint16(payload[4:6])

	var q [4]float32
	for i := 0; i < 4; i++ {
		bits := binary.LittleEndian.Uint32(payload[6+i*4 : 10+i*4])
		q[i] = math.Float32frombits(bits)
	}

	vxBits := binary.LittleEndian.Uint32(payload[22:26])
	vyBits := binary.LittleEndian.Uint32(payload[26:30])
	vzBits := binary.LittleEndian.Uint32(payload[30:34])

	failFlags := binary.LittleEndian.Uint32(payload[34:38])

	// Convert quaternion to Euler angles (degrees)
	w, x, y, z := float64(q[0]), float64(q[1]), float64(q[2]), float64(q[3])
	roll := math.Atan2(2*(w*x+y*z), 1-2*(x*x+y*y)) * 180.0 / math.Pi
	pitch := math.Asin(math.Max(-1.0, math.Min(1.0, 2*(w*y-z*x)))) * 180.0 / math.Pi
	yaw := math.Atan2(2*(w*z+x*y), 1-2*(y*y+z*z)) * 180.0 / math.Pi

	return &GimbalAttitudeStatus{
		TimeBootMS:   timeBoot,
		Flags:        flags,
		Q:            q,
		AngularVelX:  math.Float32frombits(vxBits),
		AngularVelY:  math.Float32frombits(vyBits),
		AngularVelZ:  math.Float32frombits(vzBits),
		FailureFlags: failFlags,
		YawDeg:       yaw,
		PitchDeg:     pitch,
		RollDeg:      roll,
	}, nil
}

// EncodeGimbalDeviceSetAttitude builds payload for MSG #284 (GIMBAL_DEVICE_SET_ATTITUDE).
func EncodeGimbalDeviceSetAttitude(targetSystem, targetComponent byte, flags uint16, q [4]float32, angVelX, angVelY, angVelZ float32) []byte {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint16(buf[0:2], flags)

	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(buf[2+i*4:6+i*4], math.Float32bits(q[i]))
	}

	binary.LittleEndian.PutUint32(buf[18:22], math.Float32bits(angVelX))
	binary.LittleEndian.PutUint32(buf[22:26], math.Float32bits(angVelY))
	binary.LittleEndian.PutUint32(buf[26:30], math.Float32bits(angVelZ))

	buf[30] = targetSystem
	buf[31] = targetComponent
	return buf
}

// EulerToQuaternion converts roll, pitch, yaw in degrees to quaternion [w, x, y, z].
func EulerToQuaternion(rollDeg, pitchDeg, yawDeg float64) [4]float32 {
	r := rollDeg * math.Pi / 180.0 * 0.5
	p := pitchDeg * math.Pi / 180.0 * 0.5
	y := yawDeg * math.Pi / 180.0 * 0.5

	cr := math.Cos(r)
	sr := math.Sin(r)
	cp := math.Cos(p)
	sp := math.Sin(p)
	cy := math.Cos(y)
	sy := math.Sin(y)

	return [4]float32{
		float32(cr*cp*cy + sr*sp*sy), // w
		float32(sr*cp*cy - cr*sp*sy), // x
		float32(cr*sp*cy + sr*cp*sy), // y
		float32(cr*cp*sy - sr*sp*cy), // z
	}
}

// Client is a MAVLink v2 UDP client for SIYI cameras.
type Client struct {
	remoteAddr *net.UDPAddr
	conn       *net.UDPConn
	systemID   byte
	compID     byte
	seq        byte
	mu         sync.Mutex
	closed     chan struct{}
	running    bool
	attitudeCb func(att *GimbalAttitudeStatus)
}

// NewClient creates a new MAVLink client pointing to camera IP on port 14550.
func NewClient(ip string, port ...int) (*Client, error) {
	if ip == "" {
		ip = "192.168.144.25"
	}
	p := DefaultPort
	if len(port) > 0 && port[0] > 0 {
		p = port[0]
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, p))
	if err != nil {
		return nil, fmt.Errorf("resolve MAVLink remote address: %w", err)
	}

	return &Client{
		remoteAddr: addr,
		systemID:   255, // Typical Ground Station / Mission Planner System ID
		compID:     190, // Typical Gimbal Component ID
		closed:     make(chan struct{}),
	}, nil
}

// Connect establishes the UDP endpoint and begins listening for incoming telemetry.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	conn, err := net.DialUDP("udp", nil, c.remoteAddr)
	if err != nil {
		return fmt.Errorf("dial MAVLink UDP: %w", err)
	}

	c.conn = conn
	c.running = true
	go c.readLoop()
	return nil
}

func (c *Client) readLoop() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-c.closed:
			return
		default:
		}

		_ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			frame, err := DecodeFrame(buf[:n])
			if err == nil && frame != nil {
				if frame.MessageID == MsgGimbalDeviceAttitudeStatus {
					att, err := DecodeGimbalAttitudeStatus(frame.Payload)
					if err == nil {
						c.mu.Lock()
						cb := c.attitudeCb
						c.mu.Unlock()
						if cb != nil {
							cb(att)
						}
					}
				}
			}
		}
	}
}

// OnAttitude registers a listener for incoming MAVLink gimbal attitude telemetry.
func (c *Client) OnAttitude(cb func(att *GimbalAttitudeStatus)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attitudeCb = cb
}

// SendAttitude transmits a target attitude in Euler angles (yaw, pitch) using MSG #284.
func (c *Client) SendAttitude(ctx context.Context, yawDeg, pitchDeg float64) error {
	q := EulerToQuaternion(0, pitchDeg, yawDeg)
	payload := EncodeGimbalDeviceSetAttitude(1, 154, 0, q, 0, 0, 0)

	c.mu.Lock()
	c.seq++
	seq := c.seq
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errors.New("client not connected")
	}

	frame := &Frame{
		Sequence:    seq,
		SystemID:    c.systemID,
		ComponentID: c.compID,
		MessageID:   MsgGimbalDeviceSetAttitude,
		Payload:     payload,
	}

	_, err := conn.Write(frame.Encode())
	return err
}

// Close disconnects the MAVLink client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return nil
	}
	c.running = false
	close(c.closed)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
