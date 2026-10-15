// Package siyi provides an idiomatic, full-featured Go SDK for the SIYI Gimbal Camera External SDK Protocol v0.1.1.
//
// Supports ZT30, ZT6, ZR10, ZR30, A8 mini, A2 mini, and Quad-Spectrum gimbal cameras over UDP, TCP, and UART Serial.
package siyi

import (
	"context"

	"github.com/hitraa/siyi-sdk-go/pkg/siyi"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

// Re-export core client types
type (
	Client       = siyi.Client
	ClientOption = siyi.ClientOption
	ClientConfig = siyi.ClientConfig
	TimeoutError = siyi.TimeoutError
	Unsubscribe  = siyi.Unsubscribe
)

// Re-export common errors
var (
	ErrClientClosed = siyi.ErrClientClosed
	ErrTimeout      = siyi.ErrTimeout
)

// Re-export client options
var (
	WithTimeout        = siyi.WithTimeout
	WithRetries        = siyi.WithRetries
	WithRetryBaseDelay = siyi.WithRetryBaseDelay
	WithAutoReconnect  = siyi.WithAutoReconnect
	WithLogger         = siyi.WithLogger
)

// NewClient instantiates a new Client.
func NewClient(t transport.Transport, opts ...ClientOption) *Client {
	return siyi.NewClient(t, opts...)
}

// ConnectUDP creates, connects, and returns a Client using UDP.
func ConnectUDP(ctx context.Context, ip string, port int, opts ...ClientOption) (*Client, error) {
	return siyi.ConnectUDP(ctx, ip, port, opts...)
}

// ConnectTCP creates, connects, and returns a Client using TCP (1Hz heartbeat automatically active).
func ConnectTCP(ctx context.Context, ip string, port int, opts ...ClientOption) (*Client, error) {
	return siyi.ConnectTCP(ctx, ip, port, opts...)
}

// ConnectSerial creates, connects, and returns a Client using UART serial.
func ConnectSerial(ctx context.Context, device string, baud int, opts ...ClientOption) (*Client, error) {
	return siyi.ConnectSerial(ctx, device, baud, opts...)
}
