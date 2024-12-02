package stressor

import (
	"bytes"
	"net/http"
	"net/http/httputil"
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

// todo: rename to new
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
		w.Config.Logger.Debug("Worker %d: Starting job for %s with method %s", w.ID, job.Host, job.Method)

		req, err := w.createRequest(job)
		if err != nil {
			w.handleRequestError(job, err)
			continue
		}

		w.setHeaders(req)
		w.setProtocol(req)

		start := time.Now()
		result := w.initializeResult(job, start)

		resp, err := w.Client.Do(req)
		if err != nil {
			w.handleClientError(job, result, resp, err, start)
			continue
		}

		w.processResponse(result, resp, start, job)
		resp.Body.Close()
	}
}

func WorkerPool(cfg config.Config, jobs []Job, reportChan chan<- report.Report) {
	client, err := client.NewClient(cfg)
	if err != nil {
		cfg.Logger.Error("Error creating HTTP client: %v", err)
		return
	}

	cfg.Logger.Info("Starting worker pool with %d workers", cfg.RPS)

	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan report.Result, len(jobs))

	wg := &sync.WaitGroup{}
	for i := 0; i < cfg.RPS; i++ {
		worker := newWorker(i, jobChan, resultChan, client, cfg)
		wg.Add(1)
		go worker.work(wg)
	}

	go func() {
		defer close(jobChan)
		ticker := time.NewTicker(time.Second / time.Duration(cfg.RPS))
		defer ticker.Stop()

		for _, job := range jobs {
			<-ticker.C
			jobChan <- job
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

// processResults handles result aggregation logic
func processResults(cfg config.Config, resultChan <-chan report.Result) report.Report {
	report := report.Report{}
	var totalRequests, totalBytesSent, totalBytesReceived int
	resultCodes := make(map[int]int)

	for result := range resultChan {
		resultCodes[result.ResultCode]++
		totalRequests++
		totalBytesSent += result.BytesSent
		totalBytesReceived += result.BytesReceived
		report.Results = append(report.Results, result)

		if result.ResultCode >= 400 && result.ResultCode <= 499 {
			report.ErrorBreakdown.ClientErrors++
		} else if result.ResultCode >= 500 && result.ResultCode <= 599 {
			report.ErrorBreakdown.ServerErrors++
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

	return report
}

func (w *Worker) createRequest(job Job) (*http.Request, error) {
	req, err := http.NewRequest(job.Method, job.Host, bytes.NewReader([]byte(job.Body)))
	if err != nil {
		w.Config.Logger.Error("Worker %d: Failed to create request for %s: %v", w.ID, job.Host, err)
	}
	return req, err
}

func (w *Worker) setHeaders(req *http.Request) {
	for _, h := range w.Config.ParsedHeaders {
		req.Header.Add(h.Key, h.Value)
	}
	w.Config.Logger.Debug("Worker %d: Request headers set: %v", w.ID, req.Header)
}

func (w *Worker) setProtocol(req *http.Request) {
	if !w.Config.HTTP2 {
		req.Proto = "HTTP/1.1"
		w.Config.Logger.Debug("Worker %d: Using HTTP/1.1 for request to %s", w.ID, req.URL.Host)
	}
}

func (w *Worker) initializeResult(job Job, start time.Time) report.Result {
	return report.Result{
		WorkerID:  w.ID,
		StartTime: start,
		Method:    job.Method,
		TargetURL: job.Host,
	}
}

func (w *Worker) processResponse(result report.Result, resp *http.Response, start time.Time, job Job) {
	bytesReceived, err := httputil.DumpResponse(resp, true)
	if err != nil {
		w.Config.Logger.Error("Worker %d: Failed to dump response from %s: %v", w.ID, job.Host, err)
		result.Error = err
		result.EndTime = time.Now()
		result.ElapsedTime = result.EndTime.Sub(start)
		w.Results <- result
		return
	}

	result.BytesReceived = len(bytesReceived)
	w.Config.Logger.Debug("Worker %d: Received %d bytes from %s", w.ID, result.BytesReceived, job.Host)

	result.EndTime = time.Now()
	result.ElapsedTime = result.EndTime.Sub(start)
	result.ResultCode = resp.StatusCode

	w.Config.Logger.Debug("Worker %d: Completed job for %s with status %d in %s", w.ID, job.Host, result.ResultCode, result.ElapsedTime)
	w.Results <- result
}
