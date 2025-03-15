package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/stretchr/testify/assert"
)

// Mock server to simulate HTTP requests
func mockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
}

func TestNewWorker(t *testing.T) {
	w := NewWorker(1, nil, nil, nil, config.Config{})
	if w == nil {
		t.Error("Expected worker to not be nil")
	}

	// Check worker ID
	if w.ID != 1 {
		t.Errorf("Expected worker ID to be 1, got %d", w.ID)
	}
}

func TestCreateRequest(t *testing.T) {
	mockConfig := config.Config{}
	mockClient := http.Client{}
	worker := NewWorker(1, nil, nil, &mockClient, mockConfig)

	job := Job{
		ID:     1,
		Host:   "http://example.com",
		Method: "GET",
		Body:   "",
	}

	req, err := worker.createRequest(job)

	assert.NoError(t, err)
	assert.Equal(t, req.Method, "GET")
	assert.Equal(t, req.URL.String(), "http://example.com")
}
