package protocol_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func TestFrameParser_SingleFrame(t *testing.T) {
	parser := protocol.NewFrameParser()
	frameHex := "556601010000000000598b"
	b, _ := hex.DecodeString(frameHex)

	frames, err := parser.Feed(b)
	if err != nil {
		t.Fatalf("unexpected Feed error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}

	f := frames[0]
	if f.CmdID != 0x00 {
		t.Errorf("CmdID = %d, want 0x00", f.CmdID)
	}
	if f.Ctrl != 0x01 {
		t.Errorf("Ctrl = %d, want 0x01", f.Ctrl)
	}
	if !bytes.Equal(f.Data, []byte{0x00}) {
		t.Errorf("Data = %v, want [0]", f.Data)
	}
}

func TestFrameParser_ChunkedFeed(t *testing.T) {
	parser := protocol.NewFrameParser()
	frameHex := "556601010000000000598b"
	b, _ := hex.DecodeString(frameHex)

	// Feed 1 byte at a time
	var allFrames []protocol.Frame
	for i := 0; i < len(b); i++ {
		frames, err := parser.Feed([]byte{b[i]})
		if err != nil {
			t.Fatalf("unexpected Feed error at index %d: %v", i, err)
		}
		allFrames = append(allFrames, frames...)
	}

	if len(allFrames) != 1 {
		t.Fatalf("expected 1 frame across byte-by-byte feed, got %d", len(allFrames))
	}
	if allFrames[0].CmdID != 0x00 {
		t.Errorf("CmdID = %d, want 0x00", allFrames[0].CmdID)
	}
}

func TestFrameParser_MultipleConcatenatedFrames(t *testing.T) {
	parser := protocol.NewFrameParser()

	f1 := protocol.Build(0x01, []byte{0xAA}, 1, true).ToBytes()
	f2 := protocol.Build(0x02, []byte{0xBB, 0xCC}, 2, false).ToBytes()

	stream := append(f1, f2...)
	frames, err := parser.Feed(stream)
	if err != nil {
		t.Fatalf("unexpected Feed error: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0].CmdID != 0x01 || frames[1].CmdID != 0x02 {
		t.Errorf("frame CmdIDs unexpected: %d, %d", frames[0].CmdID, frames[1].CmdID)
	}
}

func TestFrameParser_ResyncOnGarbage(t *testing.T) {
	parser := protocol.NewFrameParser()

	validFrame := protocol.Build(0x01, []byte{0x10}, 10, true).ToBytes()
	garbage := []byte{0x00, 0xFF, 0x55, 0x12, 0x55, 0x55} // garbage with false STX

	stream := append(garbage, validFrame...)
	frames, err := parser.Feed(stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame after garbage resync, got %d", len(frames))
	}
	if frames[0].CmdID != 0x01 {
		t.Errorf("CmdID = %d, want 0x01", frames[0].CmdID)
	}
}

func TestFrameParser_OversizedPayload(t *testing.T) {
	parser := protocol.NewFrameParser(64) // limit to 64 bytes

	// Construct header with 100 bytes payload length
	oversized := []byte{0x55, 0x66, 0x01, 100, 0x00, 0x00, 0x00, 0x01}
	_, err := parser.Feed(oversized)
	if err == nil {
		t.Fatalf("expected error for oversized payload")
	}
	if _, ok := err.(*protocol.FramingError); !ok {
		t.Errorf("expected *protocol.FramingError, got %T", err)
	}
}

func TestFrameParser_CRCMismatch(t *testing.T) {
	parser := protocol.NewFrameParser()
	frame := protocol.Build(0x01, []byte{0x00}, 1, true).ToBytes()
	frame[len(frame)-1] ^= 0xFF // corrupt CRC

	_, err := parser.Feed(frame)
	if err == nil {
		t.Fatalf("expected error for corrupted CRC")
	}
	if _, ok := err.(*protocol.CRCError); !ok {
		t.Errorf("expected *protocol.CRCError, got %T", err)
	}
}
