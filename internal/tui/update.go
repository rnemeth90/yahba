package tui

import (
	"fmt"
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
		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)
		return m, cmd

	case clearErrorMsg:
		m.err = nil
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

	// Pass unhandled messages (e.g. the filepicker's internal directory-read
	// results) to the filepicker while it's active.
	if m.state == stateFilepicker {
		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey dispatches key events based on current state.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateForm:
		return m.handleFormKey(msg)
	case stateFilepicker:
		return m.handleFilepickerKey(msg)
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

	case "ctrl+o":
		m.state = stateFilepicker
		m.formErr = ""
		m.err = nil
		return m, m.filepicker.Init()

	case "enter":
		cmd := m.startTest()
		return m, cmd
	}

	return m.updateFormInputs(msg)
}

// --- Filepicker key handling ---

// handleFilepickerKey lets the user browse for a test definition file. "q"
// cancels back to the form; "ctrl+c" quits the whole app; every other key is
// forwarded to the filepicker component itself (navigation, open, select).
func (m model) handleFilepickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		m.state = stateForm
		m.err = nil
		return m, nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
		runCmd := m.startDefFileTest(path)
		if runCmd == nil {
			// Validation/parsing failed; formErr is set, show it on the form.
			m.state = stateForm
			return m, nil
		}
		return m, runCmd
	}

	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		m.err = fmt.Errorf("%q is not a valid definition file (expected .yaml or .yml)", path)
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(3*time.Second))
	}

	return m, cmd
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

	if r.ResultCode > 0 {
		m.statusCodes[r.ResultCode]++
	}
	m.bytesSent += r.BytesSent
	m.bytesReceived += r.BytesReceived
	m.totalLatency += r.ElapsedTime

	m.latencies.Push(r.ElapsedTime)

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
