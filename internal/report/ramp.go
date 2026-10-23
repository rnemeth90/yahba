package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Step summarises one rate step of a ramp test.
type Step struct {
	TargetRPS   int      `json:"target_rps"`
	AchievedRPS float64  `json:"achieved_rps"`
	Requests    int      `json:"requests"`
	Failures    int      `json:"failures"`
	ErrorRate   float64  `json:"error_rate_percent"`
	P50         string   `json:"p50"`
	P95         string   `json:"p95"`
	P99         string   `json:"p99"`
	P999        string   `json:"p99_9"`
	Max         string   `json:"max"`
	Histogram   []Bucket `json:"latency_histogram"`
	Passed      bool     `json:"passed"`
	Reason      string   `json:"reason,omitempty"`
	Knee        bool     `json:"knee,omitempty"`

	P99Duration time.Duration `json:"-" yaml:"-"`
}

// Bucket counts requests with latency <= LE (cumulative buckets are not used;
// each request lands in exactly one bucket). The last bucket has LE "+Inf".
type Bucket struct {
	LE    string `json:"le"`
	Count int    `json:"count"`
}

var bucketBounds = []time.Duration{
	time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond,
	25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond,
	500 * time.Millisecond, time.Second, 5 * time.Second,
}

// LatencyHistogram buckets the results' elapsed times.
func LatencyHistogram(results []Result) []Bucket {
	buckets := make([]Bucket, len(bucketBounds)+1)
	for i, b := range bucketBounds {
		buckets[i].LE = b.String()
	}
	buckets[len(bucketBounds)].LE = "+Inf"
	for _, r := range results {
		i := 0
		for i < len(bucketBounds) && r.ElapsedTime > bucketBounds[i] {
			i++
		}
		buckets[i].Count++
	}
	return buckets
}

// kneeLatencyFactor is how much p99 must grow over the first step to mark the knee.
const kneeLatencyFactor = 2.0

// MarkKnee flags the first step whose p99 is at least kneeLatencyFactor times
// the first step's p99, i.e. where latency starts climbing before outright
// failure. It returns the knee's target RPS, or 0 if none was found.
func (r *RampReport) MarkKnee() int {
	if len(r.Steps) < 2 || r.Steps[0].P99Duration <= 0 {
		return 0
	}
	// a sub-millisecond baseline is noise; avoid flagging jitter as a knee
	base := r.Steps[0].P99Duration
	if base < time.Millisecond {
		base = time.Millisecond
	}
	for i := 1; i < len(r.Steps); i++ {
		if float64(r.Steps[i].P99Duration) >= kneeLatencyFactor*float64(base) {
			r.Steps[i].Knee = true
			r.KneeRPS = r.Steps[i].TargetRPS
			return r.KneeRPS
		}
	}
	return 0
}

// FormatStepLine is the one-line live progress output for a finished step.
func FormatStepLine(s Step) string {
	result := "PASS"
	if !s.Passed {
		result = "FAIL (" + s.Reason + ")"
	}
	return fmt.Sprintf("step %6d rps  achieved %8.1f  err %5.2f%%  p50 %-10s p99 %-10s %s",
		s.TargetRPS, s.AchievedRPS, s.ErrorRate, s.P50, s.P99, result)
}

// RampReport is the result of a stepped open-loop ramp test.
type RampReport struct {
	Host         string  `json:"host"`
	Method       string  `json:"method"`
	Steps        []Step  `json:"steps"`
	SustainedRPS int     `json:"sustained_rps"`
	KneeRPS      int     `json:"knee_rps,omitempty"`
	BreakingRPS  int     `json:"breaking_rps"`
	StopReason   string  `json:"stop_reason"`
	MaxErrorRate float64 `json:"max_error_rate_percent"`
	MaxP99       string  `json:"max_p99,omitempty"`
}

func ParseRampRaw(r RampReport) string {
	var b strings.Builder
	b.WriteString("\n==========================\n")
	b.WriteString(" YAHBA Ramp Test Report \n")
	b.WriteString("==========================\n\n")
	b.WriteString(fmt.Sprintf("Host:    %s\nMethod:  %s\n\n", r.Host, r.Method))
	b.WriteString(fmt.Sprintf("%-10s %-12s %-10s %-9s %-12s %-12s %-12s %s\n",
		"Target", "Achieved", "Requests", "Err %", "P50", "P95", "P99", "Result"))
	for _, s := range r.Steps {
		result := "PASS"
		if !s.Passed {
			result = "FAIL (" + s.Reason + ")"
		}
		if s.Knee {
			result += "  <-- knee"
		}
		b.WriteString(fmt.Sprintf("%-10d %-12.1f %-10d %-9.2f %-12s %-12s %-12s %s\n",
			s.TargetRPS, s.AchievedRPS, s.Requests, s.ErrorRate, s.P50, s.P95, s.P99, result))
	}
	b.WriteString("\nLatency histogram per step (requests per bucket, <= bound):\n")
	for _, s := range r.Steps {
		b.WriteString(fmt.Sprintf("  %6d rps:", s.TargetRPS))
		for _, h := range s.Histogram {
			if h.Count > 0 {
				b.WriteString(fmt.Sprintf("  %s:%d", h.LE, h.Count))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if r.SustainedRPS > 0 {
		b.WriteString(fmt.Sprintf("Sustained RPS:  %d\n", r.SustainedRPS))
	} else {
		b.WriteString("Sustained RPS:  none (the first step already failed)\n")
	}
	if r.KneeRPS > 0 {
		b.WriteString(fmt.Sprintf("Knee at:        %d RPS (p99 latency at least %.0fx the first step)\n", r.KneeRPS, kneeLatencyFactor))
	}
	if r.BreakingRPS > 0 {
		b.WriteString(fmt.Sprintf("Breaking RPS:   %d\n", r.BreakingRPS))
	}
	b.WriteString(fmt.Sprintf("Stopped:        %s\n\n", r.StopReason))
	return b.String()
}

func ParseRampJSON(r RampReport) (string, error) {
	out, err := json.MarshalIndent(r, "", "  ")
	return string(out), err
}

func ParseRampYAML(r RampReport) (string, error) {
	out, err := yaml.Marshal(r)
	return string(out), err
}
