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
	distributed      bool
	coordinatorAddr  string
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
}

func run(ctx context.Context, c config.Config) error {
	c.Logger.Debug("Validating configuration")
	if err := c.Validate(); err != nil {
		return err
	}
	c.Logger.Debug("Configuration validated successfully")

	if distributed {
		return runDistributed(ctx, c)
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

	// todo: do we need to create individual jobs if the jobs are all the same?
	c.Logger.Debug("Creating %d jobs for requests to %s", c.Requests, c.URL)
	jobs := make([]worker.Job, c.Requests)
	for i := 0; i < c.Requests; i++ {
		jobs[i] = worker.Job{ID: i, Host: c.URL, Method: c.Method, Body: c.Body}
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

// runDistributed submits the test to a coordinator and polls until complete.
func runDistributed(ctx context.Context, c config.Config) error {
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
		return fmt.Errorf("failed to marshal test config: %w", err)
	}

	c.Logger.Info("Submitting distributed test to coordinator at %s", coordinatorAddr)

	resp, err := http.Post(coordinatorAddr+"/api/v1/tests", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not reach coordinator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("coordinator rejected test: %s", errResp.Error)
	}

	var submission api.TestSubmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&submission); err != nil {
		return fmt.Errorf("invalid response from coordinator: %w", err)
	}

	c.Logger.Info("Test submitted (id: %s). Waiting for agents to complete...", submission.TestID)

	// Poll for completion
	pollInterval := 2 * time.Second
	for {
		select {
		case <-ctx.Done():
			return nil
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
				return generateReport(c, *status.Report)
			}
			return fmt.Errorf("test completed but no report available")

		case "error":
			return fmt.Errorf("test failed: %s", status.Error)

		default:
			c.Logger.Info("Waiting... %d/%d agents reported",
				status.ReportsReceived, status.AgentCount)
		}
	}
}

// pollTestStatus fetches the current status of a distributed test.
func pollTestStatus(testID string) (*api.TestStatusResponse, error) {
	resp, err := http.Get(coordinatorAddr + "/api/v1/tests/" + testID)
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
