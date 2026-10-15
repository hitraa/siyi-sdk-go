package siyi

import (
	"log/slog"
	"time"
)

// ClientConfig holds configuration options for Client.
type ClientConfig struct {
	DefaultTimeout time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	AutoReconnect  bool
	Logger         *slog.Logger
}

// ClientOption configures a Client.
type ClientOption func(*ClientConfig)

// WithTimeout sets default command timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *ClientConfig) {
		if d > 0 {
			c.DefaultTimeout = d
		}
	}
}

// WithRetries sets maximum retries for idempotent reads.
func WithRetries(retries int) ClientOption {
	return func(c *ClientConfig) {
		if retries >= 0 {
			c.MaxRetries = retries
		}
	}
}

// WithRetryBaseDelay sets base delay for retry backoff.
func WithRetryBaseDelay(delay time.Duration) ClientOption {
	return func(c *ClientConfig) {
		if delay > 0 {
			c.RetryBaseDelay = delay
		}
	}
}

// WithAutoReconnect enables automatic transport reconnection.
func WithAutoReconnect(enabled bool) ClientOption {
	return func(c *ClientConfig) {
		c.AutoReconnect = enabled
	}
}

// WithLogger sets custom structured logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *ClientConfig) {
		if logger != nil {
			c.Logger = logger
		}
	}
}
