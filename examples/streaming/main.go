package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
	"github.com/pion/rtp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rtspURL := stream.BuildRTSPURL(protocol.DefaultIP, models.StreamTypeMain, models.CameraGenerationOld)
	fmt.Printf("Connecting to RTSP video stream at %s...\n", rtspURL)

	packetCount := 0
	startTime := time.Now()

	s := stream.New(stream.Config{
		RTSPURL: rtspURL,
		OnPacket: func(media *description.Media, forma format.Format, pkt *rtp.Packet) {
			packetCount++
			if packetCount%100 == 0 {
				fps := float64(packetCount) / time.Since(startTime).Seconds()
				fmt.Printf("[%s] Received %d RTP packets (%.1f packets/sec)\n",
					time.Now().Format("15:04:05"), packetCount, fps)
			}
		},
	})

	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start streaming: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = s.Stop() }()

	fmt.Println("Streaming started. Press Ctrl+C to terminate.")
	<-ctx.Done()
	fmt.Printf("Stream terminated. Total packets received: %d\n", packetCount)
}
