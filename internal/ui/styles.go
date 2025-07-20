package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Underline(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Align(lipgloss.Center)

	// Tab styles
	TabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 1)

	ActiveTabStyle = TabStyle.Copy().
			Foreground(lipgloss.Color("36")).
			Bold(true).
			Underline(true)

	InactiveTabStyle = TabStyle.Copy().
				Foreground(lipgloss.Color("241"))

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	ProgressCompleteStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("36")).
				Foreground(lipgloss.Color("230"))

	ProgressEmptyStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("240"))

	// Data styles
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Align(lipgloss.Left)

	TableCellStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("230"))
)

func RenderProgressBar(percent float64, width int) string {
	if width <= 0 {
		width = 20
	}

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}

	return ProgressCompleteStyle.Render(bar[:filled]) +
		ProgressEmptyStyle.Render(bar[filled:])
}
