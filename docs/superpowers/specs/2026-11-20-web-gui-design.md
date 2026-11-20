# YAHBA Web GUI Design

## Goal

Add a `yahba web` subcommand that serves a browser UI to configure and run load tests, watch live charts, and browse and compare saved run history. It ships as part of the single yahba binary.

## Decisions

- **Scope**: configure + run + live charts + saved history and comparison.
- **Frontend**: TypeScript source in `web/`, bundled with esbuild via `make web` into `internal/web/static/`. The compiled bundle is committed so `go:embed` and `go install` work without Node. Charts use vendored uPlot.
- **Live updates**: Server-Sent Events (SSE).
- **Storage**: SQLite at `~/.yahba/history.db` via `modernc.org/sqlite` (pure Go, no cgo).
- **Security**: binds to `127.0.0.1:8080` by default (`--addr` to override), since the tool can launch load against arbitrary URLs.

## Architecture

- `cmd/web.go`: the `web` subcommand, parallel to `cmd/tui.go`.
- `internal/web/server.go`: HTTP routes and `go:embed` static assets.
- `internal/web/runner.go`: starts runs using the existing `config` and `worker` packages (the same engine as the TUI), and fans live stats out to SSE subscribers. One active run at a time.
- `internal/web/store.go`: SQLite history store; the only code that touches the DB.
- `web/`: TypeScript sources. `internal/web/static/`: compiled output (committed).

## API

| Method | Path | Purpose |
| ------ | ---- | ------- |
| POST | `/api/runs` | Start a run from config JSON; returns id (400 on invalid config, 409 if a run is active) |
| GET | `/api/runs/{id}/events` | SSE stream: RPS, latency percentiles, status codes, errors |
| DELETE | `/api/runs/{id}` | Stop the active run |
| GET | `/api/runs` | List history |
| GET | `/api/runs/{id}` | Config, final report, samples |
| GET | `/api/compare?a=&b=` | Two reports side by side |

## UI

1. **New run**: form for the common `run` flags, with the same validation rules as `config`.
2. **Live run**: uPlot charts (RPS, latency percentiles over time), status-code counts, error list, Stop button.
3. **History**: table of past runs; selecting two opens **Compare** (overlaid latency charts and a delta table).

## Data Model

- `runs`: id, started_at, duration, status (`running`/`completed`/`stopped`/`aborted`), config JSON, report JSON.
- `samples`: run_id, ts, and the live metrics, used to replay charts for past runs.

## Error Handling

- Invalid config: 400 with per-field errors.
- Second concurrent run: 409.
- Client disconnect mid-run: the run continues and its results are saved.
- On startup, runs left in `running` state by a crash are marked `aborted`.

## Testing

- Go: store (temp DB), runner (using `internal/server` as the target), HTTP handlers (`httptest`).
- TypeScript: unit tests for the compare and format helpers.

## Out of Scope

Authentication, multi-user, remote exposure hardening, and concurrent runs.
