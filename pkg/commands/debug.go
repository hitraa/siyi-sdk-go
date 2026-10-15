package commands

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeGetControlMode creates payload to query ArduPilot control mode (0x27).
func EncodeGetControlMode() []byte {
	return nil
}

// DecodeControlMode parses 1-byte response for command 0x27.
func DecodeControlMode(payload []byte) (models.ControlMode, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestControlMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.ControlMode(payload[0]), nil
}

// EncodeGetWeakThreshold creates payload to query weak control threshold (0x28).
func EncodeGetWeakThreshold() []byte {
	return nil
}

// DecodeWeakThreshold parses 6-byte response for command 0x28 (3 x int16 LE / 10).
func DecodeWeakThreshold(payload []byte) (models.WeakControlThreshold, error) {
	if len(payload) != 6 {
		return models.WeakControlThreshold{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestWeakThreshold,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	limit := int16(binary.LittleEndian.Uint16(payload[0:2]))
	voltage := int16(binary.LittleEndian.Uint16(payload[2:4]))
	angularError := int16(binary.LittleEndian.Uint16(payload[4:6]))

	return models.WeakControlThreshold{
		Limit:        float64(limit) / 10.0,
		Voltage:      float64(voltage) / 10.0,
		AngularError: float64(angularError) / 10.0,
	}, nil
}

// EncodeSetWeakThreshold encodes 6-byte weak threshold configuration (0x29).
func EncodeSetWeakThreshold(t models.WeakControlThreshold) []byte {
	buf := make([]byte, 6)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(int16(math.Round(t.Limit*10))))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(int16(math.Round(t.Voltage*10))))
	binary.LittleEndian.PutUint16(buf[4:6], uint16(int16(math.Round(t.AngularError*10))))
	return buf
}

// DecodeSetWeakThresholdAck parses 1-byte response for command 0x29.
func DecodeSetWeakThresholdAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetWeakThreshold,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetWeakThreshold, Sta: payload[0]}
	}
	return true, nil
}

// EncodeGetMotorVoltage creates payload to query motor voltages (0x2A).
func EncodeGetMotorVoltage() []byte {
	return nil
}

// DecodeMotorVoltage parses 6-byte response for command 0x2A (3 x int16 LE / 1000).
func DecodeMotorVoltage(payload []byte) (models.MotorVoltage, error) {
	if len(payload) != 6 {
		return models.MotorVoltage{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestMotorVoltage,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	yaw := int16(binary.LittleEndian.Uint16(payload[0:2]))
	pitch := int16(binary.LittleEndian.Uint16(payload[2:4]))
	roll := int16(binary.LittleEndian.Uint16(payload[4:6]))

	return models.MotorVoltage{
		Yaw:   float64(yaw) / 1000.0,
		Pitch: float64(pitch) / 1000.0,
		Roll:  float64(roll) / 1000.0,
	}, nil
}

// EncodeGetWeakControlMode creates payload to query weak control state (0x70).
func EncodeGetWeakControlMode() []byte {
	return nil
}

// DecodeWeakControlMode parses 1-byte response for command 0x70.
func DecodeWeakControlMode(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestWeakControlMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}

// EncodeSetWeakControlMode encodes set weak control mode into 1 byte (0x71).
func EncodeSetWeakControlMode(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetWeakControlModeAck parses 2-byte response for command 0x71 (sta, state).
func DecodeSetWeakControlModeAck(payload []byte) (bool, error) {
	if len(payload) != 2 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetWeakControlMode,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetWeakControlMode, Sta: payload[0]}
	}
	return payload[1] != 0, nil
}
