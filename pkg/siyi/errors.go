package siyi

import (
	"errors"
	"fmt"
)

var (
	// ErrClientClosed is returned when an operation is attempted on a closed client.
	ErrClientClosed = errors.New("client closed")

	// ErrTimeout is returned when a response is not received within timeout.
	ErrTimeout = errors.New("command timed out waiting for response")
)

// TimeoutError carries command ID and duration for timeouts.
type TimeoutError struct {
	CmdID    uint8
	Duration string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout waiting for response to command 0x%02X after %s", e.CmdID, e.Duration)
}

func (e *TimeoutError) Is(target error) bool {
	return target == ErrTimeout
}
