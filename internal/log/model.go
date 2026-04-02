package log

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/store"
)

type viewMode int

const (
	modeNormal viewMode = iota
	modeAdd
	modeEdit
	modeConfirmDelete
)

type sessionsLoadedMsg struct {
	sessions []store.SessionWithHabit
	habits   []common.Habit
}

// Model is the Bubble Tea sub-model for the log view.
type Model struct {
	store    *store.Store
	sessions []store.SessionWithHabit
	habits   []common.Habit
	cursor   int
	mode     viewMode
	input    textinput.Model
	editID   int
	parseErr string
	width    int
	height   int
	scroll   int
}

// New creates a new log model.
func New(s *store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. 30m math, 1pm-2pm programming yesterday"
	ti.CharLimit = 80
	ti.Width = 50

	return Model{
		store: s,
		input: ti,
	}
}

// Init returns a command to load sessions.
func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	sessions, _ := m.store.ListSessionsWithHabits(100)
	habits, _ := m.store.ListHabits()
	return sessionsLoadedMsg{sessions: sessions, habits: habits}
}

// IsEditing returns true if the log is in add or edit mode.
func (m Model) IsEditing() bool {
	return m.mode == modeAdd || m.mode == modeEdit
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		m.sessions = msg.sessions
		m.habits = msg.habits
		if m.cursor >= len(m.sessions) {
			m.cursor = max(0, len(m.sessions)-1)
		}
	case common.LogRefreshMsg:
		return m, m.loadData
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			return m.updateNormal(msg)
		case modeAdd:
			return m.updateInput(msg, false)
		case modeEdit:
			return m.updateInput(msg, true)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if len(m.sessions) > 0 {
			m.cursor = (m.cursor + 1) % len(m.sessions)
		}
	case "k", "up":
		if len(m.sessions) > 0 {
			m.cursor = (m.cursor - 1 + len(m.sessions)) % len(m.sessions)
		}
	case "a":
		m.mode = modeAdd
		m.input.Reset()
		m.input.Focus()
		m.parseErr = ""
		return m, textinput.Blink
	case "e":
		if len(m.sessions) > 0 {
			s := m.sessions[m.cursor]
			m.mode = modeEdit
			m.editID = s.ID
			// Pre-fill with a parseable representation
			dur := s.ActualSecs / 60
			m.input.SetValue(fmt.Sprintf("%dm %s", dur, s.HabitName))
			m.input.Focus()
			m.parseErr = ""
			return m, textinput.Blink
		}
	case "d":
		if len(m.sessions) > 0 {
			m.mode = modeConfirmDelete
		}
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg, isEdit bool) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		parsed, err := ParseSessionInput(m.input.Value(), m.habits)
		if err != nil {
			m.parseErr = err.Error()
			return m, nil
		}

		if isEdit {
			_ = m.store.UpdateSession(m.editID, parsed.HabitID, parsed.StartTime, parsed.EndTime, parsed.DurationSecs)
		} else {
			_ = m.store.CreateManualSession(parsed.HabitID, parsed.StartTime, parsed.EndTime, parsed.DurationSecs)
		}

		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		m.parseErr = ""
		return m, func() tea.Msg { return common.StatsRefreshMsg{} }

	case "esc":
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		m.parseErr = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if len(m.sessions) > 0 {
			_ = m.store.DeleteSession(m.sessions[m.cursor].ID)
		}
		m.mode = modeNormal
		return m, func() tea.Msg { return common.StatsRefreshMsg{} }
	case "x", "esc", "n":
		m.mode = modeNormal
	}
	return m, nil
}

// View renders the log view.
func (m Model) View() string {
	if len(m.sessions) == 0 && m.mode == modeNormal {
		content := common.TitleStyle.Render("Log") + "\n\n" +
			common.MutedStyle.Render("  No sessions yet. Press [a] to add one.") + "\n\n" +
			m.renderHelp()

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	var b strings.Builder
	b.WriteString(common.TitleStyle.Render("Log"))
	b.WriteString("\n\n")

	// Group sessions by date
	var lastDate string
	for i, s := range m.sessions {
		dateLabel := formatDateGroup(s.StartTime)
		if dateLabel != lastDate {
			if lastDate != "" {
				b.WriteString("\n")
			}
			b.WriteString(common.MutedStyle.Render("  " + dateLabel))
			b.WriteString("\n")
			lastDate = dateLabel
		}

		cursor := "  "
		if i == m.cursor {
			cursor = common.AccentStyle.Render("▸ ")
		}

		timeStr := fmt.Sprintf("%s - %s", s.StartTime.Format("15:04"), s.EndTime.Format("15:04"))
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(s.HabitColor))
		durStr := formatDuration(s.ActualSecs)

		line := fmt.Sprintf("%s%-13s  %-14s  %s",
			cursor,
			common.MutedStyle.Render(timeStr),
			nameStyle.Render(s.HabitName),
			common.AccentStyle.Render(durStr),
		)
		b.WriteString("  " + line + "\n")
	}

	// Modal states
	b.WriteString("\n")
	switch m.mode {
	case modeAdd:
		b.WriteString(common.MutedStyle.Render("  Add: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.parseErr != "" {
			b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.parseErr))
			b.WriteString("\n")
		}
		b.WriteString("  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel"))
	case modeEdit:
		b.WriteString(common.MutedStyle.Render("  Edit: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.parseErr != "" {
			b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.parseErr))
			b.WriteString("\n")
		}
		b.WriteString("  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel"))
	case modeConfirmDelete:
		if len(m.sessions) > 0 {
			s := m.sessions[m.cursor]
			desc := fmt.Sprintf("%s %s (%s)", s.StartTime.Format("15:04"), s.HabitName, formatDuration(s.ActualSecs))
			b.WriteString(common.WarningStyle.Render(fmt.Sprintf("  Delete \"%s\"?", desc)))
			b.WriteString("\n")
			b.WriteString("  " + helpEntry("y", "delete") + "  " + helpEntry("x", "cancel"))
		}
	default:
		b.WriteString(m.renderHelp())
	}

	content := b.String()

	// Scroll handling
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	if totalLines <= m.height {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, content)
	}

	maxScroll := totalLines - m.height
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	end := m.scroll + m.height
	if end > totalLines {
		end = totalLines
	}
	return strings.Join(lines[m.scroll:end], "\n")
}

func (m Model) renderHelp() string {
	return "  " + helpEntry("a", "add") + "  " +
		helpEntry("e", "edit") + "  " +
		helpEntry("d", "delete") + "  " +
		helpEntry("j/k", "navigate")
}

func helpEntry(key, desc string) string {
	return common.HelpKeyStyle.Render("["+key+"]") + " " + common.HelpDescStyle.Render(desc)
}

func formatDateGroup(t time.Time) string {
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	if d.Equal(today) {
		return "Today"
	}
	if d.Equal(yesterday) {
		return "Yesterday"
	}
	return t.Format("Mon, Jan 2")
}

func formatDuration(secs int) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
