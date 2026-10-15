package siyi

import (
	"context"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

// ConnectUDP creates, connects, and returns a Client using UDP.
func ConnectUDP(ctx context.Context, ip string, port int, opts ...ClientOption) (*Client, error) {
	if ip == "" {
		ip = protocol.DefaultIP
	}
	if port <= 0 {
		port = protocol.DefaultUDPPort
	}
	t := transport.NewUDPTransport(ip, port)
	client := NewClient(t, opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// ConnectTCP creates, connects, and returns a Client using TCP (1Hz heartbeat automatically active).
func ConnectTCP(ctx context.Context, ip string, port int, opts ...ClientOption) (*Client, error) {
	if ip == "" {
		ip = protocol.DefaultIP
	}
	if port <= 0 {
		port = protocol.DefaultTCPPort
	}
	t := transport.NewTCPTransport(ip, port)
	client := NewClient(t, opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// ConnectSerial creates, connects, and returns a Client using UART serial.
func ConnectSerial(ctx context.Context, device string, baud int, opts ...ClientOption) (*Client, error) {
	if baud <= 0 {
		baud = protocol.DefaultBaudRate
	}
	t := transport.NewSerialTransport(device, baud)
	client := NewClient(t, opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}
