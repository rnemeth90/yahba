//go:build linux

package config

import "syscall"

// applySocketOpts sets IP-layer socket options on an outgoing TCP connection.
// On Linux, IP_FREEBIND is set whenever a SourceIP is requested, which allows
// binding to any IPv4 address — including addresses not assigned to a local
// interface. This enables true source-IP spoofing for HTTP load tests, matching
// the behaviour of the raw packet sender.
//
// Requirements:
//   - IP_FREEBIND with a non-local address requires the server process to have
//     CAP_NET_RAW or run as root.
//   - For TCP connections to complete, return traffic must be routed back to the
//     machine (e.g. via IP aliases, policy routing, or lo:0 aliases).
func applySocketOpts(fd uintptr, sourceIP string, ttl, tos int) {
	if sourceIP != "" {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_FREEBIND, 1)
	}
	if ttl > 0 {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	}
	if tos > 0 {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, tos)
	}
}

// sourceIPFreeBindSupported reports whether arbitrary (non-local) source IPs
// are supported on this platform.
func sourceIPFreeBindSupported() bool { return true }
