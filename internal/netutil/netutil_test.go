package netutil

import (
	"context"
	"net"
	"time"
)

type MockConn struct{}

func (m *MockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *MockConn) Write(b []byte) (n int, err error)  { return 0, nil }
func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

type MockDialer struct {
	Called   bool
	AddrUsed string
	Err      error
}

func (m *MockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	m.Called = true
	m.AddrUsed = address
	if m.Err != nil {
		return nil, m.Err
	}
	return &MockConn{}, nil
}
