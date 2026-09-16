package tui

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/queue"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
)

// appState represents which screen the TUI is showing.
type appState int

const (
	stateForm       appState = iota // configuration form
	stateFilepicker                 // browsing for a definition file
	stateRunning                    // test in progress
	stateReport                     // final report
)

// latencyBufferSize bounds how many recent per-request latencies are kept
// for the graph. The graph only ever renders the last graphWidth (<=80)
// columns anyway, so this cap is generous headroom, not a display limit.
const latencyBufferSize = 512

// Form input indices
const (
	inputURL = iota
	inputMethod
	inputRequests
	inputRPS
	inputKeepAlives
	inputWorkers
	inputTimeout
	inputHeaders
	inputBody
	inputCount // sentinel: total number of inputs
)

// inputLabels maps input indices to their display labels.
var inputLabels = [inputCount]string{
	"URL",
	"Method",
	"Requests",
	"RPS",
	"Keep-Alives",
	"Workers",
	"Timeout (s)",
	"Headers",
	"Body",
}

// --- Messages ---

// resultMsg is sent when a single request result arrives from the worker pool.
type resultMsg report.Result

// reportMsg is sent when the final aggregated report is ready.
type reportMsg report.Report

// resultsCompleteMsg signals that all results have been received.
type resultsCompleteMsg struct{}

// tickMsg drives the elapsed-time display during a running test.
type tickMsg time.Time

// errMsg wraps errors surfaced as Bubble Tea messages.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Commands ---

// waitForResult listens on the progress channel for the next individual result.
func waitForResult(ch <-chan report.Result) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-ch
		if !ok {
			return resultsCompleteMsg{}
		}
		return resultMsg(r)
	}
}

// waitForReport listens on the report channel for the final aggregated report.
func waitForReport(ch <-chan report.Report) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-ch
		if !ok {
			return nil
		}
		return reportMsg(r)
	}
}

// tick returns a command that fires a tickMsg every 100ms.
func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- Model ---
type model struct {
	state  appState
	width  int
	height int

	// Form
	inputs   []textinput.Model
	focusIdx int
	formErr  string

	// Running: live stats
	cfg           config.Config
	runLabel      string // human-readable description of what's running, shown on the running screen
	totalRequests int
	completed     int
	successes     int
	failures      int
	statusCodes   map[int]int
	startTime     time.Time
	elapsed       time.Duration
	minLatency    time.Duration
	maxLatency    time.Duration
	totalLatency  time.Duration
	latencies     *queue.CircularBuffer // recent per-request latencies for the graph; bounded, oldest evicted first
	bytesSent     int
	bytesReceived int

	// Channels & cancellation
	progressChan <-chan report.Result
	reportChan   <-chan report.Report
	cancelFunc   context.CancelFunc

	// Final report
	report *report.Report

	// UI components
	progress progress.Model
	spinner  spinner.Model

	// filepicker elements, used to browse for a test definition file
	filepicker   filepicker.Model
	selectedFile string
	err          error
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

// New creates the initial TUI model in form state.
func New() model {
	inputs := make([]textinput.Model, inputCount)

	for i := range inputs {
		t := textinput.New()
		t.CharLimit = 256
		t.Width = 50
		inputs[i] = t
	}

	inputs[inputURL].Placeholder = "https://example.com"
	inputs[inputURL].Focus()

	inputs[inputMethod].Placeholder = "GET"
	inputs[inputMethod].SetValue("GET")
	inputs[inputMethod].CharLimit = 10

	inputs[inputRequests].Placeholder = "100"
	inputs[inputRequests].SetValue("100")
	inputs[inputRequests].CharLimit = 10

	inputs[inputRPS].Placeholder = "10"
	inputs[inputRPS].SetValue("10")
	inputs[inputRPS].CharLimit = 10

	inputs[inputKeepAlives].Placeholder = "true"
	inputs[inputKeepAlives].SetValue("false")
	inputs[inputKeepAlives].CharLimit = 5

	inputs[inputWorkers].Placeholder = "10"
	inputs[inputWorkers].SetValue("10")
	inputs[inputWorkers].CharLimit = 10

	inputs[inputTimeout].Placeholder = "10"
	inputs[inputTimeout].SetValue("10")
	inputs[inputTimeout].CharLimit = 10

	inputs[inputHeaders].Placeholder = "Key:Value,Key:Value"
	inputs[inputBody].Placeholder = `{"key":"value"}`

	p := progress.New(
		progress.WithScaledGradient(string(progressFullColor), string(progressEmptyColor)),
		progress.WithoutPercentage(),
	)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	fp := filepicker.New()
	fp.AllowedTypes = []string{".yaml", ".yml"}
	if cwd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = cwd
	}

	return model{
		state:      stateForm,
		inputs:     inputs,
		focusIdx:   0,
		progress:   p,
		spinner:    s,
		filepicker: fp,
		latencies:  queue.NewCircularBuffer(latencyBufferSize),
	}
}

// Init is the first Bubble Tea lifecycle method.
func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.WindowSize(), m.filepicker.Init())
}

// buildSharedConfig parses the form fields common to both ad-hoc and
// definition-file runs (worker pool size, timeout, keep-alives) into a
// config.Config template.
func (m *model) buildSharedConfig() (config.Config, error) {
	keepAlives, err := strconv.ParseBool(m.inputs[inputKeepAlives].Value())
	if err != nil {
		return config.Config{}, fmt.Errorf("keep-alives must be true or false")
	}

	workers, err := strconv.Atoi(m.inputs[inputWorkers].Value())
	if err != nil || workers <= 0 {
		return config.Config{}, fmt.Errorf("workers must be a positive integer")
	}

	timeout, err := strconv.Atoi(m.inputs[inputTimeout].Value())
	if err != nil || timeout <= 0 {
		return config.Config{}, fmt.Errorf("timeout must be a positive integer")
	}

	return config.Config{
		KeepAlive:    keepAlives,
		Workers:      workers,
		Timeout:      timeout,
		LogLevel:     "error",
		OutputFormat: "raw",
		OutputFile:   "stdout",
		Silent:       true,
		Logger:       logger.New("error", "stdout", true),
	}, nil
}

// buildConfig parses the form inputs into a config.Config and validates it.
func (m *model) buildConfig() (config.Config, error) {
	cfg, err := m.buildSharedConfig()
	if err != nil {
		return config.Config{}, err
	}

	method := m.inputs[inputMethod].Value()
	if method == "" {
		method = "GET"
	}

	requests, err := strconv.Atoi(m.inputs[inputRequests].Value())
	if err != nil || requests <= 0 {
		return config.Config{}, fmt.Errorf("requests must be a positive integer")
	}

	rps, err := strconv.Atoi(m.inputs[inputRPS].Value())
	if err != nil || rps <= 0 {
		return config.Config{}, fmt.Errorf("RPS must be a positive integer")
	}

	cfg.URL = m.inputs[inputURL].Value()
	cfg.Method = method
	cfg.Requests = requests
	cfg.RPS = rps
	cfg.Headers = m.inputs[inputHeaders].Value()
	cfg.Body = m.inputs[inputBody].Value()

	if cfg.Headers != "" {
		parsedHeaders, err := util.ParseHeaders(cfg.Headers)
		if err != nil {
			return config.Config{}, fmt.Errorf("invalid headers: %w", err)
		}
		cfg.ParsedHeaders = parsedHeaders
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

// workerFactory returns a WorkerFactory that constructs plain worker.Worker instances.
func workerFactory() worker.WorkerFactory {
	return func(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, c config.Config) worker.Worker {
		return *worker.NewWorker(id, jobChan, resultChan, client, c)
	}
}

// resetRunState clears live-run stats ahead of dispatching a test (ad-hoc or
// definition-file) and returns a fresh cancellable context for it.
func (m *model) resetRunState(total int) context.Context {
	m.totalRequests = total
	m.completed = 0
	m.successes = 0
	m.failures = 0
	m.statusCodes = make(map[int]int)
	m.startTime = time.Now()
	m.elapsed = 0
	m.minLatency = 0
	m.maxLatency = 0
	m.totalLatency = 0
	m.bytesSent = 0
	m.bytesReceived = 0
	m.latencies = queue.NewCircularBuffer(latencyBufferSize)
	m.report = nil
	m.formErr = ""

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	return ctx
}

// launchTest wires up progress/report channels for a background run
// function, transitions the model into the running state, and returns the
// commands needed to start listening for updates.
func (m *model) launchTest(total int, run func(progressChan chan<- report.Result, reportChan chan<- report.Report)) tea.Cmd {
	progressChan := make(chan report.Result, total)
	reportChan := make(chan report.Report, 1)
	m.progressChan = progressChan
	m.reportChan = reportChan

	go run(progressChan, reportChan)

	m.state = stateRunning

	return tea.Batch(
		waitForResult(progressChan),
		waitForReport(reportChan),
		tick(),
		m.spinner.Tick,
	)
}

// startTest kicks off an ad-hoc load test built from the form inputs.
func (m *model) startTest() tea.Cmd {
	cfg, err := m.buildConfig()
	if err != nil {
		m.formErr = err.Error()
		return nil
	}

	ctx := m.resetRunState(cfg.Requests)
	m.cfg = cfg
	m.runLabel = fmt.Sprintf("%s %s", cfg.Method, cfg.URL)

	jobs := make([]worker.Job, cfg.Requests)
	for i := 0; i < cfg.Requests; i++ {
		jobs[i] = worker.Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body}
	}

	factory := workerFactory()

	return m.launchTest(cfg.Requests, func(progressChan chan<- report.Result, reportChan chan<- report.Report) {
		worker.Work(ctx, cfg, jobs, reportChan, factory, progressChan)
	})
}

// startDefFileTest validates and parses a test definition file, then runs
// all of its named requests (in dependency order) as a single test,
// aggregating their results into one combined report.
func (m *model) startDefFileTest(path string) tea.Cmd {
	baseCfg, err := m.buildSharedConfig()
	if err != nil {
		m.formErr = err.Error()
		return nil
	}

	if err := config.ValidateDefFile(path); err != nil {
		m.formErr = fmt.Sprintf("invalid definition file: %v", err)
		return nil
	}

	def, err := config.ParseDefFile(path)
	if err != nil {
		m.formErr = fmt.Sprintf("error parsing definition file: %v", err)
		return nil
	}

	total := 0
	for _, r := range def.Requests {
		total += r.Requests
	}

	ctx := m.resetRunState(total)
	m.cfg = baseCfg
	m.runLabel = fmt.Sprintf("definition file: %s (%d requests)", filepath.Base(path), len(def.Requests))

	factory := workerFactory()

	return m.launchTest(total, func(progressChan chan<- report.Result, reportChan chan<- report.Report) {
		runDefFileTest(ctx, baseCfg, def, factory, progressChan, reportChan)
	})
}

// resetToForm resets the model to the form state, preserving input values.
func (m *model) resetToForm() {
	m.state = stateForm
	m.formErr = ""
	m.report = nil
	m.completed = 0
	m.successes = 0
	m.failures = 0
	m.statusCodes = nil
	m.latencies = queue.NewCircularBuffer(latencyBufferSize)
	m.progressChan = nil
	m.reportChan = nil
	m.cancelFunc = nil
	m.selectedFile = ""
	m.err = nil

	// re-focus first input
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focusIdx = 0
	m.inputs[0].Focus()
}

// latencySnapshot returns the currently buffered latencies, oldest first, as
// a plain slice for rendering.
func (m model) latencySnapshot() []time.Duration {
	items := m.latencies.Items()
	out := make([]time.Duration, len(items))
	for i, v := range items {
		out[i] = v.(time.Duration)
	}
	return out
}
