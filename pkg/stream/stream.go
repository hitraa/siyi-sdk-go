package stream

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

// PacketCallback receives raw RTP packets from the camera stream.
type PacketCallback func(media *description.Media, forma format.Format, pkt *rtp.Packet)

// Config configures RTSP stream ingestion.
type Config struct {
	RTSPURL        string
	ReconnectDelay time.Duration
	OnPacket       PacketCallback
}

// Stream manages RTSP streaming from a SIYI gimbal camera.
type Stream struct {
	cfg       Config
	mu        sync.RWMutex
	running   bool
	cancel    context.CancelFunc
	closer    func()
	callbacks []PacketCallback
}

// New creates a new RTSP stream receiver.
func New(cfg Config) *Stream {
	if cfg.ReconnectDelay <= 0 {
		cfg.ReconnectDelay = 2 * time.Second
	}
	s := &Stream{
		cfg:       cfg,
		callbacks: make([]PacketCallback, 0),
	}
	if cfg.OnPacket != nil {
		s.callbacks = append(s.callbacks, cfg.OnPacket)
	}
	return s
}

// OnPacket registers an RTP packet callback.
func (s *Stream) OnPacket(cb PacketCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, cb)
}

// IsRunning reports whether the stream is active.
func (s *Stream) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Start begins stream ingestion asynchronously.
func (s *Stream) Start(parentCtx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(parentCtx)
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go s.runLoop(ctx)
	return nil
}

func (s *Stream) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := s.connectAndStream(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.cfg.ReconnectDelay):
			}
		}
	}
}

func (s *Stream) connectAndStream(ctx context.Context) error {
	u, err := base.ParseURL(s.cfg.RTSPURL)
	if err != nil {
		return fmt.Errorf("invalid RTSP URL: %w", err)
	}

	c := &gortsplib.Client{
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, address)
		},
	}

	err = c.Start(u.Scheme, u.Host)
	if err != nil {
		return fmt.Errorf("RTSP start failed: %w", err)
	}

	var once sync.Once
	closeFn := func() {
		once.Do(func() {
			c.Close()
		})
	}

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		closeFn()
		return nil
	}
	s.closer = closeFn
	s.mu.Unlock()

	defer func() {
		closeFn()
		s.mu.Lock()
		s.closer = nil
		s.mu.Unlock()
	}()

	desc, _, err := c.Describe(u)
	if err != nil {
		return fmt.Errorf("RTSP describe failed: %w", err)
	}

	err = c.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		return fmt.Errorf("RTSP setup failed: %w", err)
	}

	c.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		s.mu.RLock()
		cbs := s.callbacks
		s.mu.RUnlock()

		for _, cb := range cbs {
			cb(medi, forma, pkt)
		}
	})

	_, err = c.Play(nil)
	if err != nil {
		return fmt.Errorf("RTSP play failed: %w", err)
	}

	waitErr := make(chan error, 1)
	go func() {
		waitErr <- c.Wait()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-waitErr:
		return err
	}
}

// Stop cleanly terminates streaming.
func (s *Stream) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	closer := s.closer
	s.closer = nil
	s.mu.Unlock()

	if closer != nil {
		closer()
	}
	return nil
}
