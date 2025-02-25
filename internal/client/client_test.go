package client

import (
	"net/http"
	"net/http/httptest"
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

func TestGETRequest(t *testing.T) {
	// Set up a mock server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	cfg := config.Config{
		URL:    testServer.URL,
		Method: http.MethodGet,
		Logger: logger.New("error", "stdout", false),
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", testServer.URL, nil)
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %v", response.StatusCode)
	}
}

// Helper function to identify timeout errors
func isTimeoutError(err error) bool {
	urlErr, ok := err.(*url.Error)
	return ok && urlErr.Timeout()
}
