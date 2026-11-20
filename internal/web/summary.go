package web

import (
	"sort"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
)

// Summary is the compact, numeric form of a finished run that is stored and
// compared. It deliberately omits the per-request results of report.Report.
type Summary struct {
	Host          string      `json:"host"`
	Method        string      `json:"method"`
	TotalRequests int         `json:"total_requests"`
	Successes     int         `json:"successes"`
	Failures      int         `json:"failures"`
	DurationMs    int64       `json:"duration_ms"`
	RPS           float64     `json:"rps"`
	MinMs         float64     `json:"min_ms"`
	AvgMs         float64     `json:"avg_ms"`
	P50Ms         float64     `json:"p50_ms"`
	P95Ms         float64     `json:"p95_ms"`
	P99Ms         float64     `json:"p99_ms"`
	MaxMs         float64     `json:"max_ms"`
	StatusCodes   map[int]int `json:"status_codes"`
	BytesSent     int         `json:"bytes_sent"`
	BytesReceived int         `json:"bytes_received"`
}

func Summarize(r report.Report) Summary {
	s := Summary{
		Host:          r.Host,
		Method:        r.Method,
		TotalRequests: r.TotalRequests,
		Successes:     r.Successes,
		Failures:      r.Failures,
		DurationMs:    r.Duration.Milliseconds(),
		StatusCodes:   r.StatusCodes,
		BytesSent:     r.Throughput.TotalBytesSent,
		BytesReceived: r.Throughput.TotalBytesReceived,
	}
	if r.Duration > 0 {
		s.RPS = float64(r.TotalRequests) / r.Duration.Seconds()
	}
	if len(r.Results) == 0 {
		return s
	}
	lat := make([]time.Duration, len(r.Results))
	var total time.Duration
	for i, res := range r.Results {
		lat[i] = res.ElapsedTime
		total += res.ElapsedTime
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	s.MinMs = ms(lat[0])
	s.MaxMs = ms(lat[len(lat)-1])
	s.AvgMs = ms(total / time.Duration(len(lat)))
	s.P50Ms = ms(percentile(lat, 50))
	s.P95Ms = ms(percentile(lat, 95))
	s.P99Ms = ms(percentile(lat, 99))
	return s
}
