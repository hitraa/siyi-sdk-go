package protocol_test

import (
	"encoding/hex"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func TestCRC16(t *testing.T) {
	tests := []struct {
		name     string
		inputHex string
		expected uint16
	}{
		{
			name:     "empty",
			inputHex: "",
			expected: 0x0000,
		},
		{
			name:     "heartbeat without crc",
			inputHex: "556601010000000000",
			expected: 0x8B59, // wire little endian 59 8B
		},
		{
			name:     "sample firmware request",
			inputHex: "5566010000000001",
			expected: 0xC464, // wire little endian 64 C4
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := hex.DecodeString(tc.inputHex)
			if err != nil {
				t.Fatalf("failed to decode hex: %v", err)
			}
			actual := protocol.CRC16(b)
			if actual != tc.expected {
				t.Errorf("CRC16(%s) = 0x%04X; want 0x%04X", tc.inputHex, actual, tc.expected)
			}
		})
	}
}

func TestCRC16Check(t *testing.T) {
	// Heartbeat frame: 55 66 01 01 00 00 00 00 00 59 8B
	frameWithoutCRC, _ := hex.DecodeString("556601010000000000")
	crcLE, _ := hex.DecodeString("598b")

	if !protocol.CRC16Check(frameWithoutCRC, crcLE) {
		t.Errorf("expected CRC16Check to return true")
	}

	badCRC, _ := hex.DecodeString("0000")
	if protocol.CRC16Check(frameWithoutCRC, badCRC) {
		t.Errorf("expected CRC16Check with bad CRC to return false")
	}

	shortCRC := []byte{0x59}
	if protocol.CRC16Check(frameWithoutCRC, shortCRC) {
		t.Errorf("expected CRC16Check with short CRC to return false")
	}
}
