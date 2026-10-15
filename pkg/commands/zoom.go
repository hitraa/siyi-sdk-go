package commands

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeManualZoom encodes zoom rate command into 1 byte (0x05).
// direction: -1=zoom out, 0=stop, 1=zoom in.
func EncodeManualZoom(direction int) ([]byte, error) {
	if direction < -1 || direction > 1 {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("direction must be in {-1,0,1}, got %d", direction),
		}
	}
	return []byte{byte(int8(direction))}, nil
}

// DecodeManualZoomAck parses 2-byte response for command 0x05.
// uint16 LE multiple (divided by 10).
func DecodeManualZoomAck(payload []byte) (float64, error) {
	if len(payload) != 2 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdManualZoomAutoFocus,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	raw := binary.LittleEndian.Uint16(payload[0:2])
	return float64(raw) / 10.0, nil
}

// EncodeAbsoluteZoom encodes target zoom level into 2 bytes (0x0F).
// zoom in [1.0, 30.0].
func EncodeAbsoluteZoom(zoom float64) ([]byte, error) {
	if zoom < 1.0 || zoom > 30.0 {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("zoom must be in [1.0, 30.0], got %f", zoom),
		}
	}

	intPart := int(math.Floor(zoom))
	floatPart := int(math.Round((zoom - float64(intPart)) * 10))

	// Handle edge case (e.g., 4.95 rounds to int=4, float=10 -> int=5, float=0)
	if floatPart == 10 {
		intPart++
		floatPart = 0
	}

	if intPart < 1 || intPart > 30 {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("zoom intPart must be in [1,30], got %d", intPart),
		}
	}
	if floatPart < 0 || floatPart > 9 {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("zoom floatPart must be in [0,9], got %d", floatPart),
		}
	}

	return []byte{byte(intPart), byte(floatPart)}, nil
}

// DecodeAbsoluteZoomAck parses 1-byte response for command 0x0F.
func DecodeAbsoluteZoomAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdAbsoluteZoomAutoFocus,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdAbsoluteZoomAutoFocus, Sta: payload[0]}
	}
	return nil
}

// EncodeZoomRange creates payload to query maximum optical zoom (0x16).
func EncodeZoomRange() []byte {
	return nil
}

// DecodeZoomRange parses 2-byte response for command 0x16 (max_int, max_float).
func DecodeZoomRange(payload []byte) (models.ZoomRange, error) {
	if len(payload) != 2 {
		return models.ZoomRange{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestZoomRange,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	return models.ZoomRange{
		MaxInt:   payload[0],
		MaxFloat: payload[1],
	}, nil
}

// EncodeCurrentZoom creates payload to query current zoom (0x18).
func EncodeCurrentZoom() []byte {
	return nil
}

// DecodeCurrentZoom parses 2-byte response for command 0x18 (integer, decimal).
func DecodeCurrentZoom(payload []byte) (models.CurrentZoom, error) {
	if len(payload) != 2 {
		return models.CurrentZoom{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestZoomMagnification,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	return models.CurrentZoom{
		Integer: payload[0],
		Decimal: payload[1],
	}, nil
}
