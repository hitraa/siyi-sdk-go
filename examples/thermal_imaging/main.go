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

	client, err := siyi.ConnectUDP(ctx, protocol.DefaultIP, protocol.DefaultUDPPort)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// 1. Measure temperature at screen center point (320, 256 for 640x512 thermal core)
	fmt.Println("Querying temperature at center point (320, 256)...")
	point, err := client.GetTempAtPoint(ctx, 320, 256, models.TempMeasureOnce)
	if err != nil {
		log.Printf("GetTempAtPoint failed (camera may not be thermal): %v", err)
	} else {
		fmt.Printf("Point Temperature at (%d, %d): %.1f °C\n", point.X, point.Y, point.TemperatureC)
	}

	// 2. Measure regional temperature box (start: 100, 100, end: 500, 400)
	fmt.Println("Querying regional temperature box...")
	region, err := client.GetLocalTemp(ctx, 100, 100, 500, 400, models.TempMeasureOnce)
	if err != nil {
		log.Printf("GetLocalTemp failed: %v", err)
	} else {
		fmt.Printf("Regional Temperature: Min=%.1f°C at (%d,%d), Max=%.1f°C at (%d,%d)\n",
			region.MinC, region.MinX, region.MinY, region.MaxC, region.MaxX, region.MaxY)
	}

	// 3. Query global full-frame temperature
	fmt.Println("Querying global temperature...")
	global, err := client.GetGlobalTemp(ctx, models.TempMeasureOnce)
	if err != nil {
		log.Printf("GetGlobalTemp failed: %v", err)
	} else {
		fmt.Printf("Global Temperature: Min=%.1f°C at (%d,%d), Max=%.1f°C at (%d,%d)\n",
			global.MinC, global.MinX, global.MinY, global.MaxC, global.MaxX, global.MaxY)
	}

	// 4. Query & set thermal pseudo-color palette
	color, err := client.GetPseudoColor(ctx)
	if err == nil {
		fmt.Printf("Current Thermal PseudoColor: %d\n", color)
		fmt.Println("Switching pseudo color to White Hot...")
		_ = client.SetPseudoColor(ctx, models.PseudoColorWhiteHot)
	}

	// 5. Query thermal gain sensitivity
	gain, err := client.GetThermalGain(ctx)
	if err == nil {
		fmt.Printf("Current Thermal Gain: %d\n", gain)
	}
}
