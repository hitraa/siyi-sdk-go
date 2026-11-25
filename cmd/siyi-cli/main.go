package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
)

type cliOpts struct {
	ip           string
	port         int
	proto        string
	device       string
	baud         int
	cmd          string
	yaw          int
	pitch        int
	yawDeg       float64
	pitchDeg     float64
	axis         int
	centerAction int
	mode         string
	captureFunc  string
	dir          int
	zoomVal      float64
	touchX       int
	touchY       int
	rebootTarget string
	set          bool
	on           bool
	streamType   int
	encType      string
	width        int
	height       int
	bitrate      int
	fileType     string
	nameType     string
	stitchMode   int
	newIP        string
	mask         string
	gateway      string
	pseudoColor  int
	thermalMode  int
	thermalGain  int
	shutter      bool
	weakThresh   int
}

func main() {
	var o cliOpts
	flag.StringVar(&o.ip, "ip", protocol.DefaultIP, "Camera IP address")
	flag.IntVar(&o.port, "port", protocol.DefaultUDPPort, "UDP/TCP port")
	flag.StringVar(&o.proto, "proto", "udp", "Protocol (udp, tcp, serial)")
	flag.StringVar(&o.device, "device", "/dev/ttyUSB0", "Serial device path for -proto serial")
	flag.IntVar(&o.baud, "baud", protocol.DefaultBaudRate, "Serial baud rate for -proto serial")
	flag.StringVar(&o.cmd, "cmd", "info", "Command: info, attitude, set-attitude, single-axis, center, rotate, mode, capture, photo, record, zoom, focus, encoding, pic-name, osd, stitching, ip-config, sys-time, sync-time, weak-control, laser, laser-state, ai, thermal, format, reboot, rtsp-url, discover, version")
	showVersion := flag.Bool("v", false, "Print version and exit")
	showVersionLong := flag.Bool("version", false, "Print version and exit")

	// Gimbal & camera control flags
	flag.IntVar(&o.yaw, "yaw", 0, "Yaw velocity (-100..100) for rotate")
	flag.IntVar(&o.pitch, "pitch", 0, "Pitch velocity (-100..100) for rotate")
	flag.Float64Var(&o.yawDeg, "yaw-deg", 0, "Target yaw angle in degrees for set-attitude / single-axis")
	flag.Float64Var(&o.pitchDeg, "pitch-deg", 0, "Target pitch angle in degrees for set-attitude")
	flag.IntVar(&o.axis, "axis", 0, "Axis for single-axis (0=yaw, 1=pitch)")
	flag.IntVar(&o.centerAction, "center-action", 1, "Centering action (1=one-key-center, 2=center+downward, 3=center, 4=downward)")
	flag.StringVar(&o.mode, "mode", "", "Gimbal mode to set: lock, follow, fpv")
	flag.StringVar(&o.captureFunc, "func", "photo", "Capture/function action: photo, hdr, record, lock, follow, fpv, hdmi, cvbs, disable-video, tilt-down, zoom-linkage")
	flag.IntVar(&o.dir, "dir", 1, "Direction: 1 (in/far), -1 (out/near), 0 (stop)")
	flag.Float64Var(&o.zoomVal, "zoom", 1.0, "Target zoom magnification for absolute zoom")
	flag.IntVar(&o.touchX, "touch-x", 960, "X coordinate for auto-focus")
	flag.IntVar(&o.touchY, "touch-y", 540, "Y coordinate for auto-focus")
	flag.StringVar(&o.rebootTarget, "reboot-target", "both", "Reboot target: camera, gimbal, or both")

	// Settings flags
	flag.BoolVar(&o.set, "set", false, "Apply setting instead of querying")
	flag.BoolVar(&o.on, "on", true, "Boolean toggle for osd, laser-state, weak-control, ai")
	flag.IntVar(&o.streamType, "stream", 1, "Stream type (0=recording, 1=main, 2=sub)")
	flag.StringVar(&o.encType, "enc", "h264", "Video codec: h264 or h265")
	flag.IntVar(&o.width, "width", 1920, "Encoding width (e.g. 1920, 1280, 3840)")
	flag.IntVar(&o.height, "height", 1080, "Encoding height (e.g. 1080, 720, 2160)")
	flag.IntVar(&o.bitrate, "bitrate", 4000, "Encoding bitrate in kbps")
	flag.StringVar(&o.fileType, "file-type", "picture", "File type for pic-name: picture, temp, video")
	flag.StringVar(&o.nameType, "name-type", "index", "File name type for pic-name: index, timestamp")
	flag.IntVar(&o.stitchMode, "stitch", -1, "Video stitching mode (0..8) to set")
	flag.StringVar(&o.newIP, "new-ip", "", "Static IP to set with ip-config -set")
	flag.StringVar(&o.mask, "mask", "255.255.255.0", "Subnet mask to set with ip-config -set")
	flag.StringVar(&o.gateway, "gateway", "192.168.144.1", "Gateway to set with ip-config -set")
	flag.IntVar(&o.pseudoColor, "palette", -1, "Thermal pseudo-color palette ID (0..10) to set")
	flag.IntVar(&o.thermalMode, "thermal-mode", -1, "Thermal output mode (0=30fps, 1=25fps+temp) to set")
	flag.IntVar(&o.thermalGain, "thermal-gain", -1, "Thermal gain (0=low, 1=high) to set")
	flag.BoolVar(&o.shutter, "shutter", false, "Trigger manual thermal shutter calibration")
	flag.IntVar(&o.weakThresh, "weak-threshold", -1, "Weak control threshold to set")

	flag.Parse()

	if *showVersion || *showVersionLong || o.cmd == "version" {
		fmt.Printf("siyi-cli version %s\n", siyi.Version)
		return
	}

	if o.cmd == "discover" {
		runDiscover()
		return
	}

	if o.cmd == "rtsp-url" {
		fmt.Printf("RTSP URLs for %s:\n", o.ip)
		fmt.Printf("  Old-gen (ZR30/ZR10/A8/A2): %s\n", stream.BuildRTSPURL(o.ip, models.StreamTypeMain, models.CameraGenerationOld))
		fmt.Printf("  New-gen Main (ZT30/ZT6):   %s\n", stream.BuildRTSPURL(o.ip, models.StreamTypeMain, models.CameraGenerationNew))
		fmt.Printf("  New-gen Sub (ZT30/ZT6):    %s\n", stream.BuildRTSPURL(o.ip, models.StreamTypeSub, models.CameraGenerationNew))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var client *siyi.Client
	var err error
	switch strings.ToLower(o.proto) {
	case "tcp":
		client, err = siyi.ConnectTCP(ctx, o.ip, o.port)
	case "serial":
		client, err = siyi.ConnectSerial(ctx, o.device, o.baud)
	default:
		client, err = siyi.ConnectUDP(ctx, o.ip, o.port)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to %s:%d: %v\n", o.ip, o.port, err)
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	if err := dispatchCommand(ctx, client, o); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runDiscover() {
	fmt.Println("Probing local network for SIYI cameras (UDP 37260 broadcast)...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	devices, err := siyi.Discover(ctx, "255.255.255.255", 2*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Println("No SIYI cameras discovered.")
		return
	}
	fmt.Printf("Discovered %d SIYI camera(s):\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] Model: %-15s IP: %-16s Port: %-5d HWID: %s\n",
			i+1, d.ModelName, d.IP, d.Port, d.HardwareID)
	}
}

func dispatchCommand(ctx context.Context, client *siyi.Client, o cliOpts) error {
	switch o.cmd {
	case "info":
		return cmdInfo(ctx, client)
	case "attitude":
		att, err := client.GetGimbalAttitude(ctx)
		if err != nil {
			return fmt.Errorf("GetGimbalAttitude failed: %w", err)
		}
		fmt.Printf("Gimbal Attitude:\n  Yaw:   %.1f° (Rate: %.1f°/s)\n  Pitch: %.1f° (Rate: %.1f°/s)\n  Roll:  %.1f° (Rate: %.1f°/s)\n",
			att.YawDeg, att.YawRateDPS, att.PitchDeg, att.PitchRateDPS, att.RollDeg, att.RollRateDPS)
	case "set-attitude":
		ack, err := client.SetGimbalAttitude(ctx, o.yawDeg, o.pitchDeg)
		if err != nil {
			return fmt.Errorf("SetGimbalAttitude failed: %w", err)
		}
		fmt.Printf("Target attitude set (yaw=%.1f°, pitch=%.1f°). Current: yaw=%.1f°, pitch=%.1f°, roll=%.1f°\n",
			o.yawDeg, o.pitchDeg, ack.YawDeg, ack.PitchDeg, ack.RollDeg)
	case "single-axis":
		ack, err := client.SetSingleAxis(ctx, o.yawDeg, o.axis)
		if err != nil {
			return fmt.Errorf("SetSingleAxis failed: %w", err)
		}
		fmt.Printf("Single-axis angle set (axis=%d, angle=%.1f°). Current: yaw=%.1f°, pitch=%.1f°, roll=%.1f°\n",
			o.axis, o.yawDeg, ack.YawDeg, ack.PitchDeg, ack.RollDeg)
	case "center":
		if err := client.OneKeyCentering(ctx, models.CenteringAction(o.centerAction)); err != nil {
			return fmt.Errorf("Centering failed: %w", err)
		}
		fmt.Printf("Gimbal centering action %s succeeded.\n", models.CenteringAction(o.centerAction))
	case "rotate":
		if err := client.Rotate(ctx, o.yaw, o.pitch); err != nil {
			return fmt.Errorf("Rotate failed: %w", err)
		}
		fmt.Printf("Rotate command dispatched (yaw: %d, pitch: %d).\n", o.yaw, o.pitch)
	case "mode":
		return cmdMode(ctx, client, o.mode)
	case "capture":
		return cmdCapture(ctx, client, o.captureFunc)
	case "photo":
		if err := client.TakePhoto(ctx); err != nil {
			return fmt.Errorf("Photo capture failed: %w", err)
		}
		fmt.Println("Photo capture command dispatched.")
	case "record":
		if err := client.StartRecording(ctx); err != nil {
			return fmt.Errorf("Record start/stop failed: %w", err)
		}
		fmt.Println("Record toggle command dispatched.")
	case "zoom":
		if o.zoomVal > 1.0 {
			if err := client.AbsoluteZoom(ctx, o.zoomVal); err != nil {
				return fmt.Errorf("Absolute zoom to %.1fx failed: %w", o.zoomVal, err)
			}
			fmt.Printf("Absolute zoom set to %.1fx.\n", o.zoomVal)
		} else {
			mag, err := client.ManualZoom(ctx, o.dir)
			if err != nil {
				return fmt.Errorf("Manual zoom failed: %w", err)
			}
			fmt.Printf("Manual zoom dispatched (dir: %d). Current zoom: %.1fx.\n", o.dir, mag)
		}
	case "focus":
		if o.touchX > 0 && o.touchY > 0 {
			if err := client.AutoFocus(ctx, 1, uint16(o.touchX), uint16(o.touchY)); err != nil {
				return fmt.Errorf("Auto focus failed: %w", err)
			}
			fmt.Printf("Auto focus triggered at (%d, %d).\n", o.touchX, o.touchY)
		} else {
			if err := client.ManualFocus(ctx, o.dir); err != nil {
				return fmt.Errorf("Manual focus failed: %w", err)
			}
			fmt.Printf("Manual focus dispatched (dir: %d).\n", o.dir)
		}
	case "encoding":
		return cmdEncoding(ctx, client, o)
	case "pic-name":
		return cmdPicName(ctx, client, o)
	case "osd":
		return cmdOSD(ctx, client, o)
	case "stitching":
		return cmdStitching(ctx, client, o)
	case "ip-config":
		return cmdIPConfig(ctx, client, o)
	case "sys-time":
		st, err := client.GetSystemTime(ctx)
		if err != nil {
			return fmt.Errorf("GetSystemTime failed: %w", err)
		}
		fmt.Printf("System Time: unix_usec=%d (%s), boot_ms=%d\n",
			st.UnixUsec, time.UnixMicro(int64(st.UnixUsec)).UTC().Format(time.RFC3339), st.BootMS)
	case "sync-time":
		ok, err := client.SetUTCTime(ctx, uint64(time.Now().UTC().UnixMicro()))
		if err != nil {
			return fmt.Errorf("SetUTCTime failed: %w", err)
		}
		fmt.Printf("UTC time synchronized: %v\n", ok)
	case "weak-control":
		return cmdWeakControl(ctx, client, o)
	case "laser":
		ld, err := client.GetLaserDistance(ctx)
		if err != nil {
			return fmt.Errorf("GetLaserDistance failed: %v", err)
		}
		if ld.DistanceM != nil {
			fmt.Printf("Laser Distance: %.1f m\n", *ld.DistanceM)
		} else {
			fmt.Println("Laser Distance: Out of range or no target.")
		}
		if coords, err := client.GetLaserTargetLatLon(ctx); err == nil && coords.LatE7 != 0 {
			fmt.Printf("Target Coordinates: Lat: %.6f, Lon: %.6f\n", coords.Latitude(), coords.Longitude())
		}
	case "laser-state":
		ok, err := client.SetLaserRangingState(ctx, o.on)
		if err != nil {
			return fmt.Errorf("SetLaserRangingState failed: %w", err)
		}
		fmt.Printf("Laser ranging set to %v (ack=%v)\n", o.on, ok)
	case "ai":
		return cmdAI(ctx, client, o)
	case "thermal":
		return cmdThermal(ctx, client, o)
	case "format":
		ok, err := client.FormatSDCard(ctx)
		if err != nil {
			return fmt.Errorf("Format SD Card failed: %w", err)
		}
		fmt.Printf("Format SD card acknowledged: %v\n", ok)
	case "reboot":
		rebootCam := o.rebootTarget == "camera" || o.rebootTarget == "both"
		rebootGim := o.rebootTarget == "gimbal" || o.rebootTarget == "both"
		camOK, gimOK, err := client.SoftReboot(ctx, rebootCam, rebootGim)
		if err != nil {
			return fmt.Errorf("Soft reboot failed: %w", err)
		}
		fmt.Printf("Soft reboot dispatched: Camera OK=%v, Gimbal OK=%v\n", camOK, gimOK)
	default:
		return fmt.Errorf("Unknown command: %s", o.cmd)
	}
	return nil
}

func cmdInfo(ctx context.Context, client *siyi.Client) error {
	fw, err := client.GetFirmwareVersion(ctx)
	if err != nil {
		return fmt.Errorf("GetFirmwareVersion failed: %w", err)
	}
	hw, err := client.GetHardwareID(ctx)
	if err != nil {
		return fmt.Errorf("GetHardwareID failed: %w", err)
	}
	pid, _ := hw.ProductID()
	fmt.Printf("Connected to SIYI %s\n", pid.String())
	fmt.Printf("  Camera FW: %s (0x%08X)\n", models.FormatWord(fw.Camera), fw.Camera)
	fmt.Printf("  Gimbal FW: %s (0x%08X)\n", models.FormatWord(fw.Gimbal), fw.Gimbal)
	fmt.Printf("  Zoom FW:   %s (0x%08X)\n", models.FormatWord(fw.Zoom), fw.Zoom)

	if camInfo, err := client.GetCameraSystemInfo(ctx); err == nil {
		fmt.Printf("  Record State: %s\n", camInfo.RecordSta.String())
		fmt.Printf("  Gimbal Mode:  %s\n", camInfo.GimbalMotionMode.String())
		fmt.Printf("  Mounting Dir: %s\n", camInfo.GimbalMountingDir.String())
		fmt.Printf("  HDMI/CVBS:    %d\n", camInfo.VideoHDMICVBS)
	}
	return nil
}

func cmdMode(ctx context.Context, client *siyi.Client, mode string) error {
	if mode == "" {
		m, err := client.GetGimbalMode(ctx)
		if err != nil {
			return fmt.Errorf("GetGimbalMode failed: %w", err)
		}
		fmt.Printf("Gimbal Mode: %s (%d)\n", m.String(), m)
		return nil
	}
	var fn models.CaptureFuncType
	switch strings.ToLower(mode) {
	case "lock":
		fn = models.CaptureFuncLockMode
	case "follow":
		fn = models.CaptureFuncFollowMode
	case "fpv":
		fn = models.CaptureFuncFPVMode
	default:
		return fmt.Errorf("unknown mode %q (expected lock, follow, or fpv)", mode)
	}
	if err := client.Capture(ctx, fn); err != nil {
		return fmt.Errorf("Set gimbal mode failed: %w", err)
	}
	fmt.Printf("Gimbal mode set to %s.\n", mode)
	return nil
}

func cmdCapture(ctx context.Context, client *siyi.Client, action string) error {
	actions := map[string]models.CaptureFuncType{
		"photo":         models.CaptureFuncPhoto,
		"hdr":           models.CaptureFuncHDRToggle,
		"record":        models.CaptureFuncStartRecord,
		"lock":          models.CaptureFuncLockMode,
		"follow":        models.CaptureFuncFollowMode,
		"fpv":           models.CaptureFuncFPVMode,
		"hdmi":          models.CaptureFuncEnableHDMI,
		"cvbs":          models.CaptureFuncEnableCVBS,
		"disable-video": models.CaptureFuncDisableHDMICVBS,
		"tilt-down":     models.CaptureFuncTiltDownward,
		"zoom-linkage":  models.CaptureFuncZoomLinkage,
	}
	fn, ok := actions[strings.ToLower(action)]
	if !ok {
		return fmt.Errorf("unknown capture action %q", action)
	}
	if err := client.Capture(ctx, fn); err != nil {
		return fmt.Errorf("Capture(%s) failed: %w", action, err)
	}
	fmt.Printf("Capture action %q dispatched.\n", action)
	return nil
}

func cmdEncoding(ctx context.Context, client *siyi.Client, o cliOpts) error {
	st := models.StreamType(o.streamType)
	if !o.set {
		enc, err := client.GetEncodingParams(ctx, st)
		if err != nil {
			return fmt.Errorf("GetEncodingParams failed: %w", err)
		}
		fmt.Printf("Encoding (%s): Codec=%d Resolution=%dx%d Bitrate=%d kbps FPS=%d\n",
			enc.StreamType.String(), enc.EncType, enc.ResolutionW, enc.ResolutionH, enc.BitrateKbps, enc.FrameRateFPS)
		return nil
	}
	encType := models.VideoEncH264
	if strings.EqualFold(o.encType, "h265") {
		encType = models.VideoEncH265
	}
	ok, err := client.SetEncodingParams(ctx, models.EncodingParams{
		StreamType:  st,
		EncType:     encType,
		ResolutionW: uint16(o.width),
		ResolutionH: uint16(o.height),
		BitrateKbps: uint16(o.bitrate),
	})
	if err != nil {
		return fmt.Errorf("SetEncodingParams failed: %w", err)
	}
	fmt.Printf("SetEncodingParams acknowledged: %v\n", ok)
	return nil
}

func cmdPicName(ctx context.Context, client *siyi.Client, o cliOpts) error {
	ft := models.FileTypePicture
	switch strings.ToLower(o.fileType) {
	case "temp":
		ft = models.FileTypeTempRaw
	case "video":
		ft = models.FileTypeRecordVideo
	}
	if !o.set {
		nt, err := client.GetPicNameType(ctx, ft)
		if err != nil {
			return fmt.Errorf("GetPicNameType failed: %w", err)
		}
		fmt.Printf("File Name Type (%s): %d (1=Index, 2=Timestamp)\n", o.fileType, nt)
		return nil
	}
	nt := models.FileNameTypeIndex
	if strings.EqualFold(o.nameType, "timestamp") {
		nt = models.FileNameTypeTimestamp
	}
	if err := client.SetPicNameType(ctx, ft, nt); err != nil {
		return fmt.Errorf("SetPicNameType failed: %w", err)
	}
	fmt.Printf("SetPicNameType (%s -> %s) succeeded.\n", o.fileType, o.nameType)
	return nil
}

func cmdOSD(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if !o.set {
		on, err := client.GetMavlinkOSDFlag(ctx)
		if err != nil {
			return fmt.Errorf("GetMavlinkOSDFlag failed: %w", err)
		}
		fmt.Printf("MAVLink/HDMI OSD Enabled: %v\n", on)
		return nil
	}
	ok, err := client.SetMavlinkOSDFlag(ctx, o.on)
	if err != nil {
		return fmt.Errorf("SetMavlinkOSDFlag failed: %w", err)
	}
	fmt.Printf("SetMavlinkOSDFlag(%v) acknowledged: %v\n", o.on, ok)
	return nil
}

func cmdStitching(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if o.stitchMode < 0 {
		m, err := client.GetVideoStitchingMode(ctx)
		if err != nil {
			return fmt.Errorf("GetVideoStitchingMode failed: %w", err)
		}
		fmt.Printf("Video Stitching Mode: %d\n", m)
		return nil
	}
	m, err := client.SetVideoStitchingMode(ctx, models.VideoStitchingMode(o.stitchMode))
	if err != nil {
		return fmt.Errorf("SetVideoStitchingMode failed: %w", err)
	}
	fmt.Printf("Video Stitching Mode set to: %d\n", m)
	return nil
}

func cmdIPConfig(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if !o.set {
		cfg, err := client.GetIP(ctx)
		if err != nil {
			return fmt.Errorf("GetIP failed: %w", err)
		}
		fmt.Printf("IP Config: IP=%s Mask=%s Gateway=%s\n", cfg.IP, cfg.Mask, cfg.Gateway)
		return nil
	}
	if o.newIP == "" {
		return fmt.Errorf("-new-ip is required with -cmd ip-config -set")
	}
	ok, err := client.SetIP(ctx, models.IPConfig{
		IP:      net.ParseIP(o.newIP),
		Mask:    net.ParseIP(o.mask),
		Gateway: net.ParseIP(o.gateway),
	})
	if err != nil {
		return fmt.Errorf("SetIP failed: %w", err)
	}
	fmt.Printf("SetIP acknowledged: %v\n", ok)
	return nil
}

func cmdWeakControl(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if o.set {
		ok, err := client.SetWeakControlMode(ctx, o.on)
		if err != nil {
			return fmt.Errorf("SetWeakControlMode failed: %w", err)
		}
		fmt.Printf("Weak Control Mode set to %v (ack=%v)\n", o.on, ok)
	} else if on, err := client.GetWeakControlMode(ctx); err == nil {
		fmt.Printf("Weak Control Mode: %v\n", on)
	} else {
		fmt.Printf("Weak Control Mode: error (%v)\n", err)
	}

	if o.weakThresh >= 0 {
		ok, err := client.SetWeakThreshold(ctx, models.WeakControlThreshold{Limit: float64(o.weakThresh)})
		if err != nil {
			return fmt.Errorf("SetWeakThreshold failed: %w", err)
		}
		fmt.Printf("Weak Threshold set to %d (ack=%v)\n", o.weakThresh, ok)
	} else if t, err := client.GetWeakThreshold(ctx); err == nil {
		fmt.Printf("Weak Threshold: %+v\n", t)
	}

	if cm, err := client.GetControlMode(ctx); err == nil {
		fmt.Printf("Control Mode: %d\n", cm)
	}
	if mv, err := client.GetMotorVoltage(ctx); err == nil {
		fmt.Printf("Motor Voltage: %+v\n", mv)
	}
	return nil
}

func cmdAI(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if o.set {
		ok, err := client.SetAIStreamOutput(ctx, o.on)
		if err != nil {
			return fmt.Errorf("SetAIStreamOutput failed: %w", err)
		}
		fmt.Printf("AI Stream Output set to %v (ack=%v)\n", o.on, ok)
		return nil
	}
	mode, err := client.GetAIMode(ctx)
	if err != nil {
		return fmt.Errorf("GetAIMode failed: %w", err)
	}
	fmt.Printf("AI Mode Active: %v\n", mode)
	if st, err := client.GetAIStreamStatus(ctx); err == nil {
		fmt.Printf("AI Stream Status: %+v\n", st)
	}
	return nil
}

func cmdThermal(ctx context.Context, client *siyi.Client, o cliOpts) error {
	if o.shutter {
		if err := client.ManualThermalShutter(ctx); err != nil {
			return fmt.Errorf("ManualThermalShutter failed: %w", err)
		}
		fmt.Println("Manual thermal shutter calibration triggered.")
		return nil
	}
	if o.pseudoColor >= 0 {
		if err := client.SetPseudoColor(ctx, models.PseudoColor(o.pseudoColor)); err != nil {
			return fmt.Errorf("SetPseudoColor failed: %w", err)
		}
		fmt.Printf("Pseudo-color set to %d.\n", o.pseudoColor)
	} else if pc, err := client.GetPseudoColor(ctx); err == nil {
		fmt.Printf("Pseudo-Color: %d\n", pc)
	}
	if o.thermalMode >= 0 {
		if err := client.SetThermalOutputMode(ctx, models.ThermalOutputMode(o.thermalMode)); err != nil {
			return fmt.Errorf("SetThermalOutputMode failed: %w", err)
		}
		fmt.Printf("Thermal output mode set to %d.\n", o.thermalMode)
	} else if tm, err := client.GetThermalOutputMode(ctx); err == nil {
		fmt.Printf("Thermal Output Mode: %d\n", tm)
	}
	if o.thermalGain >= 0 {
		if err := client.SetThermalGain(ctx, models.ThermalGain(o.thermalGain)); err != nil {
			return fmt.Errorf("SetThermalGain failed: %w", err)
		}
		fmt.Printf("Thermal gain set to %d.\n", o.thermalGain)
	} else if tg, err := client.GetThermalGain(ctx); err == nil {
		fmt.Printf("Thermal Gain: %d\n", tg)
	}
	if tg, err := client.GetGlobalTemp(ctx, models.TempMeasureOnce); err == nil {
		fmt.Printf("Global Temp: Min=%.2f°C Max=%.2f°C\n", tg.MinC, tg.MaxC)
	}
	return nil
}
