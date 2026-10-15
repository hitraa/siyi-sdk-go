# Quickstart Guide

This guide walks you through connecting to a SIYI gimbal camera, sending commands, receiving telemetry streams, capturing media, and using the CLI and Web UI.

---

## 1. Installation

```bash
go get github.com/hitraa/siyi-sdk-go
```

Requires **Go 1.22+**.

---

## 2. Choosing a Transport

The SDK supports three physical transports and one testing mock:

| Transport | Function | Use Case |
|---|---|---|
| **UDP** | `siyi.ConnectUDP(ctx, ip, port, opts...)` | Low-latency network control (Ethernet/WiFi). Recommended default. |
| **TCP** | `siyi.ConnectTCP(ctx, ip, port, opts...)` | Reliable stream over TCP; automatically runs 1Hz keepalive heartbeat. |
| **Serial**| `siyi.ConnectSerial(ctx, device, baud, opts...)` | Direct UART RS-232 link (e.g. `/dev/ttyUSB0` at 115200 baud). |
| **Mock** | `transport.NewMockTransport()` | In-memory unit and integration testing without hardware. |

---

## 3. Basic Example: Query & Centering

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
)

func main() {
	ctx := context.Background()

	// Connect over UDP
	client, err := siyi.ConnectUDP(ctx, "192.168.144.25", 37260,
		siyi.WithTimeout(2*time.Second),
		siyi.WithRetries(2),
	)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer client.Close()

	// Query Hardware ID
	hw, err := client.GetHardwareID(ctx)
	if err == nil {
		pid, _ := hw.ProductID()
		fmt.Printf("Connected Camera: %s (ID: %s)\n", pid.String(), string(hw.Raw))
	}

	// Query Firmware Versions
	fw, err := client.GetFirmwareVersion(ctx)
	if err == nil {
		fmt.Printf("Camera FW: %s, Gimbal FW: %s, Zoom FW: %s\n",
			models.FormatWord(fw.Camera), models.FormatWord(fw.Gimbal), models.FormatWord(fw.Zoom))
	}

	// Recentering
	fmt.Println("Recentering gimbal...")
	_ = client.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter)
}
```

---

## 4. Attitude Monitoring & Push Subscriptions

SIYI cameras can stream continuous attitude updates. Subscribe using `OnAttitude`:

```go
unsub := client.OnAttitude(func(att models.GimbalAttitude) {
    fmt.Printf("Attitude Push: Yaw=%.1f°, Pitch=%.1f°, Roll=%.1f° (YawRate=%.1f°/s)\n",
        att.YawDeg, att.PitchDeg, att.RollDeg, att.YawRateDPS)
})
defer unsub()

// Request continuous gimbal stream at 10Hz
_, err := client.RequestGimbalStream(ctx, models.GimbalDataTypeAttitude, models.DataStreamFreq10Hz)
if err != nil {
    log.Fatalf("RequestGimbalStream failed: %v", err)
}

time.Sleep(10 * time.Second)
```

---

## 5. Gimbal Control & Servoing

### Absolute Position Setpoints

```go
// Set absolute Yaw=45° and Pitch=-20°
ack, err := client.SetGimbalAttitude(ctx, 45.0, -20.0)
if err != nil {
    log.Fatalf("SetGimbalAttitude failed: %v", err)
}
fmt.Printf("Current position reported by ACK: Yaw=%.1f°, Pitch=%.1f°\n", ack.YawDeg, ack.PitchDeg)
```

### Velocity Rotation (Rate Control)

```go
// Pan right at 25% speed, tilt down at 10% speed
_ = client.Rotate(ctx, 25, -10)
time.Sleep(1 * time.Second)
_ = client.Rotate(ctx, 0, 0) // Stop
```

### High-Rate Servoing (50-100Hz Loops)

For visual tracking loops where round-trip ACK latency would cause frame drops, use `NoWait` methods:

```go
ticker := time.NewTicker(20 * time.Millisecond) // 50 Hz
defer ticker.Stop()

for range ticker.C {
    // Send rate command without waiting for ACK response
    _ = client.RotateNoWait(ctx, targetYawRate, targetPitchRate)
}
```

---

## 6. Photo & Video Recording

```go
// Take a still photo
_ = client.TakePhoto(ctx)

// Start recording video
_ = client.StartRecording(ctx)

time.Sleep(10 * time.Second)

// Stop recording video
_ = client.StopRecording(ctx)
```

---

## 7. Optical Zoom & Focus

```go
// Check maximum zoom
zr, _ := client.GetZoomRange(ctx)
fmt.Printf("Max Optical Zoom: %.1fx\n", zr.MaxZoom())

// Absolute zoom to 3.0x
_ = client.AbsoluteZoom(ctx, 3.0)

// Touch-to-focus at screen center
_ = client.AutoFocus(ctx, 1, 960, 540)
```

---

## 8. CLI Tool Usage (`siyi-cli`)

Inspect and control the camera from your terminal:

```bash
# Query system information
go run ./cmd/siyi-cli -cmd info

# Query live attitude
go run ./cmd/siyi-cli -cmd attitude

# Recenter gimbal
go run ./cmd/siyi-cli -cmd center

# Rotate gimbal
go run ./cmd/siyi-cli -cmd rotate -yaw 25 -pitch -10

# Take a photo
go run ./cmd/siyi-cli -cmd photo

# Toggle recording
go run ./cmd/siyi-cli -cmd record

# Set zoom
go run ./cmd/siyi-cli -cmd zoom -zoom 3.5

# Query laser rangefinder
go run ./cmd/siyi-cli -cmd laser

# Print RTSP video stream URLs
go run ./cmd/siyi-cli -cmd rtsp-url
```

---

## 9. Web UI Dashboard (`siyi-web`)

Launch the embedded dashboard:

```bash
go run ./cmd/siyi-web -ip 192.168.144.25 -port 8080
```

Open `http://localhost:8080` in your web browser.
