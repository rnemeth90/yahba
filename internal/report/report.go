package report

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/rnemeth90/yahba/internal/util"
)

type Report struct {
	Host           string         `json:"host"`
	Method         string         `json:"method"`
	Results        []Result       `json:"results"`
	ErrorBreakdown ErrorBreakdown `json:"error_breakdown"`
	Latency        Latency        `json:"latency"`
	Throughput     Throughput     `json:"throughput"`
	StatusCodes    map[int]int    `json:"status_codes"`
	TotalRequests  int            `json:"total_requests"`
	Successes      int            `json:"success"`
	Failures       int            `json:"failures"`
	StartTime      string         `json:"start_time"`
	EndTime        string         `json:"end_time"`
	Duration       time.Duration  `json:"duration"`
}

type Result struct {
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	ElapsedTime   time.Duration `json:"elapsed_time"`
	WorkerID      int           `json:"worker_id"`
	ResultCode    int           `json:"result_code"`
	Error         error         `json:"error"`
	TargetURL     string        `json:"target_url"`
	Method        string        `json:"method"`
	Timeout       bool          `json:"timeout"`
	BytesSent     int           `json:"bytes_sent"`
	BytesReceived int           `json:"bytes_received"`
}

type ErrorBreakdown struct {
	ServerErrors int `json:"server_errors"`
	ClientErrors int `json:"client_errors"`
}

type Latency struct {
	Min string `json:"min"`
	Max string `json:"max"`
	Avg string `json:"avg"`
	P50 string `json:"p50"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
}

type Throughput struct {
	TotalBytesSent         int     `json:"total_bytes_sent"`
	TotalBytesReceived     int     `json:"total_bytes_received"`
	BytesSentPerSecond     float64 `json:"bytes_sent_per_second"`
	BytesReceivedPerSecond float64 `json:"bytes_received_per_second"`
}

func (r *Report) CalculateLatencyMetrics() {
	if r.TotalRequests == 0 {
		r.Latency = Latency{}
		return
	}

	var latencies []time.Duration
	var totalLatency time.Duration
	for _, result := range r.Results {
		latencies = append(latencies, result.ElapsedTime)
		totalLatency += result.ElapsedTime
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	n := len(latencies)
	minLatency := latencies[0]
	maxLatency := latencies[n-1]
	avgLatency := totalLatency / time.Duration(n)

	r.Latency = Latency{
		Min: formatDuration(minLatency),
		Max: formatDuration(maxLatency),
		Avg: formatDuration(avgLatency),
		P50: formatDuration(latencies[percentileIndex(n, 50)]),
		P95: formatDuration(latencies[percentileIndex(n, 95)]),
		P99: formatDuration(latencies[percentileIndex(n, 99)]),
	}
}

// percentileIndex returns the index for a given percentile using the nearest-rank method.
func percentileIndex(n int, percentile float64) int {
	idx := int(math.Ceil(percentile/100*float64(n))) - 1
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

// Format duration into a readable string
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%v", d)
}

// Aggregate combines the results of multiple sub-tests (e.g. each named
// request in a definition file) into a single combined Report. Host and
// Method are left blank; callers should set them based on the sub-tests
// that were run.
func Aggregate(results []Result) Report {
	r := Report{
		Results:     results,
		StatusCodes: make(map[int]int),
	}

	var totalBytesSent, totalBytesReceived int
	for _, result := range results {
		r.TotalRequests++
		r.StatusCodes[result.ResultCode]++
		totalBytesSent += result.BytesSent
		totalBytesReceived += result.BytesReceived

		if result.ResultCode >= 400 && result.ResultCode <= 499 {
			r.ErrorBreakdown.ClientErrors++
		} else if result.ResultCode >= 500 && result.ResultCode <= 599 {
			r.ErrorBreakdown.ServerErrors++
		}

		if result.ResultCode == 0 || result.ResultCode >= 400 {
			r.Failures++
		} else {
			r.Successes++
		}
	}

	if len(r.Results) > 0 {
		earliest := r.Results[0].StartTime
		latest := r.Results[0].EndTime
		for _, res := range r.Results[1:] {
			if res.StartTime.Before(earliest) {
				earliest = res.StartTime
			}
			if res.EndTime.After(latest) {
				latest = res.EndTime
			}
		}
		r.StartTime = earliest.Format(time.RFC3339)
		r.EndTime = latest.Format(time.RFC3339)
		r.Duration = latest.Sub(earliest)
	}

	r.Throughput.TotalBytesSent = totalBytesSent
	r.Throughput.TotalBytesReceived = totalBytesReceived
	r.Throughput.BytesSentPerSecond = util.CalculateBytesPerSecond(float64(totalBytesSent), r.Duration.Seconds())
	r.Throughput.BytesReceivedPerSecond = util.CalculateBytesPerSecond(float64(totalBytesReceived), r.Duration.Seconds())

	r.CalculateLatencyMetrics()

	return r
}
