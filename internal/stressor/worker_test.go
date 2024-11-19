package stressor

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
)

// Mock server to simulate HTTP requests
func mockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
}
func TestWorker(t *testing.T) {
	// Create a mock server with a small delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.Config{
		Host:    server.URL,
		Method:  http.MethodGet,
		Timeout: 1, // Timeout for the HTTP client
		Logger:  logger.NewLogger("info", "stdout"),
	}

	jobChan := make(chan Job, 1)
	resultChan := make(chan report.Result, 1)

	// Add a single job and close the channel
	job := Job{Host: cfg.Host, Method: cfg.Method}
	jobChan <- job
	close(jobChan)

	// Create a worker
	client := &http.Client{Timeout: 2 * time.Second}
	worker := newWorker(1, jobChan, resultChan, client, cfg)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Run the worker and wait with a timeout
	done := make(chan struct{})
	go func() {
		worker.work(wg)
		close(done)
	}()

	select {
	case <-done:
		// Check results
		result := <-resultChan
		if result.WorkerID != 1 {
			t.Errorf("Expected WorkerID 1, got %d", result.WorkerID)
		}
		if result.ResultCode != http.StatusOK {
			t.Errorf("Expected ResultCode 200, got %d", result.ResultCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("TestWorker timed out waiting for worker to finish")
	}
}

//
// // Test the Worker function
// func TestWorker(t *testing.T) {
// 	server := mockServer()
// 	defer server.Close()
//
// 	// Configuration and channels
// 	cfg := config.Config{
// 		Host:    server.URL,
// 		Method:  http.MethodGet,
// 		Timeout: int(5 * time.Second),
// 		HTTP2:   true,
// 		RPS:     1,
// 		Logger:  logger.NewLogger("info", "stdout"),
// 	}
// 	jobChan := make(chan Job, 1)
// 	resultChan := make(chan report.Result, 1)
// 	client := http.DefaultClient
//
// 	// Create a worker and a job
// 	job := Job{Host: server.URL, Method: http.MethodGet}
// 	jobChan <- job
//
// 	worker := newWorker(1, jobChan, resultChan, client, cfg)
// 	wg := &sync.WaitGroup{}
// 	wg.Add(1)
//
// 	close(jobChan)
// 	close(resultChan)
//
// 	// Start the worker and wait for it to finish
// 	go worker.work(wg)
// 	wg.Wait()
//
// 	// Check the results
// 	result := <-resultChan
// 	if result.WorkerID != 1 {
// 		t.Errorf("Expected WorkerID 1, got %d", result.WorkerID)
// 	}
// 	if result.ResultCode != http.StatusOK {
// 		t.Errorf("Expected ResultCode 200, got %d", result.ResultCode)
// 	}
// 	if result.BytesReceived == 0 {
// 		t.Error("Expected non-zero BytesReceived")
// 	}
// }

//
// // Test the Worker function with a timeout
// func TestWorkerTimeout(t *testing.T) {
// 	// Simulate a server with a delay to trigger a timeout
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		time.Sleep(2 * time.Second)
// 		w.WriteHeader(http.StatusOK)
// 	}))
// 	defer server.Close()
//
// 	cfg := config.Config{
// 		Host:    server.URL,
// 		Method:  http.MethodGet,
//    Timeout:  int(5 * time.Second),
// 		Logger:  logger.NewLogger("info", "stdout"),
// 	}
//
// 	jobChan := make(chan Job, 1)
// 	resultChan := make(chan report.Result, 1)
//
// 	// Add job to the job channel
// 	job := Job{Host: server.URL, Method: http.MethodGet}
// 	jobChan <- job
//
// 	client := &http.Client{
// 		Timeout: time.Duration(cfg.Timeout) * time.Second,
// 	}
// 	worker := newWorker(1, jobChan, resultChan, client, cfg)
// 	wg := &sync.WaitGroup{}
// 	wg.Add(1)
//
// 	// Start the worker and wait for it to finish
// 	go worker.work(wg)
// 	wg.Wait()
//
// 	// Check that the result is a timeout
// 	result := <-resultChan
// 	if !result.Timeout {
// 		t.Error("Expected request to timeout, but it did not")
// 	}
// 	if result.Error == nil {
// 		t.Error("Expected error due to timeout, got nil")
// 	}
// }

// Test the WorkerPool function with multiple jobs
func TestWorkerPool(t *testing.T) {
	server := mockServer()
	defer server.Close()

	cfg := config.Config{
		Host:     server.URL,
		Method:   http.MethodGet,
		Timeout:  int(5 * time.Second),
		RPS:      2,
		Requests: 5,
		Logger:   logger.NewLogger("info", "stdout"),
	}
	reportChan := make(chan report.Report, 1)
	jobs := make([]Job, cfg.Requests)

	// Create multiple jobs
	for i := 0; i < cfg.Requests; i++ {
		jobs[i] = Job{Host: cfg.Host, Method: cfg.Method}
	}

	// Start WorkerPool
	go WorkerPool(cfg, jobs, reportChan)
	report := <-reportChan
	// close(reportChan)

	// Check the report metrics
	if report.TotalRequests != cfg.Requests {
		t.Errorf("Expected TotalRequests %d, got %d", cfg.Requests, report.TotalRequests)
	}
	if report.Successes != cfg.Requests {
		t.Errorf("Expected Successes %d, got %d", cfg.Requests, report.Successes)
	}
	if report.Failures != 0 {
		t.Errorf("Expected Failures 0, got %d", report.Failures)
	}
	if report.Throughput.TotalBytesReceived == 0 {
		t.Error("Expected non-zero TotalBytesReceived in throughput")
	}
	if report.Throughput.TotalBytesSent == 0 {
		t.Error("Expected non-zero TotalBytesSent in throughput")
	}
}
