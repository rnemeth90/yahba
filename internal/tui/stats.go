package tui

import (
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
)

// recentLatencies bounds the latency history kept for the graph, so a long or
// high-rate run does not grow memory without limit.
const recentLatencies = 10000

// liveStats aggregates results off the UI goroutine. At closed-loop rates a
// Bubble Tea message per request would throttle the test itself, so workers
// feed counters here and the UI samples a snapshot on each tick.
type liveStats struct {
	mu            sync.Mutex
	completed     int
	successes     int
	failures      int
	bytesSent     int
	bytesReceived int
	statusCodes   map[int]int
	minLatency    time.Duration
	maxLatency    time.Duration
	totalLatency  time.Duration
	ring          []time.Duration
	next          int
}

type statsSnapshot struct {
	completed, successes, failures int
	bytesSent, bytesReceived       int
	statusCodes                    map[int]int
	minLatency, maxLatency         time.Duration
	totalLatency                   time.Duration
	latencies                      []time.Duration // oldest first
}

func newLiveStats() *liveStats {
	return &liveStats{statusCodes: make(map[int]int)}
}

func (s *liveStats) add(r report.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.completed++
	if r.ResultCode > 0 && r.ResultCode < 400 {
		s.successes++
	} else {
		s.failures++
	}
	// a network error has no status code; do not report it as code 0
	if r.ResultCode > 0 {
		s.statusCodes[r.ResultCode]++
	}
	s.bytesSent += r.BytesSent
	s.bytesReceived += r.BytesReceived
	s.totalLatency += r.ElapsedTime
	if s.minLatency == 0 || r.ElapsedTime < s.minLatency {
		s.minLatency = r.ElapsedTime
	}
	if r.ElapsedTime > s.maxLatency {
		s.maxLatency = r.ElapsedTime
	}

	if len(s.ring) < recentLatencies {
		s.ring = append(s.ring, r.ElapsedTime)
	} else {
		s.ring[s.next] = r.ElapsedTime
		s.next = (s.next + 1) % recentLatencies
	}
}

func (s *liveStats) snapshot() statsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	codes := make(map[int]int, len(s.statusCodes))
	for k, v := range s.statusCodes {
		codes[k] = v
	}
	latencies := make([]time.Duration, 0, len(s.ring))
	latencies = append(latencies, s.ring[s.next:]...)
	latencies = append(latencies, s.ring[:s.next]...)

	return statsSnapshot{
		completed:     s.completed,
		successes:     s.successes,
		failures:      s.failures,
		bytesSent:     s.bytesSent,
		bytesReceived: s.bytesReceived,
		statusCodes:   codes,
		minLatency:    s.minLatency,
		maxLatency:    s.maxLatency,
		totalLatency:  s.totalLatency,
		latencies:     latencies,
	}
}

// rampLive collects finished ramp steps as the worker reports them.
type rampLive struct {
	mu    sync.Mutex
	steps []report.Step
}

func (r *rampLive) add(s report.Step) {
	r.mu.Lock()
	r.steps = append(r.steps, s)
	r.mu.Unlock()
}

func (r *rampLive) snapshot() []report.Step {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]report.Step(nil), r.steps...)
}
