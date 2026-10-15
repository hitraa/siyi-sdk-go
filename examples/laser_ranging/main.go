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
	defer client.Close()

	// 1. Enable laser rangefinder beam
	fmt.Println("Enabling laser rangefinder...")
	ok, err := client.SetLaserRangingState(ctx, true)
	if err != nil {
		log.Printf("SetLaserRangingState failed (model may lack laser): %v", err)
	} else {
		fmt.Printf("Laser rangefinder state changed: %v\n", ok)
	}

	time.Sleep(1 * time.Second)

	// 2. Query target distance
	fmt.Println("Querying laser target distance...")
	dist, err := client.GetLaserDistance(ctx)
	if err != nil {
		log.Printf("GetLaserDistance failed: %v", err)
	} else if dist.DistanceM != nil {
		fmt.Printf("Laser Target Distance: %.1f meters\n", *dist.DistanceM)
	} else {
		fmt.Println("Laser Target Distance: Out of range (or no reflection received).")
	}

	// 3. Query target geographic coordinates
	fmt.Println("Querying laser target coordinates...")
	coords, err := client.GetLaserTargetLatLon(ctx)
	if err != nil {
		log.Printf("GetLaserTargetLatLon failed: %v", err)
	} else if coords.LatE7 != 0 {
		fmt.Printf("Target Geographic Coordinates: Latitude=%.6f°, Longitude=%.6f°\n",
			coords.Latitude(), coords.Longitude())
	}

	// 4. Disable laser rangefinder when done
	fmt.Println("Disabling laser rangefinder...")
	_, _ = client.SetLaserRangingState(ctx, false)
}
