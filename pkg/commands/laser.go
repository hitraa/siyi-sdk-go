package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeLaserDistance creates payload to query laser distance (0x15).
func EncodeLaserDistance() []byte {
	return nil
}

// DecodeLaserDistance parses 2-byte response for command 0x15.
// Raw value in decimeters (divided by 10.0 for meters).
// Per spec: raw < 50 or raw == 0 -> out of range (returns nil).
func DecodeLaserDistance(payload []byte) (models.LaserDistance, error) {
	if len(payload) != 2 {
		return models.LaserDistance{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestLaserDistance,
			Reason: fmt.Sprintf("expected 2 bytes, got %d", len(payload)),
		}
	}
	raw := binary.LittleEndian.Uint16(payload[0:2])
	if raw == 0 || raw < protocol.LaserMinRawDM {
		return models.LaserDistance{DistanceM: nil}, nil
	}
	val := float64(raw) / 10.0
	return models.LaserDistance{DistanceM: &val}, nil
}

// EncodeLaserTargetLatLon creates payload to query laser target coordinates (0x17).
func EncodeLaserTargetLatLon() []byte {
	return nil
}

// DecodeLaserTargetLatLon parses 8-byte response for command 0x17 (2 x int32 LE: lon_e7, lat_e7).
func DecodeLaserTargetLatLon(payload []byte) (models.LaserTargetLatLon, error) {
	if len(payload) != 8 {
		return models.LaserTargetLatLon{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestLaserLatLon,
			Reason: fmt.Sprintf("expected 8 bytes, got %d", len(payload)),
		}
	}
	lonE7 := int32(binary.LittleEndian.Uint32(payload[0:4]))
	latE7 := int32(binary.LittleEndian.Uint32(payload[4:8]))
	return models.LaserTargetLatLon{
		LatE7: latE7,
		LonE7: lonE7,
	}, nil
}

// EncodeSetLaserRangingState encodes laser state control into 1 byte (0x32).
func EncodeSetLaserRangingState(on bool) []byte {
	var val byte
	if on {
		val = 1
	}
	return []byte{val}
}

// DecodeSetLaserRangingStateAck parses 1-byte response for command 0x32.
func DecodeSetLaserRangingStateAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetLaserRangingState,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetLaserRangingState, Sta: payload[0]}
	}
	return true, nil
}
