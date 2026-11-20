package web

import (
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
	"github.com/stretchr/testify/assert"
)

func TestSummarize(t *testing.T) {
	r := report.Report{
		Host:          "http://x",
		Method:        "GET",
		TotalRequests: 4,
		Successes:     3,
		Failures:      1,
		Duration:      2 * time.Second,
		StatusCodes:   map[int]int{200: 3, 500: 1},
		Results: []report.Result{
			{ElapsedTime: 10 * time.Millisecond},
			{ElapsedTime: 20 * time.Millisecond},
			{ElapsedTime: 30 * time.Millisecond},
			{ElapsedTime: 40 * time.Millisecond},
		},
		Throughput: report.Throughput{TotalBytesSent: 100, TotalBytesReceived: 400},
	}

	s := Summarize(r)

	assert.Equal(t, "http://x", s.Host)
	assert.Equal(t, 4, s.TotalRequests)
	assert.Equal(t, int64(2000), s.DurationMs)
	assert.InDelta(t, 2.0, s.RPS, 0.001)
	assert.InDelta(t, 10.0, s.MinMs, 0.001)
	assert.InDelta(t, 25.0, s.AvgMs, 0.001)
	assert.InDelta(t, 20.0, s.P50Ms, 0.001)
	assert.InDelta(t, 40.0, s.P95Ms, 0.001)
	assert.InDelta(t, 40.0, s.MaxMs, 0.001)
	assert.Equal(t, 100, s.BytesSent)
	assert.Equal(t, 400, s.BytesReceived)
}

func TestSummarizeEmptyReport(t *testing.T) {
	s := Summarize(report.Report{})
	assert.Equal(t, 0, s.TotalRequests)
	assert.Equal(t, 0.0, s.RPS)
	assert.Equal(t, 0.0, s.P95Ms)
}
