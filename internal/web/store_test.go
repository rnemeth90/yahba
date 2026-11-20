package web

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStoreRunLifecycle(t *testing.T) {
	s := newTestStore(t)
	started := time.Now().UTC().Truncate(time.Millisecond)

	id, err := s.CreateRun(json.RawMessage(`{"url":"http://x"}`), started)
	require.NoError(t, err)

	run, err := s.GetRun(id)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, run.Status)
	assert.True(t, started.Equal(run.StartedAt))
	assert.JSONEq(t, `{"url":"http://x"}`, string(run.Config))
	assert.Empty(t, run.Summary)

	require.NoError(t, s.AddSample(id, Sample{T: 1, Count: 5}))
	require.NoError(t, s.AddSample(id, Sample{T: 0, Count: 3}))
	require.NoError(t, s.FinishRun(id, StatusCompleted, 1500, json.RawMessage(`{"total_requests":8}`)))

	run, err = s.GetRun(id)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, run.Status)
	assert.Equal(t, int64(1500), run.DurationMs)
	assert.JSONEq(t, `{"total_requests":8}`, string(run.Summary))

	samples, err := s.Samples(id)
	require.NoError(t, err)
	require.Len(t, samples, 2)
	assert.Equal(t, 0, samples[0].T)
	assert.Equal(t, 5, samples[1].Count)
}

func TestStoreNotFoundAndEmptyLists(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetRun(42)
	assert.ErrorIs(t, err, ErrNotFound)

	runs, err := s.ListRuns()
	require.NoError(t, err)
	assert.NotNil(t, runs)
	assert.Empty(t, runs)

	samples, err := s.Samples(42)
	require.NoError(t, err)
	assert.NotNil(t, samples)
}

func TestStoreListNewestFirstAndMarkAbandoned(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.CreateRun(json.RawMessage(`{}`), time.Now())
	b, _ := s.CreateRun(json.RawMessage(`{}`), time.Now())
	require.NoError(t, s.FinishRun(a, StatusCompleted, 1, nil))

	require.NoError(t, s.MarkAbandoned())

	runs, err := s.ListRuns()
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Equal(t, b, runs[0].ID)
	assert.Equal(t, StatusAborted, runs[0].Status)
	assert.Equal(t, StatusCompleted, runs[1].Status)
}
