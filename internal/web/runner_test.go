package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRunner(t *testing.T) (*Runner, *Store) {
	t.Helper()
	store := newTestStore(t)
	r := NewRunner(store)
	t.Cleanup(r.Shutdown)
	return r, store
}

func targetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func waitForStatus(t *testing.T, store *Store, id int64, want RunStatus) Run {
	t.Helper()
	var run Run
	require.Eventually(t, func() bool {
		var err error
		run, err = store.GetRun(id)
		return err == nil && run.Status == want
	}, 15*time.Second, 50*time.Millisecond)
	return run
}

func TestRunnerCompletesAndPersists(t *testing.T) {
	r, store := newTestRunner(t)
	target := targetServer(t)

	id, err := r.Start(RunRequest{URL: target.URL, Requests: 5, RPS: 50, Workers: 2, Timeout: 5})
	require.NoError(t, err)

	run := waitForStatus(t, store, id, StatusCompleted)

	var sum Summary
	require.NoError(t, json.Unmarshal(run.Summary, &sum))
	assert.Equal(t, 5, sum.TotalRequests)
	assert.Equal(t, 5, sum.Successes)

	samples, err := store.Samples(id)
	require.NoError(t, err)
	require.NotEmpty(t, samples)
	total := 0
	for _, s := range samples {
		total += s.Count
	}
	assert.Equal(t, 5, total)
}

func TestRunnerRejectsSecondRunAndCanStop(t *testing.T) {
	r, store := newTestRunner(t)
	target := targetServer(t)

	id, err := r.Start(RunRequest{URL: target.URL, Requests: 1000, RPS: 5, Workers: 2, Timeout: 5})
	require.NoError(t, err)

	_, err = r.Start(RunRequest{URL: target.URL, Requests: 1, RPS: 1, Workers: 1, Timeout: 5})
	assert.True(t, errors.Is(err, ErrBusy))

	assert.False(t, r.Stop(id+1))
	assert.True(t, r.Stop(id))
	waitForStatus(t, store, id, StatusStopped)

	// the slot is free again
	id2, err := r.Start(RunRequest{URL: target.URL, Requests: 1, RPS: 10, Workers: 1, Timeout: 5})
	require.NoError(t, err)
	waitForStatus(t, store, id2, StatusCompleted)
}

func TestRunnerInvalidRequest(t *testing.T) {
	r, _ := newTestRunner(t)
	_, err := r.Start(RunRequest{})
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestRunnerRefusesStartAfterShutdown(t *testing.T) {
	r, _ := newTestRunner(t)
	r.Shutdown()

	_, err := r.Start(validRequest())
	assert.True(t, errors.Is(err, ErrShuttingDown), "got %v", err)
}

func TestRunnerSubscribeReceivesSamplesAndClose(t *testing.T) {
	r, store := newTestRunner(t)
	target := targetServer(t)

	id, err := r.Start(RunRequest{URL: target.URL, Requests: 5, RPS: 5, Workers: 2, Timeout: 5})
	require.NoError(t, err)

	_, ch, unsub, ok := r.Subscribe(id)
	require.True(t, ok)
	defer unsub()

	timeout := time.After(15 * time.Second)
	for open := true; open; {
		select {
		case _, open = <-ch:
		case <-timeout:
			t.Fatal("subscriber channel was not closed when the run finished")
		}
	}
	run, err := store.GetRun(id)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, run.Status)

	_, _, _, ok = r.Subscribe(id)
	assert.False(t, ok)
}
