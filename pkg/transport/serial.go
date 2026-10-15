package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"go.bug.st/serial"
)

// SerialTransport implements Transport over UART/RS-232 serial links.
type SerialTransport struct {
	device    string
	baud      int
	port      serial.Port
	streamCh  chan []byte
	closed    chan struct{}
	mu        sync.RWMutex
	connected bool
}

// NewSerialTransport creates a new SerialTransport.
func NewSerialTransport(device string, baud ...int) *SerialTransport {
	b := protocol.DefaultBaudRate
	if len(baud) > 0 && baud[0] > 0 {
		b = baud[0]
	}
	return &SerialTransport{
		device: device,
		baud:   b,
	}
}

// Connect opens the serial port.
func (s *SerialTransport) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	mode := &serial.Mode{
		BaudRate: s.baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(s.device, mode)
	if err != nil {
		return fmt.Errorf("%w: open serial port %s: %v", ErrConnectionFailed, s.device, err)
	}

	// Set read timeout
	_ = port.SetReadTimeout(500 * time.Millisecond)

	s.port = port
	s.streamCh = make(chan []byte, 100)
	s.closed = make(chan struct{})
	s.connected = true

	go s.readLoop()

	return nil
}

func (s *SerialTransport) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.closed:
			return
		default:
		}

		n, err := s.port.Read(buf)
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}

		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case s.streamCh <- chunk:
			case <-s.closed:
				return
			}
		}
	}
}

// Close closes the serial port and releases resources.
func (s *SerialTransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	s.connected = false
	close(s.closed)
	err := s.port.Close()
	close(s.streamCh)
	return err
}

// Send transmits raw bytes over the serial link.
func (s *SerialTransport) Send(ctx context.Context, data []byte) error {
	s.mu.RLock()
	if !s.connected || s.port == nil {
		s.mu.RUnlock()
		return ErrNotConnected
	}
	port := s.port
	s.mu.RUnlock()

	_, err := port.Write(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	return nil
}

// Stream returns the receive channel.
func (s *SerialTransport) Stream() <-chan []byte {
	return s.streamCh
}

// IsConnected returns whether the serial transport is connected.
func (s *SerialTransport) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// SupportsHeartbeat returns false for serial transports.
func (s *SerialTransport) SupportsHeartbeat() bool {
	return false
}
