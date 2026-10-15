package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeAutoFocus encodes auto focus request into 5 bytes (0x04).
// autoFocus: 0=disable, 1=enable. touchX, touchY: coordinates.
func EncodeAutoFocus(autoFocus uint8, touchX, touchY uint16) []byte {
	buf := make([]byte, 5)
	buf[0] = autoFocus
	binary.LittleEndian.PutUint16(buf[1:3], touchX)
	binary.LittleEndian.PutUint16(buf[3:5], touchY)
	return buf
}

// DecodeAutoFocusAck parses 1-byte response for command 0x04.
func DecodeAutoFocusAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdAutoFocus,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdAutoFocus, Sta: payload[0]}
	}
	return nil
}

// EncodeManualFocus encodes manual focus command into 1 byte (0x06).
// direction: -1=near, 0=stop, 1=far.
func EncodeManualFocus(direction int) ([]byte, error) {
	if direction < -1 || direction > 1 {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("direction must be in {-1,0,1}, got %d", direction),
		}
	}
	return []byte{byte(int8(direction))}, nil
}

// DecodeManualFocusAck parses 1-byte response for command 0x06.
func DecodeManualFocusAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdManualFocus,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdManualFocus, Sta: payload[0]}
	}
	return nil
}
