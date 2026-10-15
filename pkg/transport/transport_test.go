package transport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

func TestMockTransport_Lifecycle(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)

	if mock.IsConnected() {
		t.Errorf("expected disconnected initially")
	}

	err := mock.Send(ctx, []byte{0x01})
	if !errors.Is(err, transport.ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}

	if err := mock.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !mock.IsConnected() {
		t.Errorf("expected connected after Connect()")
	}

	testData := []byte{0x55, 0x66, 0x01}
	if err := mock.Send(ctx, testData); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	sent := mock.SentFrames()
	if len(sent) != 1 || len(sent[0]) != 3 {
		t.Fatalf("SentFrames count mismatch: %d", len(sent))
	}

	// Queue response
	mock.QueueResponse([]byte{0xAA, 0xBB})
	select {
	case resp := <-mock.Stream():
		if len(resp) != 2 || resp[0] != 0xAA {
			t.Errorf("Stream received unexpected: %v", resp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timed out waiting for queued response")
	}

	// Queue error
	mockErr := errors.New("simulated error")
	mock.QueueError(mockErr)
	if err := mock.Send(ctx, []byte{0x02}); !errors.Is(err, mockErr) {
		t.Errorf("expected mockErr, got %v", err)
	}

	// Close
	if err := mock.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if mock.IsConnected() {
		t.Errorf("expected disconnected after Close()")
	}
}

func TestMockTransport_HeartbeatFlag(t *testing.T) {
	mockUDP := transport.NewMockTransport(false)
	if mockUDP.SupportsHeartbeat() {
		t.Errorf("expected SupportsHeartbeat = false")
	}

	mockTCP := transport.NewMockTransport(true)
	if !mockTCP.SupportsHeartbeat() {
		t.Errorf("expected SupportsHeartbeat = true")
	}
}
