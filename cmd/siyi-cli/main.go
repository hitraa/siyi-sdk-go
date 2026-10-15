package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
)

func main() {
	ip := flag.String("ip", protocol.DefaultIP, "Camera IP address")
	port := flag.Int("port", protocol.DefaultUDPPort, "UDP/TCP port")
	proto := flag.String("proto", "udp", "Protocol (udp, tcp)")
	cmd := flag.String("cmd", "info", "Command: info, attitude, center, rotate, photo, record, zoom, focus, laser, format, reboot, rtsp-url")

	// Command-specific parameters
	yaw := flag.Int("yaw", 0, "Yaw velocity (-100..100) for rotate")
	pitch := flag.Int("pitch", 0, "Pitch velocity (-100..100) for rotate")
	dir := flag.Int("dir", 1, "Direction: 1 (in/far), -1 (out/near), 0 (stop)")
	zoomVal := flag.Float64("zoom", 1.0, "Target zoom magnification for absolute zoom")
	touchX := flag.Int("touch-x", 960, "X coordinate for auto-focus")
	touchY := flag.Int("touch-y", 540, "Y coordinate for auto-focus")
	rebootTarget := flag.String("reboot-target", "both", "Reboot target: camera, gimbal, or both")

	flag.Parse()

	if *cmd == "rtsp-url" {
		urlOld := stream.BuildRTSPURL(*ip, models.StreamTypeMain, models.CameraGenerationOld)
		urlNewMain := stream.BuildRTSPURL(*ip, models.StreamTypeMain, models.CameraGenerationNew)
		urlNewSub := stream.BuildRTSPURL(*ip, models.StreamTypeSub, models.CameraGenerationNew)
		fmt.Printf("RTSP URLs for %s:\n", *ip)
		fmt.Printf("  Old-gen (ZR30/ZR10/A8/A2): %s\n", urlOld)
		fmt.Printf("  New-gen Main (ZT30/ZT6):   %s\n", urlNewMain)
		fmt.Printf("  New-gen Sub (ZT30/ZT6):    %s\n", urlNewSub)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var client *siyi.Client
	var err error

	if *proto == "tcp" {
		client, err = siyi.ConnectTCP(ctx, *ip, *port)
	} else {
		client, err = siyi.ConnectUDP(ctx, *ip, *port)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to %s:%d: %v\n", *ip, *port, err)
		os.Exit(1)
	}
	defer client.Close()

	switch *cmd {
	case "info":
		fw, err := client.GetFirmwareVersion(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetFirmwareVersion failed: %v\n", err)
			os.Exit(1)
		}
		hw, err := client.GetHardwareID(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetHardwareID failed: %v\n", err)
			os.Exit(1)
		}
		pid, _ := hw.ProductID()
		fmt.Printf("Connected to SIYI %s\n", pid.String())
		fmt.Printf("  Camera FW: %s (0x%08X)\n", models.FormatWord(fw.Camera), fw.Camera)
		fmt.Printf("  Gimbal FW: %s (0x%08X)\n", models.FormatWord(fw.Gimbal), fw.Gimbal)
		fmt.Printf("  Zoom FW:   %s (0x%08X)\n", models.FormatWord(fw.Zoom), fw.Zoom)

		camInfo, err := client.GetCameraSystemInfo(ctx)
		if err == nil {
			fmt.Printf("  Record State: %s\n", camInfo.RecordSta.String())
			fmt.Printf("  Gimbal Mode:  %s\n", camInfo.GimbalMotionMode.String())
		}

	case "attitude":
		att, err := client.GetGimbalAttitude(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetGimbalAttitude failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Gimbal Attitude:\n")
		fmt.Printf("  Yaw:   %.1f° (Rate: %.1f°/s)\n", att.YawDeg, att.YawRateDPS)
		fmt.Printf("  Pitch: %.1f° (Rate: %.1f°/s)\n", att.PitchDeg, att.PitchRateDPS)
		fmt.Printf("  Roll:  %.1f° (Rate: %.1f°/s)\n", att.RollDeg, att.RollRateDPS)

	case "center":
		if err := client.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter); err != nil {
			fmt.Fprintf(os.Stderr, "Centering failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Gimbal centered successfully.")

	case "rotate":
		if err := client.Rotate(ctx, *yaw, *pitch); err != nil {
			fmt.Fprintf(os.Stderr, "Rotate failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Rotate command dispatched (yaw: %d, pitch: %d).\n", *yaw, *pitch)

	case "photo":
		if err := client.TakePhoto(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Photo capture failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Photo capture command dispatched.")

	case "record":
		if err := client.StartRecording(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Record start/stop failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Record toggle command dispatched.")

	case "zoom":
		if *zoomVal > 1.0 {
			if err := client.AbsoluteZoom(ctx, *zoomVal); err != nil {
				fmt.Fprintf(os.Stderr, "Absolute zoom to %.1fx failed: %v\n", *zoomVal, err)
				os.Exit(1)
			}
			fmt.Printf("Absolute zoom set to %.1fx.\n", *zoomVal)
		} else {
			mag, err := client.ManualZoom(ctx, *dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Manual zoom failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Manual zoom dispatched (dir: %d). Current zoom: %.1fx.\n", *dir, mag)
		}

	case "focus":
		if *touchX > 0 && *touchY > 0 {
			if err := client.AutoFocus(ctx, 1, uint16(*touchX), uint16(*touchY)); err != nil {
				fmt.Fprintf(os.Stderr, "Auto focus failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Auto focus triggered at (%d, %d).\n", *touchX, *touchY)
		} else {
			if err := client.ManualFocus(ctx, *dir); err != nil {
				fmt.Fprintf(os.Stderr, "Manual focus failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Manual focus dispatched (dir: %d).\n", *dir)
		}

	case "laser":
		ld, err := client.GetLaserDistance(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetLaserDistance failed: %v\n", err)
			os.Exit(1)
		}
		if ld.DistanceM != nil {
			fmt.Printf("Laser Distance: %.1f m\n", *ld.DistanceM)
		} else {
			fmt.Println("Laser Distance: Out of range or no target.")
		}

		coords, err := client.GetLaserTargetLatLon(ctx)
		if err == nil && coords.LatE7 != 0 {
			fmt.Printf("Target Coordinates: Lat: %.6f, Lon: %.6f\n", coords.Latitude(), coords.Longitude())
		}

	case "format":
		ok, err := client.FormatSDCard(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Format SD Card failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Format SD card acknowledged: %v\n", ok)

	case "reboot":
		rebootCam := *rebootTarget == "camera" || *rebootTarget == "both"
		rebootGim := *rebootTarget == "gimbal" || *rebootTarget == "both"
		camOK, gimOK, err := client.SoftReboot(ctx, rebootCam, rebootGim)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Soft reboot failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Soft reboot dispatched: Camera OK=%v, Gimbal OK=%v\n", camOK, gimOK)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *cmd)
		os.Exit(1)
	}
}
