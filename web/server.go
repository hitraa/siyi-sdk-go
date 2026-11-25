package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hitraa/siyi-sdk-go/pkg/media"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/siyi"
	"github.com/hitraa/siyi-sdk-go/pkg/stream"
	"github.com/hitraa/siyi-sdk-go/web/ui"
)

// Server is the web UI control server for SIYI gimbal cameras.
type Server struct {
	port       int
	cameraIP   string
	logger     *slog.Logger
	httpServer *http.Server

	mu          sync.RWMutex
	client      *siyi.Client
	mediaClient *media.Client
	stream      *stream.Stream
	connected   bool

	attMu    sync.RWMutex
	attitude map[string]float64

	frameMu     sync.RWMutex
	latestFrame []byte
	frameSubs   map[chan struct{}]struct{}
	videoCancel context.CancelFunc
	rtspURL     string

	upgrader websocket.Upgrader
}

// Config holds options for Server.
type Config struct {
	Port     int
	CameraIP string
	Logger   *slog.Logger
}

// NewServer creates a new Web UI Server.
func NewServer(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 8082
	}
	if cfg.CameraIP == "" {
		cfg.CameraIP = protocol.DefaultIP
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Server{
		port:        cfg.Port,
		cameraIP:    cfg.CameraIP,
		logger:      cfg.Logger,
		mediaClient: media.NewClient(cfg.CameraIP),
		attitude:    map[string]float64{"yaw": 0, "pitch": 0, "roll": 0},
		frameSubs:   make(map[chan struct{}]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Initialize connects to camera and starts background telemetry/streaming.
func (s *Server) Initialize(ctx context.Context, ip string) error {
	client, err := siyi.ConnectUDP(ctx, ip, protocol.DefaultUDPPort)
	if err != nil {
		s.logger.Warn("initial connection failed, watchdog will retry", "ip", ip, "err", err)
	}

	medCli := media.NewClient(ip)
	gen := models.CameraGenerationOld
	if client != nil {
		hwCtx, hwCancel := context.WithTimeout(ctx, 1*time.Second)
		if hw, hwErr := client.GetHardwareID(hwCtx); hwErr == nil {
			if pid, pidErr := hw.ProductID(); pidErr == nil {
				gen = pid.Generation()
			}
		}
		hwCancel()
	}
	rtspURL := stream.BuildRTSPURL(ip, models.StreamTypeMain, gen)
	str := stream.New(stream.Config{
		RTSPURL: rtspURL,
	})

	s.mu.Lock()
	s.cameraIP = ip
	s.rtspURL = rtspURL
	oldClient := s.client
	oldStream := s.stream
	s.client = client
	s.mediaClient = medCli
	s.stream = str
	s.connected = (err == nil)
	s.mu.Unlock()

	s.stopVideoPipeline()
	if oldClient != nil {
		_ = oldClient.Close()
	}
	if oldStream != nil {
		_ = oldStream.Stop()
	}

	if client != nil {
		client.OnAttitude(func(att models.GimbalAttitude) {
			s.attMu.Lock()
			s.attitude["yaw"] = att.YawDeg
			s.attitude["pitch"] = att.PitchDeg
			s.attitude["roll"] = att.RollDeg
			s.attMu.Unlock()
		})
		go func() {
			streamCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			_, _ = client.RequestGimbalStream(streamCtx, models.GimbalDataAttitude, models.DataStreamFreq10Hz)
		}()
	}

	_ = str.Start(context.Background())
	s.startVideoPipeline(rtspURL)
	return nil
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		_ = s.Initialize(ctx, s.cameraIP)
	}()

	mux := http.NewServeMux()

	// Static UI assets
	staticFS, err := fs.Sub(ui.FS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Web dashboard root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := ui.FS.ReadFile("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	// REST API Endpoints
	mux.HandleFunc("/api/config/ip", s.handleConfigIP)
	mux.HandleFunc("/api/gimbal/rotate", s.handleGimbalRotate)
	mux.HandleFunc("/api/gimbal/center", s.handleGimbalCenter)
	mux.HandleFunc("/api/gimbal/mode", s.handleGimbalMode)
	mux.HandleFunc("/api/gimbal/attitude", s.handleGimbalAttitude)
	mux.HandleFunc("/api/gimbal/weak", s.handleGimbalWeak)
	mux.HandleFunc("/api/camera/photo", s.handleCameraPhoto)
	mux.HandleFunc("/api/camera/record", s.handleCameraRecord)
	mux.HandleFunc("/api/camera/zoom", s.handleCameraZoom)
	mux.HandleFunc("/api/camera/focus", s.handleCameraFocus)
	mux.HandleFunc("/api/camera/encoding", s.handleCameraEncoding)
	mux.HandleFunc("/api/camera/picname", s.handleCameraPicName)
	mux.HandleFunc("/api/camera/osd", s.handleCameraOSD)
	mux.HandleFunc("/api/camera/stitching", s.handleCameraStitching)
	mux.HandleFunc("/api/camera/capture", s.handleCameraCapture)
	mux.HandleFunc("/api/storage/format", s.handleStorageFormat)
	mux.HandleFunc("/api/system/reboot", s.handleSystemReboot)
	mux.HandleFunc("/api/system/info", s.handleSystemInfo)
	mux.HandleFunc("/api/system/ip", s.handleSystemIP)
	mux.HandleFunc("/api/system/time", s.handleSystemTime)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/laser", s.handleLaser)
	mux.HandleFunc("/api/thermal", s.handleThermal)
	mux.HandleFunc("/api/ai", s.handleAI)
	mux.HandleFunc("/api/media/directories", s.handleMediaDirectories)
	mux.HandleFunc("/api/media/files", s.handleMediaFiles)
	mux.HandleFunc("/api/media/download", s.handleMediaDownload)
	mux.HandleFunc("/api/media/download-all", s.handleMediaDownloadAll)
	mux.HandleFunc("/api/stream/toggle", s.handleStreamToggle)
	mux.HandleFunc("/ws/attitude", s.handleWebSocketAttitude)
	mux.HandleFunc("/api/stream/video", s.handleStreamVideo)

	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	srv := s.httpServer
	s.mu.Unlock()

	s.logger.Info("Web UI server listening", "port", s.port)
	return srv.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.stopVideoPipeline()
	s.mu.Lock()
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.stream != nil {
		_ = s.stream.Stop()
	}
	srv := s.httpServer
	s.mu.Unlock()

	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleConfigIP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			IP string `json:"ip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go func() {
			_ = s.Initialize(context.Background(), req.IP)
		}()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "ip": req.IP})
		return
	}

	s.mu.RLock()
	ip := s.cameraIP
	connected := s.connected
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ip": ip, "connected": connected})
}

func (s *Server) handleGimbalRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Yaw   int `json:"yaw"`
		Pitch int `json:"pitch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client != nil {
		_ = client.Rotate(r.Context(), req.Yaw, req.Pitch)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleGimbalCenter(w http.ResponseWriter, r *http.Request) {
	action := models.CenteringActionOneKeyCenter
	if q := r.URL.Query().Get("action"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 1 && v <= 4 {
			action = models.CenteringAction(v)
		}
	}
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_ = client.OneKeyCentering(r.Context(), action)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleCameraPhoto(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_ = client.TakePhoto(r.Context())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleCameraRecord(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_ = client.Capture(r.Context(), models.CaptureFuncStartRecord)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleCameraZoom(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if absStr := r.URL.Query().Get("absolute"); absStr != "" {
		if absVal, err := strconv.ParseFloat(absStr, 64); err == nil && client != nil {
			_ = client.AbsoluteZoom(r.Context(), absVal)
		}
	} else {
		dir, _ := strconv.Atoi(r.URL.Query().Get("direction"))
		if client != nil {
			_, _ = client.ManualZoom(r.Context(), dir)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleCameraFocus(w http.ResponseWriter, r *http.Request) {
	dir, _ := strconv.Atoi(r.URL.Query().Get("direction"))
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_ = client.ManualFocus(r.Context(), dir)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleCameraEncoding(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if r.Method == http.MethodPost {
		var req struct {
			StreamType  int    `json:"stream_type"`
			EncType     int    `json:"enc_type"`
			Resolution  string `json:"resolution"`
			BitrateKbps int    `json:"bitrate_kbps"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if client != nil {
			parts := strings.Split(req.Resolution, "x")
			wPx, hPx := uint16(1920), uint16(1080)
			if len(parts) == 2 {
				wVal, _ := strconv.Atoi(parts[0])
				hVal, _ := strconv.Atoi(parts[1])
				wPx, hPx = uint16(wVal), uint16(hVal)
			}
			st := models.StreamTypeMain
			if req.StreamType >= 0 && req.StreamType <= 2 {
				st = models.StreamType(req.StreamType)
			}
			enc := models.VideoEncH264
			if req.EncType == int(models.VideoEncH265) {
				enc = models.VideoEncH265
			}
			_, _ = client.SetEncodingParams(r.Context(), models.EncodingParams{
				StreamType:   st,
				EncType:      enc,
				ResolutionW:  wPx,
				ResolutionH:  hPx,
				BitrateKbps:  uint16(req.BitrateKbps),
				FrameRateFPS: 30,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	if client == nil {
		http.Error(w, "camera disconnected", http.StatusServiceUnavailable)
		return
	}
	st := models.StreamTypeMain
	if q := r.URL.Query().Get("stream"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 0 && v <= 2 {
			st = models.StreamType(v)
		}
	}
	params, err := client.GetEncodingParams(r.Context(), st)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"stream_type":  params.StreamType.String(),
		"enc_type":     params.EncType,
		"resolution":   fmt.Sprintf("%dx%d", params.ResolutionW, params.ResolutionH),
		"bitrate_kbps": params.BitrateKbps,
		"frame_rate":   params.FrameRateFPS,
	})
}

func (s *Server) handleStorageFormat(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_, _ = client.FormatSDCard(r.Context())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleSystemReboot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Camera bool `json:"camera"`
		Gimbal bool `json:"gimbal"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		cam, gim, _ := client.SoftReboot(r.Context(), req.Camera, req.Gimbal)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "camera": cam, "gimbal": gim})
		return
	}
	http.Error(w, "camera disconnected", http.StatusServiceUnavailable)
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		http.Error(w, "camera disconnected", http.StatusServiceUnavailable)
		return
	}
	hw, _ := client.GetHardwareID(r.Context())
	fw, _ := client.GetFirmwareVersion(r.Context())
	pid, _ := hw.ProductID()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"camera_type": pid.String(),
		"camera_fw":   models.FormatWord(fw.Camera),
		"gimbal_fw":   models.FormatWord(fw.Gimbal),
		"zoom_fw":     models.FormatWord(fw.Zoom),
	})
}

func (s *Server) getClientOrError(w http.ResponseWriter) *siyi.Client {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client == nil {
		http.Error(w, "camera disconnected", http.StatusServiceUnavailable)
		return nil
	}
	return client
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleGimbalMode(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var fn models.CaptureFuncType
		switch strings.ToLower(req.Mode) {
		case "lock":
			fn = models.CaptureFuncLockMode
		case "follow":
			fn = models.CaptureFuncFollowMode
		case "fpv":
			fn = models.CaptureFuncFPVMode
		default:
			http.Error(w, "invalid mode", http.StatusBadRequest)
			return
		}
		if err := client.Capture(r.Context(), fn); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]string{"status": "ok", "mode": req.Mode})
		return
	}
	mode, err := client.GetGimbalMode(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]interface{}{"mode": mode.String(), "code": mode})
}

func (s *Server) handleGimbalAttitude(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Yaw   float64 `json:"yaw"`
			Pitch float64 `json:"pitch"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ack, err := client.SetGimbalAttitude(r.Context(), req.Yaw, req.Pitch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, ack)
		return
	}
	att, err := client.GetGimbalAttitude(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, att)
}

func (s *Server) handleGimbalWeak(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := client.SetWeakControlMode(r.Context(), req.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok, "enabled": req.Enabled})
		return
	}
	weakOn, err := client.GetWeakControlMode(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	resp := map[string]interface{}{"weak_mode": weakOn}
	if cm, err := client.GetControlMode(r.Context()); err == nil {
		resp["control_mode"] = cm
	}
	if mv, err := client.GetMotorVoltage(r.Context()); err == nil {
		resp["motor_voltage"] = mv
	}
	writeJSON(w, resp)
}

func (s *Server) handleCameraPicName(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			FileType int `json:"file_type"`
			NameType int `json:"name_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := client.SetPicNameType(r.Context(), models.FileType(req.FileType), models.FileNameType(req.NameType)); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}
	ft := models.FileTypePicture
	if q := r.URL.Query().Get("file_type"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			ft = models.FileType(v)
		}
	}
	nt, err := client.GetPicNameType(r.Context(), ft)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]interface{}{"file_type": ft, "name_type": nt})
}

func (s *Server) handleCameraOSD(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := client.SetMavlinkOSDFlag(r.Context(), req.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok, "enabled": req.Enabled})
		return
	}
	on, err := client.GetMavlinkOSDFlag(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]bool{"enabled": on})
}

func (s *Server) handleCameraStitching(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Mode int `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode, err := client.SetVideoStitchingMode(r.Context(), models.VideoStitchingMode(req.Mode))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "mode": mode})
		return
	}
	mode, err := client.GetVideoStitchingMode(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]interface{}{"mode": mode})
}

func (s *Server) handleCameraCapture(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	var req struct {
		Func int `json:"func"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := client.Capture(r.Context(), models.CaptureFuncType(req.Func)); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleSystemIP(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			IP      string `json:"ip"`
			Mask    string `json:"mask"`
			Gateway string `json:"gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := client.SetIP(r.Context(), models.IPConfig{
			IP:      net.ParseIP(req.IP),
			Mask:    net.ParseIP(req.Mask),
			Gateway: net.ParseIP(req.Gateway),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok})
		return
	}
	cfg, err := client.GetIP(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{
		"ip":      cfg.IP.String(),
		"mask":    cfg.Mask.String(),
		"gateway": cfg.Gateway.String(),
	})
}

func (s *Server) handleSystemTime(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		ok, err := client.SetUTCTime(r.Context(), uint64(time.Now().UTC().UnixMicro()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok})
		return
	}
	st, err := client.GetSystemTime(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, st)
}

func (s *Server) handleLaser(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := client.SetLaserRangingState(r.Context(), req.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok, "enabled": req.Enabled})
		return
	}
	ld, err := client.GetLaserDistance(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	resp := map[string]interface{}{"distance_m": ld.DistanceM}
	if coords, err := client.GetLaserTargetLatLon(r.Context()); err == nil {
		resp["lat"] = coords.Latitude()
		resp["lon"] = coords.Longitude()
	}
	writeJSON(w, resp)
}

func (s *Server) handleThermal(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			PseudoColor *int `json:"pseudo_color"`
			OutputMode  *int `json:"output_mode"`
			Gain        *int `json:"gain"`
			Shutter     bool `json:"shutter"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Shutter {
			if err := client.ManualThermalShutter(r.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		if req.PseudoColor != nil {
			if err := client.SetPseudoColor(r.Context(), models.PseudoColor(*req.PseudoColor)); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		if req.OutputMode != nil {
			if err := client.SetThermalOutputMode(r.Context(), models.ThermalOutputMode(*req.OutputMode)); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		if req.Gain != nil {
			if err := client.SetThermalGain(r.Context(), models.ThermalGain(*req.Gain)); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}
	pc, err := client.GetPseudoColor(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	resp := map[string]interface{}{"pseudo_color": pc}
	if tm, err := client.GetThermalOutputMode(r.Context()); err == nil {
		resp["output_mode"] = tm
	}
	if tg, err := client.GetThermalGain(r.Context()); err == nil {
		resp["gain"] = tg
	}
	if gtemp, err := client.GetGlobalTemp(r.Context(), models.TempMeasureOnce); err == nil {
		resp["global_temp"] = gtemp
	}
	writeJSON(w, resp)
}

func (s *Server) handleAI(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := client.SetAIStreamOutput(r.Context(), req.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"status": "ok", "ok": ok, "enabled": req.Enabled})
		return
	}
	mode, err := client.GetAIMode(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	resp := map[string]interface{}{"ai_mode": mode}
	if st, err := client.GetAIStreamStatus(r.Context()); err == nil {
		resp["stream_status"] = st
	}
	writeJSON(w, resp)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	client := s.getClientOrError(w)
	if client == nil {
		return
	}

	s.mu.RLock()
	targetIP := s.cameraIP
	s.mu.RUnlock()

	var mu sync.Mutex
	out := map[string]interface{}{
		"target_ip": targetIP,
	}

	var wg sync.WaitGroup
	probe := func(key string, fn func(context.Context) (interface{}, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pCtx, cancel := context.WithTimeout(r.Context(), 450*time.Millisecond)
			defer cancel()
			if val, err := fn(pCtx); err == nil {
				mu.Lock()
				out[key] = val
				mu.Unlock()
			}
		}()
	}

	probe("hardware", func(c context.Context) (interface{}, error) {
		hw, err := client.GetHardwareID(c)
		if err != nil {
			return nil, err
		}
		pid, _ := hw.ProductID()
		return map[string]interface{}{
			"camera_type": pid.String(),
			"product_id":  pid,
			"hardware_id": strings.TrimRight(string(hw.Raw), "\x00"),
		}, nil
	})
	probe("firmware", func(c context.Context) (interface{}, error) {
		fw, err := client.GetFirmwareVersion(c)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"camera": models.FormatWord(fw.Camera),
			"gimbal": models.FormatWord(fw.Gimbal),
			"zoom":   models.FormatWord(fw.Zoom),
		}, nil
	})
	probe("camera_info", func(c context.Context) (interface{}, error) {
		return client.GetCameraSystemInfo(c)
	})
	probe("gimbal_mode", func(c context.Context) (interface{}, error) {
		m, err := client.GetGimbalMode(c)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"code": m, "name": m.String()}, nil
	})
	probe("encoding_recording", func(c context.Context) (interface{}, error) {
		return client.GetEncodingParams(c, models.StreamTypeRecording)
	})
	probe("encoding_main", func(c context.Context) (interface{}, error) {
		return client.GetEncodingParams(c, models.StreamTypeMain)
	})
	probe("encoding_sub", func(c context.Context) (interface{}, error) {
		return client.GetEncodingParams(c, models.StreamTypeSub)
	})
	probe("pic_name_photo", func(c context.Context) (interface{}, error) {
		return client.GetPicNameType(c, models.FileTypePicture)
	})
	probe("pic_name_video", func(c context.Context) (interface{}, error) {
		return client.GetPicNameType(c, models.FileTypeRecordVideo)
	})
	probe("osd_enabled", func(c context.Context) (interface{}, error) {
		return client.GetMavlinkOSDFlag(c)
	})
	probe("stitching_mode", func(c context.Context) (interface{}, error) {
		return client.GetVideoStitchingMode(c)
	})
	probe("weak_control", func(c context.Context) (interface{}, error) {
		return client.GetWeakControlMode(c)
	})
	probe("control_mode", func(c context.Context) (interface{}, error) {
		return client.GetControlMode(c)
	})
	probe("motor_voltage", func(c context.Context) (interface{}, error) {
		return client.GetMotorVoltage(c)
	})
	probe("ip_config", func(c context.Context) (interface{}, error) {
		cfg, err := client.GetIP(c)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"ip":      cfg.IP.String(),
			"mask":    cfg.Mask.String(),
			"gateway": cfg.Gateway.String(),
		}, nil
	})
	probe("system_time", func(c context.Context) (interface{}, error) {
		return client.GetSystemTime(c)
	})
	probe("zoom_range", func(c context.Context) (interface{}, error) {
		return client.GetZoomRange(c)
	})
	probe("current_zoom", func(c context.Context) (interface{}, error) {
		return client.GetCurrentZoom(c)
	})
	probe("pseudo_color", func(c context.Context) (interface{}, error) {
		return client.GetPseudoColor(c)
	})
	probe("thermal_output_mode", func(c context.Context) (interface{}, error) {
		return client.GetThermalOutputMode(c)
	})
	probe("thermal_gain", func(c context.Context) (interface{}, error) {
		return client.GetThermalGain(c)
	})
	probe("ai_mode", func(c context.Context) (interface{}, error) {
		return client.GetAIMode(c)
	})
	probe("laser_distance", func(c context.Context) (interface{}, error) {
		return client.GetLaserDistance(c)
	})

	wg.Wait()
	writeJSON(w, out)
}

func parseMediaType(r *http.Request) models.MediaType {
	if t, err := strconv.Atoi(r.URL.Query().Get("type")); err == nil && t == int(models.MediaTypeVideos) {
		return models.MediaTypeVideos
	}
	return models.MediaTypeImages
}

func (s *Server) handleMediaDirectories(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	med := s.mediaClient
	s.mu.RUnlock()
	if med == nil {
		http.Error(w, "media client not ready", http.StatusServiceUnavailable)
		return
	}
	dirs, err := med.ListDirectories(r.Context(), parseMediaType(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dirs)
}

func (s *Server) handleMediaFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	s.mu.RLock()
	med := s.mediaClient
	s.mu.RUnlock()
	if med == nil {
		http.Error(w, "media client not ready", http.StatusServiceUnavailable)
		return
	}
	files, err := med.ListAllFiles(r.Context(), parseMediaType(r), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

func (s *Server) handleMediaDownload(w http.ResponseWriter, r *http.Request) {
	downloadURL := r.URL.Query().Get("url")
	if downloadURL == "" {
		http.Error(w, "missing url query param", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(downloadURL)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	filename := filepath.Base(u.Path)

	resp, err := http.Get(downloadURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleMediaDownloadAll(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	med := s.mediaClient
	s.mu.RUnlock()
	if med == nil {
		http.Error(w, "media client not ready", http.StatusServiceUnavailable)
		return
	}

	typeParam := r.URL.Query().Get("type")
	dirFilter := r.URL.Query().Get("path")

	var types []models.MediaType
	zipName := "siyi_media_all.zip"
	switch typeParam {
	case "0", "images", "photos":
		types = []models.MediaType{models.MediaTypeImages}
		zipName = "siyi_photos.zip"
	case "1", "videos":
		types = []models.MediaType{models.MediaTypeVideos}
		zipName = "siyi_videos.zip"
	default:
		types = []models.MediaType{models.MediaTypeImages, models.MediaTypeVideos}
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))
	w.Header().Set("Content-Type", "application/zip")

	zw := zip.NewWriter(w)
	defer func() { _ = zw.Close() }()

	httpCli := &http.Client{Timeout: 30 * time.Second}
	for _, mt := range types {
		folderPrefix := "photos"
		if mt == models.MediaTypeVideos {
			folderPrefix = "videos"
		}
		dirs, err := med.ListDirectories(r.Context(), mt)
		if err != nil {
			continue
		}
		for _, d := range dirs {
			if dirFilter != "" && d.Path != dirFilter {
				continue
			}
			files, err := med.ListAllFiles(r.Context(), mt, d.Path)
			if err != nil {
				continue
			}
			for _, f := range files {
				if r.Context().Err() != nil {
					return
				}
				req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, f.URL, nil)
				if err != nil {
					continue
				}
				resp, err := httpCli.Do(req)
				if err != nil {
					continue
				}
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					entryName := fmt.Sprintf("%s/%s/%s", folderPrefix, d.Name, f.Name)
					if fw, err := zw.Create(entryName); err == nil {
						_, _ = io.Copy(fw, resp.Body)
					}
				}
				_ = resp.Body.Close()
			}
		}
	}
}

func (s *Server) handleStreamToggle(w http.ResponseWriter, r *http.Request) {
	enabled := false
	if q := r.URL.Query().Get("enabled"); q != "" {
		enabled, _ = strconv.ParseBool(q)
	} else if r.Body != nil {
		var body struct {
			Enable *bool `json:"enable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Enable != nil {
			enabled = *body.Enable
		}
	}

	s.mu.Lock()
	str := s.stream
	rtspURL := s.rtspURL
	s.mu.Unlock()

	if enabled {
		if str != nil && !str.IsRunning() {
			_ = str.Start(context.Background())
		}
		if rtspURL != "" {
			s.startVideoPipeline(rtspURL)
		}
	} else {
		s.stopVideoPipeline()
		if str != nil && str.IsRunning() {
			_ = str.Stop()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "enabled": enabled})
}

func (s *Server) handleWebSocketAttitude(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ticker := time.NewTicker(100 * time.Millisecond) // 10Hz
	defer ticker.Stop()

	for range ticker.C {
		s.attMu.RLock()
		att := map[string]float64{
			"yaw":   s.attitude["yaw"],
			"pitch": s.attitude["pitch"],
			"roll":  s.attitude["roll"],
		}
		s.attMu.RUnlock()

		if err := conn.WriteJSON(att); err != nil {
			return
		}
	}
}

func (s *Server) publishFrame(frame []byte) {
	cp := make([]byte, len(frame))
	copy(cp, frame)

	s.frameMu.Lock()
	s.latestFrame = cp
	for ch := range s.frameSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.frameMu.Unlock()
}

func (s *Server) subscribeFrames() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	s.frameMu.Lock()
	s.frameSubs[ch] = struct{}{}
	s.frameMu.Unlock()
	return ch, func() {
		s.frameMu.Lock()
		delete(s.frameSubs, ch)
		s.frameMu.Unlock()
	}
}

func (s *Server) stopVideoPipeline() {
	s.frameMu.Lock()
	cancel := s.videoCancel
	s.videoCancel = nil
	s.frameMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Server) startVideoPipeline(rtspURL string) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		s.logger.Warn("ffmpeg not found in PATH; MJPEG video stream disabled")
		return
	}

	s.stopVideoPipeline()
	ctx, cancel := context.WithCancel(context.Background())
	s.frameMu.Lock()
	s.videoCancel = cancel
	s.frameMu.Unlock()

	go s.runFFmpegMJPEG(ctx, ffmpegPath, rtspURL)
}

func (s *Server) runFFmpegMJPEG(ctx context.Context, ffmpegPath, rtspURL string) {
	for {
		if ctx.Err() != nil {
			return
		}

		cmd := exec.CommandContext(ctx, ffmpegPath,
			"-hide_banner",
			"-loglevel", "error",
			"-rtsp_transport", "tcp",
			"-fflags", "nobuffer",
			"-flags", "low_delay",
			"-probesize", "500000",
			"-analyzeduration", "500000",
			"-i", rtspURL,
			"-f", "image2pipe",
			"-vcodec", "mjpeg",
			"-q:v", "5",
			"-r", "15",
			"pipe:1",
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		if err := cmd.Start(); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		s.readMJPEGStream(stdout)
		_ = cmd.Wait()

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *Server) readMJPEGStream(r io.Reader) {
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	var buf []byte
	chunk := make([]byte, 32*1024)
	for {
		n, err := r.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			for {
				start := bytes.Index(buf, soi)
				if start < 0 {
					if len(buf) > 1 {
						buf = buf[len(buf)-1:]
					}
					break
				}
				if start > 0 {
					buf = buf[start:]
				}
				end := bytes.Index(buf[2:], eoi)
				if end < 0 {
					break
				}
				frameEnd := 2 + end + 2
				s.publishFrame(buf[:frameEnd])
				buf = buf[frameEnd:]
			}
		}
		if err != nil {
			return
		}
	}
}

func writeMJPEGFrame(w http.ResponseWriter, frame []byte) bool {
	if _, err := fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame)); err != nil {
		return false
	}
	if _, err := w.Write(frame); err != nil {
		return false
	}
	if _, err := w.Write([]byte("\r\n")); err != nil {
		return false
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return true
}

func (s *Server) handleStreamVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	ch, unsub := s.subscribeFrames()
	defer unsub()

	s.frameMu.RLock()
	initial := s.latestFrame
	s.frameMu.RUnlock()
	if len(initial) > 0 {
		if !writeMJPEGFrame(w, initial) {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			s.frameMu.RLock()
			frame := s.latestFrame
			s.frameMu.RUnlock()
			if len(frame) > 0 {
				if !writeMJPEGFrame(w, frame) {
					return
				}
			}
		}
	}
}
