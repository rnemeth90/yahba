package web

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/worker"
)

var ErrBusy = errors.New("a run is already active")

// ErrShuttingDown is returned by Start once Shutdown has begun; the server is
// going away and must not accept new work.
var ErrShuttingDown = errors.New("server is shutting down")

type activeRun struct {
	id      int64
	cancel  context.CancelFunc
	stopped atomic.Bool
	samples []Sample
	subs    map[chan Sample]struct{}
	done    chan struct{}
}

// Runner executes at most one load test at a time, persisting its samples and
// final summary and fanning live samples out to subscribers.
type Runner struct {
	store        *Store
	mu           sync.Mutex
	active       *activeRun
	shuttingDown atomic.Bool
}

func NewRunner(store *Store) *Runner { return &Runner{store: store} }

func (r *Runner) Start(req RunRequest) (int64, error) {
	if r.shuttingDown.Load() {
		return 0, ErrShuttingDown
	}
	cfg, err := req.ToConfig()
	if err != nil {
		return 0, err
	}
	cfgJSON, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shuttingDown.Load() {
		return 0, ErrShuttingDown
	}
	if r.active != nil {
		return 0, ErrBusy
	}
	started := time.Now()
	id, err := r.store.CreateRun(cfgJSON, started)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &activeRun{id: id, cancel: cancel, subs: map[chan Sample]struct{}{}, done: make(chan struct{})}
	r.active = a
	go r.execute(ctx, a, cfg, started)
	return id, nil
}

func (r *Runner) execute(ctx context.Context, a *activeRun, cfg config.Config, started time.Time) {
	defer close(a.done)
	defer a.cancel()

	jobs := make([]worker.Job, cfg.Requests)
	for i := range jobs {
		jobs[i] = worker.Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body}
	}
	factory := func(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, c config.Config) worker.Worker {
		return *worker.NewWorker(id, jobChan, resultChan, client, c)
	}

	progress := make(chan report.Result, cfg.Requests)
	reportChan := make(chan report.Report, 1)
	workDone := make(chan struct{})
	go func() {
		defer close(workDone)
		worker.Work(ctx, cfg, jobs, reportChan, factory, progress)
	}()

	sampler := NewSampler(started)
	handle := func(res report.Result) {
		for _, s := range sampler.Add(res) {
			r.publish(a, s)
		}
	}

	// worker.Work returns without closing its channels if the HTTP client cannot
	// be built, so also watch workDone and drain whatever is buffered.
loop:
	for {
		select {
		case res, ok := <-progress:
			if !ok {
				break loop
			}
			handle(res)
		case <-workDone:
			for {
				select {
				case res, ok := <-progress:
					if !ok {
						break loop
					}
					handle(res)
				default:
					break loop
				}
			}
		}
	}
	for _, s := range sampler.Flush() {
		r.publish(a, s)
	}

	// progressChan is closed before worker.Work writes reportChan, so wait for
	// Work to fully return (workDone) before reading reportChan to avoid racing
	// its write. If Work returned early (e.g. client creation failed), workDone
	// still closes but reportChan is never written, and the non-blocking read
	// below correctly falls through to default.
	<-workDone
	status := StatusFailed
	var summary json.RawMessage
	select {
	case rep, ok := <-reportChan:
		if ok {
			status = StatusCompleted
			if a.stopped.Load() {
				status = StatusStopped
			}
			summary, _ = json.Marshal(Summarize(rep))
		}
	default:
	}

	if err := r.store.FinishRun(a.id, status, time.Since(started).Milliseconds(), summary); err != nil {
		log.Printf("web: failed to finish run %d: %v", a.id, err)
	}

	r.mu.Lock()
	for ch := range a.subs {
		close(ch)
	}
	a.subs = nil
	r.active = nil
	r.mu.Unlock()
}

func (r *Runner) publish(a *activeRun, s Sample) {
	if err := r.store.AddSample(a.id, s); err != nil {
		log.Printf("web: failed to save sample for run %d: %v", a.id, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a.samples = append(a.samples, s)
	for ch := range a.subs {
		select {
		case ch <- s:
		default: // slow subscriber; it can reconnect and replay
		}
	}
}

// Stop cancels the active run if it has the given id.
func (r *Runner) Stop(id int64) bool {
	r.mu.Lock()
	a := r.active
	r.mu.Unlock()
	if a == nil || a.id != id {
		return false
	}
	a.stopped.Store(true)
	a.cancel()
	return true
}

// Subscribe returns the samples produced so far plus a channel of new ones. The
// channel is closed when the run finishes. ok is false if id is not active.
func (r *Runner) Subscribe(id int64) (past []Sample, ch <-chan Sample, unsubscribe func(), ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.active
	if a == nil || a.id != id {
		return nil, nil, nil, false
	}
	c := make(chan Sample, 64)
	a.subs[c] = struct{}{}
	past = append([]Sample(nil), a.samples...)
	unsubscribe = func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if _, found := a.subs[c]; found {
			delete(a.subs, c)
			close(c)
		}
	}
	return past, c, unsubscribe, true
}

// Shutdown stops the active run, if any, and waits for it to be persisted.
func (r *Runner) Shutdown() {
	r.shuttingDown.Store(true)
	r.mu.Lock()
	a := r.active
	r.mu.Unlock()
	if a == nil {
		return
	}
	a.stopped.Store(true)
	a.cancel()
	<-a.done
}
