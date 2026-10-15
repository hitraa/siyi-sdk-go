# Camera Media HTTP API (Port 82)

SIYI gimbal cameras run a lightweight HTTP web server on port **82** for browsing, listing, and downloading photos and videos recorded on the TF card. This HTTP API operates independently of the binary UDP/TCP command protocol.

The `pkg/media` package provides an idiomatic Go client for this API.

---

## Connection Endpoints

| Property           | Value                                        |
| ------------------ | -------------------------------------------- |
| **Default Host**   | `192.168.144.25`                             |
| **Port**           | `82`                                         |
| **Base URL**       | `http://192.168.144.25:82/cgi-bin/media.cgi` |
| **Transport**      | HTTP/1.1 GET with query string parameters    |
| **Authentication** | None required                                |

---

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/media"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create media client
	client := media.NewClient(protocol.DefaultIP, protocol.DefaultHTTPPort)

	// 1. List date-based image directories
	dirs, err := client.ListDirectories(ctx, models.MediaTypeImages)
	if err != nil {
		log.Fatalf("ListDirectories failed: %v", err)
	}

	for _, d := range dirs {
		count, _ := client.GetMediaCount(ctx, models.MediaTypeImages, d.Path)
		fmt.Printf("Directory: %s (%s) — %d photos\n", d.Name, d.Path, count)
	}

	// 2. Fetch first batch of files
	if len(dirs) > 0 {
		files, err := client.ListFiles(ctx, models.MediaTypeImages, dirs[0].Path, 0, 10)
		if err != nil {
			log.Fatalf("ListFiles failed: %v", err)
		}
		for _, f := range files {
			fmt.Printf("File: %s -> %s\n", f.Name, f.URL)
		}
	}
}
```

---

## API Methods

### `NewClient(host string, port ...int) *Client`

Constructs a new media HTTP client. If `host` is empty, defaults to `192.168.144.25`. If `port` is omitted, defaults to `82`.

### `ListDirectories(ctx context.Context, mediaType models.MediaType) ([]models.MediaDirectory, error)`

Returns all date-based directories for the given media type (`models.MediaTypeImages` or `models.MediaTypeVideos`).

### `GetMediaCount(ctx context.Context, mediaType models.MediaType, path string) (int, error)`

Returns total file count under `path`. Pass an empty string `""` to query the total file count across all directories.

### `ListFiles(ctx context.Context, mediaType models.MediaType, path string, start, count int) ([]models.MediaFile, error)`

Returns a paginated slice of files starting at index `start` up to `count` items. Each `models.MediaFile` contains:

- `Name`: File name (e.g. `SIYI_0001.JPG`)
- `URL`: Direct HTTP download URL (e.g. `http://192.168.144.25:82/photo/20260315/SIYI_0001.JPG`)

### `ListAllFiles(ctx context.Context, mediaType models.MediaType, path string, pageSize ...int) ([]models.MediaFile, error)`

Convenience method that queries the media count and automatically paginates until all files in `path` are retrieved.
