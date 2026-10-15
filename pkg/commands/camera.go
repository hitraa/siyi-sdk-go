package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeCameraSystemInfo creates payload to request camera system info (0x0A).
func EncodeCameraSystemInfo() []byte {
	return nil
}

// DecodeCameraSystemInfo parses 7 or 8 byte response for command 0x0A.
// Cameras without optical zoom (e.g., A8 Mini) return 7 bytes (no zoom_linkage).
func DecodeCameraSystemInfo(payload []byte) (models.CameraSystemInfo, error) {
	if len(payload) != 7 && len(payload) != 8 {
		return models.CameraSystemInfo{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestCameraSystemInfo,
			Reason: fmt.Sprintf("expected 7 or 8 bytes, got %d", len(payload)),
		}
	}

	var zoomLinkage uint8
	if len(payload) == 8 {
		zoomLinkage = payload[7]
	}

	return models.CameraSystemInfo{
		ReservedA:         payload[0],
		HDRSta:            payload[1],
		ReservedB:         payload[2],
		RecordSta:         models.RecordingState(payload[3]),
		GimbalMotionMode:  models.GimbalMotionMode(payload[4]),
		GimbalMountingDir: models.MountingDirection(payload[5]),
		VideoHDMICVBS:     models.HDMICVBSOutput(payload[6]),
		ZoomLinkage:       zoomLinkage,
	}, nil
}

// DecodeFunctionFeedback parses 1-byte response for command 0x0B.
func DecodeFunctionFeedback(payload []byte) (models.FunctionFeedback, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdFunctionFeedback,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.FunctionFeedback(payload[0]), nil
}

// EncodeCapture encodes capture photo / record video function into 1 byte (0x0C).
func EncodeCapture(funcType models.CaptureFuncType) []byte {
	return []byte{byte(funcType)}
}

// EncodeGetEncodingParams encodes stream parameter request (0x20).
func EncodeGetEncodingParams(stream models.StreamType) []byte {
	return []byte{byte(stream)}
}

// DecodeGetEncodingParams parses 9-byte response for command 0x20.
// stream_type (1B) + enc_type (1B) + res_w (2B LE) + res_h (2B LE) + bitrate (2B LE) + frame_rate (1B).
func DecodeGetEncodingParams(payload []byte) (models.EncodingParams, error) {
	if len(payload) != 9 {
		return models.EncodingParams{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestEncodingParams,
			Reason: fmt.Sprintf("expected 9 bytes, got %d", len(payload)),
		}
	}

	return models.EncodingParams{
		StreamType:   models.StreamType(payload[0]),
		EncType:      models.VideoEncType(payload[1]),
		ResolutionW:  binary.LittleEndian.Uint16(payload[2:4]),
		ResolutionH:  binary.LittleEndian.Uint16(payload[4:6]),
		BitrateKbps:  binary.LittleEndian.Uint16(payload[6:8]),
		FrameRateFPS: payload[8],
	}, nil
}

// EncodeSetEncodingParams encodes 9-byte configuration payload (0x21).
func EncodeSetEncodingParams(params models.EncodingParams) ([]byte, error) {
	// Validate resolution per specification (supports 4K, 2K, 1080p, 720p)
	if !((params.ResolutionW == 3840 && params.ResolutionH == 2160) ||
		(params.ResolutionW == 2560 && params.ResolutionH == 1440) ||
		(params.ResolutionW == 1920 && params.ResolutionH == 1080) ||
		(params.ResolutionW == 1280 && params.ResolutionH == 720)) {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("resolution must be 3840x2160, 2560x1440, 1920x1080, or 1280x720, got %dx%d",
				params.ResolutionW, params.ResolutionH),
		}
	}

	buf := make([]byte, 9)
	buf[0] = byte(params.StreamType)
	buf[1] = byte(params.EncType)
	binary.LittleEndian.PutUint16(buf[2:4], params.ResolutionW)
	binary.LittleEndian.PutUint16(buf[4:6], params.ResolutionH)
	binary.LittleEndian.PutUint16(buf[6:8], params.BitrateKbps)
	buf[8] = 0 // reserve byte
	return buf, nil
}

// DecodeSetEncodingParamsAck parses 2-byte response for command 0x21 (stream_type, sta).
func DecodeSetEncodingParamsAck(payload []byte) (bool, error) {
	if len(payload) != 2 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetEncodingParams,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	if payload[1] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetEncodingParams, Sta: payload[1]}
	}
	return true, nil
}

// EncodeFormatSD encodes SD card format request (0x48). Default format_sta is 1.
func EncodeFormatSD(formatSta ...uint8) []byte {
	sta := uint8(1)
	if len(formatSta) > 0 {
		sta = formatSta[0]
	}
	return []byte{sta}
}

// DecodeFormatSDAck parses 1-byte response for command 0x48.
func DecodeFormatSDAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSDFormat,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSDFormat, Sta: payload[0]}
	}
	return true, nil
}

// EncodeGetPicNameType encodes request for picture naming scheme (0x49).
func EncodeGetPicNameType(ft models.FileType) []byte {
	return []byte{byte(ft)}
}

// DecodeGetPicNameType parses 2-byte response for command 0x49 (file_type, file_name_type).
func DecodeGetPicNameType(payload []byte) (models.FileNameType, error) {
	if len(payload) != 2 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetPicNameType,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	return models.FileNameType(payload[1]), nil
}

// EncodeSetPicNameType encodes picture naming scheme request (0x4A).
func EncodeSetPicNameType(ft models.FileType, nt models.FileNameType) []byte {
	return []byte{byte(ft), byte(nt)}
}

// DecodeSetPicNameTypeAck parses 2-byte response for command 0x4A.
func DecodeSetPicNameTypeAck(payload []byte) error {
	if len(payload) != 2 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetPicNameType,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	return nil
}

// EncodeGetOSDFlag creates payload to query HDMI OSD status (0x4B).
func EncodeGetOSDFlag() []byte {
	return nil
}

// DecodeGetOSDFlag parses 1-byte response for command 0x4B.
func DecodeGetOSDFlag(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetMavlinkOSDFlag,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}

// EncodeSetOSDFlag encodes set HDMI OSD status request (0x4C).
func EncodeSetOSDFlag(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetOSDFlagAck parses 1-byte response for command 0x4C.
func DecodeSetOSDFlagAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetMavlinkOSDFlag,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}
