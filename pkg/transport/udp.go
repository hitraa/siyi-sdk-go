package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// UDPTransport implements Transport over UDP.
type UDPTransport struct {
	ip        string
	port      int
	bindPort  int
	conn      *net.UDPConn
	streamCh  chan []byte
	closed    chan struct{}
	mu        sync.RWMutex
	connected bool
}

// NewUDPTransport creates a new UDPTransport.
// ip: target gimbal camera IP (default protocol.DefaultIP).
// port: target gimbal UDP port (default protocol.DefaultUDPPort).
// bindPort: optional local port (0 for ephemeral).
func NewUDPTransport(ip string, port int, bindPort ...int) *UDPTransport {
	if ip == "" {
		ip = protocol.DefaultIP
	}
	if port <= 0 {
		port = protocol.DefaultUDPPort
	}
	bp := 0
	if len(bindPort) > 0 && bindPort[0] > 0 {
		bp = bindPort[0]
	}
	return &UDPTransport{
		ip:       ip,
		port:     port,
		bindPort: bp,
	}
}

// Connect establishes the UDP connection endpoint.
func (u *UDPTransport) Connect(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.connected {
		return nil
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", u.ip, u.port))
	if err != nil {
		return fmt.Errorf("%w: resolve remote address: %v", ErrConnectionFailed, err)
	}

	var localAddr *net.UDPAddr
	if u.bindPort > 0 {
		localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", u.bindPort))
		if err != nil {
			return fmt.Errorf("%w: resolve local bind port: %v", ErrConnectionFailed, err)
		}
	}

	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		return fmt.Errorf("%w: dial UDP: %v", ErrConnectionFailed, err)
	}

	u.conn = conn
	u.streamCh = make(chan []byte, 100)
	u.closed = make(chan struct{})
	u.connected = true

	go u.readLoop()

	return nil
}

func (u *UDPTransport) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-u.closed:
			return
		default:
		}

		_ = u.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := u.conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-u.closed:
				return
			default:
				continue
			}
		}

		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case u.streamCh <- chunk:
			case <-u.closed:
				return
			}
		}
	}
}

// Close closes the UDP transport and releases resources.
func (u *UDPTransport) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.connected {
		return nil
	}

	u.connected = false
	close(u.closed)
	err := u.conn.Close()
	close(u.streamCh)
	return err
}

// Send transmits raw bytes over UDP.
func (u *UDPTransport) Send(ctx context.Context, data []byte) error {
	u.mu.RLock()
	if !u.connected || u.conn == nil {
		u.mu.RUnlock()
		return ErrNotConnected
	}
	conn := u.conn
	u.mu.RUnlock()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	}

	_, err := conn.Write(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	return nil
}

// Stream returns the receive channel.
func (u *UDPTransport) Stream() <-chan []byte {
	return u.streamCh
}

// IsConnected returns whether the transport is connected.
func (u *UDPTransport) IsConnected() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.connected
}

// IP returns the configured remote IP address.
func (u *UDPTransport) IP() string {
	return u.ip
}

// Port returns the configured remote port.
func (u *UDPTransport) Port() int {
	return u.port
}

// SupportsHeartbeat returns false for UDP.
func (u *UDPTransport) SupportsHeartbeat() bool {
	return false
}
