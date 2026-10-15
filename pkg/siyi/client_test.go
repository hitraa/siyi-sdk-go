package siyi_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/siyi"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

func TestClient_Lifecycle(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock)

	if client.IsConnected() {
		t.Errorf("expected disconnected initially")
	}

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !client.IsConnected() {
		t.Errorf("expected connected after Connect()")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if client.IsConnected() {
		t.Errorf("expected disconnected after Close()")
	}
}

func TestClient_UnsolicitedAttitudeStream(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock)
	_ = client.Connect(ctx)
	defer client.Close()

	var receivedAtt models.GimbalAttitude
	var wg sync.WaitGroup
	wg.Add(1)

	client.OnAttitude(func(att models.GimbalAttitude) {
		receivedAtt = att
		wg.Done()
	})

	// Push 0x0D frame: yaw=100 (10.0 deg), pitch=50 (5.0 deg), roll=0
	attPayload := []byte{
		0x64, 0x00, // 100 -> 10.0 deg
		0x32, 0x00, // 50 -> 5.0 deg
		0x00, 0x00, // 0 -> 0.0 deg
		0x00, 0x00,
		0x00, 0x00,
		0x00, 0x00,
	}
	attFrame := protocol.Build(protocol.CmdRequestGimbalAttitude, attPayload, 1, false).ToBytes()
	mock.QueueResponse(attFrame)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if receivedAtt.YawDeg != 10.0 || receivedAtt.PitchDeg != 5.0 {
			t.Errorf("received attitude unexpected: %+v", receivedAtt)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for attitude push callback")
	}
}

func TestClient_RequestResponse(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock, siyi.WithTimeout(1*time.Second))
	_ = client.Connect(ctx)
	defer client.Close()

	// Handle firmware version request in goroutine
	go func() {
		time.Sleep(20 * time.Millisecond)
		fwPayload := []byte{
			0x01, 0x02, 0x03, 0x00, // camera
			0x04, 0x05, 0x06, 0x00, // gimbal
			0x07, 0x08, 0x09, 0x00, // zoom
		}
		resp := protocol.Build(protocol.CmdRequestFirmwareVersion, fwPayload, 0, false).ToBytes()
		mock.QueueResponse(resp)
	}()

	fw, err := client.GetFirmwareVersion(ctx)
	if err != nil {
		t.Fatalf("GetFirmwareVersion failed: %v", err)
	}
	if fw.Camera != 0x00030201 || fw.Gimbal != 0x00060504 || fw.Zoom != 0x00090807 {
		t.Errorf("unexpected firmware version: %+v", fw)
	}
}

func TestClient_TimeoutAndRetry(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	// Timeout short 50ms, 2 retries
	client := siyi.NewClient(mock,
		siyi.WithTimeout(50*time.Millisecond),
		siyi.WithRetries(2),
		siyi.WithRetryBaseDelay(10*time.Millisecond),
	)
	_ = client.Connect(ctx)
	defer client.Close()

	// 0x01 is idempotent read -> should attempt 3 times total
	_, err := client.GetFirmwareVersion(ctx)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, siyi.ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}

	sent := mock.SentFrames()
	if len(sent) != 3 {
		t.Errorf("expected 3 attempts (1 initial + 2 retries), got %d", len(sent))
	}
}

func TestClient_FireAndForget(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock)
	_ = client.Connect(ctx)
	defer client.Close()

	// TakePhoto is fire-and-forget (CmdCapturePhotoRecordVideo)
	err := client.TakePhoto(ctx)
	if err != nil {
		t.Fatalf("TakePhoto error: %v", err)
	}

	sent := mock.SentFrames()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent frame, got %d", len(sent))
	}
	frame, err := protocol.FromBytes(sent[0])
	if err != nil {
		t.Fatalf("FromBytes error: %v", err)
	}
	if frame.CmdID != protocol.CmdCapturePhotoRecordVideo {
		t.Errorf("CmdID = %d, want CmdCapturePhotoRecordVideo", frame.CmdID)
	}
	if frame.Ctrl != 0 {
		t.Errorf("Ctrl = %d, want 0 for fire-and-forget", frame.Ctrl)
	}
}

func TestClient_CommandsComprehensive(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock, siyi.WithTimeout(1*time.Second))
	_ = client.Connect(ctx)
	defer client.Close()

	// 1. Hardware ID
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp := protocol.Build(protocol.CmdRequestHardwareID, []byte("730000000000"), 1, false).ToBytes()
		mock.QueueResponse(resp)
	}()
	hw, err := client.GetHardwareID(ctx)
	if err != nil {
		t.Fatalf("GetHardwareID error: %v", err)
	}
	pid, _ := hw.ProductID()
	if pid != models.ProductA8Mini {
		t.Errorf("ProductID = %v, want ProductA8Mini", pid)
	}

	// 2. Gimbal Mode
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp := protocol.Build(protocol.CmdRequestGimbalMode, []byte{byte(models.GimbalMotionModeFollow)}, 2, false).ToBytes()
		mock.QueueResponse(resp)
	}()
	gMode, err := client.GetGimbalMode(ctx)
	if err != nil || gMode != models.GimbalMotionModeFollow {
		t.Errorf("GetGimbalMode failed: %v", err)
	}

	// 3. Zoom
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp := protocol.Build(protocol.CmdRequestZoomRange, []byte{30, 0}, 3, false).ToBytes()
		mock.QueueResponse(resp)
	}()
	zr, err := client.GetZoomRange(ctx)
	if err != nil || zr.MaxZoom() != 30.0 {
		t.Errorf("GetZoomRange failed: %v", err)
	}

	// 4. IP
	go func() {
		time.Sleep(10 * time.Millisecond)
		ipBytes := []byte{192, 168, 144, 25, 255, 255, 255, 0, 192, 168, 144, 1}
		resp := protocol.Build(protocol.CmdGetIP, ipBytes, 4, false).ToBytes()
		mock.QueueResponse(resp)
	}()
	ipCfg, err := client.GetIP(ctx)
	if err != nil || !ipCfg.IP.Equal(net.ParseIP("192.168.144.25")) {
		t.Errorf("GetIP failed: %v", err)
	}
}

func TestClient_NoWaitCommands(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	if err := client.RotateNoWait(ctx, 20, -10); err != nil {
		t.Errorf("RotateNoWait failed: %v", err)
	}
	if err := client.SetGimbalAttitudeNoWait(ctx, 45.0, -15.0); err != nil {
		t.Errorf("SetGimbalAttitudeNoWait failed: %v", err)
	}
	if err := client.SetSingleAxisNoWait(ctx, 10.0, 0); err != nil {
		t.Errorf("SetSingleAxisNoWait failed: %v", err)
	}

	sent := mock.SentFrames()
	if len(sent) != 3 {
		t.Errorf("expected 3 sent frames, got %d", len(sent))
	}
}

func TestClient_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	mock := transport.NewMockTransport(false)
	client := siyi.NewClient(mock)
	_ = client.Connect(ctx)
	defer client.Close()

	var calls atomic.Int32
	unsub := client.OnAttitude(func(att models.GimbalAttitude) {
		calls.Add(1)
	})

	attPayload := []byte{
		0x64, 0x00, 0x32, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	attFrame := protocol.Build(protocol.CmdRequestGimbalAttitude, attPayload, 1, false).ToBytes()
	mock.QueueResponse(attFrame)
	time.Sleep(50 * time.Millisecond)

	if calls.Load() != 1 {
		t.Errorf("expected 1 call before unsubscribe, got %d", calls.Load())
	}

	unsub()

	mock.QueueResponse(attFrame)
	time.Sleep(50 * time.Millisecond)

	if calls.Load() != 1 {
		t.Errorf("expected still 1 call after unsubscribe, got %d", calls.Load())
	}
}

func TestClient_CreateStream(t *testing.T) {
	udp := transport.NewUDPTransport("192.168.144.25", 37260)
	client := siyi.NewClient(udp)
	s := client.CreateStream(models.StreamTypeMain, models.CameraGenerationNew)
	if s == nil {
		t.Fatal("expected non-nil stream from CreateStream")
	}
}

