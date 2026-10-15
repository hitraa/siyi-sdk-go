package protocol

import (
	"encoding/binary"
	"fmt"
)

// Frame represents a SIYI SDK protocol frame.
//
// Frame Structure (SIYI SDK Protocol):
//
//	Offset  Bytes  Field     Description
//	0       2      STX       Start marker (0x6655, wire: 0x55 0x66)
//	2       1      CTRL      Control byte (bit0=need_ack, bit1=ack_pack)
//	3       2      Data_len  Payload length (little-endian)
//	5       2      SEQ       Sequence number (little-endian)
//	7       1      CMD_ID    Command ID
//	8       N      DATA      Payload (N = Data_len)
//	8+N     2      CRC16     CRC-16/XMODEM (little-endian)
type Frame struct {
	Ctrl  uint8
	Seq   uint16
	CmdID uint8
	Data  []byte
}

// DataLen returns the length of the data payload in bytes.
func (f Frame) DataLen() int {
	return len(f.Data)
}

// ToBytes serializes the frame into its wire byte representation, including header, payload, and CRC.
func (f Frame) ToBytes() []byte {
	dataLen := len(f.Data)
	totalLen := HeaderLen + dataLen + CRCLen
	buf := make([]byte, totalLen)

	// Header: STX (LE u16) + CTRL (u8) + data_len (LE u16) + SEQ (LE u16) + CMD_ID (u8)
	binary.LittleEndian.PutUint16(buf[0:2], STX)
	buf[2] = f.Ctrl
	binary.LittleEndian.PutUint16(buf[3:5], uint16(dataLen))
	binary.LittleEndian.PutUint16(buf[5:7], f.Seq)
	buf[7] = f.CmdID

	// Payload
	copy(buf[HeaderLen:HeaderLen+dataLen], f.Data)

	// Calculate CRC-16 over Header + Payload
	crcVal := CRC16(buf[:HeaderLen+dataLen])
	binary.LittleEndian.PutUint16(buf[HeaderLen+dataLen:totalLen], crcVal)

	return buf
}

// FromBytes deserializes a complete frame from wire format.
func FromBytes(buf []byte) (Frame, error) {
	if len(buf) < MinFrameLen {
		return Frame{}, &FramingError{
			Message: fmt.Sprintf("frame too short: %d bytes, minimum is %d", len(buf), MinFrameLen),
		}
	}

	if buf[0] != STXBytes[0] || buf[1] != STXBytes[1] {
		return Frame{}, &FramingError{
			Message: fmt.Sprintf("invalid STX: expected 0x5566, got 0x%02X%02X", buf[0], buf[1]),
		}
	}

	ctrl := buf[2]
	dataLen := int(binary.LittleEndian.Uint16(buf[3:5]))
	seq := binary.LittleEndian.Uint16(buf[5:7])
	cmdID := buf[7]

	expectedLen := HeaderLen + dataLen + CRCLen
	if len(buf) < expectedLen {
		return Frame{}, &FramingError{
			Message: fmt.Sprintf("frame truncated: have %d bytes, expected %d", len(buf), expectedLen),
		}
	}

	frameWithoutCRC := buf[:HeaderLen+dataLen]
	expectedCRC := binary.LittleEndian.Uint16(buf[HeaderLen+dataLen : expectedLen])
	actualCRC := CRC16(frameWithoutCRC)

	if expectedCRC != actualCRC {
		return Frame{}, NewCRCError(expectedCRC, actualCRC, buf[:expectedLen])
	}

	dataCopy := make([]byte, dataLen)
	copy(dataCopy, buf[HeaderLen:HeaderLen+dataLen])

	return Frame{
		Ctrl:  ctrl,
		Seq:   seq,
		CmdID: cmdID,
		Data:  dataCopy,
	}, nil
}

// Build creates a new frame for sending.
// Sets CTRL byte:
// - bit[0] (CTRLNeedACK=0x01) if needACK is true
// - bit[1] (CTRLACKPack=0x02) if this is an ACK frame
func Build(cmdID uint8, data []byte, seq uint16, needACK bool) Frame {
	var ctrl uint8
	if needACK {
		ctrl = CTRLNeedACK
	}
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	return Frame{
		Ctrl:  ctrl,
		Seq:   seq,
		CmdID: cmdID,
		Data:  dataCopy,
	}
}
