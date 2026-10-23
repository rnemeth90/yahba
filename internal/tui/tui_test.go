package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStatsBoundsMemoryAndIgnoresCodeZero(t *testing.T) {
	s := newLiveStats()
	for i := 0; i < recentLatencies+500; i++ {
		s.add(report.Result{ResultCode: 200, ElapsedTime: time.Duration(i+1) * time.Microsecond})
	}
	s.add(report.Result{ResultCode: 0, ElapsedTime: time.Millisecond}) // network error

	snap := s.snapshot()
	assert.Len(t, snap.latencies, recentLatencies)
	assert.Equal(t, recentLatencies+501, snap.completed)
	assert.Equal(t, 1, snap.failures)
	assert.NotContains(t, snap.statusCodes, 0)
	assert.Equal(t, recentLatencies+500, snap.statusCodes[200])
	// oldest first: the newest latency is last
	assert.Equal(t, time.Millisecond, snap.latencies[len(snap.latencies)-1])
}

func TestFormVisibilityFollowsMode(t *testing.T) {
	m := New()
	assert.True(t, m.visible(inputRequests))
	assert.False(t, m.visible(inputDuration))
	assert.False(t, m.visible(inputRampStart))

	m.cycleMode(1)
	assert.Equal(t, config.ModeMax, m.currentMode())
	assert.False(t, m.visible(inputRequests))
	assert.True(t, m.visible(inputDuration))

	m.cycleMode(1)
	assert.Equal(t, config.ModeRamp, m.currentMode())
	assert.True(t, m.visible(inputRampStep))
	assert.False(t, m.visible(inputDuration))

	// focus skips hidden inputs
	m.focusIdx = inputMode
	m.moveFocus(1)
	assert.Equal(t, inputRampStart, m.focusIdx)
	m.moveFocus(-1)
	assert.Equal(t, inputMode, m.focusIdx)

	m.cycleMode(1)
	assert.Equal(t, config.ModeRequests, m.currentMode())
}

func TestBuildConfigPerMode(t *testing.T) {
	m := New()
	m.inputs[inputURL].SetValue("http://localhost:1234")

	cfg, err := m.buildConfig()
	require.NoError(t, err)
	assert.Equal(t, config.ModeRequests, cfg.EffectiveMode())
	assert.Equal(t, 100, cfg.Requests)

	m.cycleMode(1)
	m.inputs[inputDuration].SetValue("5s")
	cfg, err = m.buildConfig()
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, cfg.Duration)

	m.inputs[inputDuration].SetValue("soon")
	_, err = m.buildConfig()
	assert.ErrorContains(t, err, "duration")

	m.cycleMode(1)
	m.inputs[inputMaxP99].SetValue("250ms")
	cfg, err = m.buildConfig()
	require.NoError(t, err)
	assert.Equal(t, config.ModeRamp, cfg.EffectiveMode())
	assert.Equal(t, 100, cfg.RampStart)
	assert.Equal(t, 250*time.Millisecond, cfg.MaxP99)
	assert.Zero(t, cfg.RampMax)

	m.inputs[inputRampStep].SetValue("0")
	_, err = m.buildConfig()
	assert.ErrorContains(t, err, "RPS step")
}

func runToReport(t *testing.T, m model, wantRamp bool) model {
	t.Helper()
	cmd := m.startTest()
	require.Empty(t, m.formErr)
	require.NotNil(t, cmd)
	defer m.cancelFunc()

	deadline := time.After(30 * time.Second)
	for m.state == stateRunning {
		select {
		case <-deadline:
			t.Fatal("test did not finish")
		case <-time.After(50 * time.Millisecond):
		}
		updated, _ := m.Update(tickMsg(time.Now()))
		m = updated.(model)

		// deliver the completion message the way Bubble Tea would
		if wantRamp {
			select {
			case msg := <-m.rampChan:
				updated, _ = m.Update(msg)
				m = updated.(model)
			default:
			}
		} else {
			select {
			case r, ok := <-m.reportChan:
				if ok {
					updated, _ = m.Update(reportMsg(r))
					m = updated.(model)
				}
			default:
			}
		}
	}
	return m
}

func TestRunMaxModeEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()

	m := New()
	m.inputs[inputURL].SetValue(srv.URL)
	m.cycleMode(1)
	m.inputs[inputDuration].SetValue("500ms")
	m.inputs[inputKeepAlives].SetValue("true")

	m = runToReport(t, m, false)
	require.NotNil(t, m.report)
	assert.Equal(t, config.ModeMax, m.report.Mode)
	assert.Greater(t, m.completed, 0)
	assert.Equal(t, m.report.TotalRequests, m.completed)
	assert.Contains(t, m.View(), "Achieved RPS")
}

func TestRunRequestsModeShowsAchievedVsConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()

	m := New()
	m.inputs[inputURL].SetValue(srv.URL)
	m.inputs[inputRequests].SetValue("20")
	m.inputs[inputRPS].SetValue("100")

	m = runToReport(t, m, false)
	require.NotNil(t, m.report)
	assert.Equal(t, 20, m.completed)
	assert.Contains(t, m.View(), "100 configured")
}

func TestRunRampModeEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()

	m := New()
	m.inputs[inputURL].SetValue(srv.URL)
	m.cycleMode(1)
	m.cycleMode(1)
	m.inputs[inputRampStart].SetValue("10")
	m.inputs[inputRampStep].SetValue("10")
	m.inputs[inputRampMax].SetValue("20")
	m.inputs[inputStepDuration].SetValue("1s")

	m = runToReport(t, m, true)
	require.NotNil(t, m.rampReport)
	assert.Len(t, m.rampReport.Steps, 2)
	assert.Equal(t, 20, m.rampReport.SustainedRPS)

	view := m.View()
	assert.Contains(t, view, "Ramp Test Report")
	assert.Contains(t, view, "Sustained")
}

func TestRampRunningViewListsFinishedSteps(t *testing.T) {
	m := New()
	m.mode = config.ModeRamp
	m.cfg = config.Config{URL: "http://x", Method: "GET", RampStart: 100, RampStep: 100, StepDuration: 10 * time.Second, MaxErrorRate: 5}
	m.rampSteps = []report.Step{
		{TargetRPS: 100, AchievedRPS: 100, P50: "1ms", P99: "2ms", Passed: true, Histogram: report.LatencyHistogram(nil)},
	}
	view := m.viewRampRunning()
	assert.Contains(t, view, "Step 2")
	assert.Contains(t, view, "200 RPS")
	assert.True(t, strings.Contains(view, "PASS"))
}

func TestCtrlCCancelsRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()

	m := New()
	m.inputs[inputURL].SetValue(srv.URL)
	m.cycleMode(1)
	m.inputs[inputDuration].SetValue("1h")
	m.startTest()
	require.NotNil(t, m.cancelFunc)

	m.handleRunningKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	select {
	case <-m.reportChan:
	case <-time.After(10 * time.Second):
		t.Fatal("run did not stop after cancel")
	}
}
