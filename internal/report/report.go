package report

import "time"

type Report struct {
	Results        []Result       `json:"reports"`
	ErrorBreakdown ErrorBreakdown `json:"error_breakdown"`
	Latency        Latency        `json:"latency"`
	Throughput     Throughput     `json:"throughput"`
	StatusCodes    StatusCodes    `json:"status_codes"`
	TotalRequests  int            `json:"total_requests"`
	Rps            int            `json:"rps"`
	Successes      int            `json:"success"`
	Failures       int            `json:"failures"`
}

type Result struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	ElapsedTime time.Duration `json:"elapsed_time"`
	WorkerID    int           `json:"worker_id"`
	ResultCode  int           `json:"result_code"`
	Error       error         `json:"error"`
	TargetURL   string        `json:"target_url"`
	Method      string        `json:"method"`
	// DNSTime        string        `json:"dns_time"`
	// ConnectionTime string        `json:"connection_time"`
}

type ErrorBreakdown struct {
	Timeouts     int `json:"timeouts"`
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
	TotalDataSent     string `json:"total_data_sent"`
	TotalDataReceived string `json:"total_data_received"`
	AvgDataPerRequest string `json:"avg_data_per_request"`
}

type StatusCodes struct {
	Num200 int `json:"200"`
	Num400 int `json:"400"`
	Num403 int `json:"403"`
	Num404 int `json:"404"`
	Num500 int `json:"500"`
	Num502 int `json:"502"`
	Num503 int `json:"503"`
	Num504 int `json:"504"`
}
