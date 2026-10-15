package protocol

import (
	"fmt"
)

type parserState int

const (
	stateAwaitSTX1 parserState = iota
	stateAwaitSTX2
	stateReadCtrl
	stateReadDataLenLo
	stateReadDataLenHi
	stateReadSeqLo
	stateReadSeqHi
	stateReadCmdID
	stateReadData
	stateReadCRCLo
	stateReadCRCHi
	stateVerifyCRC
)

// FrameParser is a state-machine based streaming frame parser.
// It handles partial frames, stream fragmentation, automatic resynchronization,
// and CRC validation.
type FrameParser struct {
	maxPayload  int
	state       parserState
	ctrl        uint8
	dataLen     uint16
	seq         uint16
	cmdID       uint8
	data        []byte
	crcLo       uint8
	crcHi       uint8
	frameBuffer []byte
}

// NewFrameParser creates a new streaming frame parser.
// If maxPayload <= 0, DefaultMaxPayload (4096) is used.
func NewFrameParser(maxPayload ...int) *FrameParser {
	limit := DefaultMaxPayload
	if len(maxPayload) > 0 && maxPayload[0] > 0 {
		limit = maxPayload[0]
	}
	return &FrameParser{
		maxPayload:  limit,
		state:       stateAwaitSTX1,
		frameBuffer: make([]byte, 0, 256),
	}
}

// Reset clears any buffered state and returns the parser to the initial state.
func (p *FrameParser) Reset() {
	p.state = stateAwaitSTX1
	p.ctrl = 0
	p.dataLen = 0
	p.seq = 0
	p.cmdID = 0
	p.data = nil
	p.crcLo = 0
	p.crcHi = 0
	p.frameBuffer = p.frameBuffer[:0]
}

// Feed processes incoming bytes chunk-by-chunk and returns extracted complete frames.
// If a CRC mismatch or framing error occurs, the parser resynchronizes and returns
// any frames parsed prior to the error, along with the error.
func (p *FrameParser) Feed(chunk []byte) ([]Frame, error) {
	var frames []Frame

	idx := 0
	for idx < len(chunk) {
		b := chunk[idx]
		idx++

		switch p.state {
		case stateAwaitSTX1:
			if b == STXBytes[0] { // 0x55
				p.frameBuffer = append(p.frameBuffer[:0], b)
				p.state = stateAwaitSTX2
			}
			// Discard anything else

		case stateAwaitSTX2:
			switch b {
			case STXBytes[1]: // 0x66
				p.frameBuffer = append(p.frameBuffer, b)
				p.state = stateReadCtrl
			case STXBytes[0]: // 0x55
				// Potential start of new frame
				p.frameBuffer = append(p.frameBuffer[:0], b)
				// Stay in stateAwaitSTX2
			default:
				// Invalid STX, reset
				p.Reset()
			}

		case stateReadCtrl:
			p.ctrl = b
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateReadDataLenLo

		case stateReadDataLenLo:
			p.dataLen = uint16(b)
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateReadDataLenHi

		case stateReadDataLenHi:
			p.dataLen |= uint16(b) << 8
			p.frameBuffer = append(p.frameBuffer, b)

			if int(p.dataLen) > p.maxPayload {
				err := &FramingError{
					Message: fmt.Sprintf("payload size %d exceeds maximum %d", p.dataLen, p.maxPayload),
				}
				p.Reset()
				return frames, err
			}
			p.state = stateReadSeqLo

		case stateReadSeqLo:
			p.seq = uint16(b)
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateReadSeqHi

		case stateReadSeqHi:
			p.seq |= uint16(b) << 8
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateReadCmdID

		case stateReadCmdID:
			p.cmdID = b
			p.frameBuffer = append(p.frameBuffer, b)
			p.data = make([]byte, 0, p.dataLen)
			if p.dataLen > 0 {
				p.state = stateReadData
			} else {
				p.state = stateReadCRCLo
			}

		case stateReadData:
			p.data = append(p.data, b)
			p.frameBuffer = append(p.frameBuffer, b)
			if len(p.data) >= int(p.dataLen) {
				p.state = stateReadCRCLo
			}

		case stateReadCRCLo:
			p.crcLo = b
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateReadCRCHi

		case stateReadCRCHi:
			p.crcHi = b
			p.frameBuffer = append(p.frameBuffer, b)
			p.state = stateVerifyCRC
		}

		if p.state == stateVerifyCRC {
			frameWithoutCRC := p.frameBuffer[:HeaderLen+int(p.dataLen)]
			expectedCRC := uint16(p.crcLo) | (uint16(p.crcHi) << 8)
			actualCRC := CRC16(frameWithoutCRC)

			if expectedCRC != actualCRC {
				err := NewCRCError(expectedCRC, actualCRC, p.frameBuffer)
				p.Reset()
				return frames, err
			}

			// Valid frame
			dataCopy := make([]byte, len(p.data))
			copy(dataCopy, p.data)

			frames = append(frames, Frame{
				Ctrl:  p.ctrl,
				Seq:   p.seq,
				CmdID: p.cmdID,
				Data:  dataCopy,
			})

			p.Reset()
		}
	}

	return frames, nil
}
