package commands

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeGimbalAttitude creates payload to request attitude angles and rates (0x0D).
func EncodeGimbalAttitude() []byte {
	return nil
}

// DecodeGimbalAttitude parses 12-byte response for command 0x0D.
// 6 x int16 LE: yaw, pitch, roll, yaw_vel, pitch_vel, roll_vel (divided by 10).
func DecodeGimbalAttitude(payload []byte) (models.GimbalAttitude, error) {
	if len(payload) != 12 {
		return models.GimbalAttitude{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestGimbalAttitude,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}

	yaw := int16(binary.LittleEndian.Uint16(payload[0:2]))
	pitch := int16(binary.LittleEndian.Uint16(payload[2:4]))
	roll := int16(binary.LittleEndian.Uint16(payload[4:6]))
	yawVel := int16(binary.LittleEndian.Uint16(payload[6:8]))
	pitchVel := int16(binary.LittleEndian.Uint16(payload[8:10]))
	rollVel := int16(binary.LittleEndian.Uint16(payload[10:12]))

	return models.GimbalAttitude{
		YawDeg:       float64(yaw) / 10.0,
		PitchDeg:     float64(pitch) / 10.0,
		RollDeg:      float64(roll) / 10.0,
		YawRateDPS:   float64(yawVel) / 10.0,
		PitchRateDPS: float64(pitchVel) / 10.0,
		RollRateDPS:  float64(rollVel) / 10.0,
	}, nil
}

// EncodeAircraftAttitude encodes aircraft attitude into 28 bytes (0x22).
// uint32 time_boot_ms + 6 x float32 LE: roll, pitch, yaw, rollspeed, pitchspeed, yawspeed.
func EncodeAircraftAttitude(att models.AircraftAttitude) []byte {
	buf := make([]byte, 28)
	binary.LittleEndian.PutUint32(buf[0:4], att.TimeBootMS)
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(att.RollRad))
	binary.LittleEndian.PutUint32(buf[8:12], math.Float32bits(att.PitchRad))
	binary.LittleEndian.PutUint32(buf[12:16], math.Float32bits(att.YawRad))
	binary.LittleEndian.PutUint32(buf[16:20], math.Float32bits(att.RollSpeed))
	binary.LittleEndian.PutUint32(buf[20:24], math.Float32bits(att.PitchSpeed))
	binary.LittleEndian.PutUint32(buf[24:28], math.Float32bits(att.YawSpeed))
	return buf
}

// EncodeFCStream encodes FC data stream request (0x24).
func EncodeFCStream(dataType models.FCDataType, freq models.DataStreamFreq) ([]byte, error) {
	if dataType != models.FCDataAttitude && dataType != models.FCDataRCChannels {
		return nil, &ConfigurationError{Message: fmt.Sprintf("dataType must be 1 or 2, got %d", dataType)}
	}
	return []byte{byte(dataType), byte(freq)}, nil
}

// DecodeFCStreamAck parses 1-byte response for command 0x24.
func DecodeFCStreamAck(payload []byte) (models.FCDataType, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestFCDataStream,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.FCDataType(payload[0]), nil
}

// EncodeGimbalStream encodes gimbal data stream request (0x25).
func EncodeGimbalStream(dataType models.GimbalDataType, freq models.DataStreamFreq) ([]byte, error) {
	if dataType < models.GimbalDataAttitude || dataType > models.GimbalDataMotorVoltage {
		return nil, &ConfigurationError{Message: fmt.Sprintf("dataType must be 1-4, got %d", dataType)}
	}
	return []byte{byte(dataType), byte(freq)}, nil
}

// DecodeGimbalStreamAck parses 1-byte response for command 0x25.
func DecodeGimbalStreamAck(payload []byte) (models.GimbalDataType, error) {
	if len(payload) != 1 {
		return 0, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestGimbalDataStream,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	return models.GimbalDataType(payload[0]), nil
}

// EncodeMagneticEncoder creates payload to query magnetic encoders (0x26).
func EncodeMagneticEncoder() []byte {
	return nil
}

// DecodeMagneticEncoder parses 6-byte response for command 0x26 (3 x int16 LE / 10).
func DecodeMagneticEncoder(payload []byte) (models.MagneticEncoderAngles, error) {
	if len(payload) != 6 {
		return models.MagneticEncoderAngles{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestMagneticEncoder,
			Reason: fmt.Sprintf("expected 6 bytes, got %d", len(payload)),
		}
	}
	yaw := int16(binary.LittleEndian.Uint16(payload[0:2]))
	pitch := int16(binary.LittleEndian.Uint16(payload[2:4]))
	roll := int16(binary.LittleEndian.Uint16(payload[4:6]))
	return models.MagneticEncoderAngles{
		Yaw:   float64(yaw) / 10.0,
		Pitch: float64(pitch) / 10.0,
		Roll:  float64(roll) / 10.0,
	}, nil
}

// EncodeRawGPS encodes raw GPS message into 32 bytes (0x3E).
// uint32 time_boot_ms + 7 x int32 LE.
func EncodeRawGPS(gps models.RawGPS) []byte {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint32(buf[0:4], gps.TimeBootMS)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(gps.LatE7))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(gps.LonE7))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(gps.AltMSLCm))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(gps.AltEllipsoidCm))
	binary.LittleEndian.PutUint32(buf[20:24], uint32(gps.VnMMPS))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(gps.VeMMPS))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(gps.VdMMPS))
	return buf
}
