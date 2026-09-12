//go:build !linux

package config

import "syscall"

// applySocketOpts sets IP-layer socket options on an outgoing TCP connection.
// On non-Linux platforms, IP_FREEBIND is not available, so SourceIP must be
// an address already assigned to a local network interface.
func applySocketOpts(fd uintptr, sourceIP string, ttl, tos int) {
	if ttl > 0 {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	}
	if tos > 0 {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, tos)
	}
}

// sourceIPFreeBindSupported reports whether arbitrary (non-local) source IPs
// are supported on this platform.
func sourceIPFreeBindSupported() bool { return false }
