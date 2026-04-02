package timer

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TickMsg is sent every second while the timer is running.
type TickMsg struct {
	Time time.Time
}

// tickCmd returns a command that ticks every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}
