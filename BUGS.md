# BUGS.md

Known bugs and issues identified during code review.

---

## High Priority

FIXED
### 1. `httputil.DumpResponse` used for byte counting
**File:** `internal/worker/worker.go`
Reads the entire response body into memory just to count bytes. Should be replaced with:
```go
n, _ := io.Copy(io.Discard, resp.Body)
```
Impact: excessive memory usage and slow performance for large responses.

---

FIXED
### 2. Unbounded latency slice in TUI
**File:** `internal/tui/update.go:165`
Every request latency is appended to `m.latencies` forever. For long or high-volume tests this grows to hundreds of MB. Should use a fixed-size ring buffer or an online algorithm (e.g. Welford's) for O(1) space.

Fixed using the project's own `internal/queue.CircularBuffer` (bounded to `latencyBufferSize` = 512 in `internal/tui/model.go`). `handleResult` now calls `m.latencies.Push(...)`, which evicts the oldest sample once full instead of growing forever. The graph reads a non-destructive `Items()` snapshot; the "last N of TOTAL" label now uses `m.completed` (the true cumulative count) instead of the buffer's own length, since the buffer no longer holds the full history.

---

FIXED
### 3. Logger file handle never closed
**File:** `internal/logger/logger.go`
When logging to a file, the `*os.File` is opened but never closed. A `Close() error` method should be added to the logger and called during shutdown.

---

FIXED
### 13. Log file closed immediately after being opened, breaking file logging
**File:** `internal/logger/logger.go:121-126` (`SetOutputDestination`)
```go
f, err := os.OpenFile(destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
if err != nil {
    return err
}
l.Logger = log.New(f, "", log.LstdFlags)
defer f.Close()
```
The apparent fix for issue #3 introduced a new bug: `defer f.Close()` runs when `SetOutputDestination` returns, not when the logger is done — so the file is closed before any log line is ever written to it, and every subsequent `Debug`/`Info`/`Warn`/`Error` call silently writes to a closed `*os.File` (the error from `log.Logger.Output` is discarded). This currently fails `TestSetOutputDestination` (`internal/logger/logger_test.go`): `expected message to be logged to file, got ""`. The fix from #3 should be restored instead — track the `*os.File` on the `Logger` struct and close it explicitly from a `Close() error` method called at shutdown, not via `defer` inside `SetOutputDestination`.

---

FIXED
### 4. Channel buffers sized to `len(jobs)`
**File:** `internal/worker/worker.go:127-128`
Result and progress channels are buffered to the full job count. For large runs (e.g. 100k requests) this pre-allocates a large amount of memory unnecessarily. A fixed-size buffer (e.g. 1024) with natural backpressure is sufficient.

---

FIXED
### 5. HTTP request recreated per job
**File:** `internal/worker/worker.go:251-256`
A new `*http.Request` is constructed for every job even when all requests are identical. The request should be built once as a template and cloned via `req.Clone(ctx)` for each job.

---

FIXED
### 6. TUI status code map panics on result code 0
**File:** `internal/tui/update.go:161`
A network error produces `ResultCode == 0`. Writing `m.statusCodes[0]++` is valid Go but semantically wrong and causes incorrect status code reporting. The write should be guarded:
```go
if r.ResultCode > 0 {
    m.statusCodes[r.ResultCode]++
}
```

---
 
FIXED
### 12. Empty response bodies are misreported as connection errors
**File:** `internal/worker/worker.go` (`processResponse`)
```go
n, err := io.Copy(io.Discard, resp.Body)
if err != nil || n <= 0 {
    // treated as a failed request, ResultCode left at 0
}
```
A successful response with no body (e.g. `204 No Content`, a `HEAD` response, or any `200`/`201` with `Content-Length: 0`) has `n == 0` and `err == nil`, but is still routed into the "no response received" failure branch. `result.ResultCode` is never set to the real status code, so the request is counted as a connection failure (`ResultCode == 0`) in the aggregated report even though the server responded successfully. Found while smoke-testing the TUI's new definition-file runner against a local test server that returns an empty `200` body. The `n <= 0` check should be dropped (or only `err != nil` should be treated as a failure).

---

## Medium Priority

FIXED
### 7. RPS ticker does not account for response latency
**File:** `internal/worker/worker.go:141-150`
`time.NewTicker(time.Second / time.Duration(cfg.RPS))` controls dispatch rate only. When responses are slow, actual RPS will be lower than configured with no warning to the user. Should use `golang.org/x/time/rate` (token bucket) and report achieved vs. configured RPS in output.

---

FIXED
### 8. Throughput calculation uses sum of durations instead of wall time
**File:** `internal/report/report.go`
Total bytes divided by the sum of individual request durations over-reports throughput when requests run in parallel. The denominator should be `testEnd - testStart` (wall clock elapsed), not the sum of all per-request elapsed times.

---

FIXED
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

FIXED
### 11. No upper bound on `--rps` flag
**File:** `cmd/run.go`
There is no maximum validation on the `--rps` value. At 1,000,000 RPS the ticker interval drops to 1µs, which is unrealistic and will silently produce inaccurate results. A reasonable cap or warning should be added.


## Low Priority
**File:** `cmd/run.go`
When tests are passed in via a definition file, and the `--distributed` flag is passed, the tests are not ran in distributed mode. 
