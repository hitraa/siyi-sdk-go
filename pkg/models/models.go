package models

import (
	"fmt"
	"net"
	"strconv"
)

// FirmwareVersion carries camera, gimbal, and zoom packed firmware version words.
type FirmwareVersion struct {
	Camera uint32 `json:"camera"`
	Gimbal uint32 `json:"gimbal"`
	Zoom   uint32 `json:"zoom"`
}

// DecodeWord decodes a packed 32-bit firmware version word into major, minor, patch.
// Per spec: high byte is ignored, low 3 bytes (LSB to MSB) are patch, minor, major.
func DecodeWord(word uint32) (major, minor, patch int) {
	patch = int(word & 0xFF)
	minor = int((word >> 8) & 0xFF)
	major = int((word >> 16) & 0xFF)
	return major, minor, patch
}

// FormatWord formats a packed firmware word into a semantic version string (e.g., "v3.3.0"), or "n/a" if 0.
func FormatWord(word uint32) string {
	if word == 0 {
		return "n/a"
	}
	major, minor, patch := DecodeWord(word)
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

// HardwareID represents the 12-byte hardware identifier string.
type HardwareID struct {
	Raw []byte `json:"raw"`
}

// ProductID extracts the product model code from the first two ASCII hex bytes.
func (h HardwareID) ProductID() (ProductID, error) {
	if len(h.Raw) < 2 {
		return 0, fmt.Errorf("hardware ID too short: %d bytes", len(h.Raw))
	}
	code, err := strconv.ParseUint(string(h.Raw[0:2]), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid product ID hex '%s': %w", string(h.Raw[0:2]), err)
	}
	return ProductID(code), nil
}

// CameraSystemInfo carries operational state of the camera.
type CameraSystemInfo struct {
	ReservedA         uint8             `json:"reserved_a"`
	HDRSta            uint8             `json:"hdr_sta"`
	ReservedB         uint8             `json:"reserved_b"`
	RecordSta         RecordingState    `json:"record_sta"`
	GimbalMotionMode  GimbalMotionMode  `json:"gimbal_motion_mode"`
	GimbalMountingDir MountingDirection `json:"gimbal_mounting_dir"`
	VideoHDMICVBS     HDMICVBSOutput    `json:"video_hdmi_or_cvbs"`
	ZoomLinkage       uint8             `json:"zoom_linkage"`
}

// GimbalAttitude carries 3-axis Euler angles and rates (0x0D response).
type GimbalAttitude struct {
	YawDeg       float64 `json:"yaw_deg"`
	PitchDeg     float64 `json:"pitch_deg"`
	RollDeg      float64 `json:"roll_deg"`
	YawRateDPS   float64 `json:"yaw_rate_dps"`
	PitchRateDPS float64 `json:"pitch_rate_dps"`
	RollRateDPS  float64 `json:"roll_rate_dps"`
}

// SetAttitudeAck is the response from attitude target commands (0x0E / 0x41).
type SetAttitudeAck struct {
	YawDeg   float64 `json:"yaw_deg"`
	PitchDeg float64 `json:"pitch_deg"`
	RollDeg  float64 `json:"roll_deg"`
}

// AircraftAttitude represents vehicle orientation sent to gimbal (0x22).
type AircraftAttitude struct {
	TimeBootMS uint32  `json:"time_boot_ms"`
	RollRad    float32 `json:"roll_rad"`
	PitchRad   float32 `json:"pitch_rad"`
	YawRad     float32 `json:"yaw_rad"`
	RollSpeed  float32 `json:"rollspeed"`
	PitchSpeed float32 `json:"pitchspeed"`
	YawSpeed   float32 `json:"yawspeed"`
}

// RCChannels carries 18 RC channel values in microseconds (0x23).
type RCChannels struct {
	Channels   [18]uint16 `json:"channels"`
	ChanCount  uint8      `json:"chancount"`
	RSSI       uint8      `json:"rssi"`
}

// MagneticEncoderAngles carries raw magnetic encoder angles in degrees (0x26).
type MagneticEncoderAngles struct {
	Yaw   float64 `json:"yaw"`
	Pitch float64 `json:"pitch"`
	Roll  float64 `json:"roll"`
}

// MotorVoltage carries motor drive voltages in Volts (0x2A).
type MotorVoltage struct {
	Yaw   float64 `json:"yaw"`
	Pitch float64 `json:"pitch"`
	Roll  float64 `json:"roll"`
}

// WeakControlThreshold parameters (0x28).
type WeakControlThreshold struct {
	Limit        float64 `json:"limit"`
	Voltage      float64 `json:"voltage"`
	AngularError float64 `json:"angular_error"`
}

// EncodingParams carries video compression and stream output configuration (0x20).
type EncodingParams struct {
	StreamType   StreamType   `json:"stream_type"`
	EncType      VideoEncType `json:"enc_type"`
	ResolutionW  uint16       `json:"resolution_w"`
	ResolutionH  uint16       `json:"resolution_h"`
	BitrateKbps  uint16       `json:"bitrate_kbps"`
	FrameRateFPS uint8        `json:"frame_rate"`
}

// TempPoint represents thermal measurement at an (X, Y) point (0x12).
type TempPoint struct {
	X            uint16  `json:"x"`
	Y            uint16  `json:"y"`
	TemperatureC float64 `json:"temperature_c"`
}

// TempRegion represents thermal measurement across a bounding rectangle (0x13).
type TempRegion struct {
	StartX uint16  `json:"start_x"`
	StartY uint16  `json:"start_y"`
	EndX   uint16  `json:"end_x"`
	EndY   uint16  `json:"end_y"`
	MaxC   float64 `json:"max_c"`
	MinC   float64 `json:"min_c"`
	MaxX   uint16  `json:"max_x"`
	MaxY   uint16  `json:"max_y"`
	MinX   uint16  `json:"min_x"`
	MinY   uint16  `json:"min_y"`
}

// TempGlobal represents full-frame thermal measurement (0x14).
type TempGlobal struct {
	MaxC float64 `json:"max_c"`
	MinC float64 `json:"min_c"`
	MaxX uint16  `json:"max_x"`
	MaxY uint16  `json:"max_y"`
	MinX uint16  `json:"min_x"`
	MinY uint16  `json:"min_y"`
}

// EnvCorrectionParams environmental compensation parameters (0x39/0x3A).
type EnvCorrectionParams struct {
	DistanceM     float64 `json:"distance_m"`
	EmissivityPct float64 `json:"emissivity_pct"`
	HumidityPct   float64 `json:"humidity_pct"`
	AmbientC      float64 `json:"ambient_c"`
	ReflectiveC   float64 `json:"reflective_c"`
}

// LaserDistance carries distance measurement in meters, or nil if out of range (0x15).
type LaserDistance struct {
	DistanceM *float64 `json:"distance_m"`
}

// LaserTargetLatLon target coordinates (0x17).
type LaserTargetLatLon struct {
	LatE7 int32 `json:"lat_e7"`
	LonE7 int32 `json:"lon_e7"`
}

// Latitude returns the target latitude in degrees.
func (l LaserTargetLatLon) Latitude() float64 {
	return float64(l.LatE7) / 1e7
}

// Longitude returns the target longitude in degrees.
func (l LaserTargetLatLon) Longitude() float64 {
	return float64(l.LonE7) / 1e7
}

// ZoomRange carries maximum optical zoom capability (0x16).
type ZoomRange struct {
	MaxInt   uint8 `json:"max_int"`
	MaxFloat uint8 `json:"max_float"`
}

// MaxZoom calculates the maximum zoom as a floating point value.
func (z ZoomRange) MaxZoom() float64 {
	return float64(z.MaxInt) + float64(z.MaxFloat)/10.0
}

// CurrentZoom carries current zoom magnification (0x18).
type CurrentZoom struct {
	Integer uint8 `json:"integer"`
	Decimal uint8 `json:"decimal"`
}

// Zoom calculates current zoom magnification as a floating point value.
func (c CurrentZoom) Zoom() float64 {
	return float64(c.Integer) + float64(c.Decimal)/10.0
}

// RawGPS represents GPS telemetry sent to camera (0x3E).
type RawGPS struct {
	TimeBootMS     uint32 `json:"time_boot_ms"`
	LatE7          int32  `json:"lat_e7"`
	LonE7          int32  `json:"lon_e7"`
	AltMSLCm       int32  `json:"alt_msl_cm"`
	AltEllipsoidCm int32  `json:"alt_ellipsoid_cm"`
	VnMMPS         int32  `json:"vn_mmps"`
	VeMMPS         int32  `json:"ve_mmps"`
	VdMMPS         int32  `json:"vd_mmps"`
}

// SystemTime represents camera internal clocks (0x40).
type SystemTime struct {
	UnixUsec uint64 `json:"unix_usec"`
	BootMS   uint32 `json:"boot_ms"`
}

// GimbalSystemInfo gimbal subsystem state (0x31).
type GimbalSystemInfo struct {
	LaserState bool `json:"laser_state"`
}

// IRThreshRegion thermal threshold display zone configuration.
type IRThreshRegion struct {
	Switch  uint8 `json:"switch"` // 0=hide, 1=display
	TempMin int16 `json:"temp_min"`
	TempMax int16 `json:"temp_max"`
	ColorR  uint8 `json:"color_r"`
	ColorG  uint8 `json:"color_g"`
	ColorB  uint8 `json:"color_b"`
}

// IRThreshParams carries 3 threshold color regions (0x44/0x45).
type IRThreshParams struct {
	Region1 IRThreshRegion `json:"region_1"`
	Region2 IRThreshRegion `json:"region_2"`
	Region3 IRThreshRegion `json:"region_3"`
}

// AITrackingTarget carries bounding box and status for an AI object (0x50).
type AITrackingTarget struct {
	X        uint16        `json:"x"`
	Y        uint16        `json:"y"`
	W        uint16        `json:"w"`
	H        uint16        `json:"h"`
	TargetID AITargetID    `json:"target_id"`
	Status   AITrackStatus `json:"status"`
}

// IPConfig network configuration for the gimbal Ethernet port (0x81/0x82).
type IPConfig struct {
	IP      net.IP `json:"ip"`
	Mask    net.IP `json:"mask"`
	Gateway net.IP `json:"gateway"`
}

// MediaDirectory represents a folder entry on the TF card web interface.
type MediaDirectory struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// MediaFile represents a file entry on the TF card web interface.
type MediaFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
