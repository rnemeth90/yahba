package worker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFactory(id int, jobs <-chan Job, results chan<- report.Result, c *http.Client, cfg config.Config) Worker {
	return *NewWorker(id, jobs, results, c, cfg)
}

func baseConfig(url string) config.Config {
	return config.Config{
		URL:     url,
		Method:  "GET",
		Timeout: 5,
		Workers: 20,
		Logger:  logger.New("error", "stdout", true),
	}
}

func TestWorkPacingAndWallTimeThroughput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("0123456789"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.RPS = 100
	cfg.Requests = 100

	jobs := make([]Job, cfg.Requests)
	for i := range jobs {
		jobs[i] = Job{ID: i, Host: cfg.URL, Method: "GET"}
	}
	ch := make(chan report.Report, 1)
	Work(context.Background(), cfg, jobs, ch, testFactory, nil)
	rep := <-ch

	assert.Equal(t, 100, rep.TotalRequests)
	assert.Equal(t, 100, rep.Successes)
	assert.Equal(t, 1000, rep.Throughput.TotalBytesReceived)
	assert.InDelta(t, 100, rep.AchievedRPS, 15)
	assert.Equal(t, 100, rep.ConfiguredRPS)

	// wall time, not the sum of overlapping per-request durations
	expected := float64(rep.Throughput.TotalBytesReceived) / rep.Duration.Seconds()
	assert.InDelta(t, expected, rep.Throughput.BytesReceivedPerSecond, 0.001)
	assert.Greater(t, rep.Duration, 900*time.Millisecond)
}

func TestEmptyResponseIsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.RPS = 1000
	jobs := []Job{{Host: srv.URL, Method: "GET"}, {Host: srv.URL, Method: "GET"}}
	ch := make(chan report.Report, 1)
	Work(context.Background(), cfg, jobs, ch, testFactory, nil)
	rep := <-ch

	assert.Equal(t, 2, rep.Successes)
	assert.Equal(t, 0, rep.Failures)
	assert.Equal(t, 2, rep.StatusCodes[200])
}

func TestLatencyIncludesQueueing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("x"))
	}))
	defer srv.Close()

	// one worker at 50 RPS against a 100ms server: the queue grows, and the
	// open-model latency must reflect that rather than a flat ~100ms
	cfg := baseConfig(srv.URL)
	cfg.RPS = 50
	cfg.Workers = 1
	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{ID: i, Host: srv.URL, Method: "GET"}
	}
	ch := make(chan report.Report, 1)
	Work(context.Background(), cfg, jobs, ch, testFactory, nil)
	rep := <-ch

	assert.Greater(t, report.LatencyPercentile(rep.Results, 99), 300*time.Millisecond)
}

func TestRunMaxClosedLoop(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.Duration = 500 * time.Millisecond
	cfg.Workers = 10
	cfg.KeepAlive = true

	ch := make(chan report.Report, 1)
	start := time.Now()
	RunMax(context.Background(), cfg, ch, testFactory, nil)
	rep := <-ch

	assert.Less(t, time.Since(start), 3*time.Second)
	assert.Equal(t, config.ModeMax, rep.Mode)
	assert.Equal(t, int(hits.Load()), rep.TotalRequests)
	assert.Equal(t, 0, rep.Failures)
	// far above what any paced default (1 RPS) would deliver
	assert.Greater(t, rep.TotalRequests, 100)
}

func TestRunMaxStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.Duration = time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch := make(chan report.Report, 1)
	done := make(chan struct{})
	go func() { RunMax(ctx, cfg, ch, testFactory, nil); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunMax did not stop after context cancellation")
	}
}

func TestRunRampFindsBreakingPoint(t *testing.T) {
	// the server fails every request once the offered rate exceeds its capacity
	var window atomic.Int64
	var windowStart atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UnixNano()
		if now-windowStart.Load() > int64(time.Second) {
			windowStart.Store(now)
			window.Store(0)
		}
		if window.Add(1) > 50 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.RampStart = 20
	cfg.RampStep = 40
	cfg.StepDuration = 2 * time.Second
	cfg.MaxErrorRate = 5

	rr, err := RunRamp(context.Background(), cfg, testFactory)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rr.Steps), 2)

	assert.True(t, rr.Steps[0].Passed)
	assert.Equal(t, 20, rr.SustainedRPS)
	assert.Equal(t, 60, rr.BreakingRPS)
	assert.Contains(t, rr.StopReason, "error rate")
}

func TestRunRampStopsAtMax(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.RampStart = 10
	cfg.RampStep = 10
	cfg.RampMax = 20
	cfg.StepDuration = time.Second
	cfg.MaxErrorRate = 5

	rr, err := RunRamp(context.Background(), cfg, testFactory)
	require.NoError(t, err)
	assert.Len(t, rr.Steps, 2)
	assert.Equal(t, 20, rr.SustainedRPS)
	assert.Zero(t, rr.BreakingRPS)
	assert.Contains(t, rr.StopReason, "--rps-max")
}

func TestEvaluateStepP99AndShortfall(t *testing.T) {
	cfg := baseConfig("http://x")
	cfg.StepDuration = time.Second
	cfg.MaxErrorRate = 5
	cfg.MaxP99 = 100 * time.Millisecond

	slow := report.Report{TotalRequests: 100, Results: make([]report.Result, 100)}
	for i := range slow.Results {
		slow.Results[i].ElapsedTime = 200 * time.Millisecond
	}
	assert.Contains(t, evaluateStep(cfg, 100, slow).Reason, "p99")

	cfg.MaxP99 = 0
	assert.Contains(t, evaluateStep(cfg, 1000, slow).Reason, "achieved")
}

func TestRequestTemplateReuse(t *testing.T) {
	var bodies, agents, custom sync.Map
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies.Store(string(b), true)
		agents.Store(r.Header.Get("User-Agent"), true)
		if r.Header.Get("X-Test") == "1" {
			custom.Store("seen", true)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := baseConfig(srv.URL)
	cfg.Method = "POST"
	cfg.Body = `{"a":1}`
	cfg.ParsedHeaders = []util.Header{{Key: "X-Test", Value: "1"}}
	cfg.RandomUserAgent = true

	jobs := make(chan Job, 50)
	results := make(chan report.Result, 50)
	w := NewWorker(0, jobs, results, srv.Client(), cfg)
	for i := 0; i < 50; i++ {
		w.processJob(Job{ID: i, Host: srv.URL, Method: "POST", Body: cfg.Body})
	}

	// a different job rebuilds the template
	results2 := make(chan report.Result, 1)
	w.Results = results2
	w.processJob(Job{Host: srv.URL + "/other", Method: "POST", Body: cfg.Body})
	assert.Equal(t, srv.URL+"/other", w.tmpl.host)
	close(results)

	n := 0
	for r := range results {
		n++
		assert.Equal(t, 200, r.ResultCode)
		assert.Greater(t, r.BytesSent, 0)
	}
	assert.Equal(t, 50, n)

	_, ok := bodies.Load(`{"a":1}`)
	assert.True(t, ok)
	count := 0
	bodies.Range(func(_, _ any) bool { count++; return true })
	assert.Equal(t, 1, count, "every request must carry the full body")
	_, ok = custom.Load("seen")
	assert.True(t, ok)
	ua := 0
	agents.Range(func(_, _ any) bool { ua++; return true })
	assert.Greater(t, ua, 1, "random user agent must vary per request")
}
