package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("run not found")

type RunStatus string

const (
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusStopped   RunStatus = "stopped"
	StatusAborted   RunStatus = "aborted"
	StatusFailed    RunStatus = "failed"
)

type Run struct {
	ID         int64           `json:"id"`
	StartedAt  time.Time       `json:"started_at"`
	DurationMs int64           `json:"duration_ms"`
	Status     RunStatus       `json:"status"`
	Config     json.RawMessage `json:"config"`
	Summary    json.RawMessage `json:"summary,omitempty"`
}

const schema = `
CREATE TABLE IF NOT EXISTS runs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	started_at  TEXT NOT NULL,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	status      TEXT NOT NULL,
	config      TEXT NOT NULL,
	summary     TEXT
);
CREATE TABLE IF NOT EXISTS samples (
	run_id INTEGER NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
	t      INTEGER NOT NULL,
	data   TEXT NOT NULL,
	PRIMARY KEY (run_id, t)
);`

// Store persists runs and their per-second samples in SQLite.
type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CreateRun(cfg json.RawMessage, startedAt time.Time) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO runs (started_at, status, config) VALUES (?, ?, ?)`,
		startedAt.UTC().Format(time.RFC3339Nano), StatusRunning, string(cfg),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) AddSample(runID int64, smp Sample) error {
	data, err := json.Marshal(smp)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO samples (run_id, t, data) VALUES (?, ?, ?)`, runID, smp.T, string(data))
	return err
}

func (s *Store) FinishRun(id int64, status RunStatus, durationMs int64, summary json.RawMessage) error {
	var sum any
	if len(summary) > 0 {
		sum = string(summary)
	}
	_, err := s.db.Exec(`UPDATE runs SET status = ?, duration_ms = ?, summary = ? WHERE id = ?`, status, durationMs, sum, id)
	return err
}

func (s *Store) GetRun(id int64) (Run, error) {
	row := s.db.QueryRow(`SELECT id, started_at, duration_ms, status, config, summary FROM runs WHERE id = ?`, id)
	run, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	return run, err
}

func (s *Store) ListRuns() ([]Run, error) {
	rows, err := s.db.Query(`SELECT id, started_at, duration_ms, status, config, summary FROM runs ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []Run{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) Samples(runID int64) ([]Sample, error) {
	rows, err := s.db.Query(`SELECT data FROM samples WHERE run_id = ? ORDER BY t`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Sample{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var smp Sample
		if err := json.Unmarshal([]byte(data), &smp); err != nil {
			return nil, err
		}
		out = append(out, smp)
	}
	return out, rows.Err()
}

// MarkAbandoned flags runs left `running` by a crash or kill as aborted.
func (s *Store) MarkAbandoned() error {
	_, err := s.db.Exec(`UPDATE runs SET status = ? WHERE status = ?`, StatusAborted, StatusRunning)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRun(r rowScanner) (Run, error) {
	var (
		run     Run
		started string
		cfg     string
		summary sql.NullString
	)
	if err := r.Scan(&run.ID, &started, &run.DurationMs, &run.Status, &cfg, &summary); err != nil {
		return Run{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, started)
	if err != nil {
		return Run{}, err
	}
	run.StartedAt = t
	run.Config = json.RawMessage(cfg)
	if summary.Valid {
		run.Summary = json.RawMessage(summary.String)
	}
	return run, nil
}
