package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 1. Center the gimbal
	fmt.Println("Recentering gimbal...")
	if err := client.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter); err != nil {
		log.Printf("Centering failed: %v", err)
	}
	time.Sleep(1 * time.Second)

	// 2. Query initial attitude
	att, err := client.GetGimbalAttitude(ctx)
	if err == nil {
		fmt.Printf("Initial Attitude: Yaw=%.1f°, Pitch=%.1f°, Roll=%.1f°\n", att.YawDeg, att.PitchDeg, att.RollDeg)
	}

	// 3. Set absolute gimbal attitude (Yaw=30°, Pitch=-15°)
	fmt.Println("Setting attitude setpoint: Yaw=30°, Pitch=-15°...")
	ack, err := client.SetGimbalAttitude(ctx, 30.0, -15.0)
	if err != nil {
		log.Printf("SetGimbalAttitude failed: %v", err)
	} else {
		fmt.Printf("SetAttitude ACK: Current gimbal position: Yaw=%.1f°, Pitch=%.1f°\n", ack.YawDeg, ack.PitchDeg)
	}
	time.Sleep(2 * time.Second)

	// 4. Rate-based velocity rotation (pan right at 20% speed for 1 second)
	fmt.Println("Rotating yaw at rate 20 for 1 second...")
	_ = client.Rotate(ctx, 20, 0)
	time.Sleep(1 * time.Second)
	_ = client.Rotate(ctx, 0, 0) // Stop rotation
	time.Sleep(500 * time.Millisecond)

	// 5. High-rate servoing with RotateNoWait
	fmt.Println("Running high-rate servoing loop with RotateNoWait (50Hz)...")
	ticker := time.NewTicker(20 * time.Millisecond) // 50 Hz
	defer ticker.Stop()
	loopEnd := time.Now().Add(1 * time.Second)

	for range ticker.C {
		if time.Now().After(loopEnd) {
			break
		}
		_ = client.RotateNoWait(ctx, -15, 0)
	}
	_ = client.RotateNoWait(ctx, 0, 0) // Stop

	// 6. Recenter at end
	fmt.Println("Recentering gimbal...")
	_ = client.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter)
}
