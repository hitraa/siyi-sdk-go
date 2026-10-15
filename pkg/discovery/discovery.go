package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hitraa/siyi-sdk-go/pkg/commands"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// Device represents a discovered SIYI camera on the network.
type Device struct {
	IP         string           `json:"ip"`
	Port       int              `json:"port"`
	HardwareID string           `json:"hardware_id"`
	ProductID  models.ProductID `json:"product_id"`
	ModelName  string           `json:"model_name"`
}

// Discover probes the local network for SIYI cameras via UDP broadcast.
// broadcastAddr can be "255.255.255.255", "192.168.144.255", or empty (defaults to 255.255.255.255).
func Discover(ctx context.Context, broadcastAddr string, timeout time.Duration) ([]Device, error) {
	if broadcastAddr == "" {
		broadcastAddr = "255.255.255.255"
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	laddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("resolve local UDP: %w", err)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	defer conn.Close()

	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, protocol.DefaultUDPPort))
	if err != nil {
		return nil, fmt.Errorf("resolve broadcast UDP: %w", err)
	}

	// Build Hardware ID query frame: STX(2) + CTRL(1) + DataLen(2) + SEQ(2) + CMD_ID(1) + CRC(2)
	reqFrame := protocol.Build(protocol.CmdRequestHardwareID, commands.EncodeHardwareID(), 1, true).ToBytes()

	_, err = conn.WriteTo(reqFrame, raddr)
	if err != nil {
		return nil, fmt.Errorf("send broadcast probe: %w", err)
	}

	var devices []Device
	seen := make(map[string]bool)

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		timeLeft := time.Until(deadline)
		if timeLeft <= 0 {
			break
		}

		_ = conn.SetReadDeadline(time.Now().Add(timeLeft))
		n, remote, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			continue
		}

		if n >= protocol.MinFrameLen {
			frame, err := protocol.FromBytes(buf[:n])
			if err == nil && frame.CmdID == protocol.CmdRequestHardwareID {
				hw, err := commands.DecodeHardwareID(frame.Data)
				if err == nil {
					udpAddr, ok := remote.(*net.UDPAddr)
					if ok {
						ipStr := udpAddr.IP.String()
						if !seen[ipStr] {
							seen[ipStr] = true
							pid, _ := hw.ProductID()
							devices = append(devices, Device{
								IP:         ipStr,
								Port:       udpAddr.Port,
								HardwareID: string(hw.Raw),
								ProductID:  pid,
								ModelName:  pid.String(),
							})
						}
					}
				}
			}
		}
	}

	return devices, nil
}
