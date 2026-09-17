/*
Copyright © 2025 Ryan Nemeth

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rnemeth90/yahba/internal/api"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
	"github.com/spf13/cobra"
)

var (
	distributed     bool
	coordinatorAddr string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an HTTP performance test",
	Run: func(cmd *cobra.Command, args []string) {
		c.Logger = logger.New(c.LogLevel, c.OutputFile, c.Silent)

		ctx, cancel := context.WithCancel(context.Background())
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-shutdown
			c.Logger.Info("Shutting down...")
			cancel()
		}()

		if c.OutputFormat == "json" || c.OutputFormat == "yaml" {
			c.Logger.Silent = true
		}

		c.Logger.Debug("Starting YAHBA")
		if err := run(ctx, c); err != nil {
			c.Logger.Error("Application encountered a critical error: %v", err)
			return
		}
	},
}

func init() {
	runCmd.PersistentFlags().StringVarP(&c.URL, "url", "u", "", "The target URL to stress test")
	runCmd.PersistentFlags().IntVarP(&c.Requests, "requests", "r", 4, "Total number of requests")
	runCmd.PersistentFlags().StringVarP(&c.Method, "method", "m", "GET", "HTTP method (GET, POST, PUT)")
	runCmd.PersistentFlags().StringVarP(&c.Headers, "headers", "H", "", "Custom headers (Key1:Value1,Key2:Value2)")
	runCmd.PersistentFlags().StringVarP(&c.Body, "body", "b", "", "Request body for POST/PUT methods")
	runCmd.PersistentFlags().IntVarP(&c.Timeout, "timeout", "t", 10, "Request timeout in seconds")
	runCmd.PersistentFlags().IntVar(&c.RPS, "rps", 1, "Requests per second")
	runCmd.PersistentFlags().BoolVarP(&c.Insecure, "insecure", "i", false, "Disable SSL/TLS verification")
	runCmd.PersistentFlags().StringVar(&c.Resolver, "resolver", "", "Custom DNS resolver (IP:Port)")
	runCmd.PersistentFlags().StringVarP(&c.Proxy, "proxy", "P", "", "Proxy server (IP:Port)")
	runCmd.PersistentFlags().BoolVarP(&c.KeepAlive, "keep-alive", "k", false, "Enable HTTP keep-alive")
	runCmd.PersistentFlags().BoolVar(&c.HTTP2, "http2", false, "Enable HTTP/2 support")
	runCmd.PersistentFlags().StringVarP(&c.LogLevel, "log-level", "l", "error", "Logging level (debug, info, warn, error)")
	runCmd.PersistentFlags().BoolVar(&c.Compression, "compression", false, "Enable HTTP compression (gzip)")
	runCmd.PersistentFlags().StringVar(&c.ProxyUser, "proxy-user", "", "Proxy authentication username")
	runCmd.PersistentFlags().StringVar(&c.ProxyPassword, "proxy-password", "", "Proxy authentication password")
	runCmd.PersistentFlags().IntVarP(&c.Sleep, "sleep", "s", 0, "Additional sleep between requests in seconds (default: 0)")
	runCmd.PersistentFlags().BoolVar(&c.SkipDNS, "skip-dns", false, "Skip DNS resolution (requires direct IP)")
	runCmd.PersistentFlags().StringVarP(&c.OutputFormat, "format", "f", "raw", "Output format (json, yaml, raw)")
	runCmd.PersistentFlags().StringVar(&c.OutputFile, "out", "stdout", "Output file (default: stdout)")
	runCmd.PersistentFlags().StringVar(&c.FileName, "filename", "", "Specify a file name when using --out file")
	runCmd.PersistentFlags().BoolVarP(&c.ReuseConnections, "reuse-connections", "R", false, "Multiplex connections, only works with HTTP2")
	runCmd.PersistentFlags().IntVarP(&c.Workers, "workers", "w", 10, "Number of concurrent workers. Default: 10")
	runCmd.PersistentFlags().BoolVar(&c.RandomUserAgent, "random-user-agent", false, "Randomize the User-Agent header per request")
	runCmd.PersistentFlags().BoolVar(&distributed, "distributed", false, "Run test distributed across agents via a coordinator")
	runCmd.PersistentFlags().StringVar(&coordinatorAddr, "coordinator", "http://localhost:9090", "Coordinator address for distributed mode")
	runCmd.PersistentFlags().StringVar(&c.TestFile, "file-name", "", "Name of test definition file")
}

func run(ctx context.Context, c config.Config) error {
	if c.TestFile != "" {
		specs, err := loadDefFileSpecs(c)
		if err != nil {
			return err
		}
		return runRequests(ctx, c, specs, distributed)
	}

	c.Logger.Debug("Validating configuration")
	if err := c.Validate(); err != nil {
		return err
	}
	c.Logger.Debug("Configuration validated successfully")

	if distributed {
		return runRequests(ctx, c, []requestSpec{{
			Name:     "ad-hoc",
			URL:      c.URL,
			Method:   c.Method,
			Body:     c.Body,
			RPS:      c.RPS,
			Requests: c.Requests,
		}}, true)
	}

	// todo: do we need to parse headers HERE? why?
	if c.Headers != "" {
		c.Logger.Debug("Parsing headers: %s", c.Headers)
		parsedHeaders, err := util.ParseHeaders(c.Headers)
		if err != nil {
			return fmt.Errorf("error parsing headers: %w", err)
		}
		c.ParsedHeaders = parsedHeaders
	}

	// worker.Work dispatches one job per rate-limiter tick, so a distinct
	// slice entry is needed per request even though the content is
	// identical; the per-job ID isn't read anywhere downstream.
	c.Logger.Debug("Creating %d jobs for requests to %s", c.Requests, c.URL)
	job := worker.Job{Host: c.URL, Method: c.Method, Body: c.Body}
	jobs := make([]worker.Job, c.Requests)
	for i := range jobs {
		jobs[i] = job
	}

	factory := func(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, cfg config.Config) worker.Worker {
		return *worker.NewWorker(id, jobChan, resultChan, client, cfg)
	}

	reportChan := make(chan report.Report, c.Requests)
	go worker.Work(ctx, c, jobs, reportChan, factory, nil)

	select {
	case <-ctx.Done():
		c.Logger.Debug("Shutdown signal received. Cleaning up.")
		return nil
	case r := <-reportChan:
		return generateReport(c, r)
	}
}

// requestSpec describes a single named request to execute, either the
// top-level ad hoc request or one of several named requests parsed from a
// definition file.
type requestSpec struct {
	Name     string
	URL      string
	Method   string
	Body     string
	RPS      int
	Requests int
	Headers  map[string]string
}

// loadDefFileSpecs validates and parses a test definition file into the
// request specs it describes.
func loadDefFileSpecs(c config.Config) ([]requestSpec, error) {
	c.Logger.Debug("Validating definition file")
	if err := config.ValidateDefFile(c.TestFile); err != nil {
		return nil, fmt.Errorf("invalid definition file: %w", err)
	}

	def, err := config.ParseDefFile(c.TestFile)
	if err != nil {
		return nil, fmt.Errorf("error while parsing def file: %w", err)
	}
	c.Logger.Debug("Parsed %d request(s) from definition file %s", len(def.Requests), c.TestFile)

	specs := make([]requestSpec, 0, len(def.Requests))
	for _, reqDef := range def.Requests {
		specs = append(specs, requestSpec{
			Name:     reqDef.Name,
			URL:      reqDef.URL,
			Method:   reqDef.Method,
			Body:     reqDef.Body,
			RPS:      reqDef.RPS,
			Requests: reqDef.Requests,
			Headers:  reqDef.Headers,
		})
	}
	return specs, nil
}

// runRequests runs each request spec in order -- either locally via the
// worker pool or remotely via a distributed coordinator -- and aggregates
// all results into a single combined report.
func runRequests(ctx context.Context, c config.Config, specs []requestSpec, distributed bool) error {
	factory := func(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, cfg config.Config) worker.Worker {
		return *worker.NewWorker(id, jobChan, resultChan, client, cfg)
	}

	var allResults []report.Result
	hosts := make([]string, 0, len(specs))
	methods := make(map[string]bool)
	rpsValues := make(map[int]bool)

	for _, spec := range specs {
		select {
		case <-ctx.Done():
			c.Logger.Debug("Shutdown signal received. Cleaning up.")
			return nil
		default:
		}

		reqConfig := c
		reqConfig.URL = spec.URL
		reqConfig.Method = spec.Method
		reqConfig.Body = spec.Body
		reqConfig.RPS = spec.RPS
		reqConfig.Requests = spec.Requests
		if spec.Headers != nil {
			reqConfig.Headers = ""
			reqConfig.ParsedHeaders = util.HeaderMapToParsedHeaders(spec.Headers)
		}

		if err := reqConfig.Validate(); err != nil {
			return fmt.Errorf("invalid configuration for request %q: %w", spec.Name, err)
		}

		c.Logger.Debug("Running request %q: %s %s (rps=%d, requests=%d)", spec.Name, reqConfig.Method, reqConfig.URL, reqConfig.RPS, reqConfig.Requests)

		var r *report.Report
		var err error
		if distributed {
			r, err = submitDistributedRequest(ctx, reqConfig)
		} else {
			r, err = runLocalRequest(ctx, reqConfig, factory)
		}
		if err != nil {
			return err
		}
		if r == nil {
			c.Logger.Debug("Shutdown signal received. Cleaning up.")
			return nil
		}

		allResults = append(allResults, r.Results...)
		hosts = append(hosts, reqConfig.URL)
		methods[reqConfig.Method] = true
		rpsValues[reqConfig.RPS] = true
	}

	combined := report.Aggregate(allResults)
	combined.Host = strings.Join(hosts, ", ")
	if len(methods) == 1 {
		for m := range methods {
			combined.Method = m
		}
	} else {
		combined.Method = "MULTI"
	}
	if len(rpsValues) == 1 {
		for rps := range rpsValues {
			combined.TargetRPS = rps
		}
	}

	return generateReport(c, combined)
}

// runLocalRequest runs a single request configuration against the local
// worker pool and returns its report. A nil report and nil error means the
// context was cancelled before the run completed.
func runLocalRequest(ctx context.Context, c config.Config, factory worker.WorkerFactory) (*report.Report, error) {
	jobs := make([]worker.Job, c.Requests)
	for i := 0; i < c.Requests; i++ {
		jobs[i] = worker.Job{ID: i, Host: c.URL, Method: c.Method, Body: c.Body}
	}

	reportChan := make(chan report.Report, 1)
	go worker.Work(ctx, c, jobs, reportChan, factory, nil)

	select {
	case <-ctx.Done():
		return nil, nil
	case r := <-reportChan:
		return &r, nil
	}
}

// submitDistributedRequest submits a single request configuration to the
// coordinator and polls until it completes, returning the resulting report.
// A nil report and nil error means the context was cancelled before the run
// completed.
func submitDistributedRequest(ctx context.Context, c config.Config) (*report.Report, error) {
	tc := api.TestConfig{
		URL:              c.URL,
		Method:           c.Method,
		Headers:          c.Headers,
		Body:             c.Body,
		Requests:         c.Requests,
		RPS:              c.RPS,
		Workers:          c.Workers,
		Timeout:          c.Timeout,
		Insecure:         c.Insecure,
		KeepAlive:        c.KeepAlive,
		HTTP2:            c.HTTP2,
		Compression:      c.Compression,
		ReuseConnections: c.ReuseConnections,
		RandomUserAgent:  c.RandomUserAgent,
	}

	body, err := json.Marshal(tc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test config: %w", err)
	}

	c.Logger.Info("Submitting distributed test to coordinator at %s", coordinatorAddr)

	resp, err := http.Post("http://"+coordinatorAddr+"/api/v1/tests", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("could not reach coordinator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("coordinator rejected test: %s", errResp.Error)
	}

	var submission api.TestSubmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&submission); err != nil {
		return nil, fmt.Errorf("invalid response from coordinator: %w", err)
	}

	c.Logger.Info("Test submitted (id: %s). Waiting for agents to complete...", submission.TestID)

	// Poll for completion
	pollInterval := 2 * time.Second
	for {
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(pollInterval):
		}

		status, err := pollTestStatus(submission.TestID)
		if err != nil {
			c.Logger.Error("Error polling test status: %v", err)
			continue
		}

		switch status.Status {
		case "complete":
			c.Logger.Info("Distributed test complete (%d agents)", status.AgentCount)
			if status.Report != nil {
				return status.Report, nil
			}
			return nil, fmt.Errorf("test completed but no report available")

		case "error":
			return nil, fmt.Errorf("test failed: %s", status.Error)

		default:
			c.Logger.Info("Waiting... %d/%d agents reported",
				status.ReportsReceived, status.AgentCount)
		}
	}
}

// pollTestStatus fetches the current status of a distributed test.
func pollTestStatus(testID string) (*api.TestStatusResponse, error) {
	resp, err := http.Get("http://" + coordinatorAddr + "/api/v1/tests/" + testID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var status api.TestStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func generateReport(c config.Config, r report.Report) error {
	c.Logger.Debug("Generating report in %s format", c.OutputFormat)

	var reportOutput string
	var err error

	switch c.OutputFormat {
	case "json":
		reportOutput, err = report.ParseJSON(r)
	case "yaml":
		reportOutput, err = report.ParseYAML(r)
	default:
		reportOutput, err = report.ParseRaw(r)
	}

	if err != nil {
		return fmt.Errorf("error generating report: %w", err)
	}

	c.Logger.Debug("Report generated successfully")
	fmt.Fprintln(c.Logger.Writer(), reportOutput)
	return nil
}
