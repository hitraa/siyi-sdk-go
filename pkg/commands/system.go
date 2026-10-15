package commands

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeHeartbeat creates payload for TCP keep-alive (0x00).
func EncodeHeartbeat() []byte {
	return []byte{0x00}
}

// EncodeFirmwareVersion creates payload to request firmware versions (0x01).
func EncodeFirmwareVersion() []byte {
	return nil
}

// DecodeFirmwareVersion parses the 12-byte response for command 0x01.
func DecodeFirmwareVersion(payload []byte) (models.FirmwareVersion, error) {
	if len(payload) != 12 {
		return models.FirmwareVersion{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestFirmwareVersion,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}
	camera := binary.LittleEndian.Uint32(payload[0:4])
	gimbal := binary.LittleEndian.Uint32(payload[4:8])
	zoom := binary.LittleEndian.Uint32(payload[8:12])
	return models.FirmwareVersion{
		Camera: camera,
		Gimbal: gimbal,
		Zoom:   zoom,
	}, nil
}

// EncodeHardwareID creates payload to request hardware ID (0x02).
func EncodeHardwareID() []byte {
	return nil
}

// DecodeHardwareID parses the 12-byte hardware ID string response for command 0x02.
func DecodeHardwareID(payload []byte) (models.HardwareID, error) {
	if len(payload) != 12 {
		return models.HardwareID{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestHardwareID,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}
	raw := make([]byte, 12)
	copy(raw, payload)
	return models.HardwareID{Raw: raw}, nil
}

// EncodeSetUTCTime encodes the 64-bit UTC timestamp in microseconds (0x30).
func EncodeSetUTCTime(unixUsec uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, unixUsec)
	return buf
}

// DecodeSetUTCTimeAck parses the 1-byte status response for command 0x30.
func DecodeSetUTCTimeAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetUTCTime,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetUTCTime, Sta: payload[0]}
	}
	return true, nil
}

// EncodeGimbalSystemInfo creates payload to request gimbal system state (0x31).
func EncodeGimbalSystemInfo() []byte {
	return nil
}

// DecodeGimbalSystemInfo parses the 1-byte response for command 0x31.
func DecodeGimbalSystemInfo(payload []byte) (models.GimbalSystemInfo, error) {
	if len(payload) != 1 {
		return models.GimbalSystemInfo{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestGimbalSystemInfo,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	laserState := (payload[0] & 1) != 0
	return models.GimbalSystemInfo{LaserState: laserState}, nil
}

// EncodeSystemTime creates payload to query system time (0x40).
func EncodeSystemTime() []byte {
	return nil
}

// DecodeSystemTime parses the 12-byte response for command 0x40 (uint64 unix_usec, uint32 boot_ms).
func DecodeSystemTime(payload []byte) (models.SystemTime, error) {
	if len(payload) != 12 {
		return models.SystemTime{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdRequestSystemTime,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}
	unixUsec := binary.LittleEndian.Uint64(payload[0:8])
	bootMS := binary.LittleEndian.Uint32(payload[8:12])
	return models.SystemTime{
		UnixUsec: unixUsec,
		BootMS:   bootMS,
	}, nil
}

// EncodeSoftReboot creates a 1-byte bitmask for soft reboot (0x80).
// Bit 7: camera reboot, Bit 6: gimbal reboot.
func EncodeSoftReboot(camera, gimbal bool) []byte {
	var mask byte
	if camera {
		mask |= (1 << 7)
	}
	if gimbal {
		mask |= (1 << 6)
	}
	return []byte{mask}
}

// DecodeSoftRebootAck parses the 1-byte response for command 0x80.
func DecodeSoftRebootAck(payload []byte) (cameraOK, gimbalOK bool, err error) {
	if len(payload) != 1 {
		return false, false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSoftReboot,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	cam := ((payload[0] >> 7) & 1) != 0
	gim := ((payload[0] >> 6) & 1) != 0
	return cam, gim, nil
}

// EncodeGetIP creates payload to query network configuration (0x81).
func EncodeGetIP() []byte {
	return nil
}

// DecodeGetIP parses the 12-byte response for command 0x81 (4 bytes IP, 4 bytes Mask, 4 bytes Gateway).
func DecodeGetIP(payload []byte) (models.IPConfig, error) {
	if len(payload) != 12 {
		return models.IPConfig{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdGetIP,
			Reason: fmt.Sprintf("expected 12 bytes, got %d", len(payload)),
		}
	}
	ip := net.IPv4(payload[0], payload[1], payload[2], payload[3])
	mask := net.IPv4(payload[4], payload[5], payload[6], payload[7])
	gw := net.IPv4(payload[8], payload[9], payload[10], payload[11])
	return models.IPConfig{
		IP:      ip,
		Mask:    mask,
		Gateway: gw,
	}, nil
}

// EncodeSetIP encodes IP configuration into 12 bytes (0x82).
func EncodeSetIP(cfg models.IPConfig) ([]byte, error) {
	ip4 := cfg.IP.To4()
	if ip4 == nil {
		return nil, &ConfigurationError{Message: fmt.Sprintf("invalid IPv4 address: %v", cfg.IP)}
	}
	mask4 := cfg.Mask.To4()
	if mask4 == nil {
		return nil, &ConfigurationError{Message: fmt.Sprintf("invalid IPv4 subnet mask: %v", cfg.Mask)}
	}
	gw4 := cfg.Gateway.To4()
	if gw4 == nil {
		return nil, &ConfigurationError{Message: fmt.Sprintf("invalid IPv4 gateway: %v", cfg.Gateway)}
	}

	buf := make([]byte, 12)
	copy(buf[0:4], ip4)
	copy(buf[4:8], mask4)
	copy(buf[8:12], gw4)
	return buf, nil
}

// DecodeSetIPAck parses the 1-byte response for command 0x82.
func DecodeSetIPAck(payload []byte) (bool, error) {
	if len(payload) != 1 {
		return false, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSetIP,
			Reason: fmt.Sprintf("expected 1 byte, got %d", len(payload)),
		}
	}
	if payload[0] == 0 {
		return false, &ResponseError{CmdID: protocol.CmdSetIP, Sta: payload[0]}
	}
	return true, nil
}
