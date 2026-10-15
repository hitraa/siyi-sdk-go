package stream

import (
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// BuildRTSPURL constructs the RTSP stream URL for the specified gimbal camera.
// Old generation cameras (ZR30/ZR10/A8Mini/A2Mini/R1M) stream on /main.264.
// New generation cameras (ZT30/ZT6/etc.) stream on /video1 (main) and /video2 (sub).
func BuildRTSPURL(host string, stream models.StreamType, gen models.CameraGeneration) string {
	if host == "" {
		host = protocol.DefaultIP
	}

	if gen == models.CameraGenerationOld {
		return fmt.Sprintf("rtsp://%s:8554/main.264", host)
	}

	path := "video1"
	if stream == models.StreamTypeSub {
		path = "video2"
	}
	return fmt.Sprintf("rtsp://%s:8554/%s", host, path)
}
