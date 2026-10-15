package transport

import (
	"context"
	"sync"
)

// MockTransport is an in-memory transport implementation for automated tests.
type MockTransport struct {
	mu                sync.RWMutex
	connected         bool
	supportsHeartbeat bool
	sentFrames        [][]byte
	streamCh          chan []byte
	errCh             chan error
	onSend            func(data []byte)
}

// NewMockTransport creates a new MockTransport instance.
func NewMockTransport(supportsHeartbeat ...bool) *MockTransport {
	hb := false
	if len(supportsHeartbeat) > 0 {
		hb = supportsHeartbeat[0]
	}
	return &MockTransport{
		supportsHeartbeat: hb,
		streamCh:          make(chan []byte, 100),
		errCh:             make(chan error, 10),
		sentFrames:        make([][]byte, 0),
	}
}

// Connect marks the mock transport as connected.
func (m *MockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Close disconnects the mock transport and closes the stream channel.
func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return nil
	}
	m.connected = false
	close(m.streamCh)
	return nil
}

// Send records the sent frame and returns any queued error.
func (m *MockTransport) Send(ctx context.Context, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return ErrNotConnected
	}
	select {
	case err := <-m.errCh:
		return err
	default:
	}

	frameCopy := make([]byte, len(data))
	copy(frameCopy, data)
	m.sentFrames = append(m.sentFrames, frameCopy)
	if m.onSend != nil {
		fn := m.onSend
		go fn(frameCopy)
	}
	return nil
}

// SetOnSend configures an optional callback triggered whenever a frame is sent.
func (m *MockTransport) SetOnSend(fn func(data []byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSend = fn
}

// Stream returns the channel yielding queued responses.
func (m *MockTransport) Stream() <-chan []byte {
	return m.streamCh
}

// IsConnected returns whether the mock transport is connected.
func (m *MockTransport) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// SupportsHeartbeat returns the heartbeat support flag.
func (m *MockTransport) SupportsHeartbeat() bool {
	return m.supportsHeartbeat
}

// QueueResponse pushes a mock response payload into the stream channel.
func (m *MockTransport) QueueResponse(data []byte) {
	frameCopy := make([]byte, len(data))
	copy(frameCopy, data)
	m.streamCh <- frameCopy
}

// QueueError queues an error to be returned on the next Send() call.
func (m *MockTransport) QueueError(err error) {
	m.errCh <- err
}

// SentFrames returns a copy of all frames transmitted through Send().
func (m *MockTransport) SentFrames() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([][]byte, len(m.sentFrames))
	for i, f := range m.sentFrames {
		res[i] = make([]byte, len(f))
		copy(res[i], f)
	}
	return res
}
