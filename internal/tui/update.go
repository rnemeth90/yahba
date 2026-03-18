package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rnemeth90/yahba/internal/report"
)

// Update handles all incoming messages and returns the updated model + commands.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(msg.Width-20, 80)
		return m, nil

	case resultMsg:
		return m.handleResult(report.Result(msg))

	case resultsCompleteMsg:
		// All results received; report will arrive shortly.
		return m, nil

	case reportMsg:
		r := report.Report(msg)
		m.report = &r
		m.state = stateReport
		return m, nil

	case tickMsg:
		if m.state == stateRunning {
			m.elapsed = time.Since(m.startTime)
			return m, tick()
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == stateRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case progress.FrameMsg:
		p, cmd := m.progress.Update(msg)
		m.progress = p.(progress.Model)
		return m, cmd

	case errMsg:
		m.formErr = msg.Error()
		return m, nil
	}

	// Pass unhandled messages to active text inputs in form state.
	if m.state == stateForm {
		return m.updateFormInputs(msg)
	}

	return m, nil
}

// handleKey dispatches key events based on current state.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateForm:
		return m.handleFormKey(msg)
	case stateRunning:
		return m.handleRunningKey(msg)
	case stateReport:
		return m.handleReportKey(msg)
	}
	return m, nil
}

//	handleFormKey for key handling
//
// todo: support vim bindings (j/k to navigate, etc.)
func (m model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "tab", "down":
		m.focusIdx = (m.focusIdx + 1) % inputCount
		return m.focusInput()

	case "shift+tab", "up":
		m.focusIdx = (m.focusIdx - 1 + inputCount) % inputCount
		return m.focusInput()

	case "enter":
		cmd := m.startTest()
		return m, cmd
	}

	return m.updateFormInputs(msg)
}

// focusInput blurs all inputs and focuses the one at focusIdx.
func (m model) focusInput() (tea.Model, tea.Cmd) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.inputs[m.focusIdx].Focus()
	return m, nil
}

// updateFormInputs forwards messages to the currently focused text input.
func (m model) updateFormInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	return m, cmd
}

// --- Running key handling ---

func (m model) handleRunningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.cancelFunc != nil {
			m.cancelFunc()
		}
		// Don't quit immediately; wait for the report to arrive.
		return m, nil
	}
	return m, nil
}

// --- Report key handling ---

func (m model) handleReportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "r":
		m.resetToForm()
		return m, nil
	}
	return m, nil
}

// --- Result handling ---

func (m model) handleResult(r report.Result) (tea.Model, tea.Cmd) {
	m.completed++

	if r.ResultCode > 0 && r.ResultCode < 400 {
		m.successes++
	} else {
		m.failures++
	}

	m.statusCodes[r.ResultCode]++
	m.bytesSent += r.BytesSent
	m.bytesReceived += r.BytesReceived
	m.totalLatency += r.ElapsedTime
	m.latencies = append(m.latencies, r.ElapsedTime)

	if m.minLatency == 0 || r.ElapsedTime < m.minLatency {
		m.minLatency = r.ElapsedTime
	}
	if r.ElapsedTime > m.maxLatency {
		m.maxLatency = r.ElapsedTime
	}

	pct := float64(m.completed) / float64(m.totalRequests)
	cmd := m.progress.SetPercent(pct)

	return m, tea.Batch(cmd, waitForResult(m.progressChan))
}
