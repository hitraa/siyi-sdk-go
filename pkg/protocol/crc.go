package protocol

import "encoding/binary"

// CRC16 calculates CRC-16/XMODEM checksum using the lookup table defined in
// the SIYI SDK protocol specification Chapter 4.
//
// Algorithm: CRC-16/XMODEM
// - Polynomial: X^16 + X^12 + X^5 + 1 = 0x1021
// - Initial value: 0x0000
// - No reflect-in, no reflect-out, no final XOR
func CRC16(buf []byte, init ...uint16) uint16 {
	crc := CRC16Init
	if len(init) > 0 {
		crc = init[0]
	}

	for _, b := range buf {
		temp := byte(crc >> 8)
		crc = (crc << 8) ^ CRC16Table[b^temp]
	}
	return crc
}

// CRC16Check verifies CRC-16/XMODEM checksum.
// It computes the CRC of the frame data and compares against the provided
// 2-byte little-endian CRC slice.
func CRC16Check(frameWithoutCRC []byte, crcLE []byte) bool {
	if len(crcLE) < 2 {
		return false
	}
	expected := binary.LittleEndian.Uint16(crcLE)
	actual := CRC16(frameWithoutCRC)
	return expected == actual
}
