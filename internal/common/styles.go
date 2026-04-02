package common

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/theme"
)

var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Background(theme.ColorBg).
			Foreground(theme.ColorFg)

	// Tab bar
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorAccent).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(theme.ColorMuted).
				Padding(0, 2)

	// Panels
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorBorder).
			Padding(1, 2)

	// Text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorAccent)

	MutedStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)

	AccentStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(theme.ColorWarning)

	OvertimeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorOvertime).
			Bold(true)

	// Help bar
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)
)
