package web

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAPI(t *testing.T) (*httptest.Server, *Runner, *Store) {
	t.Helper()
	runner, store := newTestRunner(t)
	api := httptest.NewServer(NewServer(store, runner))
	t.Cleanup(api.Close)
	return api, runner, store
}

func postRun(t *testing.T, api *httptest.Server, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, api.URL+"/api/runs", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestServerStartRunValidation(t *testing.T) {
	api, _, _ := newTestAPI(t)

	resp := postRun(t, api, `{"url":"","requests":1,"rps":1,"workers":1,"timeout":1}`, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "url", body["field"])
	assert.NotEmpty(t, body["error"])
}

func TestServerRejectsUnknownFieldsAndBadContentType(t *testing.T) {
	api, _, _ := newTestAPI(t)

	resp := postRun(t, api, `{"nope":1}`, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = postRun(t, api, `{}`, map[string]string{"Content-Type": "text/plain"})
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestServerRejectsCrossOriginMutations(t *testing.T) {
	api, _, _ := newTestAPI(t)

	resp := postRun(t, api, `{}`, map[string]string{"Origin": "http://evil.example"})
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	req, _ := http.NewRequest(http.MethodDelete, api.URL+"/api/runs/1", nil)
	req.Header.Set("Origin", "http://evil.example")
	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)
}

func TestServerRunFlowAndEvents(t *testing.T) {
	api, _, _ := newTestAPI(t)
	target := targetServer(t)

	resp := postRun(t, api, `{"url":"`+target.URL+`","requests":5,"rps":5,"workers":2,"timeout":5}`, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct{ ID int64 }
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))

	// a second start while active is a conflict
	resp = postRun(t, api, `{"url":"`+target.URL+`","requests":1,"rps":1,"workers":1,"timeout":5}`, nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	client := &http.Client{Timeout: 20 * time.Second}
	ev, err := client.Get(api.URL + "/api/runs/" + itoa(created.ID) + "/events")
	require.NoError(t, err)
	defer ev.Body.Close()
	assert.Equal(t, "text/event-stream", ev.Header.Get("Content-Type"))

	var events []string
	sc := bufio.NewScanner(ev.Body)
	for sc.Scan() {
		if name, ok := strings.CutPrefix(sc.Text(), "event: "); ok {
			events = append(events, name)
			if name == "done" {
				break
			}
		}
	}
	require.NotEmpty(t, events)
	assert.Equal(t, "done", events[len(events)-1])
	assert.Contains(t, events, "sample")

	// finished run is readable via the API
	var detail RunDetail
	getJSON(t, api.URL+"/api/runs/"+itoa(created.ID), &detail)
	assert.Equal(t, StatusCompleted, detail.Run.Status)
	assert.NotEmpty(t, detail.Samples)

	var runs []Run
	getJSON(t, api.URL+"/api/runs", &runs)
	require.Len(t, runs, 1)

	// events for a finished run replay samples then done
	ev2, err := client.Get(api.URL + "/api/runs/" + itoa(created.ID) + "/events")
	require.NoError(t, err)
	defer ev2.Body.Close()
	all, err := io.ReadAll(ev2.Body)
	require.NoError(t, err)
	assert.Contains(t, string(all), "event: sample")
	assert.Contains(t, string(all), "event: done")
}

func TestServerNotFoundAndCompare(t *testing.T) {
	api, _, store := newTestAPI(t)

	for _, path := range []string{"/api/runs/999", "/api/runs/999/events"} {
		resp, err := http.Get(api.URL + path)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
	}

	resp, err := http.Get(api.URL + "/api/compare?a=x&b=2")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	a, _ := store.CreateRun(json.RawMessage(`{}`), time.Now())
	b, _ := store.CreateRun(json.RawMessage(`{}`), time.Now())
	var cmp struct{ A, B RunDetail }
	getJSON(t, api.URL+"/api/compare?a="+itoa(a)+"&b="+itoa(b), &cmp)
	assert.Equal(t, a, cmp.A.Run.ID)
	assert.Equal(t, b, cmp.B.Run.ID)
}

func TestServerServesIndex(t *testing.T) {
	api, _, _ := newTestAPI(t)
	resp, err := http.Get(api.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "YAHBA")
}

func TestServerRejectsForeignHostHeader(t *testing.T) {
	api, _, _ := newTestAPI(t)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req, err := http.NewRequest(method, api.URL+"/api/runs", strings.NewReader("{}"))
		require.NoError(t, err)
		// Simulate DNS rebinding: the attacker page's Origin matches the
		// spoofed Host it sent, but neither is a real hostname for this server.
		req.Host = "evil.example"
		req.Header.Set("Origin", "http://evil.example")
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, method)
		resp.Body.Close()
		assert.Equal(t, http.StatusMisdirectedRequest, resp.StatusCode, method)
	}
}

func TestServerAcceptsLoopbackHostHeaders(t *testing.T) {
	api, _, _ := newTestAPI(t)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(api.URL, "http://"))
	require.NoError(t, err)

	for _, host := range []string{"127.0.0.1:" + port, "localhost:" + port} {
		req, err := http.NewRequest(http.MethodGet, api.URL+"/", nil)
		require.NoError(t, err)
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, host)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, host)
	}
}

func TestServerWithAllowedHostsAcceptsExtraHost(t *testing.T) {
	runner, store := newTestRunner(t)
	api := httptest.NewServer(NewServer(store, runner, WithAllowedHosts("example.lan")))
	t.Cleanup(api.Close)

	req, err := http.NewRequest(http.MethodGet, api.URL+"/", nil)
	require.NoError(t, err)
	req.Host = "example.lan"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	req2, err := http.NewRequest(http.MethodGet, api.URL+"/", nil)
	require.NoError(t, err)
	req2.Host = "other.example"
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	resp2.Body.Close()
	assert.Equal(t, http.StatusMisdirectedRequest, resp2.StatusCode)
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
