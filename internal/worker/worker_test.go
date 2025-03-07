package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/util"
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

// func TestWatch(t *testing.T) {
//
// }

func TestProcessJob(t *testing.T) {
	worker := newWorker(1, nil, nil, nil, config.Config{})
	job := Job{}

	request, err := worker.createRequest(job)
	if err != nil {
		t.Fatal("failed to create request: %v\n", err)
	}



}

func TestWork(t *testing.T) {

}

func TestProcessResults(t *testing.T) {

}

func TestCreateRequest(t *testing.T) {

}

func TestSetHeaders(t *testing.T) {
	cfg := config.Config{
		Headers: "Content-Type: application/json,Authorization: Bearer token,User-Agent: yahba",
	}

	var err error
	cfg.ParsedHeaders, err = util.ParseHeaders(cfg.Headers)
	if err != nil {
		t.Fatalf("failed to parse headers: %v", err)
	}

	request := http.Request{}
	for _, header := range cfg.ParsedHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	if request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type header to be set")
	}

	if request.Header.Get("Authorization") != "Bearer token" {
		t.Errorf("expected Authorization header to be set")
	}

	if request.Header.Get("User-Agent") != "yahba" {
		t.Errorf("expected User-Agent header to be set")
	}
}

func TestInitializeResult(t *testing.T) {

}

func TestProcessResponse(t *testing.T) {

}
