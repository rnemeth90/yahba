package report

import (
	"testing"
	"time"
)

// Test for CalculateLatencyMetrics
func TestCalculateLatencyMetrics(t *testing.T) {
	// Create a sample Report with different latencies
	report := Report{
		Results: []Result{
			{ElapsedTime: 100 * time.Millisecond},
			{ElapsedTime: 200 * time.Millisecond},
			{ElapsedTime: 300 * time.Millisecond},
			{ElapsedTime: 500 * time.Millisecond},
			{ElapsedTime: 1500 * time.Millisecond},
		},
		TotalRequests: 5,
	}
	report.CalculateLatencyMetrics()

	// Assertions
	expectedMin := "100ms"
	expectedMax := "1.5s"
	expectedAvg := "520ms"
	expectedP50 := "300ms"
	expectedP95 := "1.5s"
	expectedP99 := "1.5s"

	if report.Latency.Min != expectedMin {
		t.Errorf("expected min latency %s, got %s", expectedMin, report.Latency.Min)
	}
	if report.Latency.Max != expectedMax {
		t.Errorf("expected max latency %s, got %s", expectedMax, report.Latency.Max)
	}
	if report.Latency.Avg != expectedAvg {
		t.Errorf("expected average latency %s, got %s", expectedAvg, report.Latency.Avg)
	}
	if report.Latency.P50 != expectedP50 {
		t.Errorf("expected P50 latency %s, got %s", expectedP50, report.Latency.P50)
	}
	if report.Latency.P95 != expectedP95 {
		t.Errorf("expected P95 latency %s, got %s", expectedP95, report.Latency.P95)
	}
	if report.Latency.P99 != expectedP99 {
		t.Errorf("expected P99 latency %s, got %s", expectedP99, report.Latency.P99)
	}
}

// Test for StatusCodes map assignment
func TestStatusCodes(t *testing.T) {
	statusCodeMap := map[int]int{
		200: 50,
		201: 10,
		204: 5,
		400: 3,
		403: 2,
		404: 7,
		500: 1,
		503: 4,
		504: 6,
	}

	report := Report{}
	report.StatusCodes = statusCodeMap

	expected := map[int]int{
		200: 50, 201: 10, 204: 5, 400: 3,
		403: 2, 404: 7, 500: 1, 503: 4, 504: 6,
	}
	for code, count := range expected {
		if report.StatusCodes[code] != count {
			t.Errorf("expected %d for status %d, got %d", count, code, report.StatusCodes[code])
		}
	}
}

// Test formatDuration helper function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{100 * time.Millisecond, "100ms"},
		{2 * time.Second, "2s"},
		{500 * time.Microsecond, "500µs"},
	}

	for _, test := range tests {
		result := formatDuration(test.input)
		if result != test.expected {
			t.Errorf("expected %s, got %s", test.expected, result)
		}
	}
}
