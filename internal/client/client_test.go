package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnemeth90/yahba/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if client == nil {
		t.Fatalf("client is nil")
	}
}

func TestSendRequest(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer testServer.Close()

	req := httptest.NewRequest("GET", testServer.URL, nil)
	req.RequestURI = ""

	cfg := config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if client == nil {
		t.Fatalf("client is nil")
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

}
