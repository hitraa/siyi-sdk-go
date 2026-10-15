package commands

import (
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeGetVideoStitchingMode creates payload to query video stitching mode (0x10).
func EncodeGetVideoStitchingMode() []byte {
	return nil
}

// DecodeVideoStitchingMode parses 1-byte response for command 0x10.
func DecodeVideoStitchingMode(payload []byte) (models.VideoStitchingMode, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestVideoStitchingMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.VideoStitchingMode(payload[0]), nil
}

// EncodeSetVideoStitchingMode encodes video stitching mode request (0x11).
func EncodeSetVideoStitchingMode(mode models.VideoStitchingMode) []byte {
	return []byte{byte(mode)}
}

// DecodeSetVideoStitchingModeAck parses 1-byte response for command 0x11.
func DecodeSetVideoStitchingModeAck(payload []byte) (models.VideoStitchingMode, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetVideoStitchingMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.VideoStitchingMode(payload[0]), nil
}
