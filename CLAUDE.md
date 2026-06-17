# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**YAHBA** (Yet Another HTTP Benchmark Application) — a CLI HTTP load testing tool written in Go. It dispatches concurrent HTTP requests at configurable rates and reports aggregated latency/throughput metrics.

## Commands

```bash
make build          # build binary: ./yahba
make test           # go test ./...
make format         # gofmt -w -s .
make clean          # remove build artifacts
make release VERSION=X.Y.Z  # cross-compile for 8 platforms
```

Run a single test package: `go test ./internal/worker/...`  
Run with race detector (same as CI): `go test -v -race ./...`

CI also runs `go vet ./...` and a `gofmt` diff check — run these before submitting.

## Architecture

Entry: `main.go` → `cmd.Execute()` (Cobra root command)

**Subcommands** (`cmd/`):
- `run` — primary load test orchestrator
- `tui` — Bubble Tea interactive dashboard
- `server` — local test HTTP server

**Data flow for `run`**:
1. Cobra flags → `config.Config` struct
2. `config.Validate()` enforces constraints (required URL, method allowlist, POST/PUT require body, proxy/DNS format, conflicting flag checks)
3. Job queue built (one `worker.Job` per request)
4. Worker pool (`internal/worker`) dispatches jobs via a ticker at the configured RPS; results collected concurrently
5. `internal/report` aggregates results into latency percentiles (min/max/avg/P50/P95/P99), throughput, and status-code breakdown
6. Output formatted as raw/JSON/YAML to stdout or file

**Internal packages**:

| Package | Role |
|---|---|
| `internal/config` | `Config` struct, validation, proxy/resolver setup |
| `internal/worker` | Worker pool, ticker-based RPS, context cancellation |
| `internal/client` | HTTP client factory: TLS, HTTP/2, compression, proxy, custom DNS |
| `internal/report` | Result aggregation, percentile calculation, multi-format output |
| `internal/logger` | Leveled logger (debug/info/warn/error); silent mode for structured output |
| `internal/util` | Header parsing, request size calculation, user-agent generation |
| `internal/tui` | Bubble Tea model/view/update + lipgloss styles |
| `internal/rawsocket` | Raw socket support (in-progress, platform-specific via build tags) |

**Key types**:
- `config.Config` — all test parameters
- `worker.Job` — per-request metadata (ID, host, method, body)
- `report.Result` — single request outcome (timing, status, bytes, error)
- `report.Report` — aggregated results

## Keeping CLAUDE.md current

After any change that affects the information in this file — new/removed subcommands, packages, flags, key types, build commands, or architectural patterns — update the relevant section of CLAUDE.md in the same commit or PR.

## Notable constraints

- Workers default to 10, max 1000.
- HTTP/2 and HTTP/3 flags are mutually exclusive; `--skip-dns` and `--resolver` are mutually exclusive.
- `rawsocket` uses build tags (`sockopt_linux.go` / `sockopt_other.go`) for platform differences.
- Logger must be set to silent mode when producing JSON/YAML output to avoid mixing log lines into structured output.
