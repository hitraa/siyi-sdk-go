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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// 1. Query optical zoom range
	fmt.Println("Querying maximum optical zoom range...")
	zr, err := client.GetZoomRange(ctx)
	if err != nil {
		log.Printf("GetZoomRange failed: %v", err)
	} else {
		fmt.Printf("Maximum Optical Zoom: %.1fx (max_int: %d, max_float: %d)\n",
			zr.MaxZoom(), zr.MaxInt, zr.MaxFloat)
	}

	// 2. Query current zoom magnification
	fmt.Println("Querying current zoom magnification...")
	cz, err := client.GetCurrentZoom(ctx)
	if err != nil {
		log.Printf("GetCurrentZoom failed: %v", err)
	} else {
		fmt.Printf("Current Zoom Magnification: %.1fx\n", cz.Zoom())
	}

	// 3. Absolute zoom to 2.5x
	fmt.Println("Setting absolute zoom to 2.5x...")
	if err := client.AbsoluteZoom(ctx, 2.5); err != nil {
		log.Printf("AbsoluteZoom failed: %v", err)
	} else {
		fmt.Println("Absolute zoom set to 2.5x successfully.")
	}
	time.Sleep(2 * time.Second)

	// 4. Manual zoom step (zoom in, dir=1)
	fmt.Println("Triggering manual zoom in step...")
	currentZoom, err := client.ManualZoom(ctx, 1)
	if err != nil {
		log.Printf("ManualZoom failed: %v", err)
	} else {
		fmt.Printf("ManualZoom ACK: Current zoom = %.1fx\n", currentZoom)
	}
	time.Sleep(1 * time.Second)

	// Stop manual zoom
	_, _ = client.ManualZoom(ctx, 0)

	// 5. Trigger Auto Focus at screen coordinate (960, 540)
	fmt.Println("Triggering auto-focus at center screen (960, 540)...")
	if err := client.AutoFocus(ctx, 1, 960, 540); err != nil {
		log.Printf("AutoFocus failed: %v", err)
	} else {
		fmt.Println("AutoFocus triggered successfully.")
	}
	time.Sleep(2 * time.Second)

	// 6. Manual focus step (far, dir=1)
	fmt.Println("Triggering manual focus step (far)...")
	if err := client.ManualFocus(ctx, 1); err != nil {
		log.Printf("ManualFocus failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	_ = client.ManualFocus(ctx, 0) // Stop
}
