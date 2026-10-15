package commands

import (
	"encoding/binary"
	"fmt"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// EncodeRCChannels encodes RC channels send (0x23).
// 38-byte payload: 18 x uint16 LE + chancount (1B) + rssi (1B).
func EncodeRCChannels(ch models.RCChannels) []byte {
	buf := make([]byte, 38)
	for i := 0; i < 18; i++ {
		binary.LittleEndian.PutUint16(buf[i*2:(i+1)*2], ch.Channels[i])
	}
	buf[36] = ch.ChanCount
	buf[37] = ch.RSSI
	return buf
}

// DecodeRCChannels parses 38-byte payload for command 0x23.
func DecodeRCChannels(payload []byte) (models.RCChannels, error) {
	if len(payload) != 38 {
		return models.RCChannels{}, &protocol.MalformedPayloadError{
			CmdID:  protocol.CmdSendRCChannels,
			Reason: fmt.Sprintf("expected 38 bytes, got %d", len(payload)),
		}
	}
	var res models.RCChannels
	for i := 0; i < 18; i++ {
		res.Channels[i] = binary.LittleEndian.Uint16(payload[i*2 : (i+1)*2])
	}
	res.ChanCount = payload[36]
	res.RSSI = payload[37]
	return res, nil
}
