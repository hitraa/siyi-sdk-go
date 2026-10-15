package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// NetworkConfig represents camera IP network configuration on port 80.
type NetworkConfig struct {
	IP      string `json:"ip"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
	DNS     string `json:"dns"`
}

// VideoSettings represents camera video encoder and output settings.
type VideoSettings struct {
	MainResolution string `json:"main_resolution"` // e.g. "1920x1080", "1280x720"
	MainBitrate    int    `json:"main_bitrate"`    // kbps
	MainFPS        int    `json:"main_fps"`
	Codec          string `json:"codec"` // "H264" or "H265"
	HDMIEnabled    bool   `json:"hdmi_enabled"`
	CVBSEnabled    bool   `json:"cvbs_enabled"`
}

// Client communicates with the SIYI camera port 80 web administration server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new administration client for the camera on port 80.
func NewClient(host string, port ...int) *Client {
	if host == "" {
		host = protocol.DefaultIP
	}
	p := 80
	if len(port) > 0 && port[0] > 0 {
		p = port[0]
	}
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, p),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetNetworkConfig queries the camera's network settings.
func (c *Client) GetNetworkConfig(ctx context.Context) (*NetworkConfig, error) {
	reqURL := fmt.Sprintf("%s/api/v1/network", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network config request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from camera admin API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cfg NetworkConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		// Fallback for form-encoded or key-value responses
		return nil, fmt.Errorf("decode network config JSON: %w", err)
	}
	return &cfg, nil
}

// SetNetworkConfig applies new network configuration.
func (c *Client) SetNetworkConfig(ctx context.Context, cfg NetworkConfig) error {
	data := url.Values{
		"ip":      []string{cfg.IP},
		"netmask": []string{cfg.Netmask},
		"gateway": []string{cfg.Gateway},
		"dns":     []string{cfg.DNS},
	}

	reqURL := fmt.Sprintf("%s/api/v1/network", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("set network config failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from camera admin API", resp.StatusCode)
	}
	return nil
}

// Reboot triggers an administrative reboot of the camera system.
func (c *Client) Reboot(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/api/v1/system/reboot", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("reboot request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
