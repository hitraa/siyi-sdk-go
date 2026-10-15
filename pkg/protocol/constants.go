package protocol

// Protocol framing constants for the SIYI SDK.
const (
	// STX is the start marker (0x6655, wire representation: 0x55 0x66).
	STX uint16 = 0x6655

	// Control byte bit flags
	CTRLNeedACK uint8 = 0x01 // bit[0]: sender needs receiver to respond
	CTRLACKPack uint8 = 0x02 // bit[1]: this frame IS a response/ACK

	// HeaderLen is the fixed size of the frame header:
	// STX(2) + CTRL(1) + DataLen(2) + SEQ(2) + CMD_ID(1)
	HeaderLen = 8

	// CRCLen is the 2-byte CRC-16/XMODEM checksum length.
	CRCLen = 2

	// MinFrameLen is the minimum frame size with empty payload: HeaderLen + CRCLen.
	MinFrameLen = 10

	// SeqMax is the maximum 16-bit sequence number before wrapping.
	SeqMax uint16 = 0xFFFF

	// DefaultMaxPayload is the maximum accepted frame payload size in bytes.
	DefaultMaxPayload = 4096
)

// Wire STX bytes.
var STXBytes = [2]byte{0x55, 0x66}

// CRC16 constants
const (
	CRC16Poly uint16 = 0x1021 // X^16 + X^12 + X^5 + 1
	CRC16Init uint16 = 0x0000 // Initial CRC value
)

// CRC16Table lookup table from SIYI SDK Protocol Chapter 4.
var CRC16Table = [256]uint16{
	0x0000, 0x1021, 0x2042, 0x3063, 0x4084, 0x50A5, 0x60C6, 0x70E7,
	0x8108, 0x9129, 0xA14A, 0xB16B, 0xC18C, 0xD1AD, 0xE1CE, 0xF1EF,
	0x1231, 0x0210, 0x3273, 0x2252, 0x52B5, 0x4294, 0x72F7, 0x62D6,
	0x9339, 0x8318, 0xB37B, 0xA35A, 0xD3BD, 0xC39C, 0xF3FF, 0xE3DE,
	0x2462, 0x3443, 0x0420, 0x1401, 0x64E6, 0x74C7, 0x44A4, 0x5485,
	0xA56A, 0xB54B, 0x8528, 0x9509, 0xE5EE, 0xF5CF, 0xC5AC, 0xD58D,
	0x3653, 0x2672, 0x1611, 0x0630, 0x76D7, 0x66F6, 0x5695, 0x46B4,
	0xB75B, 0xA77A, 0x9719, 0x8738, 0xF7DF, 0xE7FE, 0xD79D, 0xC7BC,
	0x48C4, 0x58E5, 0x6886, 0x78A7, 0x0840, 0x1861, 0x2802, 0x3823,
	0xC9CC, 0xD9ED, 0xE98E, 0xF9AF, 0x8948, 0x9969, 0xA90A, 0xB92B,
	0x5AF5, 0x4AD4, 0x7AB7, 0x6A96, 0x1A71, 0x0A50, 0x3A33, 0x2A12,
	0xDBFD, 0xCBDC, 0xFBBF, 0xEB9E, 0x9B79, 0x8B58, 0xBB3B, 0xAB1A,
	0x6CA6, 0x7C87, 0x4CE4, 0x5CC5, 0x2C22, 0x3C03, 0x0C60, 0x1C41,
	0xEDAE, 0xFD8F, 0xCDEC, 0xDDCD, 0xAD2A, 0xBD0B, 0x8D68, 0x9D49,
	0x7E97, 0x6EB6, 0x5ED5, 0x4EF4, 0x3E13, 0x2E32, 0x1E51, 0x0E70,
	0xFF9F, 0xEFBE, 0xDFDD, 0xCFFC, 0xBF1B, 0xAF3A, 0x9F59, 0x8F78,
	0x9188, 0x81A9, 0xB1CA, 0xA1EB, 0xD10C, 0xC12D, 0xF14E, 0xE16F,
	0x1080, 0x00A1, 0x30C2, 0x20E3, 0x5004, 0x4025, 0x7046, 0x6067,
	0x83B9, 0x9398, 0xA3FB, 0xB3DA, 0xC33D, 0xD31C, 0xE37F, 0xF35E,
	0x02B1, 0x1290, 0x22F3, 0x32D2, 0x4235, 0x5214, 0x6277, 0x7256,
	0xB5EA, 0xA5CB, 0x95A8, 0x8589, 0xF56E, 0xE54F, 0xD52C, 0xC50D,
	0x34E2, 0x24C3, 0x14A0, 0x0481, 0x7466, 0x6447, 0x5424, 0x4405,
	0xA7DB, 0xB7FA, 0x8799, 0x97B8, 0xE75F, 0xF77E, 0xC71D, 0xD73C,
	0x26D3, 0x36F2, 0x0691, 0x16B0, 0x6657, 0x7676, 0x4615, 0x5634,
	0xD94C, 0xC96D, 0xF90E, 0xE92F, 0x99C8, 0x89E9, 0xB98A, 0xA9AB,
	0x5844, 0x4865, 0x7806, 0x6827, 0x18C0, 0x08E1, 0x3882, 0x28A3,
	0xCB7D, 0xDB5C, 0xEB3F, 0xFB1E, 0x8BF9, 0x9BD8, 0xABBB, 0xBB9A,
	0x4A75, 0x5A54, 0x6A37, 0x7A16, 0x0AF1, 0x1AD0, 0x2AB3, 0x3A92,
	0xFD2E, 0xED0F, 0xDD6C, 0xCD4D, 0xBDAA, 0xAD8B, 0x9DE8, 0x8DC9,
	0x7C26, 0x6C07, 0x5C64, 0x4C45, 0x3CA2, 0x2C83, 0x1CE0, 0x0CC1,
	0xEF1F, 0xFF3E, 0xCF5D, 0xDF7C, 0xAF9B, 0xBFBA, 0x8FD9, 0x9FF8,
	0x6E17, 0x7E36, 0x4E55, 0x5E74, 0x2E93, 0x3EB2, 0x0ED1, 0x1EF0,
}

// Default network endpoints
const (
	DefaultIP       = "192.168.144.25"
	DefaultUDPPort  = 37260
	DefaultTCPPort  = 37260
	DefaultBaudRate = 115200
	DefaultHTTPPort = 82
)

// Pre-built wire frame bytes
var (
	// HeartbeatFrame wire bytes: 55 66 01 01 00 00 00 00 00 59 8B
	HeartbeatFrame = []byte{0x55, 0x66, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x59, 0x8B}
)

// Camera timing constants
const (
	CameraBootSeconds = 30
)

// Laser distance bounds
const (
	LaserMinM     = 5
	LaserMaxM     = 1200
	LaserMinRawDM = 50 // Minimum raw decimeter reading (5.0m)
)

// Command IDs from the SIYI SDK Protocol Specification
const (
	CmdTCPHeartbeat              uint8 = 0x00
	CmdRequestFirmwareVersion    uint8 = 0x01
	CmdRequestHardwareID         uint8 = 0x02
	CmdAutoFocus                 uint8 = 0x04
	CmdManualZoomAutoFocus       uint8 = 0x05
	CmdManualFocus               uint8 = 0x06
	CmdGimbalRotation            uint8 = 0x07
	CmdOneKeyCentering           uint8 = 0x08
	CmdRequestCameraSystemInfo   uint8 = 0x0A
	CmdFunctionFeedback          uint8 = 0x0B
	CmdCapturePhotoRecordVideo   uint8 = 0x0C
	CmdRequestGimbalAttitude     uint8 = 0x0D
	CmdSetGimbalAttitude         uint8 = 0x0E
	CmdAbsoluteZoomAutoFocus     uint8 = 0x0F
	CmdRequestVideoStitchingMode uint8 = 0x10
	CmdSetVideoStitchingMode     uint8 = 0x11
	CmdGetTempAtPoint            uint8 = 0x12
	CmdLocalTempMeasurement      uint8 = 0x13
	CmdGlobalTempMeasurement     uint8 = 0x14
	CmdRequestLaserDistance      uint8 = 0x15
	CmdRequestZoomRange          uint8 = 0x16
	CmdRequestLaserLatLon        uint8 = 0x17
	CmdRequestZoomMagnification  uint8 = 0x18
	CmdRequestGimbalMode         uint8 = 0x19
	CmdRequestPseudoColor        uint8 = 0x1A
	CmdSetPseudoColor            uint8 = 0x1B
	CmdRequestEncodingParams     uint8 = 0x20
	CmdSetEncodingParams         uint8 = 0x21
	CmdSendAircraftAttitude      uint8 = 0x22
	CmdSendRCChannels            uint8 = 0x23
	CmdRequestFCDataStream       uint8 = 0x24
	CmdRequestGimbalDataStream   uint8 = 0x25
	CmdRequestMagneticEncoder    uint8 = 0x26
	CmdRequestControlMode        uint8 = 0x27
	CmdRequestWeakThreshold      uint8 = 0x28
	CmdSetWeakThreshold          uint8 = 0x29
	CmdRequestMotorVoltage       uint8 = 0x2A
	CmdSetUTCTime                uint8 = 0x30
	CmdRequestGimbalSystemInfo   uint8 = 0x31
	CmdSetLaserRangingState      uint8 = 0x32
	CmdRequestThermalOutputMode  uint8 = 0x33
	CmdSetThermalOutputMode      uint8 = 0x34
	CmdGetSingleTempFrame        uint8 = 0x35
	CmdRequestThermalGain        uint8 = 0x37
	CmdSetThermalGain            uint8 = 0x38
	CmdRequestEnvCorrectionParams uint8 = 0x39
	CmdSetEnvCorrectionParams   uint8 = 0x3A
	CmdRequestEnvCorrectionSwitch uint8 = 0x3B
	CmdSetEnvCorrectionSwitch   uint8 = 0x3C
	CmdSendRawGPS                uint8 = 0x3E
	CmdRequestSystemTime         uint8 = 0x40
	CmdSingleAxisAttitude        uint8 = 0x41
	CmdGetIRThreshMapSta         uint8 = 0x42
	CmdSetIRThreshMapSta         uint8 = 0x43
	CmdGetIRThreshParam          uint8 = 0x44
	CmdSetIRThreshParam          uint8 = 0x45
	CmdGetIRThreshPrecision      uint8 = 0x46
	CmdSetIRThreshPrecision      uint8 = 0x47
	CmdSDFormat                  uint8 = 0x48
	CmdGetPicNameType            uint8 = 0x49
	CmdSetPicNameType            uint8 = 0x4A
	CmdGetMavlinkOSDFlag         uint8 = 0x4B
	CmdSetMavlinkOSDFlag         uint8 = 0x4C
	CmdGetAIModeSta              uint8 = 0x4D
	CmdGetAITrackStreamSta       uint8 = 0x4E
	CmdManualThermalShutter      uint8 = 0x4F
	CmdAITrackStream             uint8 = 0x50
	CmdSetAITrackStreamSta       uint8 = 0x51
	CmdRequestWeakControlMode    uint8 = 0x70
	CmdSetWeakControlMode        uint8 = 0x71
	CmdSoftReboot                uint8 = 0x80
	CmdGetIP                     uint8 = 0x81
	CmdSetIP                     uint8 = 0x82
)

// Hardware ID Product Codes (first byte of hardware ID)
const (
	HWIDZR10          uint8 = 0x6B
	HWIDA8Mini        uint8 = 0x73
	HWIDA2Mini        uint8 = 0x75
	HWIDZR30          uint8 = 0x78
	HWIDQuadSpectrum  uint8 = 0x7A
)
