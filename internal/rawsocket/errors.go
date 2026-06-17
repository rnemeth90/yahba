package rawsocket

import "errors"

var (
	ErrInvalidSrcIP = errors.New("invalid source IP address")
	ErrInvalidDstIP = errors.New("invalid destination IP address")
	ErrSocketCreate = errors.New("failed to create raw socket")
	ErrSocketSend   = errors.New("failed to send packet")
)
