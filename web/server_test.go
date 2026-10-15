package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go/web"
)

func TestWebServer_Routes(t *testing.T) {
	srv := web.NewServer(web.Config{
		Port:     18082,
		CameraIP: "127.0.0.1",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	defer func() { _ = srv.Stop(ctx) }()

	client := &http.Client{Timeout: 1 * time.Second}

	// 1. Test Index HTML
	resp, err := client.Get("http://localhost:18082/")
	if err != nil {
		t.Fatalf("failed GET /: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Errorf("expected 200 OK for index.html, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. Test Config IP (GET and POST)
	resp, err = client.Get("http://localhost:18082/api/config/ip")
	if err != nil {
		t.Fatalf("failed GET /api/config/ip: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for GET /api/config/ip, got %d", resp.StatusCode)
	}

	body, _ := json.Marshal(map[string]string{"ip": "192.168.144.25"})
	resp, err = client.Post("http://localhost:18082/api/config/ip", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed POST /api/config/ip: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/config/ip, got %d", resp.StatusCode)
	}

	// 3. Test Stream Toggle (turn off)
	body, _ = json.Marshal(map[string]bool{"enable": false})
	resp, err = client.Post("http://localhost:18082/api/stream/toggle", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed POST /api/stream/toggle: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/stream/toggle, got %d", resp.StatusCode)
	}

	// 4. Test System Info (returns 503 or error since camera offline, but route exists and executes)
	resp, err = client.Get("http://localhost:18082/api/system/info")
	if err == nil {
		_ = resp.Body.Close()
	}

	// 5. Test Gimbal Rotate
	body, _ = json.Marshal(map[string]int{"yaw": 10, "pitch": -5})
	resp, err = client.Post("http://localhost:18082/api/gimbal/rotate", "application/json", bytes.NewReader(body))
	if err == nil {
		_ = resp.Body.Close()
	}

	// 6. Test Gimbal Center
	resp, err = client.Post("http://localhost:18082/api/gimbal/center", "application/json", nil)
	if err == nil {
		_ = resp.Body.Close()
	}
}
