package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Step summarises one rate step of a ramp test.
type Step struct {
	TargetRPS   int     `json:"target_rps"`
	AchievedRPS float64 `json:"achieved_rps"`
	Requests    int     `json:"requests"`
	Failures    int     `json:"failures"`
	ErrorRate   float64 `json:"error_rate_percent"`
	P50         string  `json:"p50"`
	P95         string  `json:"p95"`
	P99         string  `json:"p99"`
	Passed      bool    `json:"passed"`
	Reason      string  `json:"reason,omitempty"`
}

// RampReport is the result of a stepped open-loop ramp test.
type RampReport struct {
	Host         string  `json:"host"`
	Method       string  `json:"method"`
	Steps        []Step  `json:"steps"`
	SustainedRPS int     `json:"sustained_rps"`
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
		b.WriteString(fmt.Sprintf("%-10d %-12.1f %-10d %-9.2f %-12s %-12s %-12s %s\n",
			s.TargetRPS, s.AchievedRPS, s.Requests, s.ErrorRate, s.P50, s.P95, s.P99, result))
	}
	b.WriteString("\n")
	if r.SustainedRPS > 0 {
		b.WriteString(fmt.Sprintf("Sustained RPS:  %d\n", r.SustainedRPS))
	} else {
		b.WriteString("Sustained RPS:  none (the first step already failed)\n")
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
