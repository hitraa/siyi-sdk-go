package commands

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeRotation encodes rotation rate commands into 2 bytes (0x07).
// yaw and pitch rates in range [-100, 100].
func EncodeRotation(yaw, pitch int) ([]byte, error) {
	if yaw < models.GimbalRateCmdMin || yaw > models.GimbalRateCmdMax {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("yaw must be in [%d,%d], got %d", models.GimbalRateCmdMin, models.GimbalRateCmdMax, yaw),
		}
	}
	if pitch < models.GimbalRateCmdMin || pitch > models.GimbalRateCmdMax {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("pitch must be in [%d,%d], got %d", models.GimbalRateCmdMin, models.GimbalRateCmdMax, pitch),
		}
	}
	return []byte{byte(int8(yaw)), byte(int8(pitch))}, nil
}

// DecodeRotationAck parses 1-byte response for command 0x07.
func DecodeRotationAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGimbalRotation,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdGimbalRotation, Sta: payload[0]}
	}
	return nil
}

// EncodeOneKeyCentering encodes one-key centering command (0x08).
func EncodeOneKeyCentering(action models.CenteringAction) ([]byte, error) {
	if action < models.CenteringActionOneKeyCenter || action > models.CenteringActionDownward {
		return nil, &ConfigurationError{
			Message: fmt.Sprintf("action must be a valid CenteringAction, got %d", action),
		}
	}
	return []byte{byte(action)}, nil
}

// DecodeOneKeyCenteringAck parses 1-byte response for command 0x08.
func DecodeOneKeyCenteringAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdOneKeyCentering,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdOneKeyCentering, Sta: payload[0]}
	}
	return nil
}

// EncodeSetAttitude encodes target yaw and pitch angles into 4 bytes (0x0E).
// Angles are scaled by 10 into int16 LE.
func EncodeSetAttitude(yawDeg, pitchDeg float64) []byte {
	yawRaw := int16(math.Round(yawDeg * 10))
	pitchRaw := int16(math.Round(pitchDeg * 10))

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(yawRaw))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(pitchRaw))
	return buf
}

// DecodeSetAttitudeAck parses 6-byte response for command 0x0E.
// 3 x int16 LE: yaw, pitch, roll in degrees (divided by 10).
func DecodeSetAttitudeAck(payload []byte) (models.SetAttitudeAck, error) {
	if len(payload) != 6 {
		return models.SetAttitudeAck{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetGimbalAttitude,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	yawRaw := int16(binary.LittleEndian.Uint16(payload[0:2]))
	pitchRaw := int16(binary.LittleEndian.Uint16(payload[2:4]))
	rollRaw := int16(binary.LittleEndian.Uint16(payload[4:6]))

	return models.SetAttitudeAck{
		YawDeg:   float64(yawRaw) / 10.0,
		PitchDeg: float64(pitchRaw) / 10.0,
		RollDeg:  float64(rollRaw) / 10.0,
	}, nil
}

// EncodeGimbalMode creates payload to request gimbal motion mode (0x19).
func EncodeGimbalMode() []byte {
	return nil
}

// DecodeGimbalMode parses 1-byte response for command 0x19.
func DecodeGimbalMode(payload []byte) (models.GimbalMotionMode, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestGimbalMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.GimbalMotionMode(payload[0]), nil
}

// EncodeSingleAxis encodes single axis attitude angle (0x41).
// axis: 0=yaw, 1=pitch.
func EncodeSingleAxis(angleDeg float64, axis int) ([]byte, error) {
	if axis != 0 && axis != 1 {
		return nil, &ConfigurationError{Message: fmt.Sprintf("axis must be 0 (yaw) or 1 (pitch), got %d", axis)}
	}
	angleRaw := int16(math.Round(angleDeg * 10))

	buf := make([]byte, 3)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(angleRaw))
	buf[2] = byte(axis)
	return buf, nil
}

// DecodeSingleAxisAck parses 6-byte response for command 0x41.
func DecodeSingleAxisAck(payload []byte) (models.SetAttitudeAck, error) {
	if len(payload) != 6 {
		return models.SetAttitudeAck{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSingleAxisAttitude,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	yawRaw := int16(binary.LittleEndian.Uint16(payload[0:2]))
	pitchRaw := int16(binary.LittleEndian.Uint16(payload[2:4]))
	rollRaw := int16(binary.LittleEndian.Uint16(payload[4:6]))

	return models.SetAttitudeAck{
		YawDeg:   float64(yawRaw) / 10.0,
		PitchDeg: float64(pitchRaw) / 10.0,
		RollDeg:  float64(rollRaw) / 10.0,
	}, nil
}
