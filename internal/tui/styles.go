package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors (Catppuccin Macchiato palette)
	// todo: HOw will these look on light terminals? Maybe add a light theme later?
	colorPrimary = lipgloss.Color("#7DC4E4")
	colorAccent  = lipgloss.Color("#C6A0F6")
	colorSuccess = lipgloss.Color("#A6DA95")
	colorDanger  = lipgloss.Color("#ED8796")
	colorWarning = lipgloss.Color("#EED49F")
	colorMuted   = lipgloss.Color("#6E738D")
	colorText    = lipgloss.Color("#CAD3F5")
	colorSubtext = lipgloss.Color("#A5ADCB")

	// Title banner
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 4).
			MarginBottom(1).
			Align(lipgloss.Center)

	// Form
	formLabelStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Width(16).
			Align(lipgloss.Right).
			MarginRight(2)

	formLabelFocusedStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Width(16).
				Align(lipgloss.Right).
				MarginRight(2)

	formErrorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			MarginTop(1)

	// Panels (bordered boxes for dashboard sections)
	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// Stats
	statLabelStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Width(16).
			Align(lipgloss.Right).
			MarginRight(1)

	statValueStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	statSuccessStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	statDangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// progress bar colors
	progressFullColor  = colorPrimary
	progressEmptyColor = lipgloss.Color("#363A4F")

	// target info
	targetStyle = lipgloss.NewStyle().
			Foreground(colorText)

	targetValueStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// Help / footer
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// report
	reportHeaderStyle = lipgloss.NewStyle().
				Foreground(colorText)

	reportValueStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	// latency graph
	// todo: the graph scolls too fast with larger jobs
	graphBarStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)

	graphBarHighStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	graphBarDangerStyle = lipgloss.NewStyle().
				Foreground(colorDanger)

	graphAxisStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	graphLabelStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Width(10).
			Align(lipgloss.Right)
)
