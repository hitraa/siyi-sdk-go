package media_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/media"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
)

func TestMediaClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/media.cgi/api/v1/getdirectories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"success": true,
			"data": {
				"directories": [
					{"name": "2026-10-15", "path": "/DCIM/100SIYI"}
				]
			}
		}`))
	})

	mux.HandleFunc("/cgi-bin/media.cgi/api/v1/getmediacount", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"success": true,
			"data": {
				"count": 2
			}
		}`))
	})

	mux.HandleFunc("/cgi-bin/media.cgi/api/v1/getmedialist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"success": true,
			"data": {
				"list": [
					{"name": "IMG_0001.JPG", "url": "http://192.168.144.25:82/DCIM/100SIYI/IMG_0001.JPG"},
					{"name": "IMG_0002.JPG", "url": "http://192.168.144.25:82/DCIM/100SIYI/IMG_0002.JPG"}
				]
			}
		}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())

	client := media.NewClient(u.Hostname(), port)
	ctx := context.Background()

	// List directories
	dirs, err := client.ListDirectories(ctx, models.MediaTypeImages)
	if err != nil {
		t.Fatalf("ListDirectories failed: %v", err)
	}
	if len(dirs) != 1 || dirs[0].Name != "2026-10-15" {
		t.Errorf("ListDirectories unexpected: %+v", dirs)
	}

	// Count
	cnt, err := client.GetMediaCount(ctx, models.MediaTypeImages, dirs[0].Path)
	if err != nil || cnt != 2 {
		t.Errorf("GetMediaCount = %d, want 2", cnt)
	}

	// List all files
	files, err := client.ListAllFiles(ctx, models.MediaTypeImages, dirs[0].Path, 1)
	if err != nil || len(files) != 2 {
		t.Fatalf("ListAllFiles failed: %v, count=%d", err, len(files))
	}
	if files[0].Name != "IMG_0001.JPG" || files[1].Name != "IMG_0002.JPG" {
		t.Errorf("ListAllFiles unexpected files: %+v", files)
	}
}
