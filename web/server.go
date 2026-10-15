package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
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
	frameNotify chan struct{}

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
		attitude:    map[string]float64{"yaw": 0, "pitch": 0, "roll": 0},
		frameNotify: make(chan struct{}, 1),
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
	rtspURL := stream.BuildRTSPURL(ip, models.StreamTypeMain, models.CameraGenerationOld)
	str := stream.New(stream.Config{
		RTSPURL: rtspURL,
	})

	s.mu.Lock()
	s.cameraIP = ip
	oldClient := s.client
	oldStream := s.stream
	s.client = client
	s.mediaClient = medCli
	s.stream = str
	s.connected = (err == nil)
	s.mu.Unlock()

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
	mux.HandleFunc("/api/camera/photo", s.handleCameraPhoto)
	mux.HandleFunc("/api/camera/record", s.handleCameraRecord)
	mux.HandleFunc("/api/camera/zoom", s.handleCameraZoom)
	mux.HandleFunc("/api/camera/focus", s.handleCameraFocus)
	mux.HandleFunc("/api/camera/encoding", s.handleCameraEncoding)
	mux.HandleFunc("/api/storage/format", s.handleStorageFormat)
	mux.HandleFunc("/api/system/reboot", s.handleSystemReboot)
	mux.HandleFunc("/api/system/info", s.handleSystemInfo)
	mux.HandleFunc("/api/media/directories", s.handleMediaDirectories)
	mux.HandleFunc("/api/media/files", s.handleMediaFiles)
	mux.HandleFunc("/api/media/download", s.handleMediaDownload)
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
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_ = client.OneKeyCentering(r.Context(), models.CenteringActionOneKeyCenter)
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
	dir, _ := strconv.Atoi(r.URL.Query().Get("direction"))
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	if client != nil {
		_, _ = client.ManualZoom(r.Context(), dir)
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
			_, _ = client.SetEncodingParams(r.Context(), models.EncodingParams{
				StreamType:   models.StreamTypeMain,
				EncType:      models.VideoEncH264,
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
	params, err := client.GetEncodingParams(r.Context(), models.StreamTypeMain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"stream_type":  params.StreamType.String(),
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

func (s *Server) handleMediaDirectories(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	med := s.mediaClient
	s.mu.RUnlock()
	if med == nil {
		http.Error(w, "media client not ready", http.StatusServiceUnavailable)
		return
	}
	dirs, err := med.ListDirectories(r.Context(), models.MediaTypeImages)
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
	files, err := med.ListFiles(r.Context(), models.MediaTypeImages, path, 0, 50)
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
	defer resp.Body.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleStreamToggle(w http.ResponseWriter, r *http.Request) {
	enabled, _ := strconv.ParseBool(r.URL.Query().Get("enabled"))
	s.mu.Lock()
	if enabled && s.stream != nil && !s.stream.IsRunning() {
		_ = s.stream.Start(context.Background())
	} else if !enabled && s.stream != nil && s.stream.IsRunning() {
		_ = s.stream.Stop()
	}
	s.mu.Unlock()

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

func (s *Server) handleStreamVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.frameNotify:
			s.frameMu.RLock()
			frame := s.latestFrame
			s.frameMu.RUnlock()

			if len(frame) > 0 {
				_, _ = fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\n\r\n%s\r\n", frame)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	}
}
