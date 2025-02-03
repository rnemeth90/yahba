package report

import (
	"fmt"
	"sort"
	"time"
)

type Report struct {
	Host           string         `json:"host"`
	Method         string         `json:"method"`
	Results        []Result       `json:"results"`
	ErrorBreakdown ErrorBreakdown `json:"error_breakdown"`
	Latency        Latency        `json:"latency"`
	Throughput     Throughput     `json:"throughput"`
	StatusCodes    StatusCodes    `json:"status_codes"`
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

type StatusCodes struct {
	Num200 int `json:"200"`
	Num201 int `json:"201"`
	Num204 int `json:"204"`
	Num400 int `json:"400"`
	Num403 int `json:"403"`
	Num404 int `json:"404"`
	Num408 int `json:"408"`
	Num429 int `json:"429"`
	Num500 int `json:"500"`
	Num502 int `json:"502"`
	Num503 int `json:"503"`
	Num504 int `json:"504"`
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

	minLatency := latencies[0]
	maxLatency := latencies[len(latencies)-1]
	avgLatency := totalLatency / time.Duration(len(latencies))
	p50 := latencies[len(latencies)*50/100]
	p95 := latencies[len(latencies)*95/100]
	p99 := latencies[len(latencies)*99/100]

	r.Latency = Latency{
		Min: formatDuration(minLatency),
		Max: formatDuration(maxLatency),
		Avg: formatDuration(avgLatency),
		P50: formatDuration(p50),
		P95: formatDuration(p95),
		P99: formatDuration(p99),
	}
}

// Format duration into a readable string
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%v", d)
}

func (r *Report) ConvertResultCodes(m map[int]int) {
	statusCodes := StatusCodes{}

	for resultCode, count := range m {
		switch resultCode {
		case 200:
			statusCodes.Num200 = count
		case 201:
			statusCodes.Num201 = count
		case 204:
			statusCodes.Num204 = count
		case 400:
			statusCodes.Num400 = count
		case 403:
			statusCodes.Num403 = count
		case 404:
			statusCodes.Num404 = count
		case 408:
			statusCodes.Num408 = count
		case 429:
			statusCodes.Num429 = count
		case 500:
			statusCodes.Num500 = count
		case 502:
			statusCodes.Num502 = count
		case 503:
			statusCodes.Num503 = count
		case 504:
			statusCodes.Num504 = count
		}
	}

	r.StatusCodes = statusCodes
}
