package mavlink

// CRCAccumulate updates a MAVLink X.25 CRC with data byte.
func CRCAccumulate(b byte, crc uint16) uint16 {
	ch := b ^ byte(crc&0xFF)
	ch = ch ^ (ch << 4)
	return (crc >> 8) ^ (uint16(ch) << 8) ^ (uint16(ch) << 3) ^ (uint16(ch) >> 4)
}

// CalculateCRC computes the MAVLink X.25 CRC-16 over data and includes the message-specific CRC extra byte.
func CalculateCRC(data []byte, crcExtra byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc = CRCAccumulate(b, crc)
	}
	crc = CRCAccumulate(crcExtra, crc)
	return crc
}
