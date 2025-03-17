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
	assert.NotNil(t, w)
	assert.Equal(t, w.ID, 1)
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

func TestProcessJob(t *testing.T) {
	mockConfig := config.Config{}
	mockClient := http.Client{}
	worker := NewWorker(1, nil, nil, &mockClient, mockConfig)

	job := Job{
		ID:     1,
		Host:   mockServer().URL,
		Method: "GET",
		Body:   "",
	}

	req, err := http.NewRequest(job.Method, job.Host, nil)
	assert.NoError(t, err)

	worker.setHeaders(req)

	resp, err := worker.Client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	defer resp.Body.Close()
}
