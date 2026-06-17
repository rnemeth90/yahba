# BUGS.md

Known bugs and issues identified during code review.

---

## High Priority

### 1. `httputil.DumpResponse` used for byte counting
**File:** `internal/worker/worker.go`
Reads the entire response body into memory just to count bytes. Should be replaced with:
```go
n, _ := io.Copy(io.Discard, resp.Body)
```
Impact: excessive memory usage and slow performance for large responses.

---

### 2. Unbounded latency slice in TUI
**File:** `internal/tui/update.go:165`
Every request latency is appended to `m.latencies` forever. For long or high-volume tests this grows to hundreds of MB. Should use a fixed-size ring buffer or an online algorithm (e.g. Welford's) for O(1) space.

---

### 3. Logger file handle never closed
**File:** `internal/logger/logger.go`
When logging to a file, the `*os.File` is opened but never closed. A `Close() error` method should be added to the logger and called during shutdown.

---

### 4. Channel buffers sized to `len(jobs)`
**File:** `internal/worker/worker.go:127-128`
Result and progress channels are buffered to the full job count. For large runs (e.g. 100k requests) this pre-allocates a large amount of memory unnecessarily. A fixed-size buffer (e.g. 1024) with natural backpressure is sufficient.

---

### 5. HTTP request recreated per job
**File:** `internal/worker/worker.go:251-256`
A new `*http.Request` is constructed for every job even when all requests are identical. The request should be built once as a template and cloned via `req.Clone(ctx)` for each job.

---

### 6. TUI status code map panics on result code 0
**File:** `internal/tui/update.go:161`
A network error produces `ResultCode == 0`. Writing `m.statusCodes[0]++` is valid Go but semantically wrong and causes incorrect status code reporting. The write should be guarded:
```go
if r.ResultCode > 0 {
    m.statusCodes[r.ResultCode]++
}
```

---

## Medium Priority

### 7. RPS ticker does not account for response latency
**File:** `internal/worker/worker.go:141-150`
`time.NewTicker(time.Second / time.Duration(cfg.RPS))` controls dispatch rate only. When responses are slow, actual RPS will be lower than configured with no warning to the user. Should use `golang.org/x/time/rate` (token bucket) and report achieved vs. configured RPS in output.

---

### 8. Throughput calculation uses sum of durations instead of wall time
**File:** `internal/report/report.go`
Total bytes divided by the sum of individual request durations over-reports throughput when requests run in parallel. The denominator should be `testEnd - testStart` (wall clock elapsed), not the sum of all per-request elapsed times.

---

### 9. Logger test leaves `yahba.log` in working directory
**File:** `internal/logger/logger_test.go:98`
The test creates `yahba.log` in the current working directory and may not clean it up if the test fails early. Should use `t.TempDir()` so cleanup is guaranteed.

---

### 10. Unresolved TODO comments
**Files:** `cmd/run.go:101`, `cmd/run.go:111`
Two TODO comments indicate unresolved design questions:
- "do we need to parse headers HERE? why?"
- "do we need to create individual jobs if the jobs are all the same?"

The second question is answered by issue #5 above (no — clone a template instead). These should be resolved or tracked as issues.

---

### 11. No upper bound on `--rps` flag
**File:** `cmd/run.go`
There is no maximum validation on the `--rps` value. At 1,000,000 RPS the ticker interval drops to 1µs, which is unrealistic and will silently produce inaccurate results. A reasonable cap or warning should be added.
