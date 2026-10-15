package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeGetAIMode creates payload to query AI mode state (0x4D).
func EncodeGetAIMode() []byte {
	return nil
}

// DecodeAIMode parses 1-byte response for command 0x4D.
func DecodeAIMode(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetAIModeSta,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}

// EncodeGetAIStreamStatus creates payload to query AI coordinate stream output state (0x4E).
func EncodeGetAIStreamStatus() []byte {
	return nil
}

// DecodeAIStreamStatus parses 1-byte response for command 0x4E.
func DecodeAIStreamStatus(payload []byte) (models.AIStreamStatus, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetAITrackStreamSta,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.AIStreamStatus(payload[0]), nil
}

// DecodeAITracking parses 10-byte unsolicited frame push for command 0x50.
// 4 x uint16 LE (x, y, w, h) + 2 x uint8 (target_id, status).
func DecodeAITracking(payload []byte) (models.AITrackingTarget, error) {
	if len(payload) != 10 {
		return models.AITrackingTarget{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdAITrackStream,
			Reason: fmt.Sprintf("expected 10 bytes, got %d", len(payload)),
		}
	}
	x := binary.LittleEndian.Uint16(payload[0:2])
	y := binary.LittleEndian.Uint16(payload[2:4])
	w := binary.LittleEndian.Uint16(payload[4:6])
	h := binary.LittleEndian.Uint16(payload[6:8])
	targetID := payload[8]
	status := payload[9]

	return models.AITrackingTarget{
		X:        x,
		Y:        y,
		W:        w,
		H:        h,
		TargetID: models.AITargetID(targetID),
		Status:   models.AITrackStatus(status),
	}, nil
}

// EncodeSetAIStreamOutput encodes AI stream output control into 1 byte (0x51).
func EncodeSetAIStreamOutput(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetAIStreamOutputAck parses 1-byte response for command 0x51.
func DecodeSetAIStreamOutputAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetAITrackStreamSta,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetAITrackStreamSta, Sta: payload[0]}
	}
	return true, nil
}
