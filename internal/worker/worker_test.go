package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnemeth90/yahba/internal/config"
)

// Mock server to simulate HTTP requests
func mockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
}

func TestNewWorker(t *testing.T) {
	// just test the worker ID
	want := &Worker{
		ID: 1,
	}

	got := newWorker(1, nil, nil, nil, config.Config{})

	if got.ID != want.ID {
		t.Errorf("Expected ID %d, got %d", want.ID, got.ID)
	}
}
