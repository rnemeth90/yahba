package config

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestSetupProxy(t *testing.T) {
	want := &url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		User:   url.UserPassword("user", "password"),
	}

	c := Config{
		Proxy:         "http://localhost:8080",
		ProxyUser:     "user",
		ProxyPassword: "password",
	}

	got, err := c.SetupProxy()
	if err != nil {
		t.Fatalf("SetupProxy() returned an error: %v", err)
	}

	if got.String() != want.String() {
		t.Fatalf("SetupProxy() = %v; want %v", got, want)
	}
}

func TestSkipNameResolution(t *testing.T) {
	c := Config{
		URL:     "http://127.0.0.1:8080",
		SkipDNS: true,
	}
	transport := &http.Transport{}

	mockDialFunc := func(ctx context.Context, network string, addr string) (net.Conn, error) {
		if addr == "127.0.0.1:8080" {
			return nil, nil
		}

		return nil, errors.New("unexpected address")
	}

	c.SkipNameResolution(transport, mockDialFunc)

	// Ensure that DialContext was set
	if transport.DialContext == nil {
		t.Fatalf("SkipNameResolution() did not set the DialContext field")
	}

	// Test that the function correctly bypasses DNS resolution
	_, err := transport.DialContext(context.Background(), "tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("SkipNameResolution() failed to bypass DNS resolution: %v", err)
	}

	// Test with an unexpected address
	_, err = transport.DialContext(context.Background(), "tcp", "10.0.0.1:8080")
	if err == nil {
		t.Fatalf("SkipNameResolution() should fail for unexpected address but didn't")
	}
}

func TestSetupCustomResolver(t *testing.T) {

}
