// Package agent implements the client side of distributed load testing.
// An agent registers with a coordinator, polls for work, runs tests
// using the existing worker pool, and reports results back.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rnemeth90/yahba/internal/api"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
)

// Agent connects to a coordinator, receives work, and executes load tests.
type Agent struct {
	id             string
	name           string
	coordinatorURL string
	logger         *logger.Logger
	pollInterval   time.Duration
}

// New creates a new Agent.
func New(name, coordinatorURL string, log *logger.Logger) *Agent {
	return &Agent{
		name:           name,
		coordinatorURL: coordinatorURL,
		logger:         log,
		pollInterval:   2 * time.Second,
	}
}

// Run registers with the coordinator and enters the poll-execute loop.
// It blocks until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	if err := a.register(); err != nil {
		return fmt.Errorf("failed to register with coordinator: %w", err)
	}
	a.logger.Info("Registered with coordinator as %q (id: %s)", a.name, a.id)
	a.logger.Info("Polling for work every %s...", a.pollInterval)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Agent shutting down")
			return nil
		default:
		}

		assignment, err := a.pollForWork()
		if err != nil {
			a.logger.Error("Error polling for work: %v", err)
			time.Sleep(a.pollInterval)
			continue
		}

		if assignment == nil {
			time.Sleep(a.pollInterval)
			continue
		}

		a.logger.Info("Received work: test %s — %d requests @ %d RPS to %s",
			assignment.TestID,
			assignment.Config.Requests,
			assignment.Config.RPS,
			assignment.Config.URL,
		)

		rpt, err := a.executeTest(ctx, assignment)
		if err != nil {
			a.logger.Error("Test execution failed: %v", err)
			continue
		}

		if err := a.reportResults(assignment.TestID, rpt); err != nil {
			a.logger.Error("Failed to report results: %v", err)
			continue
		}

		a.logger.Info("Test %s complete, results reported to coordinator", assignment.TestID)
	}
}

// register sends a registration request to the coordinator.
func (a *Agent) register() error {
	reg := api.AgentRegistration{Name: a.name}
	body, err := json.Marshal(reg)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		a.coordinatorURL+"/api/v1/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("could not reach coordinator at %s: %w", a.coordinatorURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: HTTP %d", resp.StatusCode)
	}

	var result api.AgentRegistration
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	a.id = result.ID
	return nil
}

// pollForWork asks the coordinator for a work assignment.
// Returns nil, nil when no work is available.
func (a *Agent) pollForWork() (*api.WorkAssignment, error) {
	resp, err := http.Get(a.coordinatorURL + "/api/v1/work/" + a.id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d polling for work", resp.StatusCode)
	}

	var assignment api.WorkAssignment
	if err := json.NewDecoder(resp.Body).Decode(&assignment); err != nil {
		return nil, err
	}

	return &assignment, nil
}

// executeTest runs the load test locally using the existing worker pool.
func (a *Agent) executeTest(ctx context.Context, assignment *api.WorkAssignment) (*report.Report, error) {
	tc := assignment.Config

	cfg := config.Config{
		URL:              tc.URL,
		Method:           tc.Method,
		Headers:          tc.Headers,
		Body:             tc.Body,
		Requests:         tc.Requests,
		RPS:              tc.RPS,
		Workers:          tc.Workers,
		Timeout:          tc.Timeout,
		Insecure:         tc.Insecure,
		KeepAlive:        tc.KeepAlive,
		HTTP2:            tc.HTTP2,
		Compression:      tc.Compression,
		ReuseConnections: tc.ReuseConnections,
		RandomUserAgent:  tc.RandomUserAgent,
		LogLevel:         "info",
		OutputFormat:     "raw",
		OutputFile:       "stdout",
		Logger:           a.logger,
	}

	// Parse headers if provided
	if cfg.Headers != "" {
		parsedHeaders, err := util.ParseHeaders(cfg.Headers)
		if err != nil {
			return nil, fmt.Errorf("error parsing headers: %w", err)
		}
		cfg.ParsedHeaders = parsedHeaders
	}

	// Build jobs
	jobs := make([]worker.Job, cfg.Requests)
	for i := 0; i < cfg.Requests; i++ {
		jobs[i] = worker.Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body}
	}

	factory := func(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, c config.Config) worker.Worker {
		return *worker.NewWorker(id, jobChan, resultChan, client, c)
	}

	reportChan := make(chan report.Report, 1)
	go worker.Work(ctx, cfg, jobs, reportChan, factory, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-reportChan:
		return &r, nil
	}
}

// reportResults sends the completed report back to the coordinator.
func (a *Agent) reportResults(testID string, rpt *report.Report) error {
	body, err := json.Marshal(rpt)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/results/%s?test_id=%s", a.coordinatorURL, a.id, testID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("coordinator rejected results: HTTP %d", resp.StatusCode)
	}

	return nil
}
