package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

//go:embed static
var staticFS embed.FS

// RunDetail is a run plus its per-second samples.
type RunDetail struct {
	Run     Run      `json:"run"`
	Samples []Sample `json:"samples"`
}

type server struct {
	store        *Store
	runner       *Runner
	allowedHosts map[string]struct{}
}

// ServerOption customizes NewServer. It is primarily used to extend the
// Host-header allowlist beyond the loopback defaults.
type ServerOption func(*server)

// WithAllowedHosts adds extra hostnames (no port, case-insensitive) to the
// Host-header allowlist enforced on every request. The loopback names
// (localhost, 127.0.0.1, ::1) are always allowed regardless of this option.
func WithAllowedHosts(hosts ...string) ServerOption {
	return func(s *server) {
		for _, h := range hosts {
			h = strings.ToLower(strings.TrimSpace(h))
			if h == "" {
				continue
			}
			s.allowedHosts[h] = struct{}{}
		}
	}
}

func defaultAllowedHosts() map[string]struct{} {
	return map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"::1":       {},
	}
}

func NewServer(store *Store, runner *Runner, opts ...ServerOption) http.Handler {
	s := &server{store: store, runner: runner, allowedHosts: defaultAllowedHosts()}
	for _, opt := range opts {
		opt(s)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/runs", s.startRun)
	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.getRun)
	mux.HandleFunc("GET /api/runs/{id}/events", s.events)
	mux.HandleFunc("DELETE /api/runs/{id}", s.stopRun)
	mux.HandleFunc("GET /api/compare", s.compare)

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embedded path is fixed at compile time
	}
	mux.Handle("GET /", http.FileServerFS(static))
	return allowedHostOnly(sameOriginOnly(mux), s.allowedHosts)
}

// allowedHostOnly blocks DNS-rebinding attacks: it rejects every request
// (including GET) whose Host header does not match an allowed hostname,
// regardless of port. Without this, a page on an attacker-controlled domain
// could resolve to 127.0.0.1 and issue same-origin requests that this server
// would otherwise treat as trusted.
func allowedHostOnly(next http.Handler, allowed map[string]struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := allowed[hostnameOnly(r.Host)]; !ok {
			writeError(w, http.StatusMisdirectedRequest, "host not allowed", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostnameOnly strips the port (and IPv6 brackets) from a Host header value
// and lowercases the result for case-insensitive comparison.
func hostnameOnly(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else {
		host = strings.Trim(host, "[]")
	}
	return strings.ToLower(host)
}

// sameOriginOnly blocks cross-site requests from a browser: any mutating
// request carrying an Origin must match the Host it was sent to.
func sameOriginOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host {
					writeError(w, http.StatusForbidden, "cross-origin request blocked", "")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg, field string) {
	body := map[string]string{"error": msg}
	if field != "" {
		body["field"] = field
	}
	writeJSON(w, status, body)
}

func parseID(w http.ResponseWriter, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid run id", "")
		return 0, false
	}
	return id, true
}

func (s *server) startRun(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json", "")
		return
	}
	var req RunRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error(), "")
		return
	}

	id, err := s.runner.Start(req)
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Message, ve.Field)
	case errors.Is(err, ErrBusy):
		writeError(w, http.StatusConflict, err.Error(), "")
	case errors.Is(err, ErrShuttingDown):
		writeError(w, http.StatusServiceUnavailable, err.Error(), "")
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error(), "")
	default:
		writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func (s *server) listRuns(w http.ResponseWriter, _ *http.Request) {
	runs, err := s.store.ListRuns()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *server) detail(id int64) (RunDetail, error) {
	run, err := s.store.GetRun(id)
	if err != nil {
		return RunDetail{}, err
	}
	samples, err := s.store.Samples(id)
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{Run: run, Samples: samples}, nil
}

func (s *server) writeDetailError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error(), "")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error(), "")
}

func (s *server) getRun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	d, err := s.detail(id)
	if err != nil {
		s.writeDetailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *server) stopRun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !s.runner.Stop(id) {
		writeError(w, http.StatusNotFound, "no active run with that id", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) compare(w http.ResponseWriter, r *http.Request) {
	a, okA := parseID(w, r.URL.Query().Get("a"))
	if !okA {
		return
	}
	b, okB := parseID(w, r.URL.Query().Get("b"))
	if !okB {
		return
	}
	da, err := s.detail(a)
	if err != nil {
		s.writeDetailError(w, err)
		return
	}
	db, err := s.detail(b)
	if err != nil {
		s.writeDetailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]RunDetail{"a": da, "b": db})
}

func sendEvent(w http.ResponseWriter, f http.Flusher, event string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported", "")
		return
	}

	past, ch, unsubscribe, active := s.runner.Subscribe(id)
	if !active {
		d, err := s.detail(id)
		if err != nil {
			s.writeDetailError(w, err)
			return
		}
		startStream(w)
		for _, smp := range d.Samples {
			sendEvent(w, flusher, "sample", smp)
		}
		sendEvent(w, flusher, "done", d.Run)
		return
	}
	defer unsubscribe()

	startStream(w)
	for _, smp := range past {
		sendEvent(w, flusher, "sample", smp)
	}
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case smp, open := <-ch:
			if !open {
				if run, err := s.store.GetRun(id); err == nil {
					sendEvent(w, flusher, "done", run)
				}
				return
			}
			sendEvent(w, flusher, "sample", smp)
		}
	}
}

func startStream(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}
