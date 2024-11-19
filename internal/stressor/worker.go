package stressor

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/client"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
)

type Worker struct {
	ID      int
	Jobs    <-chan Job
	Results chan<- report.Result
	Client  *http.Client
	Config  config.Config
}

type Job struct {
	Host   string
	Method string
	Body   string
}

func newWorker(id int, jobs <-chan Job, results chan<- report.Result, client *http.Client, cfg config.Config) *Worker {
	return &Worker{
		ID:      id,
		Jobs:    jobs,
		Results: results,
		Client:  client,
		Config:  cfg,
	}
}

func (w *Worker) work(wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range w.Jobs {
		// Start processing the job
		w.Config.Logger.Debug("Worker %d: Starting job for %s with method %s", w.ID, job.Host, job.Method)

		// Create the HTTP request
		req, err := http.NewRequest(job.Method, job.Host, bytes.NewReader([]byte(job.Body)))
		if err != nil {
			w.Config.Logger.Error("Worker %d: Failed to create request for %s: %v", w.ID, job.Host, err)
			w.Results <- report.Result{WorkerID: w.ID, Error: err}
			continue
		}

		// Add headers to the request and log them
		for _, h := range w.Config.ParsedHeaders {
			req.Header.Add(h.Key, h.Value)
		}
		w.Config.Logger.Debug("Worker %d: Request headers set: %v", w.ID, req.Header)

		// Set HTTP protocol if HTTP/2 is disabled
		if !w.Config.HTTP2 {
			req.Proto = "HTTP/1.1"
			w.Config.Logger.Debug("Worker %d: Using HTTP/1.1 for request to %s", w.ID, job.Host)
		}

		// Track start time and bytes sent
		start := time.Now()
		result := report.Result{
			WorkerID:  w.ID,
			StartTime: start,
			Method:    job.Method,
			TargetURL: job.Host,
		}

		// Log request data and count bytes sent
		bytesSent, err := httputil.DumpRequest(req, true)
		if err != nil {
			w.Config.Logger.Warn("Worker %d: Failed to dump request for %s: %v", w.ID, job.Host, err)
			w.Results <- report.Result{WorkerID: w.ID, Error: err}
			continue
		}
		result.BytesSent = len(bytesSent)
		w.Config.Logger.Debug("Worker %d: Sent %d bytes to %s", w.ID, result.BytesSent, job.Host)

		// Execute the HTTP request
		resp, err := w.Client.Do(req)
		if err != nil {
			if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
				w.Config.Logger.Warn("Worker %d: Request to %s timed out", w.ID, job.Host)
				result.Timeout = true
				w.Results <- result
				continue
			}
			w.Config.Logger.Error("Worker %d: Request to %s failed: %v", w.ID, job.Host, err)
			result.Error = err
			w.Results <- result
			continue
		}

		bytesReceived, err := httputil.DumpResponse(resp, true)
		if err != nil {
			w.Config.Logger.Error("Worker %d: Failed to dump response from %s: %v", w.ID, job.Host, err)
			w.Results <- report.Result{WorkerID: w.ID, Error: err}
			resp.Body.Close()
			continue
		}
		result.BytesReceived = len(bytesReceived)
		w.Config.Logger.Debug("Worker %d: Received %d bytes from %s", w.ID, result.BytesReceived, job.Host)

		// Record response time and status
		result.EndTime = time.Now()
		result.ElapsedTime = result.EndTime.Sub(start)
		result.ResultCode = resp.StatusCode
		w.Config.Logger.Info("Worker %d: Completed job for %s with status %d in %s", w.ID, job.Host, result.ResultCode, result.ElapsedTime)

		// Send result and close the response body
		w.Results <- result
		resp.Body.Close()
	}
}

func WorkerPool(cfg config.Config, jobs []Job, reportChan chan<- report.Report) {
	client, err := client.NewClient(cfg)
	if err != nil {
		cfg.Logger.Error("Error creating HTTP client: %v", err)
		return
	}

	cfg.Logger.Info("Worker pool starting with %d workers", cfg.RPS)
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan report.Result, len(jobs))
	wg := &sync.WaitGroup{}

	for i := 0; i < cfg.RPS; i++ {
		worker := newWorker(i, jobChan, resultChan, client, cfg)
		wg.Add(1)
		go worker.work(wg)
	}

	// Throttle the rate at which jobs are added to the job channel based on cfg.RPS
	ticker := time.NewTicker(time.Second / time.Duration(cfg.RPS))
	defer ticker.Stop()

	// put jobs on the job channel
	go func() {
		for _, job := range jobs {
			<-ticker.C // Limit the job addition based on the RPS
			cfg.Logger.Debug("Dispatching job to %s", job.Host)
			jobChan <- job
		}
		close(jobChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	report := report.Report{}
	cfg.Logger.Info("Aggregating results into report")
	var totalRequests int
	var totalBytesSent int
	var totalBytesReceived int
	resultCodes := make(map[int]int)

	for result := range resultChan {
		resultCodes[result.ResultCode]++
		totalRequests++
		totalBytesSent += result.BytesSent
		totalBytesReceived += result.BytesReceived
		report.Results = append(report.Results, result)
		if result.ResultCode >= 400 && result.ResultCode <= 499 {
			report.ErrorBreakdown.ClientErrors += 1
		} else if result.ResultCode >= 500 && result.ResultCode <= 599 {
			report.ErrorBreakdown.ServerErrors += 1
		}

		if result.ResultCode >= 400 {
			report.Failures++
			cfg.Logger.Warn("Request failed with status code %d", result.ResultCode)
		} else {
			report.Successes++
		}
	}

	report.TotalRequests = totalRequests
	report.Throughput.TotalBytesSent = totalBytesSent
	report.Throughput.TotalBytesReceived = totalBytesReceived

	report.ConvertResultCodes(resultCodes)
	report.CalculateLatencyMetrics()

	cfg.Logger.Info("Report aggregation complete")
	reportChan <- report
	close(reportChan)
}
