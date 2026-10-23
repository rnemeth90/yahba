package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
)

// appState represents which screen the TUI is showing.
type appState int

const (
	stateForm    appState = iota // configuration form
	stateRunning                 // test in progress
	stateReport                  // final report
)

// Form input indices, in display order
const (
	inputURL = iota
	inputMethod
	inputMode
	inputRequests
	inputRPS
	inputDuration
	inputRampStart
	inputRampStep
	inputRampMax
	inputStepDuration
	inputMaxErrorRate
	inputMaxP99
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
	"Mode",
	"Requests",
	"RPS",
	"Duration",
	"RPS Start",
	"RPS Step",
	"RPS Max (0=none)",
	"Step Duration",
	"Max Error Rate %",
	"Max P99 (0=off)",
	"Keep-Alives",
	"Workers",
	"Timeout (s)",
	"Headers",
	"Body",
}

var modes = []string{config.ModeRequests, config.ModeMax, config.ModeRamp}

// --- Messages ---

// reportMsg is sent when the final aggregated report is ready.
type reportMsg report.Report

// rampDoneMsg is sent when a ramp test finishes.
type rampDoneMsg struct {
	report report.RampReport
	err    error
}

// tickMsg drives the elapsed-time display during a running test.
type tickMsg time.Time

// errMsg wraps errors surfaced as Bubble Tea messages.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Commands ---

// waitForRamp listens for the finished ramp report.
func waitForRamp(ch <-chan rampDoneMsg) tea.Cmd {
	return func() tea.Msg { return <-ch }
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

	// Running: live stats, sampled from stats on each tick
	cfg           config.Config
	mode          string
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
	latencies     []time.Duration // most recent per-request latencies for the graph
	bytesSent     int
	bytesReceived int

	stats     *liveStats
	drained   chan struct{}
	ramp      *rampLive
	rampSteps []report.Step
	stepStart time.Time

	// Channels & cancellation
	reportChan <-chan report.Report
	rampChan   <-chan rampDoneMsg
	cancelFunc context.CancelFunc

	// Final report (one of the two is set)
	report     *report.Report
	rampReport *report.RampReport

	// UI components
	progress progress.Model
	spinner  spinner.Model
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

	inputs[inputMode].Placeholder = "requests | max | ramp"
	inputs[inputMode].SetValue(config.ModeRequests)
	inputs[inputMode].CharLimit = 10

	inputs[inputDuration].Placeholder = "30s"
	inputs[inputDuration].SetValue("30s")
	inputs[inputDuration].CharLimit = 10

	inputs[inputRampStart].SetValue("100")
	inputs[inputRampStep].SetValue("100")
	inputs[inputRampMax].SetValue("0")
	inputs[inputStepDuration].SetValue("10s")
	inputs[inputMaxErrorRate].SetValue("5")
	inputs[inputMaxP99].SetValue("0")
	for _, i := range []int{inputRampStart, inputRampStep, inputRampMax, inputStepDuration, inputMaxErrorRate, inputMaxP99} {
		inputs[i].CharLimit = 10
	}

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

	return model{
		state:    stateForm,
		inputs:   inputs,
		focusIdx: 0,
		progress: p,
		spinner:  s,
	}
}

// Init is the first Bubble Tea lifecycle method.
func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.WindowSize())
}
