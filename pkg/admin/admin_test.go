package admin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/admin"
)

func TestAdminClient_NetworkAndReboot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/network":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ip":"192.168.144.25","netmask":"255.255.255.0","gateway":"192.168.144.1","dns":"8.8.8.8"}`))
				return
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				return
			}
		case "/api/v1/system/reboot":
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	parts := strings.Split(ts.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	client := admin.NewClient("127.0.0.1", port)
	ctx := context.Background()

	cfg, err := client.GetNetworkConfig(ctx)
	if err != nil {
		t.Fatalf("GetNetworkConfig failed: %v", err)
	}
	if cfg.IP != "192.168.144.25" {
		t.Errorf("expected IP 192.168.144.25, got %s", cfg.IP)
	}

	err = client.SetNetworkConfig(ctx, *cfg)
	if err != nil {
		t.Errorf("SetNetworkConfig failed: %v", err)
	}

	err = client.Reboot(ctx)
	if err != nil {
		t.Errorf("Reboot failed: %v", err)
	}
}
