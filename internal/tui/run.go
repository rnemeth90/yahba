package tui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
)

// currentMode returns the mode typed in the form, defaulting to requests.
func (m *model) currentMode() string {
	v := strings.ToLower(strings.TrimSpace(m.inputs[inputMode].Value()))
	if v == "" {
		return config.ModeRequests
	}
	return v
}

// visible reports whether a form input applies to the selected mode.
func (m *model) visible(i int) bool {
	mode := m.currentMode()
	switch i {
	case inputRequests, inputRPS:
		return mode != config.ModeMax && mode != config.ModeRamp
	case inputDuration:
		return mode == config.ModeMax
	case inputRampStart, inputRampStep, inputRampMax, inputStepDuration, inputMaxErrorRate, inputMaxP99:
		return mode == config.ModeRamp
	}
	return true
}

// moveFocus moves focus to the next (dir=1) or previous (dir=-1) visible input.
func (m *model) moveFocus(dir int) {
	for n := 0; n < inputCount; n++ {
		m.focusIdx = (m.focusIdx + dir + inputCount) % inputCount
		if m.visible(m.focusIdx) {
			break
		}
	}
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.inputs[m.focusIdx].Focus()
}

// cycleMode switches the mode field to the next or previous mode.
func (m *model) cycleMode(dir int) {
	cur := 0
	for i, name := range modes {
		if name == m.currentMode() {
			cur = i
		}
	}
	m.inputs[inputMode].SetValue(modes[(cur+dir+len(modes))%len(modes)])
}

func (m *model) positiveInt(idx int, name string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(m.inputs[idx].Value()))
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return v, nil
}

func (m *model) duration(idx int, name string, allowZero bool) (time.Duration, error) {
	raw := strings.TrimSpace(m.inputs[idx].Value())
	if allowZero && (raw == "" || raw == "0") {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a duration such as 10s or 500ms", name)
	}
	return d, nil
}

// buildConfig parses the form inputs into a config.Config and validates it.
func (m *model) buildConfig() (config.Config, error) {
	method := m.inputs[inputMethod].Value()
	if method == "" {
		method = "GET"
	}

	keepAlives, err := strconv.ParseBool(m.inputs[inputKeepAlives].Value())
	if err != nil {
		return config.Config{}, fmt.Errorf("keep-alives must be true or false")
	}

	workers, err := m.positiveInt(inputWorkers, "workers")
	if err != nil {
		return config.Config{}, err
	}

	timeout, err := m.positiveInt(inputTimeout, "timeout")
	if err != nil {
		return config.Config{}, err
	}

	cfg := config.Config{
		URL:          m.inputs[inputURL].Value(),
		Method:       method,
		Mode:         m.currentMode(),
		KeepAlive:    keepAlives,
		Workers:      workers,
		Timeout:      timeout,
		Headers:      m.inputs[inputHeaders].Value(),
		Body:         m.inputs[inputBody].Value(),
		LogLevel:     "error",
		OutputFormat: "raw",
		OutputFile:   "stdout",
		Silent:       true,
		Logger:       logger.New("error", "stdout", true),
	}

	switch cfg.Mode {
	case config.ModeRequests:
		if cfg.Requests, err = m.positiveInt(inputRequests, "requests"); err != nil {
			return config.Config{}, err
		}
		if cfg.RPS, err = m.positiveInt(inputRPS, "RPS"); err != nil {
			return config.Config{}, err
		}
	case config.ModeMax:
		if cfg.Duration, err = m.duration(inputDuration, "duration", false); err != nil {
			return config.Config{}, err
		}
	case config.ModeRamp:
		if err := m.buildRamp(&cfg); err != nil {
			return config.Config{}, err
		}
	}

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

func (m *model) buildRamp(cfg *config.Config) error {
	var err error
	if cfg.RampStart, err = m.positiveInt(inputRampStart, "RPS start"); err != nil {
		return err
	}
	if cfg.RampStep, err = m.positiveInt(inputRampStep, "RPS step"); err != nil {
		return err
	}
	if v := strings.TrimSpace(m.inputs[inputRampMax].Value()); v != "" && v != "0" {
		if cfg.RampMax, err = m.positiveInt(inputRampMax, "RPS max"); err != nil {
			return err
		}
	}
	if cfg.StepDuration, err = m.duration(inputStepDuration, "step duration", false); err != nil {
		return err
	}
	if cfg.MaxErrorRate, err = strconv.ParseFloat(strings.TrimSpace(m.inputs[inputMaxErrorRate].Value()), 64); err != nil {
		return fmt.Errorf("max error rate must be a number")
	}
	cfg.MaxP99, err = m.duration(inputMaxP99, "max p99", true)
	return err
}

func workerFactory(id int, jobChan <-chan worker.Job, resultChan chan<- report.Result, client *http.Client, c config.Config) worker.Worker {
	return *worker.NewWorker(id, jobChan, resultChan, client, c)
}

// startTest kicks off the load test and returns initial commands.
func (m *model) startTest() tea.Cmd {
	cfg, err := m.buildConfig()
	if err != nil {
		m.formErr = err.Error()
		return nil
	}

	m.resetRun()
	m.cfg = cfg
	m.mode = cfg.EffectiveMode()
	m.totalRequests = cfg.Requests
	m.startTime = time.Now()
	m.stepStart = m.startTime

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	cmds := []tea.Cmd{tick(), m.spinner.Tick}
	if m.mode == config.ModeRamp {
		cmds = append(cmds, m.startRamp(ctx, cfg))
	} else {
		cmds = append(cmds, m.startPooled(ctx, cfg))
	}

	m.state = stateRunning
	return tea.Batch(cmds...)
}

func (m *model) startRamp(ctx context.Context, cfg config.Config) tea.Cmd {
	m.ramp = &rampLive{}
	done := make(chan rampDoneMsg, 1)
	m.rampChan = done
	live := m.ramp
	go func() {
		rr, err := worker.RunRamp(ctx, cfg, workerFactory, live.add)
		done <- rampDoneMsg{report: rr, err: err}
	}()
	return waitForRamp(done)
}

// startPooled runs the requests or max mode, draining results into liveStats
// so the load generator is never slowed by the UI.
func (m *model) startPooled(ctx context.Context, cfg config.Config) tea.Cmd {
	m.stats = newLiveStats()
	progressChan := make(chan report.Result, 1024)
	reportChan := make(chan report.Report, 1)
	m.reportChan = reportChan

	drained := make(chan struct{})
	m.drained = drained
	stats := m.stats
	go func() {
		defer close(drained)
		for r := range progressChan {
			stats.add(r)
		}
	}()

	if m.mode == config.ModeMax {
		go worker.RunMax(ctx, cfg, reportChan, workerFactory, progressChan)
	} else {
		jobs := make([]worker.Job, cfg.Requests)
		for i := range jobs {
			jobs[i] = worker.Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body}
		}
		go worker.Work(ctx, cfg, jobs, reportChan, workerFactory, progressChan)
	}
	return waitForReport(reportChan)
}

// sample copies the live counters into the model fields the views render.
func (m *model) sample() {
	if m.stats != nil {
		s := m.stats.snapshot()
		m.completed, m.successes, m.failures = s.completed, s.successes, s.failures
		m.bytesSent, m.bytesReceived = s.bytesSent, s.bytesReceived
		m.statusCodes = s.statusCodes
		m.minLatency, m.maxLatency, m.totalLatency = s.minLatency, s.maxLatency, s.totalLatency
		m.latencies = s.latencies
	}
	if m.ramp != nil {
		steps := m.ramp.snapshot()
		if len(steps) != len(m.rampSteps) {
			m.stepStart = time.Now()
		}
		m.rampSteps = steps
	}
}

// progressFraction is how far through the test we are, by mode.
func (m model) progressFraction() float64 {
	var f float64
	switch m.mode {
	case config.ModeMax:
		f = float64(m.elapsed) / float64(m.cfg.Duration)
	case config.ModeRamp:
		f = float64(time.Since(m.stepStart)) / float64(m.cfg.StepDuration)
	default:
		if m.totalRequests > 0 {
			f = float64(m.completed) / float64(m.totalRequests)
		}
	}
	return min(max(f, 0), 1)
}

// resetRun clears all per-run state.
func (m *model) resetRun() {
	m.formErr = ""
	m.report = nil
	m.rampReport = nil
	m.rampSteps = nil
	m.completed, m.successes, m.failures = 0, 0, 0
	m.bytesSent, m.bytesReceived = 0, 0
	m.statusCodes = make(map[int]int)
	m.elapsed = 0
	m.minLatency, m.maxLatency, m.totalLatency = 0, 0, 0
	m.latencies = nil
	m.stats = nil
	m.ramp = nil
	m.drained = nil
	m.reportChan = nil
	m.rampChan = nil
	m.cancelFunc = nil
}

// resetToForm resets the model to the form state, preserving input values.
func (m *model) resetToForm() {
	m.state = stateForm
	m.resetRun()

	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focusIdx = 0
	m.inputs[0].Focus()
}
