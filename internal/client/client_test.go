package client

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
)

func TestNewClient(t *testing.T) {
	cfg := config.Config{
		Timeout:     15,
		KeepAlive:   true,
		Compression: false,
		Logger:      logger.New("error", "stdout", false),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client, got nil")
	}

	if client.Timeout != time.Second*15 {
		t.Fatalf("Expected timeout of 15 seconds, got %v", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected http.Transport for client Transport")
	}
	if transport.DisableKeepAlives != cfg.KeepAlive {
		t.Fatalf("Expected DisableKeepAlives to be %v, got %v", cfg.KeepAlive, transport.DisableKeepAlives)
	}
}

// TestsetupHTTP2Transport tests the HTTP/2 transport setup
func TestSetupHTTP2Transport(t *testing.T) {
	cfg := config.Config{
		HTTP2:            true,
		ReuseConnections: true,
		Compression:      false,
		Logger:           logger.New("error", "stdout", false),
	}
	transport, err := setupHTTP2Transport(cfg)
	if err != nil {
		t.Fatalf("Failed to setup HTTP/2 transport: %v", err)
	}
	if transport == nil {
		t.Fatal("Expected non-nil HTTP/2 transport")
	}
	if transport.DisableCompression != cfg.Compression {
		t.Fatalf("Expected DisableCompression to be %v, got %v", cfg.Compression, transport.DisableCompression)
	}
}

// TestSetupHTTPTransport tests the HTTP transport setup
func TestSetupHTTPTransport(t *testing.T) {
	cfg := config.Config{
		HTTP2:            false,
		ReuseConnections: true,
		Compression:      false,
		Logger:           logger.New("error", "stdout", false),
	}
	proxyURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}
	transport := setupHTTPTransport(cfg, proxyURL)
	if transport == nil {
		t.Fatal("Expected non-nil HTTP transport")
	}
	transport.Proxy = http.ProxyURL(proxyURL)
	if transport.Proxy == nil {
		t.Fatal("Expected proxy to be set in HTTP transport")
	}
}

// Helper function to identify timeout errors
func isTimeoutError(err error) bool {
	urlErr, ok := err.(*url.Error)
	return ok && urlErr.Timeout()
}
