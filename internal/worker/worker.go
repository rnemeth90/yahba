package worker

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/client"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
)

type Worker struct {
	ID      int
	Jobs    <-chan Job
	Results chan<- report.Result
	Client  *http.Client
	Config  config.Config
}

type watcher interface {
	watch(context.Context, *sync.WaitGroup)
}

type jobExecutor interface {
	processJob(Job)
	initializeResult(Job, time.Time) report.Result
	processResponse(report.Result, *http.Response, time.Time, time.Time, Job, int)
}

type requestProcessor interface {
	createRequest(Job) (*http.Request, error)
	setHeaders(*http.Request)
}

type Job struct {
	ID     int
	Host   string
	Method string
	Body   string
}

type WorkerFactory func(i int, jobs <-chan Job, results chan<- report.Result, client *http.Client, cfg config.Config) Worker

// Create a new worker instance
func NewWorker(id int, jobs <-chan Job, results chan<- report.Result, client *http.Client, cfg config.Config) *Worker {
	return &Worker{
		ID:      id,
		Jobs:    jobs,
		Results: results,
		Client:  client,
		Config:  cfg,
	}
}

// worker loop for processing jobs
func (w *Worker) watch(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			w.Config.Logger.Info("worker %d shutting down", w.ID)
			return
		case job, ok := <-w.Jobs:
			if !ok {
				return
			}
			w.processJob(job)
		}
	}
}

// Process a single job
func (w *Worker) processJob(job Job) {
	w.Config.Logger.Debug("worker %d: Starting job for %s with method %s", w.ID, job.Host, job.Method)
	req, err := w.createRequest(job)
	if err != nil {
		w.handleRequestError(job, err)
		return
	}

	reqSize, err := util.CalculateRawRequestSize(req)
	if err != nil {
		w.Config.Logger.Error("failed to obtain request size: %v", err)
	}

	w.setHeaders(req)

	start := time.Now()
	result := w.initializeResult(job, start)

	resp, err := w.Client.Do(req)
	if err != nil {
		end := time.Now()
		w.handleClientError(job, result, resp, err, start, end)
		return
	}
	end := time.Now()

	defer resp.Body.Close()
	w.processResponse(result, resp, start, end, job, reqSize)
}

// todo: create a comment for this function
func Work(ctx context.Context, cfg config.Config, jobs []Job, reportChan chan<- report.Report, factory WorkerFactory) {
	client, err := client.NewClient(cfg)
	if err != nil {
		cfg.Logger.Error("Error creating HTTP client: %v", err)
		return
	}

	numWorkers := cfg.RPS * 10
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan report.Result, len(jobs))

	wg := &sync.WaitGroup{}

	cfg.Logger.Info("Starting worker pool with %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := factory(i, jobChan, resultChan, client, cfg)
		wg.Add(1)
		go worker.watch(ctx, wg)
	}

	go func() {
		defer close(jobChan)
		ticker := time.NewTicker(time.Second / time.Duration(cfg.RPS))
		defer ticker.Stop()

		for _, job := range jobs {
			select {
			case <-ctx.Done():
				return
			case jobChan <- job:
				<-ticker.C
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	cfg.Logger.Info("Aggregating results into report")
	report := processResults(cfg, resultChan)

	cfg.Logger.Info("Report aggregation complete")
	reportChan <- report
	close(reportChan)
}

// Process results from workers
func processResults(cfg config.Config, resultChan <-chan report.Result) report.Report {
	report := report.Report{}
	var totalRequests, totalBytesSent, totalBytesReceived int
	resultCodes := make(map[int]int)
	var duration time.Duration

	for result := range resultChan {
		totalRequests++
		resultCodes[result.ResultCode]++
		totalBytesSent += result.BytesSent
		totalBytesReceived += result.BytesReceived
		duration += result.ElapsedTime

		report.Results = append(report.Results, result)

		// check client errors vs. server errors
		if result.ResultCode >= 400 && result.ResultCode <= 499 {
			report.ErrorBreakdown.ClientErrors++
		} else if result.ResultCode >= 500 && result.ResultCode <= 599 {
			report.ErrorBreakdown.ServerErrors++
		}

		// count all failed requests
		if result.ResultCode >= 400 {
			report.Failures++
			cfg.Logger.Warn("Request failed with status code %d", result.ResultCode)
		} else {
			// count all successes
			report.Successes++
		}

		// provide occassional status updates
		if totalRequests%1000 == 0 {
			cfg.Logger.Info("Processed %d requests...", totalRequests)
		}
	}

	report.TotalRequests = totalRequests
	report.Throughput.TotalBytesSent = totalBytesSent
	report.Throughput.TotalBytesReceived = totalBytesReceived
	report.Throughput.BytesSentPerSecond = util.CalculateBytesPerSecond(float64(totalBytesSent), duration.Seconds())
	report.Throughput.BytesReceivedPerSecond = util.CalculateBytesPerSecond(float64(totalBytesReceived), duration.Seconds())
	report.ConvertResultCodes(resultCodes)
	report.CalculateLatencyMetrics()

	return report
}

// Create a new HTTP request
func (w *Worker) createRequest(job Job) (*http.Request, error) {
	req, err := http.NewRequest(job.Method, job.Host, bytes.NewReader([]byte(job.Body)))
	if err != nil {
		w.Config.Logger.Error("worker %d: Failed to create request for %s: %v", w.ID, job.Host, err)
	}
	return req, err
}

// Set headers for the HTTP request
func (w *Worker) setHeaders(req *http.Request) {
	for _, h := range w.Config.ParsedHeaders {
		req.Header.Add(h.Key, h.Value)
	}
	w.Config.Logger.Debug("worker %d: Request headers set: %v", w.ID, req.Header)
}

// Initialize the result object
func (w *Worker) initializeResult(job Job, start time.Time) report.Result {
	return report.Result{
		WorkerID:  w.ID,
		StartTime: start,
		Method:    job.Method,
		TargetURL: job.Host,
	}
}

// Process the HTTP response
func (w *Worker) processResponse(result report.Result, resp *http.Response, start time.Time, end time.Time, job Job, bytesSent int) {
	if resp == nil {
		w.Config.Logger.Error("worker %d: No response received for %s", w.ID, job.Host)
		result.Error = fmt.Errorf("no response received")
		result.EndTime = end
		result.ElapsedTime = result.EndTime.Sub(start)
		w.Results <- result
		return
	}

	bytesReceived, err := httputil.DumpResponse(resp, true)
	if err != nil {
		w.Config.Logger.Error("worker %d: Failed to dump response from %s: %v", w.ID, job.Host, err)
		result.Error = err
		result.EndTime = time.Now()
		result.ElapsedTime = result.EndTime.Sub(start)
		w.Results <- result
		return
	}

	result.BytesReceived = len(bytesReceived)
	result.BytesSent = bytesSent
	w.Config.Logger.Debug("worker %d: Received %d bytes from %s", w.ID, result.BytesReceived, job.Host)

	result.EndTime = end
	result.ElapsedTime = result.EndTime.Sub(start)
	result.ResultCode = resp.StatusCode

	w.Config.Logger.Debug("worker %d: Completed job for %s with status %d in %s", w.ID, job.Host, result.ResultCode, result.ElapsedTime)
	w.Results <- result
}
