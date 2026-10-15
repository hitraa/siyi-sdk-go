package commands_test

import (
	"net"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/commands"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
)

func TestSystemCommands(t *testing.T) {
	// Heartbeat
	if hb := commands.EncodeHeartbeat(); len(hb) != 1 {
		t.Errorf("EncodeHeartbeat len = %d", len(hb))
	}

	// Firmware Version
	fwPayload := []byte{
		0x03, 0x02, 0x03, 0x06, // camera
		0x04, 0x03, 0x02, 0x01, // gimbal
		0x08, 0x07, 0x06, 0x05, // zoom
	}
	fw, err := commands.DecodeFirmwareVersion(fwPayload)
	if err != nil {
		t.Fatalf("DecodeFirmwareVersion error: %v", err)
	}
	if fw.Camera != 0x06030203 || fw.Gimbal != 0x01020304 || fw.Zoom != 0x05060708 {
		t.Errorf("unexpected FW: %+v", fw)
	}

	// Hardware ID
	hwPayload := []byte("731234567890")
	hw, err := commands.DecodeHardwareID(hwPayload)
	if err != nil || len(hw.Raw) != 12 {
		t.Errorf("DecodeHardwareID error: %v", err)
	}

	// Set UTC Time
	utcPayload := commands.EncodeSetUTCTime(1609459200000000)
	if len(utcPayload) != 8 {
		t.Errorf("EncodeSetUTCTime len = %d", len(utcPayload))
	}
	ok, err := commands.DecodeSetUTCTimeAck([]byte{0x01})
	if err != nil || !ok {
		t.Errorf("DecodeSetUTCTimeAck failed: %v", err)
	}
	if _, err := commands.DecodeSetUTCTimeAck([]byte{0x00}); err == nil {
		t.Errorf("expected error on sta=0")
	}

	// Soft Reboot
	rebootPayload := commands.EncodeSoftReboot(true, false)
	if rebootPayload[0] != 0x80 {
		t.Errorf("EncodeSoftReboot = 0x%02X, want 0x80", rebootPayload[0])
	}
	cam, gim, err := commands.DecodeSoftRebootAck([]byte{0x80})
	if err != nil || !cam || gim {
		t.Errorf("DecodeSoftRebootAck = (%v,%v), want (true,false)", cam, gim)
	}

	// IP Config
	ipCfg := models.IPConfig{
		IP:      net.ParseIP("192.168.144.25"),
		Mask:    net.ParseIP("255.255.255.0"),
		Gateway: net.ParseIP("192.168.144.1"),
	}
	encodedIP, err := commands.EncodeSetIP(ipCfg)
	if err != nil || len(encodedIP) != 12 {
		t.Fatalf("EncodeSetIP error: %v", err)
	}
	decodedIP, err := commands.DecodeGetIP(encodedIP)
	if err != nil || !decodedIP.IP.Equal(ipCfg.IP) {
		t.Errorf("DecodeGetIP mismatch: %+v", decodedIP)
	}
}

func TestAttitudeCommands(t *testing.T) {
	// Gimbal Attitude (0x0D): 6 x int16 LE (division by 10)
	// Yaw=100 (10.0 deg), Pitch=-200 (-20.0 deg), Roll=50 (5.0 deg)
	// YawRate=15 (1.5 dps), PitchRate=-30 (-3.0 dps), RollRate=0 (0 dps)
	payload := []byte{
		0x64, 0x00, // 100
		0x38, 0xFF, // -200
		0x32, 0x00, // 50
		0x0F, 0x00, // 15
		0xE2, 0xFF, // -30
		0x00, 0x00, // 0
	}
	att, err := commands.DecodeGimbalAttitude(payload)
	if err != nil {
		t.Fatalf("DecodeGimbalAttitude error: %v", err)
	}
	if att.YawDeg != 10.0 || att.PitchDeg != -20.0 || att.RollDeg != 5.0 {
		t.Errorf("Angles mismatch: %+v", att)
	}
	if att.YawRateDPS != 1.5 || att.PitchRateDPS != -3.0 || att.RollRateDPS != 0.0 {
		t.Errorf("Rates mismatch: %+v", att)
	}

	// Aircraft Attitude
	airAtt := models.AircraftAttitude{
		TimeBootMS: 12345,
		RollRad:    0.1,
		PitchRad:   -0.2,
		YawRad:     1.5,
	}
	encodedAir := commands.EncodeAircraftAttitude(airAtt)
	if len(encodedAir) != 28 {
		t.Errorf("EncodeAircraftAttitude len = %d", len(encodedAir))
	}

	// Stream control
	fcBytes, err := commands.EncodeFCStream(models.FCDataAttitude, models.DataStreamFreq10Hz)
	if err != nil || len(fcBytes) != 2 {
		t.Errorf("EncodeFCStream error: %v", err)
	}
	gimBytes, err := commands.EncodeGimbalStream(models.GimbalDataAttitude, models.DataStreamFreq50Hz)
	if err != nil || len(gimBytes) != 2 {
		t.Errorf("EncodeGimbalStream error: %v", err)
	}

	// Raw GPS
	gps := models.RawGPS{
		TimeBootMS: 5000,
		LatE7:      377749000,
		LonE7:      -1224194000,
	}
	gpsBytes := commands.EncodeRawGPS(gps)
	if len(gpsBytes) != 32 {
		t.Errorf("EncodeRawGPS len = %d", len(gpsBytes))
	}
}

func TestGimbalCommands(t *testing.T) {
	// Rotation
	rot, err := commands.EncodeRotation(50, -50)
	if err != nil || len(rot) != 2 {
		t.Errorf("EncodeRotation failed: %v", err)
	}
	if _, err := commands.EncodeRotation(101, 0); err == nil {
		t.Errorf("expected error for yaw out of bounds")
	}

	// One-key centering
	center, err := commands.EncodeOneKeyCentering(models.CenteringActionOneKeyCenter)
	if err != nil || len(center) != 1 {
		t.Errorf("EncodeOneKeyCentering failed: %v", err)
	}

	// Set Attitude
	setAtt := commands.EncodeSetAttitude(15.5, -30.2)
	if len(setAtt) != 4 {
		t.Errorf("EncodeSetAttitude len = %d", len(setAtt))
	}
	ackPayload := []byte{
		0x9B, 0x00, // 155 -> 15.5 deg
		0xD2, 0xFE, // -302 -> -30.2 deg
		0x00, 0x00, // 0.0 deg
	}
	ack, err := commands.DecodeSetAttitudeAck(ackPayload)
	if err != nil {
		t.Fatalf("DecodeSetAttitudeAck error: %v", err)
	}
	if ack.YawDeg != 15.5 || ack.PitchDeg != -30.2 || ack.RollDeg != 0.0 {
		t.Errorf("SetAttitudeAck mismatch: %+v", ack)
	}

	// Single Axis
	single, err := commands.EncodeSingleAxis(45.0, 0)
	if err != nil || len(single) != 3 {
		t.Errorf("EncodeSingleAxis failed: %v", err)
	}
}

func TestCameraCommands(t *testing.T) {
	// 8-byte camera system info
	payload8 := []byte{0, 1, 0, 1, 1, 1, 0, 1}
	info8, err := commands.DecodeCameraSystemInfo(payload8)
	if err != nil || info8.HDRSta != 1 || info8.ZoomLinkage != 1 {
		t.Errorf("DecodeCameraSystemInfo 8-byte failed: %v", err)
	}

	// 7-byte camera system info (e.g. A8 Mini without optical zoom)
	payload7 := []byte{0, 1, 0, 1, 1, 1, 0}
	info7, err := commands.DecodeCameraSystemInfo(payload7)
	if err != nil || info7.HDRSta != 1 || info7.ZoomLinkage != 0 {
		t.Errorf("DecodeCameraSystemInfo 7-byte failed: %v", err)
	}

	// Capture photo
	capPhoto := commands.EncodeCapture(models.CaptureFuncPhoto)
	if len(capPhoto) != 1 || capPhoto[0] != 0 {
		t.Errorf("EncodeCapture failed: %v", capPhoto)
	}

	// Encoding Params
	encCfg := models.EncodingParams{
		StreamType:   models.StreamTypeMain,
		EncType:      models.VideoEncH264,
		ResolutionW:  1920,
		ResolutionH:  1080,
		BitrateKbps:  4000,
		FrameRateFPS: 30,
	}
	encBytes, err := commands.EncodeSetEncodingParams(encCfg)
	if err != nil || len(encBytes) != 9 {
		t.Fatalf("EncodeSetEncodingParams failed: %v", err)
	}

	// Invalid resolution
	badEnc := encCfg
	badEnc.ResolutionW = 800
	badEnc.ResolutionH = 600
	if _, err := commands.EncodeSetEncodingParams(badEnc); err == nil {
		t.Errorf("expected error for unsupported resolution")
	}

	// Format SD
	sd := commands.EncodeFormatSD()
	if len(sd) != 1 || sd[0] != 1 {
		t.Errorf("EncodeFormatSD unexpected: %v", sd)
	}
	ok, err := commands.DecodeFormatSDAck([]byte{0x01})
	if err != nil || !ok {
		t.Errorf("DecodeFormatSDAck failed: %v", err)
	}
}

func TestZoomAndFocusCommands(t *testing.T) {
	// Zoom
	manZoom, err := commands.EncodeManualZoom(1)
	if err != nil || manZoom[0] != 1 {
		t.Errorf("EncodeManualZoom failed: %v", err)
	}

	zoomMag, err := commands.DecodeManualZoomAck([]byte{0x1E, 0x00}) // 30 -> 3.0x
	if err != nil || zoomMag != 3.0 {
		t.Errorf("DecodeManualZoomAck = %f, want 3.0", zoomMag)
	}

	absZoom, err := commands.EncodeAbsoluteZoom(15.5)
	if err != nil || absZoom[0] != 15 || absZoom[1] != 5 {
		t.Errorf("EncodeAbsoluteZoom failed: %v", absZoom)
	}

	// Focus
	autoFocus := commands.EncodeAutoFocus(1, 640, 360)
	if len(autoFocus) != 5 {
		t.Errorf("EncodeAutoFocus len = %d", len(autoFocus))
	}
	manFocus, err := commands.EncodeManualFocus(-1)
	if err != nil || manFocus[0] != 0xFF { // -1 in two's complement uint8
		t.Errorf("EncodeManualFocus failed: %v", err)
	}
}

func TestLaserCommands(t *testing.T) {
	// Laser distance: 100 decimeters = 10.0 meters
	distPayload := []byte{0x64, 0x00}
	dist, err := commands.DecodeLaserDistance(distPayload)
	if err != nil || dist.DistanceM == nil || *dist.DistanceM != 10.0 {
		t.Errorf("DecodeLaserDistance failed: %+v", dist)
	}

	// Out of range (raw < 50)
	outOfRangePayload := []byte{0x20, 0x00} // 32 < 50
	outOfRange, err := commands.DecodeLaserDistance(outOfRangePayload)
	if err != nil || outOfRange.DistanceM != nil {
		t.Errorf("expected out of range nil, got %+v", outOfRange)
	}

	// Target lat lon
	latLonPayload := []byte{
		0xD0, 0x86, 0x07, 0x00, // lon
		0xA0, 0x86, 0x01, 0x00, // lat
	}
	coords, err := commands.DecodeLaserTargetLatLon(latLonPayload)
	if err != nil {
		t.Fatalf("DecodeLaserTargetLatLon error: %v", err)
	}
	if coords.LonE7 != 493264 || coords.LatE7 != 100000 {
		t.Errorf("LaserTargetLatLon mismatch: %+v", coords)
	}
}

func TestRCAndVideoCommands(t *testing.T) {
	var rc models.RCChannels
	for i := 0; i < 18; i++ {
		rc.Channels[i] = uint16(1000 + i*50)
	}
	rc.ChanCount = 18
	rc.RSSI = 200

	encodedRC := commands.EncodeRCChannels(rc)
	if len(encodedRC) != 38 {
		t.Fatalf("EncodeRCChannels len = %d", len(encodedRC))
	}
	decodedRC, err := commands.DecodeRCChannels(encodedRC)
	if err != nil || decodedRC.ChanCount != 18 || decodedRC.RSSI != 200 {
		t.Errorf("DecodeRCChannels mismatch: %+v", decodedRC)
	}

	// Video Stitching
	vMode, err := commands.DecodeVideoStitchingMode([]byte{0x02})
	if err != nil || vMode != models.VideoStitchingMode2 {
		t.Errorf("DecodeVideoStitchingMode failed: %v", err)
	}
}

func TestAIAndDebugCommands(t *testing.T) {
	// AI Tracking (0x50): 10 bytes
	aiPayload := []byte{
		0x00, 0x01, // x = 256
		0x00, 0x02, // y = 512
		0x64, 0x00, // w = 100
		0xC8, 0x00, // h = 200
		0x00,       // target_id = human
		0x00,       // status = normal_ai
	}
	aiTarget, err := commands.DecodeAITracking(aiPayload)
	if err != nil {
		t.Fatalf("DecodeAITracking error: %v", err)
	}
	if aiTarget.X != 256 || aiTarget.Y != 512 || aiTarget.W != 100 || aiTarget.H != 200 {
		t.Errorf("AI bounding box mismatch: %+v", aiTarget)
	}

	// Weak control threshold
	thresh := models.WeakControlThreshold{
		Limit:        2.5,
		Voltage:      3.2,
		AngularError: 15.0,
	}
	threshBytes := commands.EncodeSetWeakThreshold(thresh)
	decodedThresh, err := commands.DecodeWeakThreshold(threshBytes)
	if err != nil || decodedThresh.Limit != 2.5 || decodedThresh.Voltage != 3.2 || decodedThresh.AngularError != 15.0 {
		t.Errorf("WeakThreshold mismatch: %+v", decodedThresh)
	}
}

func TestThermalCommands(t *testing.T) {
	// Point temp: 2500 -> 25.0 C
	ptPayload := []byte{
		0x64, 0x00, // x = 100
		0xC8, 0x00, // y = 200
		0xC4, 0x09, // 2500 -> 25.0 C
	}
	pt, err := commands.DecodeTempAtPoint(ptPayload)
	if err != nil || pt.TemperatureC != 25.0 || pt.X != 100 || pt.Y != 200 {
		t.Errorf("DecodeTempAtPoint failed: %+v", pt)
	}

	// Env Correction
	envCfg := models.EnvCorrectionParams{
		DistanceM:     5.5,
		EmissivityPct: 98.0,
		HumidityPct:   50.0,
		AmbientC:      22.5,
		ReflectiveC:   20.0,
	}
	envBytes := commands.EncodeSetEnvCorrectionParams(envCfg)
	decodedEnv, err := commands.DecodeEnvCorrectionParams(envBytes)
	if err != nil || decodedEnv.DistanceM != 5.5 || decodedEnv.EmissivityPct != 98.0 {
		t.Errorf("DecodeEnvCorrectionParams mismatch: %+v", decodedEnv)
	}

	// Pseudo color
	pcBytes := commands.EncodeSetPseudoColor(models.PseudoColorIronbow)
	if len(pcBytes) != 1 || pcBytes[0] != byte(models.PseudoColorIronbow) {
		t.Errorf("EncodeSetPseudoColor failed: %v", pcBytes)
	}
	err = commands.DecodeSetPseudoColorAck([]byte{0x01})
	if err != nil {
		t.Errorf("DecodeSetPseudoColorAck error: %v", err)
	}

	// IR threshold params
	params := models.IRThreshParams{
		Region1: models.IRThreshRegion{Switch: 1, TempMin: 200, TempMax: 800, ColorR: 255, ColorG: 0, ColorB: 0},
		Region2: models.IRThreshRegion{Switch: 0, TempMin: 100, TempMax: 500, ColorR: 0, ColorG: 255, ColorB: 0},
		Region3: models.IRThreshRegion{Switch: 1, TempMin: 300, TempMax: 900, ColorR: 0, ColorG: 0, ColorB: 255},
	}
	threshParamsBytes := commands.EncodeSetIRThreshParams(params)
	if len(threshParamsBytes) != 24 {
		t.Fatalf("EncodeSetIRThreshParams len = %d", len(threshParamsBytes))
	}
	decodedParams, err := commands.DecodeIRThreshParams(threshParamsBytes)
	if err != nil || decodedParams.Region1.ColorR != 255 || decodedParams.Region3.ColorB != 255 {
		t.Errorf("DecodeIRThreshParams mismatch: %+v", decodedParams)
	}
}
