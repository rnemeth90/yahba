// Package rawsocket provides primitives for crafting and sending raw IPv4
// packets with hand-built IP headers via a raw socket.
//
// Requires root / CAP_NET_RAW on Linux, or administrator rights on macOS.
package rawsocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
)

// IPHeader represents a simplified IPv4 header (20 bytes, no options).
type IPHeader struct {
	VersionIHL uint8  // Version (4 bits) + IHL (4 bits)
	TOS        uint8  // Type of Service
	Length     uint16 // Total packet length (header + payload)
	ID         uint16 // Identification
	FlagsFO    uint16 // Flags (3 bits) + Fragment Offset (13 bits)
	TTL        uint8  // Time to Live
	Protocol   uint8  // Protocol (IPPROTO_ICMP, IPPROTO_TCP, …)
	Checksum   uint16 // Header checksum (computed; 0 during calculation)
	SrcAddr    [4]byte
	DstAddr    [4]byte
}

// NewIPHeader builds an IPv4 header and computes its checksum.
func NewIPHeader(srcIP, dstIP net.IP, protocol, ttl uint8, payloadLen int) (*IPHeader, error) {
	src := srcIP.To4()
	if src == nil {
		return nil, ErrInvalidSrcIP
	}
	dst := dstIP.To4()
	if dst == nil {
		return nil, ErrInvalidDstIP
	}

	h := &IPHeader{
		VersionIHL: 0x45, // IPv4, IHL = 5 × 4 = 20 bytes
		TOS:        0,
		Length:     uint16(20 + payloadLen),
		ID:         1,
		FlagsFO:    0,
		TTL:        ttl,
		Protocol:   protocol,
		Checksum:   0, // zero while computing
	}
	copy(h.SrcAddr[:], src)
	copy(h.DstAddr[:], dst)

	h.Checksum = IPChecksum(h.Bytes())
	return h, nil
}

// Bytes serializes the header to network (big-endian) byte order.
func (h *IPHeader) Bytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, h.VersionIHL)
	binary.Write(&buf, binary.BigEndian, h.TOS)
	binary.Write(&buf, binary.BigEndian, h.Length)
	binary.Write(&buf, binary.BigEndian, h.ID)
	binary.Write(&buf, binary.BigEndian, h.FlagsFO)
	binary.Write(&buf, binary.BigEndian, h.TTL)
	binary.Write(&buf, binary.BigEndian, h.Protocol)
	binary.Write(&buf, binary.BigEndian, h.Checksum)
	buf.Write(h.SrcAddr[:])
	buf.Write(h.DstAddr[:])
	return buf.Bytes()
}

// IPChecksum computes the one's-complement checksum used for IPv4 and ICMP
// headers. Pass a header serialized with Checksum = 0.
func IPChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(data); i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 != 0 {
		sum += uint32(data[len(data)-1]) << 8
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return ^uint16(sum)
}

// ICMPEchoRequest builds an 8-byte ICMP Echo Request payload (type 8, code 0)
// with the given identifier and sequence number.
func ICMPEchoRequest(id, seq uint16) []byte {
	msg := make([]byte, 8)
	msg[0] = 8 // type: Echo Request
	msg[1] = 0 // code: 0
	// msg[2:4] checksum – left zero during calculation
	binary.BigEndian.PutUint16(msg[4:6], id)
	binary.BigEndian.PutUint16(msg[6:8], seq)
	binary.BigEndian.PutUint16(msg[2:4], IPChecksum(msg))
	return msg
}

// CreateRawSocket opens an AF_INET raw socket and sets IP_HDRINCL so the
// caller provides a complete IP header.
func CreateRawSocket() (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrSocketCreate, err)
	}
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd)
		return 0, fmt.Errorf("%w: IP_HDRINCL: %v", ErrSocketCreate, err)
	}
	return fd, nil
}

// SendPacket assembles the final wire packet and transmits it.
func SendPacket(fd int, header *IPHeader, payload []byte) error {
	packet := append(header.Bytes(), payload...)
	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], header.DstAddr[:])
	if err := syscall.Sendto(fd, packet, 0, addr); err != nil {
		return fmt.Errorf("%w: %v", ErrSocketSend, err)
	}
	return nil
}
