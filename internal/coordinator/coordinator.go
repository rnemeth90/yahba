// Package coordinator implements the server side of distributed load testing.
// It manages agent registration, distributes work, and aggregates results.
package coordinator

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/api"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
)

// agentState tracks a connected agent.
type agentState struct {
	Info         api.AgentRegistration
	Status       string // idle, running
	RegisteredAt time.Time
}

// testState tracks a distributed test.
type testState struct {
	ID               string
	Config           api.TestConfig
	Status           string // pending, running, complete, error
	Agents           []string
	Reports          map[string]*report.Report
	AggregatedReport *report.Report
	CreatedAt        time.Time
	Error            string
}

// Coordinator manages agents and distributed tests.
type Coordinator struct {
	mu     sync.RWMutex
	agents map[string]*agentState
	tests  map[string]*testState
	work   map[string]*api.WorkAssignment // agentID -> pending assignment
	logger *logger.Logger
	port   string
}

// New creates a new Coordinator.
func New(port string, log *logger.Logger) *Coordinator {
	return &Coordinator{
		agents: make(map[string]*agentState),
		tests:  make(map[string]*testState),
		work:   make(map[string]*api.WorkAssignment),
		logger: log,
		port:   port,
	}
}

// Run starts the coordinator HTTP server.
func (c *Coordinator) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/register", c.handleRegister)
	mux.HandleFunc("GET /api/v1/agents", c.handleListAgents)
	mux.HandleFunc("POST /api/v1/tests", c.handleSubmitTest)
	mux.HandleFunc("GET /api/v1/tests/{id}", c.handleGetTest)
	mux.HandleFunc("GET /api/v1/work/{agentID}", c.handleGetWork)
	mux.HandleFunc("POST /api/v1/results/{agentID}", c.handlePostResults)

	c.logger.Info("Coordinator listening on %s", c.port)
	return http.ListenAndServe(c.port, mux)
}

// --- Handlers ---

// handleRegister registers a new agent.
func (c *Coordinator) handleRegister(w http.ResponseWriter, r *http.Request) {
	var reg api.AgentRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body"})
		return
	}

	reg.ID = generateID()

	c.mu.Lock()
	c.agents[reg.ID] = &agentState{
		Info:         reg,
		Status:       "idle",
		RegisteredAt: time.Now(),
	}
	c.mu.Unlock()

	c.logger.Info("Agent registered: %s (id: %s)", reg.Name, reg.ID)
	writeJSON(w, http.StatusOK, reg)
}

// handleListAgents returns all registered agents.
func (c *Coordinator) handleListAgents(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agents := make([]api.AgentInfo, 0, len(c.agents))
	for _, a := range c.agents {
		agents = append(agents, api.AgentInfo{
			ID:     a.Info.ID,
			Name:   a.Info.Name,
			Status: a.Status,
		})
	}

	writeJSON(w, http.StatusOK, agents)
}

// handleSubmitTest accepts a new test, splits work across idle agents.
func (c *Coordinator) handleSubmitTest(w http.ResponseWriter, r *http.Request) {
	var tc api.TestConfig
	if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body"})
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Find idle agents
	var idleAgents []string
	for id, a := range c.agents {
		if a.Status == "idle" {
			idleAgents = append(idleAgents, id)
		}
	}

	if len(idleAgents) == 0 {
		writeJSON(w, http.StatusConflict, api.ErrorResponse{Error: "no idle agents available"})
		return
	}

	testID := generateID()
	agentCount := len(idleAgents)
	requestsPerAgent := tc.Requests / agentCount
	rpsPerAgent := tc.RPS / agentCount
	remainder := tc.Requests % agentCount

	// Ensure at least 1 RPS per agent
	if rpsPerAgent < 1 {
		rpsPerAgent = 1
	}

	c.logger.Info("Test %s: distributing %d requests across %d agents (%d each, %d remainder)",
		testID, tc.Requests, agentCount, requestsPerAgent, remainder)

	ts := &testState{
		ID:        testID,
		Config:    tc,
		Status:    "running",
		Agents:    idleAgents,
		Reports:   make(map[string]*report.Report),
		CreatedAt: time.Now(),
	}
	c.tests[testID] = ts

	// Create work assignments for each agent
	for i, agentID := range idleAgents {
		agentConfig := tc
		agentConfig.Requests = requestsPerAgent
		agentConfig.RPS = rpsPerAgent

		// First agent picks up the remainder
		if i == 0 {
			agentConfig.Requests += remainder
		}

		c.work[agentID] = &api.WorkAssignment{
			TestID: testID,
			Config: agentConfig,
		}
		c.agents[agentID].Status = "running"
	}

	writeJSON(w, http.StatusOK, api.TestSubmissionResponse{TestID: testID})
}

// handleGetTest returns the status of a test, including the aggregated report if complete.
func (c *Coordinator) handleGetTest(w http.ResponseWriter, r *http.Request) {
	testID := r.PathValue("id")

	c.mu.RLock()
	defer c.mu.RUnlock()

	ts, ok := c.tests[testID]
	if !ok {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "test not found"})
		return
	}

	resp := api.TestStatusResponse{
		ID:              ts.ID,
		Status:          ts.Status,
		AgentCount:      len(ts.Agents),
		ReportsReceived: len(ts.Reports),
		Report:          ts.AggregatedReport,
		Error:           ts.Error,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetWork returns a pending work assignment for the agent, or 204 if none.
func (c *Coordinator) handleGetWork(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.agents[agentID]; !ok {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "agent not found"})
		return
	}

	assignment, ok := c.work[agentID]
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	delete(c.work, agentID)
	writeJSON(w, http.StatusOK, assignment)
}

// handlePostResults receives an agent's report after it finishes its work.
func (c *Coordinator) handlePostResults(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	testID := r.URL.Query().Get("test_id")

	if testID == "" {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "missing test_id query parameter"})
		return
	}

	var rpt report.Report
	if err := json.NewDecoder(r.Body).Decode(&rpt); err != nil {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: fmt.Sprintf("invalid report: %v", err)})
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	agent, ok := c.agents[agentID]
	if !ok {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "agent not found"})
		return
	}
	agent.Status = "idle"

	ts, ok := c.tests[testID]
	if !ok {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "test not found"})
		return
	}

	ts.Reports[agentID] = &rpt
	c.logger.Info("Test %s: received report from agent %s (%d/%d)",
		testID, agentID, len(ts.Reports), len(ts.Agents))

	// If all agents have reported, aggregate
	if len(ts.Reports) == len(ts.Agents) {
		merged := mergeReports(ts.Reports, ts.Config)
		ts.AggregatedReport = &merged
		ts.Status = "complete"
		c.logger.Info("Test %s: all agents reported, test complete", testID)
	}

	w.WriteHeader(http.StatusOK)
}

// mergeReports combines reports from multiple agents into a single aggregated report.
func mergeReports(reports map[string]*report.Report, cfg api.TestConfig) report.Report {
	merged := report.Report{
		Host:        cfg.URL,
		Method:      cfg.Method,
		StatusCodes: make(map[int]int),
	}

	var earliest, latest time.Time

	for _, r := range reports {
		merged.TotalRequests += r.TotalRequests
		merged.Successes += r.Successes
		merged.Failures += r.Failures
		merged.ErrorBreakdown.ServerErrors += r.ErrorBreakdown.ServerErrors
		merged.ErrorBreakdown.ClientErrors += r.ErrorBreakdown.ClientErrors
		merged.Throughput.TotalBytesSent += r.Throughput.TotalBytesSent
		merged.Throughput.TotalBytesReceived += r.Throughput.TotalBytesReceived
		merged.Results = append(merged.Results, r.Results...)

		for code, count := range r.StatusCodes {
			merged.StatusCodes[code] += count
		}

		if start, err := time.Parse(time.RFC3339, r.StartTime); err == nil {
			if earliest.IsZero() || start.Before(earliest) {
				earliest = start
			}
		}
		if end, err := time.Parse(time.RFC3339, r.EndTime); err == nil {
			if latest.IsZero() || end.After(latest) {
				latest = end
			}
		}
	}

	if !earliest.IsZero() && !latest.IsZero() {
		merged.StartTime = earliest.Format(time.RFC3339)
		merged.EndTime = latest.Format(time.RFC3339)
		merged.Duration = latest.Sub(earliest)
	}

	if merged.Duration > 0 {
		seconds := merged.Duration.Seconds()
		merged.Throughput.BytesSentPerSecond = float64(merged.Throughput.TotalBytesSent) / seconds
		merged.Throughput.BytesReceivedPerSecond = float64(merged.Throughput.TotalBytesReceived) / seconds
	}

	merged.CalculateLatencyMetrics()

	return merged
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
