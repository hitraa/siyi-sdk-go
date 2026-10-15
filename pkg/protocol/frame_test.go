package protocol_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func TestFrame_Serialization(t *testing.T) {
	// Frame: ctrl=1, seq=0, cmd=0x00, data=empty
	// Expected wire: 5566010000000000598b (wait, in python heartbeat: ctrl=1, data_len=1, data=b'\x00' -> 556601010000000000598b)
	frame := protocol.Frame{
		Ctrl:  0x01,
		Seq:   0x0000,
		CmdID: 0x00,
		Data:  []byte{0x00},
	}

	encoded := frame.ToBytes()
	expectedHex := "556601010000000000598b"
	if hex.EncodeToString(encoded) != expectedHex {
		t.Fatalf("ToBytes() = %s, want %s", hex.EncodeToString(encoded), expectedHex)
	}

	decoded, err := protocol.FromBytes(encoded)
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if decoded.Ctrl != frame.Ctrl {
		t.Errorf("Ctrl = %d, want %d", decoded.Ctrl, frame.Ctrl)
	}
	if decoded.Seq != frame.Seq {
		t.Errorf("Seq = %d, want %d", decoded.Seq, frame.Seq)
	}
	if decoded.CmdID != frame.CmdID {
		t.Errorf("CmdID = %d, want %d", decoded.CmdID, frame.CmdID)
	}
	if !bytes.Equal(decoded.Data, frame.Data) {
		t.Errorf("Data = %v, want %v", decoded.Data, frame.Data)
	}
}

func TestFrame_Build(t *testing.T) {
	frame := protocol.Build(0x01, []byte{0x01, 0x02}, 42, true)
	if frame.Ctrl != protocol.CTRLNeedACK {
		t.Errorf("Ctrl = %d, want %d", frame.Ctrl, protocol.CTRLNeedACK)
	}
	if frame.CmdID != 0x01 {
		t.Errorf("CmdID = %d, want 0x01", frame.CmdID)
	}
	if frame.Seq != 42 {
		t.Errorf("Seq = %d, want 42", frame.Seq)
	}
	if len(frame.Data) != 2 || frame.Data[0] != 0x01 || frame.Data[1] != 0x02 {
		t.Errorf("Data unexpected: %v", frame.Data)
	}

	frameNoACK := protocol.Build(0x0C, []byte{0x00}, 10, false)
	if frameNoACK.Ctrl != 0 {
		t.Errorf("Ctrl = %d, want 0", frameNoACK.Ctrl)
	}
}

func TestFrame_FromBytes_Errors(t *testing.T) {
	// Truncated (less than MinFrameLen)
	_, err := protocol.FromBytes([]byte{0x55, 0x66, 0x00})
	if err == nil {
		t.Errorf("expected error for truncated frame")
	}

	// Bad STX
	badSTX := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err = protocol.FromBytes(badSTX)
	if err == nil {
		t.Errorf("expected error for bad STX")
	}

	// Bad CRC
	badCRC, _ := hex.DecodeString("556601010000000000ffff")
	_, err = protocol.FromBytes(badCRC)
	if err == nil {
		t.Errorf("expected CRCError for corrupted CRC")
	}
	if _, ok := err.(*protocol.CRCError); !ok {
		t.Errorf("expected *protocol.CRCError, got %T", err)
	}

	// Truncated payload
	truncatedPayload, _ := hex.DecodeString("556601050000000000") // claims 5 bytes data but only 1 byte given
	_, err = protocol.FromBytes(truncatedPayload)
	if err == nil {
		t.Errorf("expected error for truncated payload")
	}
}
