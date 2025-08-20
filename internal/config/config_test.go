package config

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/logger"
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
	Resolver *net.Resolver
	Timeout  time.Duration
}

func (m *MockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	m.Called = true
	m.AddrUsed = address
	if m.Err != nil {
		return nil, m.Err
	}
	return &MockConn{}, nil

}

func TestSetupProxy(t *testing.T) {
	want, _ := url.Parse("http://www.google.com")
	config := &Config{
		Proxy:  "http://www.google.com",
		Logger: logger.New("error", "stdout", false),
	}

	proxy, err := config.SetupProxy()
	if err != nil {
		t.Fatalf("SetupProxy() error = %v, wantErr %v", err, nil)
	}

	if proxy.String() != want.String() {
		t.Errorf("SetupProxy() got = %v, want %v", proxy.String(), want.String())
	}
}

func TestSkipNameResolution(t *testing.T) {
	cfg := &Config{
		URL:    "127.0.0.1",
		Logger: logger.New("error", "stdout", false),
	}

	tr := &http.Transport{}
	mockDialer := &MockDialer{}
	cfg.SkipNameResolution(tr, mockDialer)

	dial := tr.DialContext
	if dial == nil {
		t.Fatal("DialContext not set")
	}

	ctx := context.Background()
	_, _ = dial(ctx, "tcp", "somehost:9999")

	if !mockDialer.Called {
		t.Error("Expected DialContext to be called")
	}

	expected := "127.0.0.1:9999"
	if mockDialer.AddrUsed != expected {
		t.Errorf("Expected address %s, got %s", expected, mockDialer.AddrUsed)
	}
}

func TestSetupCustomResolver(t *testing.T) {
	config := &Config{
		Resolver: "8.8.8.8",
		Logger:   logger.New("error", "stdout", false),
	}
	tr := &http.Transport{}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &MockDialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := &MockDialer{
						Timeout: 2 * time.Second,
					}
					return d.DialContext(ctx, "udp", config.Resolver)
				},
			},
		}
		return dialer.DialContext(ctx, network, addr)
	}

	if tr.DialContext == nil {
		t.Fatal("DialContext not set")
	}
}
