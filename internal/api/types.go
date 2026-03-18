// Package api defines the REST types shared between the coordinator and agents.
package api

import "github.com/rnemeth90/yahba/internal/report"

// TestConfig is the load test configuration exchanged over the wire.
// The coordinator receives the full config from the user, then sends
// a modified copy to each agent with requests/RPS split evenly.
type TestConfig struct {
	URL              string `json:"url"`
	Method           string `json:"method"`
	Headers          string `json:"headers"`
	Body             string `json:"body"`
	Requests         int    `json:"requests"`
	RPS              int    `json:"rps"`
	Workers          int    `json:"workers"`
	Timeout          int    `json:"timeout"`
	Insecure         bool   `json:"insecure"`
	KeepAlive        bool   `json:"keep_alive"`
	HTTP2            bool   `json:"http2"`
	Compression      bool   `json:"compression"`
	ReuseConnections bool   `json:"reuse_connections"`
	RandomUserAgent  bool   `json:"random_user_agent"`
}

// AgentRegistration is the request body for agent registration
// and the response (with ID populated by the coordinator).
type AgentRegistration struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AgentInfo describes a connected agent and its current status.
type AgentInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"` // idle, running
}

// WorkAssignment is returned when an agent polls for work.
type WorkAssignment struct {
	TestID string     `json:"test_id"`
	Config TestConfig `json:"config"`
}

// TestSubmissionResponse is returned when a test is submitted.
type TestSubmissionResponse struct {
	TestID string `json:"test_id"`
}

// TestStatusResponse describes the state of a distributed test.
type TestStatusResponse struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"` // pending, running, complete, error
	AgentCount      int            `json:"agent_count"`
	ReportsReceived int            `json:"reports_received"`
	Report          *report.Report `json:"report,omitempty"`
	Error           string         `json:"error,omitempty"`
}

// ErrorResponse is a generic error body.
type ErrorResponse struct {
	Error string `json:"error"`
}
