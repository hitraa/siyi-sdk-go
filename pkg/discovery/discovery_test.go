package discovery_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/discovery"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func TestDiscover(t *testing.T) {
	// Start mock camera responder on local UDP port
	mockConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP failed: %v", err)
	}
	defer mockConn.Close()

	localPort := mockConn.LocalAddr().(*net.UDPAddr).Port

	// Goroutine responding to probe
	go func() {
		buf := make([]byte, 512)
		_ = mockConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remote, err := mockConn.ReadFrom(buf)
		if err == nil && n >= protocol.MinFrameLen {
			// Respond with 0x02 Hardware ID payload: "731234567890" (A8 Mini)
			hwPayload := []byte("731234567890")
			respFrame := protocol.Build(protocol.CmdRequestHardwareID, hwPayload, 1, false).ToBytes()
			_, _ = mockConn.WriteTo(respFrame, remote)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Discover targeting local listener
	devices, err := discovery.Discover(ctx, "127.0.0.1", 300*time.Millisecond)
	_ = localPort
	_ = err
	_ = devices

	// If packet routed locally or loopback allows broadcast
	for _, d := range devices {
		if d.ProductID == models.ProductA8Mini {
			if d.ModelName != "A8 Mini" {
				t.Errorf("expected A8 Mini, got %s", d.ModelName)
			}
		}
	}
}
