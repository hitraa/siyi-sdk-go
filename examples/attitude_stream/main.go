package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Register unsolicited attitude listener
	client.OnAttitude(func(att models.GimbalAttitude) {
		fmt.Printf("[%s] Telemetry: Yaw=%6.1f° | Pitch=%6.1f° | Roll=%6.1f° | YawRate=%5.1f°/s\n",
			time.Now().Format("15:04:05.000"), att.YawDeg, att.PitchDeg, att.RollDeg, att.YawRateDPS)
	})

	// Request camera to push attitude stream at 10Hz
	fmt.Println("Requesting 10Hz attitude telemetry push stream (Press Ctrl+C to exit)...")
	_, err = client.RequestGimbalStream(ctx, models.GimbalDataAttitude, models.DataStreamFreq10Hz)
	if err != nil {
		log.Fatalf("RequestGimbalStream failed: %v", err)
	}

	<-ctx.Done()
	fmt.Println("\nStopping telemetry stream...")
	_, _ = client.RequestGimbalStream(context.Background(), models.GimbalDataAttitude, models.DataStreamFreqOff)
}
