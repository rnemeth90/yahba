package tui

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current screen, centered in the terminal.
func (m model) View() string {
	var content string
	switch m.state {
	case stateForm:
		content = m.viewForm()
	case stateRunning:
		content = m.viewRunning()
	case stateReport:
		content = m.viewReport()
	}

	if m.width == 0 || m.height == 0 {
		return content
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// --- Form View ---

func (m model) viewForm() string {
	var b strings.Builder

	title := titleStyle.Render("YAHBA Load Tester")
	b.WriteString(title)
	b.WriteString("\n\n")

	for i := 0; i < inputCount; i++ {
		label := inputLabels[i]
		var labelStr string
		if i == m.focusIdx {
			labelStr = formLabelFocusedStyle.Render(label)
		} else {
			labelStr = formLabelStyle.Render(label)
		}
		b.WriteString(labelStr)
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.formErr != "" {
		b.WriteString(formErrorStyle.Render("  Error: " + m.formErr))
		b.WriteString("\n")
	}

	help := helpStyle.Render("enter start  |  tab/shift+tab navigate  |  esc quit")
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

// --- Running View ---

// viewRunning renders the dashboard while the load test is in progress, showing real-time stats and a latency graph.
func (m model) viewRunning() string {
	var b strings.Builder

	title := titleStyle.Render("YAHBA Load Test in Progress")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Target info
	b.WriteString(targetStyle.Render("  Target: "))
	b.WriteString(targetValueStyle.Render(m.cfg.URL))
	b.WriteString("    ")
	b.WriteString(targetStyle.Render("Method: "))
	b.WriteString(targetValueStyle.Render(m.cfg.Method))
	b.WriteString("\n\n")

	// Progress bar
	// todo: the progress bar scrolls too fast with larger jobs,
	// consider updating it on a timer instead of every request
	// or perhaps some type of sliding scale
	pct := float64(m.completed) / float64(m.totalRequests)
	progressStr := fmt.Sprintf("  %s %d/%d (%.0f%%)",
		m.progress.View(),
		m.completed, m.totalRequests,
		pct*100,
	)
	b.WriteString(progressStr)
	b.WriteString("\n\n")

	// Latency graph
	graphWidth := min(m.width-20, 80)
	if graphWidth < 20 {
		graphWidth = 20
	}
	b.WriteString(renderLatencyGraph(m.latencies, graphWidth))
	b.WriteString("\n\n")

	// Panels
	summaryPanel := m.renderSummaryPanel()
	latencyPanel := m.renderLatencyPanel()
	statusPanel := m.renderStatusPanel()

	row := lipgloss.JoinHorizontal(lipgloss.Top, summaryPanel, "   ", latencyPanel)
	b.WriteString(row)
	b.WriteString("\n\n")
	b.WriteString(statusPanel)
	b.WriteString("\n")

	help := helpStyle.Render("ctrl+c cancel")
	b.WriteString(help)

	return b.String()
}

func (m model) renderSummaryPanel() string {
	var avgLatency time.Duration
	if m.completed > 0 {
		avgLatency = m.totalLatency / time.Duration(m.completed)
	}
	_ = avgLatency

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		statLabelStyle.Render("Elapsed:"),
		statValueStyle.Render(formatElapsed(m.elapsed)),
		statLabelStyle.Render("Completed:"),
		statValueStyle.Render(fmt.Sprintf("%d", m.completed)),
		statLabelStyle.Render("Successes:"),
		statSuccessStyle.Render(fmt.Sprintf("%d", m.successes)),
		statLabelStyle.Render("Failures:"),
		statDangerStyle.Render(fmt.Sprintf("%d", m.failures)),
		statLabelStyle.Render("Bytes Recv:"),
		statValueStyle.Render(formatBytes(m.bytesReceived)),
	)

	return panelStyle.Width(38).Render(
		panelTitleStyle.Render("Summary") + "\n" + content,
	)
}

func (m model) renderLatencyPanel() string {
	var minStr, maxStr, avgStr string

	if m.completed > 0 {
		minStr = formatDuration(m.minLatency)
		maxStr = formatDuration(m.maxLatency)
		avgStr = formatDuration(m.totalLatency / time.Duration(m.completed))
	} else {
		minStr = "--"
		maxStr = "--"
		avgStr = "--"
	}

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s",
		statLabelStyle.Render("Min:"),
		statValueStyle.Render(minStr),
		statLabelStyle.Render("Max:"),
		statValueStyle.Render(maxStr),
		statLabelStyle.Render("Avg:"),
		statValueStyle.Render(avgStr),
	)

	return panelStyle.Width(36).Render(
		panelTitleStyle.Render("Latency") + "\n" + content,
	)
}

func (m model) renderStatusPanel() string {
	if len(m.statusCodes) == 0 {
		return panelStyle.Width(38).Render(
			panelTitleStyle.Render("Status Codes") + "\n" +
				statLabelStyle.Render("  waiting for responses..."),
		)
	}

	codes := sortedCodes(m.statusCodes)
	var lines []string
	for _, code := range codes {
		statusText := http.StatusText(code)
		if statusText == "" {
			statusText = "Unknown"
		}
		line := fmt.Sprintf("  %s  %s",
			statLabelStyle.Render(fmt.Sprintf("%d %s:", code, statusText)),
			statValueStyle.Render(fmt.Sprintf("%d", m.statusCodes[code])),
		)
		lines = append(lines, line)
	}

	return panelStyle.Render(
		panelTitleStyle.Render("Status Codes") + "\n" + strings.Join(lines, "\n"),
	)
}

// --- Report View ---

func (m model) viewReport() string {
	if m.report == nil {
		return "No report available."
	}

	r := m.report
	var b strings.Builder

	title := titleStyle.Render("YAHBA Stress Test Report")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Summary header
	var successRate, failureRate float64
	if r.TotalRequests > 0 {
		successRate = float64(r.Successes) / float64(r.TotalRequests) * 100
		failureRate = float64(r.Failures) / float64(r.TotalRequests) * 100
	}

	headerLines := []struct {
		label string
		value string
	}{
		{"Host", r.Host},
		{"Method", r.Method},
		{"Total Requests", fmt.Sprintf("%d", r.TotalRequests)},
		{"Successes", fmt.Sprintf("%d (%.2f%%)", r.Successes, successRate)},
		{"Failures", fmt.Sprintf("%d (%.2f%%)", r.Failures, failureRate)},
		{"Start Time", r.StartTime},
		{"End Time", r.EndTime},
		{"Duration", r.Duration.String()},
	}

	for _, h := range headerLines {
		label := reportHeaderStyle.Width(20).Align(lipgloss.Right).Render(h.label + ":")
		value := reportValueStyle.Render("  " + h.value)
		b.WriteString(label)
		b.WriteString(value)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Latency graph (full history from the test run)
	graphWidth := min(m.width-20, 80)
	if graphWidth < 20 {
		graphWidth = 20
	}
	b.WriteString(renderLatencyGraph(m.latencies, graphWidth))
	b.WriteString("\n\n")

	// Latency panel
	latencyContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		statLabelStyle.Render("Min:"), statValueStyle.Render(r.Latency.Min),
		statLabelStyle.Render("Max:"), statValueStyle.Render(r.Latency.Max),
		statLabelStyle.Render("Avg:"), statValueStyle.Render(r.Latency.Avg),
		statLabelStyle.Render("P50:"), statValueStyle.Render(r.Latency.P50),
		statLabelStyle.Render("P95:"), statValueStyle.Render(r.Latency.P95),
		statLabelStyle.Render("P99:"), statValueStyle.Render(r.Latency.P99),
	)
	latencyPanel := panelStyle.Width(36).Render(
		panelTitleStyle.Render("Latency") + "\n" + latencyContent,
	)

	// Throughput panel
	throughputContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		statLabelStyle.Render("Bytes Sent:"), statValueStyle.Render(formatBytes(r.Throughput.TotalBytesSent)),
		statLabelStyle.Render("Bytes Recv:"), statValueStyle.Render(formatBytes(r.Throughput.TotalBytesReceived)),
		statLabelStyle.Render("Sent/Sec:"), statValueStyle.Render(fmt.Sprintf("%.0f", r.Throughput.BytesSentPerSecond)),
		statLabelStyle.Render("Recv/Sec:"), statValueStyle.Render(fmt.Sprintf("%.0f", r.Throughput.BytesReceivedPerSecond)),
	)
	throughputPanel := panelStyle.Width(38).Render(
		panelTitleStyle.Render("Throughput") + "\n" + throughputContent,
	)

	row := lipgloss.JoinHorizontal(lipgloss.Top, latencyPanel, "   ", throughputPanel)
	b.WriteString(row)
	b.WriteString("\n\n")

	// Status codes panel
	if len(r.StatusCodes) > 0 {
		codes := sortedCodes(r.StatusCodes)
		var codeLines []string
		for _, code := range codes {
			statusText := http.StatusText(code)
			if statusText == "" {
				statusText = "Unknown"
			}
			line := fmt.Sprintf("  %s  %s",
				statLabelStyle.Render(fmt.Sprintf("%d %s:", code, statusText)),
				statValueStyle.Render(fmt.Sprintf("%d", r.StatusCodes[code])),
			)
			codeLines = append(codeLines, line)
		}
		statusPanel := panelStyle.Render(
			panelTitleStyle.Render("Status Codes") + "\n" + strings.Join(codeLines, "\n"),
		)
		b.WriteString(statusPanel)
		b.WriteString("  ")
	}

	// Error breakdown panel
	errorContent := fmt.Sprintf(
		"%s %s\n%s %s",
		statLabelStyle.Render("Server Errors:"), statDangerStyle.Render(fmt.Sprintf("%d", r.ErrorBreakdown.ServerErrors)),
		statLabelStyle.Render("Client Errors:"), statDangerStyle.Render(fmt.Sprintf("%d", r.ErrorBreakdown.ClientErrors)),
	)
	errorPanel := panelStyle.Width(36).Render(
		panelTitleStyle.Render("Errors") + "\n" + errorContent,
	)
	b.WriteString(errorPanel)
	b.WriteString("\n")

	help := helpStyle.Render("q quit  |  r new test")
	b.WriteString(help)

	return b.String()
}

// --- Latency Graph ---

// sparkBlocks are the Unicode block elements used for the sparkline, from lowest to tallest.
var sparkBlocks = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// renderLatencyGraph draws a sparkline chart of per-request latencies.
// maxWidth controls the maximum number of columns for the chart area.
func renderLatencyGraph(latencies []time.Duration, maxWidth int) string {
	if len(latencies) == 0 {
		return panelStyle.Width(maxWidth + 4).Render(
			panelTitleStyle.Render("Latency per Request") + "\n" +
				statLabelStyle.Render("  waiting for data..."),
		)
	}

	chartWidth := maxWidth
	if chartWidth < 10 {
		chartWidth = 10
	}

	// If we have more data points than columns, downsample by taking a
	// sliding window of the most recent values, or bucket-average.
	data := latencies
	if len(data) > chartWidth {
		data = data[len(data)-chartWidth:]
	}

	// Find min and max for scaling
	minVal := data[0]
	maxVal := data[0]
	for _, v := range data[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Calculate P95 threshold for coloring spikes
	p95Threshold := maxVal
	if len(latencies) > 1 {
		sorted := make([]time.Duration, len(latencies))
		copy(sorted, latencies)
		sortDurations(sorted)
		p95Idx := int(float64(len(sorted)) * 0.95)
		if p95Idx >= len(sorted) {
			p95Idx = len(sorted) - 1
		}
		p95Threshold = sorted[p95Idx]
	}

	spread := maxVal - minVal
	levels := len(sparkBlocks) - 1

	var bar strings.Builder
	for _, v := range data {
		var level int
		if spread == 0 {
			level = levels / 2
		} else {
			level = int(float64(v-minVal) / float64(spread) * float64(levels))
			if level > levels {
				level = levels
			}
		}

		block := sparkBlocks[level]
		if v >= p95Threshold && spread > 0 {
			bar.WriteString(graphBarDangerStyle.Render(block))
		} else if level >= 6 {
			bar.WriteString(graphBarHighStyle.Render(block))
		} else {
			bar.WriteString(graphBarStyle.Render(block))
		}
	}

	// Y-axis labels
	maxLabel := graphLabelStyle.Render(formatDuration(maxVal))
	minLabel := graphLabelStyle.Render(formatDuration(minVal))

	// Build the chart with axis labels and a baseline
	baseline := graphAxisStyle.Render(strings.Repeat("─", len(data)))

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Latency per Request"))
	b.WriteString("\n")
	b.WriteString(maxLabel + " " + graphAxisStyle.Render("┤") + "\n")
	b.WriteString(strings.Repeat(" ", 11) + graphAxisStyle.Render("│") + bar.String() + "\n")
	b.WriteString(minLabel + " " + graphAxisStyle.Render("┤") + baseline)

	// X-axis annotation
	reqCount := fmt.Sprintf("%d requests", len(latencies))
	if len(latencies) > len(data) {
		reqCount = fmt.Sprintf("last %d of %d", len(data), len(latencies))
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", 12))
	b.WriteString(graphAxisStyle.Render(reqCount))

	return panelStyle.Render(b.String())
}

// sortDurations sorts a slice of time.Duration in ascending order.
func sortDurations(d []time.Duration) {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
}

// --- Helpers ---

func formatElapsed(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func formatBytes(b int) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func sortedCodes(m map[int]int) []int {
	codes := make([]int, 0, len(m))
	for c := range m {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	return codes
}
