package web

import (
	"math"
	"sort"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
)

// Sample is one second of aggregated results.
type Sample struct {
	T           int         `json:"t"`
	Count       int         `json:"count"`
	Errors      int         `json:"errors"`
	AvgMs       float64     `json:"avg_ms"`
	P50Ms       float64     `json:"p50_ms"`
	P95Ms       float64     `json:"p95_ms"`
	P99Ms       float64     `json:"p99_ms"`
	StatusCodes map[int]int `json:"status_codes"`
}

// Sampler groups results into one-second buckets keyed by completion time.
type Sampler struct {
	start time.Time
	cur   int
	lat   []time.Duration
	errs  int
	codes map[int]int
}

func NewSampler(start time.Time) *Sampler {
	return &Sampler{start: start, codes: map[int]int{}}
}

// isFailure matches report.Report.Failures: no response, or status >= 400.
func isFailure(code int) bool { return code == 0 || code >= 400 }

// Add records r and returns the previous second's Sample when r begins a later
// second. Results arriving for an earlier second are folded into the current one.
func (s *Sampler) Add(r report.Result) []Sample {
	sec := int(r.EndTime.Sub(s.start) / time.Second)
	var out []Sample
	if sec > s.cur {
		out = s.Flush()
		s.cur = sec
	}
	s.lat = append(s.lat, r.ElapsedTime)
	s.codes[r.ResultCode]++
	if isFailure(r.ResultCode) {
		s.errs++
	}
	return out
}

// Flush returns the in-progress second as a Sample and resets it; nil if empty.
func (s *Sampler) Flush() []Sample {
	if len(s.lat) == 0 {
		return nil
	}
	sorted := append([]time.Duration(nil), s.lat...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var total time.Duration
	for _, d := range sorted {
		total += d
	}
	smp := Sample{
		T:           s.cur,
		Count:       len(sorted),
		Errors:      s.errs,
		AvgMs:       ms(total / time.Duration(len(sorted))),
		P50Ms:       ms(percentile(sorted, 50)),
		P95Ms:       ms(percentile(sorted, 95)),
		P99Ms:       ms(percentile(sorted, 99)),
		StatusCodes: s.codes,
	}
	s.lat = s.lat[:0]
	s.errs = 0
	s.codes = map[int]int{}
	return []Sample{smp}
}

// percentile uses the nearest-rank method on an ascending slice.
func percentile(sorted []time.Duration, p float64) time.Duration {
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
