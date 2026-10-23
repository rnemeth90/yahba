package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rnemeth90/yahba/internal/report"
)

// viewRampRunning shows finished steps as they arrive, so the point where
// latency starts climbing is visible while the test is still going.
func (m model) viewRampRunning() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("YAHBA Ramp Test in Progress"))
	b.WriteString("\n\n")

	b.WriteString(targetStyle.Render("  Target: "))
	b.WriteString(targetValueStyle.Render(m.cfg.URL))
	b.WriteString("    ")
	b.WriteString(targetStyle.Render("Method: "))
	b.WriteString(targetValueStyle.Render(m.cfg.Method))
	b.WriteString("\n\n")

	current := m.cfg.RampStart + len(m.rampSteps)*m.cfg.RampStep
	b.WriteString(fmt.Sprintf("  %s Step %d: offering %d RPS for %s\n  %s\n\n",
		m.spinner.View(), len(m.rampSteps)+1, current, m.cfg.StepDuration,
		m.progress.ViewAs(m.progressFraction())))

	b.WriteString(renderStepTable(m.rampSteps, nil))
	b.WriteString("\n\n")

	limits := fmt.Sprintf("Stops when error rate > %.1f%%", m.cfg.MaxErrorRate)
	if m.cfg.MaxP99 > 0 {
		limits += fmt.Sprintf(", p99 > %s", m.cfg.MaxP99)
	}
	limits += ", or a step reaches under 90% of its target"
	b.WriteString(statLabelStyle.Render("  " + limits))
	b.WriteString("\n")
	b.WriteString(statLabelStyle.Render(fmt.Sprintf("  Elapsed: %s", formatElapsed(m.elapsed))))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("ctrl+c stop"))

	return b.String()
}

// viewRampReport is the final ramp screen.
func (m model) viewRampReport() string {
	r := m.rampReport
	var b strings.Builder

	b.WriteString(titleStyle.Render("YAHBA Ramp Test Report"))
	b.WriteString("\n\n")

	headers := []struct{ label, value string }{
		{"Host", r.Host},
		{"Method", r.Method},
		{"Stopped", r.StopReason},
	}
	for _, h := range headers {
		b.WriteString(reportHeaderStyle.Width(20).Align(lipgloss.Right).Render(h.label + ":"))
		b.WriteString(reportValueStyle.Render("  " + h.value))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(renderStepTable(r.Steps, r))
	b.WriteString("\n\n")

	sustained := "none (the first step already failed)"
	if r.SustainedRPS > 0 {
		sustained = fmt.Sprintf("%d RPS", r.SustainedRPS)
	}
	lines := []string{
		statLabelStyle.Render("Sustained:") + " " + statSuccessStyle.Render(sustained),
	}
	if r.KneeRPS > 0 {
		lines = append(lines, statLabelStyle.Render("Knee:")+"      "+statValueStyle.Render(fmt.Sprintf("%d RPS (p99 at least 2x the first step)", r.KneeRPS)))
	}
	if r.BreakingRPS > 0 {
		lines = append(lines, statLabelStyle.Render("Breaking:")+"   "+statDangerStyle.Render(fmt.Sprintf("%d RPS", r.BreakingRPS)))
	}
	b.WriteString(panelStyle.Render(panelTitleStyle.Render("Result") + "\n" + strings.Join(lines, "\n")))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("q quit  |  r new test"))

	return b.String()
}

// renderStepTable draws one row per ramp step with a latency histogram
// sparkline. rr, when non-nil, supplies the knee marker.
func renderStepTable(steps []report.Step, rr *report.RampReport) string {
	if len(steps) == 0 {
		return panelStyle.Render(panelTitleStyle.Render("Steps") + "\n" + statLabelStyle.Render("  waiting for the first step to finish..."))
	}

	var lines []string
	lines = append(lines, statLabelStyle.Render(fmt.Sprintf("%8s %9s %7s %10s %10s  %-12s %s",
		"target", "achieved", "err %", "p50", "p99", "latency", "")))
	for _, s := range steps {
		row := fmt.Sprintf("%8d %9.1f %7.2f %10s %10s  %-12s ",
			s.TargetRPS, s.AchievedRPS, s.ErrorRate, s.P50, s.P99, histSpark(s.Histogram))
		switch {
		case !s.Passed:
			row += statDangerStyle.Render("FAIL " + s.Reason)
		case s.Knee:
			row += graphBarHighStyle.Render("PASS  <-- knee")
		default:
			row += statSuccessStyle.Render("PASS")
		}
		lines = append(lines, row)
	}

	title := "Steps"
	if len(steps[0].Histogram) > 0 {
		title += "  (latency: " + steps[0].Histogram[0].LE + " ... " + steps[0].Histogram[len(steps[0].Histogram)-2].LE + ", +Inf)"
	}
	return panelStyle.Render(panelTitleStyle.Render(title) + "\n" + strings.Join(lines, "\n"))
}

// histSpark renders a histogram as a sparkline, scaled to its fullest bucket.
func histSpark(buckets []report.Bucket) string {
	peak := 0
	for _, b := range buckets {
		peak = max(peak, b.Count)
	}
	var s strings.Builder
	for _, b := range buckets {
		if b.Count == 0 || peak == 0 {
			s.WriteString("·")
			continue
		}
		level := b.Count * (len(sparkBlocks) - 1) / peak
		s.WriteString(sparkBlocks[level])
	}
	return s.String()
}
