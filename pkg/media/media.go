package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

const defaultBasePath = "/cgi-bin/media.cgi"

// Client is the HTTP client for the SIYI camera web server media API (port 82).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new media client.
func NewClient(host string, port ...int) *Client {
	if host == "" {
		host = protocol.DefaultIP
	}
	p := protocol.DefaultHTTPPort
	if len(port) > 0 && port[0] > 0 {
		p = port[0]
	}
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d%s", host, p, defaultBasePath),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) get(ctx context.Context, endpoint string, params url.Values, out interface{}) error {
	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create media request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("media request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from camera media API", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var res apiResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return fmt.Errorf("decode media API JSON: %w", err)
	}

	if !res.Success {
		msg := res.Message
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("camera media API error: %s", msg)
	}

	if out != nil && len(res.Data) > 0 {
		if err := json.Unmarshal(res.Data, out); err != nil {
			return fmt.Errorf("decode data payload: %w", err)
		}
	}

	return nil
}

// ListDirectories returns date-based directories for the given media type.
func (c *Client) ListDirectories(ctx context.Context, mediaType models.MediaType) ([]models.MediaDirectory, error) {
	params := url.Values{
		"media_type": []string{strconv.Itoa(int(mediaType))},
	}
	var data struct {
		Directories []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"directories"`
	}

	if err := c.get(ctx, "/api/v1/getdirectories", params, &data); err != nil {
		return nil, err
	}

	dirs := make([]models.MediaDirectory, len(data.Directories))
	for i, d := range data.Directories {
		dirs[i] = models.MediaDirectory{
			Name: d.Name,
			Path: d.Path,
		}
	}
	return dirs, nil
}

// GetMediaCount returns the number of files under path (or total if path is empty).
func (c *Client) GetMediaCount(ctx context.Context, mediaType models.MediaType, path string) (int, error) {
	params := url.Values{
		"media_type": []string{strconv.Itoa(int(mediaType))},
		"path":       []string{path},
	}
	var data struct {
		Count int `json:"count"`
	}
	if err := c.get(ctx, "/api/v1/getmediacount", params, &data); err != nil {
		return 0, err
	}
	return data.Count, nil
}

// ListFiles returns a paginated list of media files under path.
func (c *Client) ListFiles(ctx context.Context, mediaType models.MediaType, path string, start, count int) ([]models.MediaFile, error) {
	params := url.Values{
		"media_type": []string{strconv.Itoa(int(mediaType))},
		"path":       []string{path},
		"start":      []string{strconv.Itoa(start)},
		"count":      []string{strconv.Itoa(count)},
	}
	var data struct {
		List []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"list"`
	}
	if err := c.get(ctx, "/api/v1/getmedialist", params, &data); err != nil {
		return nil, err
	}
	files := make([]models.MediaFile, len(data.List))
	for i, f := range data.List {
		files[i] = models.MediaFile{
			Name: f.Name,
			URL:  f.URL,
		}
	}
	return files, nil
}

// ListAllFiles fetches all files under path by automatically paginating.
func (c *Client) ListAllFiles(ctx context.Context, mediaType models.MediaType, path string, pageSize ...int) ([]models.MediaFile, error) {
	ps := 50
	if len(pageSize) > 0 && pageSize[0] > 0 {
		ps = pageSize[0]
	}

	total, err := c.GetMediaCount(ctx, mediaType, path)
	if err != nil {
		return nil, err
	}

	var allFiles []models.MediaFile
	start := 0
	for start < total {
		batch, err := c.ListFiles(ctx, mediaType, path, start, ps)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		allFiles = append(allFiles, batch...)
		start += len(batch)
	}
	return allFiles, nil
}
