package commands

import "fmt"

// ResponseError indicates that the device returned a non-success status code (sta=0).
type ResponseError struct {
	CmdID uint8
	Sta   uint8
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("command 0x%02X failed with status=%d", e.CmdID, e.Sta)
}

// ConfigurationError indicates that an invalid parameter or out-of-range value was supplied.
type ConfigurationError struct {
	Message string
}

func (e *ConfigurationError) Error() string {
	return "configuration error: " + e.Message
}
