//go:build linux || darwin

package rawsocket

import (
	"fmt"
	"syscall"
)

// FD is a raw socket file descriptor.
type FD int

// CreateRawSocket opens an AF_INET raw socket and sets IP_HDRINCL so the
// caller provides a complete IP header.
func CreateRawSocket() (FD, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrSocketCreate, err)
	}
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd)
		return 0, fmt.Errorf("%w: IP_HDRINCL: %v", ErrSocketCreate, err)
	}
	return FD(fd), nil
}

// SendPacket assembles the final wire packet and transmits it.
func SendPacket(fd FD, header *IPHeader, payload []byte) error {
	packet := append(header.Bytes(), payload...)
	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], header.DstAddr[:])
	if err := syscall.Sendto(int(fd), packet, 0, addr); err != nil {
		return fmt.Errorf("%w: %v", ErrSocketSend, err)
	}
	return nil
}

// Close releases the raw socket.
func Close(fd FD) error {
	return syscall.Close(int(fd))
}
