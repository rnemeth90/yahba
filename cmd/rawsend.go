/*
Copyright © 2025 Ryan Nemeth

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"net"
	"syscall"

	"github.com/rnemeth90/yahba/internal/rawsocket"
	"github.com/spf13/cobra"
)

var (
	rawSrcIP    string
	rawDstIP    string
	rawTTL      int
	rawCount    int
	rawProtocol string
)

var rawSendCmd = &cobra.Command{
	Use:   "rawsend",
	Short: "Send raw IPv4 packets with custom headers",
	Long: `Send raw IPv4 packets with fully hand-crafted IPv4 headers via a raw socket.

Supported protocols: icmp (default), tcp, udp.
For ICMP, an Echo Request payload is automatically generated.
For TCP/UDP, an empty payload is sent (extend as needed).

Requires root / CAP_NET_RAW privileges on Linux, or administrator rights on macOS/Windows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srcIP := net.ParseIP(rawSrcIP)
		if srcIP == nil {
			return fmt.Errorf("invalid source IP: %s", rawSrcIP)
		}
		dstIP := net.ParseIP(rawDstIP)
		if dstIP == nil {
			return fmt.Errorf("invalid destination IP: %s", rawDstIP)
		}
		if rawTTL < 1 || rawTTL > 255 {
			return fmt.Errorf("--ttl must be between 1 and 255")
		}
		if rawCount < 1 {
			return fmt.Errorf("--count must be at least 1")
		}

		var proto uint8
		switch rawProtocol {
		case "icmp":
			proto = syscall.IPPROTO_ICMP
		case "tcp":
			proto = syscall.IPPROTO_TCP
		case "udp":
			proto = syscall.IPPROTO_UDP
		default:
			return fmt.Errorf("unsupported protocol %q – use icmp, tcp, or udp", rawProtocol)
		}

		fd, err := rawsocket.CreateRawSocket()
		if err != nil {
			return err
		}
		defer syscall.Close(fd)

		for i := 0; i < rawCount; i++ {
			var payload []byte
			if rawProtocol == "icmp" {
				// Sequence number increments per packet; identifier is fixed.
				payload = rawsocket.ICMPEchoRequest(1, uint16(i+1))
			}

			header, err := rawsocket.NewIPHeader(srcIP, dstIP, proto, uint8(rawTTL), len(payload))
			if err != nil {
				return err
			}

			if err := rawsocket.SendPacket(fd, header, payload); err != nil {
				return fmt.Errorf("packet %d/%d: %w", i+1, rawCount, err)
			}
			fmt.Printf("Sent packet %d/%d → %s (proto=%s ttl=%d)\n",
				i+1, rawCount, rawDstIP, rawProtocol, rawTTL)
		}
		return nil
	},
}

func init() {
	rawSendCmd.Flags().StringVar(&rawSrcIP, "src", "", "Source IP address (required)")
	rawSendCmd.Flags().StringVar(&rawDstIP, "dst", "", "Destination IP address (required)")
	rawSendCmd.Flags().IntVar(&rawTTL, "ttl", 64, "IP Time to Live (1–255)")
	rawSendCmd.Flags().IntVar(&rawCount, "count", 1, "Number of packets to send")
	rawSendCmd.Flags().StringVar(&rawProtocol, "protocol", "icmp", "Protocol: icmp, tcp, or udp")
	rawSendCmd.MarkFlagRequired("src")
	rawSendCmd.MarkFlagRequired("dst")
	rootCmd.AddCommand(rawSendCmd)
}
