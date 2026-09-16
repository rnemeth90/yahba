//go:build windows

package rawsocket

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// ipprotoRaw is the IANA "raw IP packet" pseudo-protocol number (255). It is
// not exposed by golang.org/x/sys/windows, so it is defined locally.
const ipprotoRaw = 255

// FD is a raw socket handle.
type FD windows.Handle

// CreateRawSocket opens an AF_INET raw socket and sets IP_HDRINCL so the
// caller provides a complete IP header.
func CreateRawSocket() (FD, error) {
	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, ipprotoRaw)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrSocketCreate, err)
	}
	if err := windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1); err != nil {
		windows.Closesocket(fd)
		return 0, fmt.Errorf("%w: IP_HDRINCL: %v", ErrSocketCreate, err)
	}
	return FD(fd), nil
}

// SendPacket assembles the final wire packet and transmits it.
func SendPacket(fd FD, header *IPHeader, payload []byte) error {
	packet := append(header.Bytes(), payload...)
	addr := &windows.SockaddrInet4{}
	copy(addr.Addr[:], header.DstAddr[:])
	if err := windows.Sendto(windows.Handle(fd), packet, 0, addr); err != nil {
		return fmt.Errorf("%w: %v", ErrSocketSend, err)
	}
	return nil
}

// Close releases the raw socket.
func Close(fd FD) error {
	return windows.Closesocket(windows.Handle(fd))
}
