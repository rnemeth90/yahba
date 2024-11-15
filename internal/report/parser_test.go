package report

import (
	"strings"
	"testing"
	"time"
)

// Helper function to create a sample Report for testing
func sampleReport() Report {
	return Report{
		TotalRequests: 100,
		Successes:     90,
		Failures:      10,
		Latency: Latency{
			Min: "20ms",
			Max: "1.5s",
			Avg: "200ms",
			P50: "180ms",
			P95: "1s",
			P99: "1.4s",
		},
		Throughput: Throughput{
			TotalBytesSent:     1000000,
			TotalBytesReceived: 5000000,
		},
		StatusCodes: StatusCodes{
			Num200: 90,
			Num400: 5,
			Num403: 2,
			Num404: 2,
			Num500: 1,
			Num502: 0,
			Num503: 0,
			Num504: 0,
		},
		ErrorBreakdown: ErrorBreakdown{
			ServerErrors: 5,
			ClientErrors: 5,
		},
		Results: []Result{
			{
				WorkerID:    1,
				ResultCode:  200,
				ElapsedTime: time.Duration(200 * time.Millisecond),
				TargetURL:   "http://example.com",
				Timeout:     false,
			},
			{
				WorkerID:    2,
				ResultCode:  500,
				ElapsedTime: time.Duration(1 * time.Second),
				TargetURL:   "http://example.com",
				Timeout:     false,
			},
		},
	}
}

func TestParseRaw(t *testing.T) {
	reportChan := make(chan Report, 1)
	reportChan <- sampleReport()
	close(reportChan)

	output, err := ParseRaw(reportChan)
	if err != nil {
		t.Fatalf("unexpected error in ParseRaw: %v", err)
	}

	if !strings.Contains(output, "Total Requests:      100") {
		t.Errorf("expected 'Total Requests:      100' in raw output, got: %s", output)
	}
	if !strings.Contains(output, "Latency:") {
		t.Errorf("expected 'Latency' section in raw output, got: %s", output)
	}
	if !strings.Contains(output, "Status Code Breakdown:") {
		t.Errorf("expected 'Status Code Breakdown' section in raw output, got: %s", output)
	}
	if !strings.Contains(output, "Worker 1 | Status: 200") {
		t.Errorf("expected individual result for Worker 1 in raw output, got: %s", output)
	}
}

func TestParseRawChannelClosed(t *testing.T) {
	reportChan := make(chan Report)
	close(reportChan)

	_, err := ParseRaw(reportChan)
	if err == nil || !strings.Contains(err.Error(), "channel unexpectedly closed") {
		t.Errorf("expected error for closed channel, got: %v", err)
	}
}

func TestParseJSON(t *testing.T) {
	reportChan := make(chan Report, 1)
	reportChan <- sampleReport()
	close(reportChan)

	output, err := ParseJSON(reportChan)
	if err != nil {
		t.Fatalf("unexpected error in ParseJSON: %v", err)
	}

	if !strings.Contains(output, `"TotalRequests": 100`) {
		t.Errorf("expected 'TotalRequests': 100 in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"Latency"`) {
		t.Errorf("expected 'Latency' section in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"StatusCodes"`) {
		t.Errorf("expected 'StatusCodes' section in JSON output, got: %s", output)
	}
}

func TestParseJSONChannelClosed(t *testing.T) {
	reportChan := make(chan Report)
	close(reportChan)

	_, err := ParseJSON(reportChan)
	if err == nil || !strings.Contains(err.Error(), "channel unexpectedly closed") {
		t.Errorf("expected error for closed channel, got: %v", err)
	}
}

func TestParseYAML(t *testing.T) {
	reportChan := make(chan Report, 1)
	reportChan <- sampleReport()
	close(reportChan)

	output, err := ParseYAML(reportChan)
	if err != nil {
		t.Fatalf("unexpected error in ParseYAML: %v", err)
	}

	if !strings.Contains(output, "TotalRequests: 100") {
		t.Errorf("expected 'TotalRequests: 100' in YAML output, got: %s", output)
	}
	if !strings.Contains(output, "Latency:") {
		t.Errorf("expected 'Latency' section in YAML output, got: %s", output)
	}
	if !strings.Contains(output, "StatusCodes:") {
		t.Errorf("expected 'StatusCodes' section in YAML output, got: %s", output)
	}
}

func TestParseYAMLChannelClosed(t *testing.T) {
	reportChan := make(chan Report)
	close(reportChan)

	_, err := ParseYAML(reportChan)
	if err == nil || !strings.Contains(err.Error(), "channel unexpectedly closed") {
		t.Errorf("expected error for closed channel, got: %v", err)
	}
}
