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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect over UDP (default 192.168.144.25:37260)
	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Query firmware version
	fw, err := client.GetFirmwareVersion(ctx)
	if err != nil {
		log.Printf("GetFirmwareVersion error: %v", err)
	} else {
		fmt.Printf("Camera FW: %s, Gimbal FW: %s, Zoom FW: %s\n",
			models.FormatWord(fw.Camera), models.FormatWord(fw.Gimbal), models.FormatWord(fw.Zoom))
	}

	// Query hardware identification
	hw, err := client.GetHardwareID(ctx)
	if err != nil {
		log.Printf("GetHardwareID error: %v", err)
	} else {
		pid, _ := hw.ProductID()
		fmt.Printf("Hardware ID: %s (Product: %s)\n", string(hw.Raw), pid.String())
	}

	// Query current attitude
	att, err := client.GetGimbalAttitude(ctx)
	if err != nil {
		log.Printf("GetGimbalAttitude error: %v", err)
	} else {
		fmt.Printf("Attitude: Yaw=%.1f° Pitch=%.1f° Roll=%.1f° (YawRate=%.1f°/s)\n",
			att.YawDeg, att.PitchDeg, att.RollDeg, att.YawRateDPS)
	}

	// Center camera
	fmt.Println("Recentering gimbal...")
	_ = client.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter)
}
