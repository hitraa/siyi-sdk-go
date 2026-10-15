package transport

import (
	"context"
	"errors"
)

var (
	// ErrNotConnected is returned when an operation is attempted on an unopened or closed transport.
	ErrNotConnected = errors.New("transport not connected")

	// ErrConnectionFailed is returned when establishing a transport connection fails.
	ErrConnectionFailed = errors.New("failed to establish connection")

	// ErrSendFailed is returned when an underlying I/O error prevents sending data.
	ErrSendFailed = errors.New("failed to send data")
)

// Transport defines the common interface for all communication channels
// (UDP, TCP, Serial UART, and Mock).
type Transport interface {
	// Connect establishes the physical or logical connection to the device.
	Connect(ctx context.Context) error

	// Close gracefully terminates the connection and releases underlying resources.
	Close() error

	// Send transmits raw bytes to the device.
	Send(ctx context.Context, data []byte) error

	// Stream returns a read-only channel delivering incoming raw byte chunks from the device.
	// The channel is closed when the transport is closed.
	Stream() <-chan []byte

	// IsConnected returns whether the transport is currently open and healthy.
	IsConnected() bool

	// SupportsHeartbeat indicates whether this transport requires 1 Hz heartbeat packets (true for TCP).
	SupportsHeartbeat() bool
}
