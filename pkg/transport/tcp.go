package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// TCPTransport implements Transport over TCP (requires 1Hz heartbeat).
type TCPTransport struct {
	ip        string
	port      int
	conn      net.Conn
	streamCh  chan []byte
	closed    chan struct{}
	mu        sync.RWMutex
	connected bool
}

// NewTCPTransport creates a new TCPTransport.
func NewTCPTransport(ip string, port int) *TCPTransport {
	if ip == "" {
		ip = protocol.DefaultIP
	}
	if port <= 0 {
		port = protocol.DefaultTCPPort
	}
	return &TCPTransport{
		ip:   ip,
		port: port,
	}
}

// Connect establishes the TCP connection.
func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", t.ip, t.port))
	if err != nil {
		return fmt.Errorf("%w: TCP dial: %v", ErrConnectionFailed, err)
	}

	t.conn = conn
	t.streamCh = make(chan []byte, 100)
	t.closed = make(chan struct{})
	t.connected = true

	go t.readLoop()

	return nil
}

func (t *TCPTransport) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-t.closed:
			return
		default:
		}

		_ = t.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := t.conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if err == io.EOF {
				t.mu.Lock()
				t.connected = false
				t.mu.Unlock()
				return
			}
			select {
			case <-t.closed:
				return
			default:
				continue
			}
		}

		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case t.streamCh <- chunk:
			case <-t.closed:
				return
			}
		}
	}
}

// Close closes the TCP connection and releases resources.
func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.connected = false
	close(t.closed)
	err := t.conn.Close()
	close(t.streamCh)
	return err
}

// Send transmits raw bytes over TCP.
func (t *TCPTransport) Send(ctx context.Context, data []byte) error {
	t.mu.RLock()
	if !t.connected || t.conn == nil {
		t.mu.RUnlock()
		return ErrNotConnected
	}
	conn := t.conn
	t.mu.RUnlock()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	}

	_, err := conn.Write(data)
	if err != nil {
		t.mu.Lock()
		t.connected = false
		t.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	return nil
}

// Stream returns the receive channel.
func (t *TCPTransport) Stream() <-chan []byte {
	return t.streamCh
}

// IsConnected returns whether the TCP transport is connected.
func (t *TCPTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// IP returns the configured remote IP address.
func (t *TCPTransport) IP() string {
	return t.ip
}

// Port returns the configured remote port.
func (t *TCPTransport) Port() int {
	return t.port
}

// SupportsHeartbeat returns true for TCP (heartbeat required).
func (t *TCPTransport) SupportsHeartbeat() bool {
	return true
}
