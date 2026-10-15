package commands

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeTempAtPoint encodes point temperature request (0x12).
func EncodeTempAtPoint(x, y uint16, flag models.TempMeasureFlag) []byte {
	buf := make([]byte, 5)
	binary.LittleEndian.PutUint16(buf[0:2], x)
	binary.LittleEndian.PutUint16(buf[2:4], y)
	buf[4] = byte(flag)
	return buf
}

// DecodeTempAtPoint parses 6-byte response for command 0x12 (x, y, temp/100).
func DecodeTempAtPoint(payload []byte) (models.TempPoint, error) {
	if len(payload) != 6 {
		return models.TempPoint{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetTempAtPoint,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	x := binary.LittleEndian.Uint16(payload[0:2])
	y := binary.LittleEndian.Uint16(payload[2:4])
	rawTemp := binary.LittleEndian.Uint16(payload[4:6])
	return models.TempPoint{
		X:            x,
		Y:            y,
		TemperatureC: float64(rawTemp) / 100.0,
	}, nil
}

// EncodeLocalTemp encodes regional temperature request (0x13).
func EncodeLocalTemp(startX, startY, endX, endY uint16, flag models.TempMeasureFlag) []byte {
	buf := make([]byte, 9)
	binary.LittleEndian.PutUint16(buf[0:2], startX)
	binary.LittleEndian.PutUint16(buf[2:4], startY)
	binary.LittleEndian.PutUint16(buf[4:6], endX)
	binary.LittleEndian.PutUint16(buf[6:8], endY)
	buf[8] = byte(flag)
	return buf
}

// DecodeLocalTemp parses 24-byte response for command 0x13.
func DecodeLocalTemp(payload []byte) (models.TempRegion, error) {
	if len(payload) != 24 {
		return models.TempRegion{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdLocalTempMeasurement,
			Reason: fmt.Sprintf("expected 24 bytes, got %d", len(payload)),
		}
	}
	startX := binary.LittleEndian.Uint16(payload[0:2])
	startY := binary.LittleEndian.Uint16(payload[2:4])
	endX := binary.LittleEndian.Uint16(payload[4:6])
	endY := binary.LittleEndian.Uint16(payload[6:8])
	maxTemp := binary.LittleEndian.Uint16(payload[8:10])
	minTemp := binary.LittleEndian.Uint16(payload[10:12])
	maxX := binary.LittleEndian.Uint16(payload[12:14])
	maxY := binary.LittleEndian.Uint16(payload[14:16])
	minX := binary.LittleEndian.Uint16(payload[16:18])
	minY := binary.LittleEndian.Uint16(payload[18:20])

	return models.TempRegion{
		StartX: startX,
		StartY: startY,
		EndX:   endX,
		EndY:   endY,
		MaxC:   float64(maxTemp) / 100.0,
		MinC:   float64(minTemp) / 100.0,
		MaxX:   maxX,
		MaxY:   maxY,
		MinX:   minX,
		MinY:   minY,
	}, nil
}

// EncodeGlobalTemp encodes global thermal measurement request (0x14).
func EncodeGlobalTemp(flag models.TempMeasureFlag) []byte {
	return []byte{byte(flag)}
}

// DecodeGlobalTemp parses 12-byte response for command 0x14.
func DecodeGlobalTemp(payload []byte) (models.TempGlobal, error) {
	if len(payload) != 12 {
		return models.TempGlobal{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGlobalTempMeasurement,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}
	maxTemp := binary.LittleEndian.Uint16(payload[0:2])
	minTemp := binary.LittleEndian.Uint16(payload[2:4])
	maxX := binary.LittleEndian.Uint16(payload[4:6])
	maxY := binary.LittleEndian.Uint16(payload[6:8])
	minX := binary.LittleEndian.Uint16(payload[8:10])
	minY := binary.LittleEndian.Uint16(payload[10:12])

	return models.TempGlobal{
		MaxC: float64(maxTemp) / 100.0,
		MinC: float64(minTemp) / 100.0,
		MaxX: maxX,
		MaxY: maxY,
		MinX: minX,
		MinY: minY,
	}, nil
}

// EncodeGetPseudoColor creates payload to query thermal palette (0x1A).
func EncodeGetPseudoColor() []byte {
	return nil
}

// DecodePseudoColor parses 1-byte response for command 0x1A.
func DecodePseudoColor(payload []byte) (models.PseudoColor, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestPseudoColor,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.PseudoColor(payload[0]), nil
}

// EncodeSetPseudoColor encodes set pseudo color palette into 1 byte (0x1B).
func EncodeSetPseudoColor(color models.PseudoColor) []byte {
	return []byte{byte(color)}
}

// DecodeSetPseudoColorAck parses 1-byte response for command 0x1B.
func DecodeSetPseudoColorAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetPseudoColor,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetPseudoColor, Sta: payload[0]}
	}
	return nil
}

// EncodeGetThermalOutputMode creates payload to query thermal output mode (0x33).
func EncodeGetThermalOutputMode() []byte {
	return nil
}

// DecodeThermalOutputMode parses 1-byte response for command 0x33.
func DecodeThermalOutputMode(payload []byte) (models.ThermalOutputMode, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestThermalOutputMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.ThermalOutputMode(payload[0]), nil
}

// EncodeSetThermalOutputMode encodes thermal output mode into 1 byte (0x34).
func EncodeSetThermalOutputMode(mode models.ThermalOutputMode) []byte {
	return []byte{byte(mode)}
}

// DecodeSetThermalOutputModeAck parses 1-byte response for command 0x34.
func DecodeSetThermalOutputModeAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetThermalOutputMode,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetThermalOutputMode, Sta: payload[0]}
	}
	return nil
}

// EncodeGetSingleTempFrame creates payload to query single temperature frame (0x35).
func EncodeGetSingleTempFrame() []byte {
	return nil
}

// DecodeSingleTempFrameAck parses 1-byte response for command 0x35.
func DecodeSingleTempFrameAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetSingleTempFrame,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdGetSingleTempFrame, Sta: payload[0]}
	}
	return nil
}

// EncodeGetThermalGain creates payload to query thermal gain (0x37).
func EncodeGetThermalGain() []byte {
	return nil
}

// DecodeThermalGain parses 1-byte response for command 0x37.
func DecodeThermalGain(payload []byte) (models.ThermalGain, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestThermalGain,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.ThermalGain(payload[0]), nil
}

// EncodeSetThermalGain encodes thermal gain into 1 byte (0x38).
func EncodeSetThermalGain(gain models.ThermalGain) []byte {
	return []byte{byte(gain)}
}

// DecodeSetThermalGainAck parses 1-byte response for command 0x38.
func DecodeSetThermalGainAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetThermalGain,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetThermalGain, Sta: payload[0]}
	}
	return nil
}

// EncodeGetEnvCorrectionParams creates payload to query environmental correction (0x39).
func EncodeGetEnvCorrectionParams() []byte {
	return nil
}

// DecodeEnvCorrectionParams parses 10-byte response for command 0x39 (5 x uint16 LE / 100).
func DecodeEnvCorrectionParams(payload []byte) (models.EnvCorrectionParams, error) {
	if len(payload) != 10 {
		return models.EnvCorrectionParams{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestEnvCorrectionParams,
			Reason: fmt.Sprintf("expected 10 bytes, got %d", len(payload)),
		}
	}
	dist := binary.LittleEndian.Uint16(payload[0:2])
	emiss := binary.LittleEndian.Uint16(payload[2:4])
	humid := binary.LittleEndian.Uint16(payload[4:6])
	amb := binary.LittleEndian.Uint16(payload[6:8])
	ref := binary.LittleEndian.Uint16(payload[8:10])

	return models.EnvCorrectionParams{
		DistanceM:     float64(dist) / 100.0,
		EmissivityPct: float64(emiss) / 100.0,
		HumidityPct:   float64(humid) / 100.0,
		AmbientC:      float64(amb) / 100.0,
		ReflectiveC:   float64(ref) / 100.0,
	}, nil
}

// EncodeSetEnvCorrectionParams encodes 10-byte environmental parameters (0x3A).
func EncodeSetEnvCorrectionParams(p models.EnvCorrectionParams) []byte {
	buf := make([]byte, 10)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(math.Round(p.DistanceM*100)))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(math.Round(p.EmissivityPct*100)))
	binary.LittleEndian.PutUint16(buf[4:6], uint16(math.Round(p.HumidityPct*100)))
	binary.LittleEndian.PutUint16(buf[6:8], uint16(math.Round(p.AmbientC*100)))
	binary.LittleEndian.PutUint16(buf[8:10], uint16(math.Round(p.ReflectiveC*100)))
	return buf
}

// DecodeSetEnvCorrectionParamsAck parses 1-byte response for command 0x3A.
func DecodeSetEnvCorrectionParamsAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetEnvCorrectionParams,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetEnvCorrectionParams, Sta: payload[0]}
	}
	return nil
}

// EncodeGetEnvCorrectionSwitch creates payload to query environmental compensation switch (0x3B).
func EncodeGetEnvCorrectionSwitch() []byte {
	return nil
}

// DecodeEnvCorrectionSwitch parses 1-byte response for command 0x3B.
func DecodeEnvCorrectionSwitch(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestEnvCorrectionSwitch,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}

// EncodeSetEnvCorrectionSwitch encodes environmental switch into 1 byte (0x3C).
func EncodeSetEnvCorrectionSwitch(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetEnvCorrectionSwitchAck parses 1-byte response for command 0x3C.
func DecodeSetEnvCorrectionSwitchAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetEnvCorrectionSwitch,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetEnvCorrectionSwitch, Sta: payload[0]}
	}
	return nil
}

// EncodeGetIRThreshMapState creates payload to query IR threshold display switch (0x42).
func EncodeGetIRThreshMapState() []byte {
	return nil
}

// DecodeIRThreshMapState parses 1-byte response for command 0x42.
func DecodeIRThreshMapState(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetIRThreshMapSta,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return payload[0] != 0, nil
}

// EncodeSetIRThreshMapState encodes IR threshold display switch into 1 byte (0x43).
func EncodeSetIRThreshMapState(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetIRThreshMapStateAck parses 1-byte response for command 0x43.
func DecodeSetIRThreshMapStateAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetIRThreshMapSta,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetIRThreshMapSta, Sta: payload[0]}
	}
	return nil
}

// EncodeGetIRThreshParams creates payload to query IR threshold regions (0x44).
func EncodeGetIRThreshParams() []byte {
	return nil
}

func decodeIRThreshRegion(data []byte) models.IRThreshRegion {
	return models.IRThreshRegion{
		Switch:  data[0],
		TempMin: int16(binary.LittleEndian.Uint16(data[1:3])),
		TempMax: int16(binary.LittleEndian.Uint16(data[3:5])),
		ColorR:  data[5],
		ColorG:  data[6],
		ColorB:  data[7],
	}
}

func encodeIRThreshRegion(r models.IRThreshRegion) []byte {
	buf := make([]byte, 8)
	buf[0] = r.Switch
	binary.LittleEndian.PutUint16(buf[1:3], uint16(r.TempMin))
	binary.LittleEndian.PutUint16(buf[3:5], uint16(r.TempMax))
	buf[5] = r.ColorR
	buf[6] = r.ColorG
	buf[7] = r.ColorB
	return buf
}

// DecodeIRThreshParams parses 24-byte response for command 0x44 (3 regions x 8 bytes).
func DecodeIRThreshParams(payload []byte) (models.IRThreshParams, error) {
	if len(payload) != 24 {
		return models.IRThreshParams{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetIRThreshParam,
			Reason: fmt.Sprintf("expected 24 bytes, got %d", len(payload)),
		}
	}
	return models.IRThreshParams{
		Region1: decodeIRThreshRegion(payload[0:8]),
		Region2: decodeIRThreshRegion(payload[8:16]),
		Region3: decodeIRThreshRegion(payload[16:24]),
	}, nil
}

// EncodeSetIRThreshParams encodes 24-byte payload for command 0x45.
func EncodeSetIRThreshParams(p models.IRThreshParams) []byte {
	buf := make([]byte, 24)
	copy(buf[0:8], encodeIRThreshRegion(p.Region1))
	copy(buf[8:16], encodeIRThreshRegion(p.Region2))
	copy(buf[16:24], encodeIRThreshRegion(p.Region3))
	return buf
}

// DecodeSetIRThreshParamsAck parses 1-byte response for command 0x45.
func DecodeSetIRThreshParamsAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetIRThreshParam,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetIRThreshParam, Sta: payload[0]}
	}
	return nil
}

// EncodeGetIRThreshPrecision creates payload to query IR threshold precision (0x46).
func EncodeGetIRThreshPrecision() []byte {
	return nil
}

// DecodeIRThreshPrecision parses 1-byte response for command 0x46.
func DecodeIRThreshPrecision(payload []byte) (models.IRThreshPrecision, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetIRThreshPrecision,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.IRThreshPrecision(payload[0]), nil
}

// EncodeSetIRThreshPrecision encodes IR threshold precision into 1 byte (0x47).
func EncodeSetIRThreshPrecision(precision models.IRThreshPrecision) []byte {
	return []byte{byte(precision)}
}

// DecodeSetIRThreshPrecisionAck parses 1-byte response for command 0x47.
func DecodeSetIRThreshPrecisionAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetIRThreshPrecision,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdSetIRThreshPrecision, Sta: payload[0]}
	}
	return nil
}

// EncodeManualThermalShutter creates payload to trigger manual thermal shutter calibration (0x4F).
func EncodeManualThermalShutter() []byte {
	return nil
}

// DecodeManualThermalShutterAck parses 1-byte response for command 0x4F.
func DecodeManualThermalShutterAck(payload []byte) error {
	if len(payload) != 1 {
		return &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdManualThermalShutter,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return &ResponseError{CmdID: protocol.CmdManualThermalShutter, Sta: payload[0]}
	}
	return nil
}
