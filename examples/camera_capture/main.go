package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 1. Take a still photo
	fmt.Println("Taking a photo...")
	if err := client.TakePhoto(ctx); err != nil {
		log.Fatalf("Photo capture failed: %v", err)
	}
	fmt.Println("Photo captured successfully.")

	// Wait 2 seconds before recording
	time.Sleep(2 * time.Second)

	// 2. Start recording video
	fmt.Println("Starting video recording...")
	if err := client.StartRecording(ctx); err != nil {
		log.Fatalf("Start recording failed: %v", err)
	}
	fmt.Println("Recording started. Recording for 5 seconds...")

	time.Sleep(5 * time.Second)

	// 3. Stop recording video
	fmt.Println("Stopping video recording...")
	if err := client.StopRecording(ctx); err != nil {
		log.Fatalf("Stop recording failed: %v", err)
	}
	fmt.Println("Recording stopped successfully.")
}
