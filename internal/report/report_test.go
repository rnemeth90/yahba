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

	// Calculate latency metrics
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

// Test for ConvertResultCodes
func TestConvertResultCodes(t *testing.T) {
	// Sample status code map
	statusCodeMap := map[int]int{
		200: 50,
		201: 10,
		204: 5,
		400: 3,
		403: 2,
		404: 7,
		500: 1,
		502: 0,
		503: 4,
		504: 6,
	}

	// Create a Report and convert result codes
	report := Report{}
	report.ConvertResultCodes(statusCodeMap)

	// Assertions
	if report.StatusCodes.Num200 != 50 {
		t.Errorf("expected 50 for 200 OK, got %d", report.StatusCodes.Num200)
	}
	if report.StatusCodes.Num201 != 10 {
		t.Errorf("expected 10 for 201 Created, got %d", report.StatusCodes.Num201)
	}
	if report.StatusCodes.Num204 != 5 {
		t.Errorf("expected 5 for 204 No Content, got %d", report.StatusCodes.Num204)
	}
	if report.StatusCodes.Num400 != 3 {
		t.Errorf("expected 3 for 400 Bad Request, got %d", report.StatusCodes.Num400)
	}
	if report.StatusCodes.Num403 != 2 {
		t.Errorf("expected 2 for 403 Forbidden, got %d", report.StatusCodes.Num403)
	}
	if report.StatusCodes.Num404 != 7 {
		t.Errorf("expected 7 for 404 Not Found, got %d", report.StatusCodes.Num404)
	}
	if report.StatusCodes.Num500 != 1 {
		t.Errorf("expected 1 for 500 Internal Server Error, got %d", report.StatusCodes.Num500)
	}
	if report.StatusCodes.Num502 != 0 {
		t.Errorf("expected 0 for 502 Bad Gateway, got %d", report.StatusCodes.Num502)
	}
	if report.StatusCodes.Num503 != 4 {
		t.Errorf("expected 4 for 503 Service Unavailable, got %d", report.StatusCodes.Num503)
	}
	if report.StatusCodes.Num504 != 6 {
		t.Errorf("expected 6 for 504 Gateway Timeout, got %d", report.StatusCodes.Num504)
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
