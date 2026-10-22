package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
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

	tmpl *requestTemplate
}

// requestTemplate is built once per distinct job and cloned for each send.
type requestTemplate struct {
	host, method, body string
	req                *http.Request
	size               int
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
	// Scheduled is when an open-model dispatcher intended to send the job.
	// Latency is measured from it so queueing delay is not hidden
	// (coordinated omission). Zero means measure from the actual send.
	Scheduled time.Time
}

type WorkerFactory func(id int, jobs <-chan Job, results chan<- report.Result, client *http.Client, cfg config.Config) Worker

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
	req, reqSize, err := w.newRequest(job)
	if err != nil {
		w.handleRequestError(job, err)
		return
	}

	start := time.Now()
	if !job.Scheduled.IsZero() && job.Scheduled.Before(start) {
		start = job.Scheduled
	}
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

// Work dispatches jobs to a worker pool at the configured RPS and collects results
// into a final report. If progressChan is non-nil, each individual result is also
// forwarded there for real-time progress tracking (e.g. from a TUI).
func Work(ctx context.Context, cfg config.Config, jobs []Job, reportChan chan<- report.Report, factory WorkerFactory, progressChan chan<- report.Result) {
	client, err := client.NewClient(cfg)
	if err != nil {
		cfg.Logger.Error("Error creating HTTP client: %v", err)
		return
	}

	sleep := time.Duration(cfg.Sleep) * time.Second
	produce := pacedProducer(cfg.RPS, len(jobs), 0, sleep, func(i int) Job { return jobs[i] })
	rep := runPool(ctx, cfg, client, factory, produce, numWorkers(cfg), progressChan)
	rep.Mode = config.ModeRequests
	rep.ConfiguredRPS = cfg.RPS
	warnIfBelowTarget(cfg, rep)

	reportChan <- rep
	close(reportChan)
}

// producer feeds jobs into the pool and returns when it is done or ctx ends.
type producer func(ctx context.Context, jobs chan<- Job)

const (
	resultBuffer = 1024
	// sends due within this window are not slept for; the timer error would exceed it
	minSleep = 100 * time.Microsecond
)

func numWorkers(cfg config.Config) int {
	const maxWorkers = 1000
	n := cfg.Workers
	if n <= 0 {
		n = cfg.RPS * 10
	}
	if n > maxWorkers {
		n = maxWorkers
	}
	if n < 1 {
		n = 1
	}
	return n
}

// runPool starts workers, runs the producer, and aggregates results into a report.
func runPool(ctx context.Context, cfg config.Config, client *http.Client, factory WorkerFactory, produce producer, workers int, progressChan chan<- report.Result) report.Report {
	return runPoolBuffered(ctx, cfg, client, factory, produce, workers, workers, progressChan)
}

// runPoolBuffered is runPool with an explicit dispatch channel capacity;
// 0 gives a strict closed loop.
func runPoolBuffered(ctx context.Context, cfg config.Config, client *http.Client, factory WorkerFactory, produce producer, workers, jobBuffer int, progressChan chan<- report.Result) report.Report {
	jobChan := make(chan Job, jobBuffer)
	resultChan := make(chan report.Result, resultBuffer)
	wg := &sync.WaitGroup{}

	cfg.Logger.Info("Starting worker pool with %d workers", workers)
	for i := 0; i < workers; i++ {
		worker := factory(i, jobChan, resultChan, client, cfg)
		wg.Add(1)
		go worker.watch(ctx, wg)
	}

	go func() {
		defer close(jobChan)
		produce(ctx, jobChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	cfg.Logger.Info("Aggregating results into report")
	rep := processResults(cfg, resultChan, progressChan)
	cfg.Logger.Info("Report aggregation complete")
	return rep
}

// pacedProducer dispatches job(i) at a constant rate against an absolute
// schedule (start + i/rps), so a slow response never lowers the offered rate
// and the dispatcher catches up when workers free up. It stops after count
// jobs or once duration has elapsed (0 disables either limit).
func pacedProducer(rps, count int, duration, sleep time.Duration, job func(i int) Job) producer {
	return func(ctx context.Context, jobs chan<- Job) {
		start := time.Now()
		var offset time.Duration
		for i := 0; count == 0 || i < count; i++ {
			rel := time.Duration(float64(i) * float64(time.Second) / float64(rps))
			if duration > 0 && rel >= duration {
				return
			}
			due := start.Add(offset + rel)
			if !sleepUntil(ctx, due) {
				return
			}

			j := job(i)
			j.Scheduled = due
			select {
			case <-ctx.Done():
				return
			case jobs <- j:
			}

			if sleep > 0 {
				offset += sleep
				if !sleepUntil(ctx, time.Now().Add(sleep)) {
					return
				}
			}
		}
	}
}

// sleepUntil waits until t, returning false if ctx ended first.
func sleepUntil(ctx context.Context, t time.Time) bool {
	d := time.Until(t)
	if d <= minSleep {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func warnIfBelowTarget(cfg config.Config, rep report.Report) {
	if rep.ConfiguredRPS > 0 && rep.AchievedRPS < float64(rep.ConfiguredRPS)*0.95 {
		cfg.Logger.Warn("achieved %.1f RPS, below the configured %d RPS: increase --workers or the server is saturated", rep.AchievedRPS, rep.ConfiguredRPS)
	}
}

// Process results from workers, optionally forwarding each result to progressChan.
func processResults(cfg config.Config, resultChan <-chan report.Result, progressChan chan<- report.Result) report.Report {
	report := report.Report{}
	var totalRequests, totalBytesSent, totalBytesReceived int
	resultCodes := make(map[int]int)

	for result := range resultChan {
		if progressChan != nil {
			progressChan <- result
		}
		totalRequests++
		resultCodes[result.ResultCode]++
		totalBytesSent += result.BytesSent
		totalBytesReceived += result.BytesReceived

		report.Results = append(report.Results, result)

		// check client errors vs. server errors
		if result.ResultCode >= 400 && result.ResultCode <= 499 {
			report.ErrorBreakdown.ClientErrors++
		} else if result.ResultCode >= 500 && result.ResultCode <= 599 {
			report.ErrorBreakdown.ServerErrors++
		}

		// count all failed requests
		if result.ResultCode == 0 || result.ResultCode >= 400 {
			report.Failures++
			if result.ResultCode == 0 {
				cfg.Logger.Warn("Request failed with no response (connection error)")
			} else {
				cfg.Logger.Warn("Request failed with status code %d", result.ResultCode)
			}
		} else {
			// count all successes
			report.Successes++
		}

		// provide occassional status updates
		if totalRequests%1000 == 0 {
			cfg.Logger.Info("Processed %d requests...", totalRequests)
		}
	}

	if progressChan != nil {
		close(progressChan)
	}

	report.Host = cfg.URL
	report.Method = cfg.Method
	report.TotalRequests = totalRequests
	report.Throughput.TotalBytesSent = totalBytesSent
	report.Throughput.TotalBytesReceived = totalBytesReceived
	report.StatusCodes = resultCodes
	report.Finalize()
	report.CalculateLatencyMetrics()

	return report
}

// newRequest returns a request for the job and its raw size, cloning a cached
// template instead of rebuilding the request and headers for every job.
func (w *Worker) newRequest(job Job) (*http.Request, int, error) {
	t := w.tmpl
	if t == nil || t.host != job.Host || t.method != job.Method || t.body != job.Body {
		req, err := w.createRequest(job)
		if err != nil {
			return nil, 0, err
		}
		w.setHeaders(req)
		size, err := util.CalculateRawRequestSize(req)
		if err != nil {
			w.Config.Logger.Error("failed to obtain request size: %v", err)
		}
		t = &requestTemplate{host: job.Host, method: job.Method, body: job.Body, req: req, size: size}
		w.tmpl = t
	}

	req := t.req.Clone(context.Background())
	if t.body != "" {
		// Clone shares the template's body reader, which must not be reused
		req.Body = io.NopCloser(bytes.NewReader([]byte(t.body)))
	}

	size := t.size
	if w.Config.RandomUserAgent {
		req.Header.Set("User-Agent", util.RandomUserAgent())
		var err error
		if size, err = util.CalculateRawRequestSize(req); err != nil {
			w.Config.Logger.Error("failed to obtain request size: %v", err)
		}
	}
	return req, size, nil
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
	if w.Config.RandomUserAgent {
		req.Header.Set("User-Agent", util.RandomUserAgent())
	} else {
		req.Header.Set("User-Agent", util.DefaultUserAgent)
	}

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

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		w.Config.Logger.Error("worker %d: Failed to read response from %s: %v", w.ID, job.Host, err)
		result.Error = err
		result.EndTime = time.Now()
		result.ElapsedTime = result.EndTime.Sub(start)
		w.Results <- result
		return
	}

	result.BytesReceived = int(n)
	result.BytesSent = bytesSent
	w.Config.Logger.Debug("worker %d: Received %d bytes from %s", w.ID, result.BytesReceived, job.Host)

	result.EndTime = end
	result.ElapsedTime = result.EndTime.Sub(start)
	result.ResultCode = resp.StatusCode

	w.Config.Logger.Debug("worker %d: Completed job for %s with status %d in %s", w.ID, job.Host, result.ResultCode, result.ElapsedTime)
	w.Results <- result
}
