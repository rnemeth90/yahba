package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseRaw generates a raw-text summary of the report
func ParseRaw(report Report) (string, error) {
	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString("\n")
	builder.WriteString("YAHBA Stress Test Report\n")
	builder.WriteString("========================\n\n")
	builder.WriteString(fmt.Sprintf("Total Requests:      %d\n", report.TotalRequests))
	builder.WriteString(fmt.Sprintf("Successes:           %d\n", report.Successes))
	builder.WriteString(fmt.Sprintf("Failures:            %d\n\n", report.Failures))

	builder.WriteString("Latency:\n")
	builder.WriteString(fmt.Sprintf("  Min: %s\n", report.Latency.Min))
	builder.WriteString(fmt.Sprintf("  Max: %s\n", report.Latency.Max))
	builder.WriteString(fmt.Sprintf("  Avg: %s\n", report.Latency.Avg))
	builder.WriteString(fmt.Sprintf("  P50: %s\n", report.Latency.P50))
	builder.WriteString(fmt.Sprintf("  P95: %s\n", report.Latency.P95))
	builder.WriteString(fmt.Sprintf("  P99: %s\n\n", report.Latency.P99))

	builder.WriteString("Throughput:\n")
	builder.WriteString(fmt.Sprintf("  Total Bytes Sent:     %d\n", report.Throughput.TotalBytesSent))
	builder.WriteString(fmt.Sprintf("  Total Bytes Received: %d\n\n", report.Throughput.TotalBytesReceived))

	builder.WriteString("Status Code Breakdown:\n")
	builder.WriteString(fmt.Sprintf("  200 OK:                 %d\n", report.StatusCodes.Num200))
	builder.WriteString(fmt.Sprintf("  400 Bad Request:        %d\n", report.StatusCodes.Num400))
	builder.WriteString(fmt.Sprintf("  403 Forbidden:          %d\n", report.StatusCodes.Num403))
	builder.WriteString(fmt.Sprintf("  404 Not Found:          %d\n", report.StatusCodes.Num404))
	builder.WriteString(fmt.Sprintf("  408 Request Timed Out:  %d\n", report.StatusCodes.Num408))
	builder.WriteString(fmt.Sprintf("  429 Too Many Requests:  %d\n", report.StatusCodes.Num429))
	builder.WriteString(fmt.Sprintf("  500 Internal Server Error: %d\n", report.StatusCodes.Num500))
	builder.WriteString(fmt.Sprintf("  502 Bad Gateway:        %d\n", report.StatusCodes.Num502))
	builder.WriteString(fmt.Sprintf("  503 Service Unavailable: %d\n", report.StatusCodes.Num503))
	builder.WriteString(fmt.Sprintf("  504 Gateway Timeout:    %d\n\n", report.StatusCodes.Num504))

	builder.WriteString("Error Breakdown:\n")
	builder.WriteString(fmt.Sprintf("  Server Errors:          %d\n", report.ErrorBreakdown.ServerErrors))
	builder.WriteString(fmt.Sprintf("  Client Errors:          %d\n", report.ErrorBreakdown.ClientErrors))
	builder.WriteString("\n")

	return builder.String(), nil
}

func ParseJSON(report Report) (string, error) {
	jsonStr, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonStr), nil
}

func ParseYAML(report Report) (string, error) {
	yamlStr, err := yaml.Marshal(report)
	if err != nil {
		return "", err
	}

	return string(yamlStr), nil
}
