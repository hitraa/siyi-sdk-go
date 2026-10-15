package stream_test

import (
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
)

func TestBuildRTSPURL(t *testing.T) {
	// Old generation
	oldURL := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeMain, models.CameraGenerationOld)
	if oldURL != "rtsp://192.168.144.25:8554/main.264" {
		t.Errorf("BuildRTSPURL old = %s, want rtsp://192.168.144.25:8554/main.264", oldURL)
	}

	// New generation main
	newMain := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeMain, models.CameraGenerationNew)
	if newMain != "rtsp://192.168.144.25:8554/video1" {
		t.Errorf("BuildRTSPURL new main = %s, want rtsp://192.168.144.25:8554/video1", newMain)
	}

	// New generation sub
	newSub := stream.BuildRTSPURL("192.168.144.25", models.StreamTypeSub, models.CameraGenerationNew)
	if newSub != "rtsp://192.168.144.25:8554/video2" {
		t.Errorf("BuildRTSPURL new sub = %s, want rtsp://192.168.144.25:8554/video2", newSub)
	}
}

func TestStream_Lifecycle(t *testing.T) {
	s := stream.New(stream.Config{
		RTSPURL: "rtsp://127.0.0.1:8554/video1",
	})
	if s.IsRunning() {
		t.Errorf("expected IsRunning = false initially")
	}
	_ = s.Stop()
}
