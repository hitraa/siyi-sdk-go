package protocol

import (
	"encoding/hex"
	"fmt"
)

// FramingError indicates an invalid frame structure (bad STX, truncated frame, etc.).
type FramingError struct {
	Message string
}

func (e *FramingError) Error() string {
	return "framing error: " + e.Message
}

// CRCError indicates CRC-16 checksum mismatch.
type CRCError struct {
	Expected uint16
	Actual   uint16
	FrameHex string
}

func NewCRCError(expected, actual uint16, raw []byte) *CRCError {
	return &CRCError{
		Expected: expected,
		Actual:   actual,
		FrameHex: hex.EncodeToString(raw),
	}
}

func (e *CRCError) Error() string {
	return fmt.Sprintf("CRC mismatch: expected=0x%04X actual=0x%04X frame=%s",
		e.Expected, e.Actual, e.FrameHex)
}

// MalformedPayloadError indicates that a payload does not match the expected length or format.
type MalformedPayloadError struct {
	CmdID  uint8
	Reason string
}

func (e *MalformedPayloadError) Error() string {
	return fmt.Sprintf("malformed payload for command 0x%02X: %s", e.CmdID, e.Reason)
}

// UnknownCommandError indicates that an unexpected or unrecognized command ID was received.
type UnknownCommandError struct {
	CmdID uint8
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("unknown command ID: 0x%02X", e.CmdID)
}
