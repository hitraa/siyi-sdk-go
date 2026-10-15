# SIYI Camera RTSP Video Streaming (Go Implementation)

The `pkg/stream` package provides a pure Go RTSP stream receiver built on top of `github.com/bluenviron/gortsplib/v4`.

It does not require system GStreamer plugins or OpenCV CGo bindings; it compiles to static binaries and runs across Linux, macOS, and Windows.

---

## 1. Prerequisites

- Camera is reachable via IP network (default: `192.168.144.25`).
- Port `8554` is open on the camera (RTSP server is always running on the camera hardware).
- No command is needed to start RTSP streaming on the camera; connecting to the URL begins session negotiation.

---

## 2. Camera Generations and Stream URLs

SIYI cameras fall into two generations with different RTSP path conventions:

| Generation | Camera Models | Main Stream URL | Sub Stream URL |
|---|---|---|---|
| **Old Generation** | ZR30, ZR10, A8 mini, A2 mini, R1M | `rtsp://<ip>:8554/main.264` | *N/A* |
| **New Generation** | ZT30, ZT6 | `rtsp://<ip>:8554/video1` | `rtsp://<ip>:8554/video2` |

Use `stream.BuildRTSPURL`:

```go
urlOld := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeMain, models.CameraGenerationOld)
// "rtsp://192.168.144.25:8554/main.264"

urlNew := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeMain, models.CameraGenerationNew)
// "rtsp://192.168.144.25:8554/video1"

urlSub := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeSub, models.CameraGenerationNew)
// "rtsp://192.168.144.25:8554/video2"
```

---

## 3. Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
	"github.com/pion/rtp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rtspURL := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeMain, models.CameraGenerationNew)

	s := stream.New(stream.Config{
		RTSPURL:        rtspURL,
		ReconnectDelay: 2 * time.Second,
		OnPacket: func(media *description.Media, forma format.Format, pkt *rtp.Packet) {
			fmt.Printf("Received RTP packet: seq=%d, payload_bytes=%d, timestamp=%d\n",
				pkt.SequenceNumber, len(pkt.Payload), pkt.Timestamp)
		},
	})

	if err := s.Start(ctx); err != nil {
		log.Fatalf("Failed to start stream: %v", err)
	}

	time.Sleep(10 * time.Second)
	_ = s.Stop()
}
```

---

## 4. Automatic Reconnection & Resiliency

The `Stream` runner runs in an asynchronous loop. If the camera reboots, network drops, or frames temporarily disconnect, `Stream` waits for `ReconnectDelay` and automatically re-establishes RTSP negotiation, maintaining registered callbacks seamlessly.
