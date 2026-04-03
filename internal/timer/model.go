package timer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/theme"
)

// TimerState represents the current state of the timer.
type TimerState int

const (
	Idle TimerState = iota
	Running
	Paused
	Overtime
	Confirming // waiting for save/discard after stop
)

// bigDigits are 3-line tall ASCII representations of digits 0-9.
var bigDigits = [10][]string{
	{"█▀█", "█ █", "▀▀▀"}, // 0
	{" ▀█", "  █", "  ▀"}, // 1
	{"▀▀█", "█▀▀", "▀▀▀"}, // 2
	{"▀▀█", " ▀█", "▀▀▀"}, // 3
	{"█ █", "▀▀█", "  ▀"}, // 4
	{"█▀▀", "▀▀█", "▀▀▀"}, // 5
	{"█▀▀", "█▀█", "▀▀▀"}, // 6
	{"▀▀█", "  █", "  ▀"}, // 7
	{"█▀█", "█▀█", "▀▀▀"}, // 8
	{"█▀█", "▀▀█", "▀▀▀"}, // 9
}

// bigColon is the 3-line tall colon separator.
var bigColon = []string{" ", ":", " "}

// Model is the timer sub-model for the pomodoro app.
type Model struct {
	state          TimerState
	plannedMinutes int
	remaining      int // seconds remaining (counts down)
	overtime       int // seconds of overtime (counts up)
	elapsed        int // total seconds elapsed
	habit          common.Habit
	habits         []common.Habit // all available habits
	habitIdx       int            // current index in habits slice
	client         *api.Client
	width, height  int
	progress       progress.Model

	// Confirming state: stash values while waiting for y/x
	confirmIsReset   bool // true = reset confirmation, false = stop confirmation
	stashedCompleted bool
	lastErr          string
}

// New creates a new timer model with default settings.
func New(c *api.Client) Model {
	p := progress.New(
		progress.WithGradient(string(theme.ColorAccent), string(theme.ColorMagenta)),
		progress.WithoutPercentage(),
	)
	m := Model{
		state:          Idle,
		plannedMinutes: 25,
		remaining:      25 * 60,
		client:         c,
		progress:       p,
	}
	// Load habits so j/k cycling works on the timer page
	if habits, err := c.ListHabits(); err == nil && len(habits) > 0 {
		m.habits = habits
		m.habit = habits[0]
		m.habitIdx = 0
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		// Confirming state handles its own keys
		if m.state == Confirming {
			return m.handleConfirm(msg)
		}

		switch msg.String() {
		case " ":
			return m.handleStartPause()
		case "s":
			return m.handleStop()
		case "+", "=":
			if m.state == Idle {
				m.plannedMinutes += 5
				m.remaining = m.plannedMinutes * 60
			}
		case "-":
			if m.state == Idle && m.plannedMinutes > 5 {
				m.plannedMinutes -= 5
				m.remaining = m.plannedMinutes * 60
			}
		case "r":
			return m.handleReset()
		case "j", "down":
			return m.cycleHabit(1), nil
		case "k", "up":
			return m.cycleHabit(-1), nil
		}

	case TickMsg:
		return m.handleTick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(msg.Width-10, 40)

	case common.HabitSelectedMsg:
		m.setHabitByID(msg.ID)
	}

	return m, nil
}

func (m Model) cycleHabit(dir int) Model {
	if len(m.habits) == 0 {
		return m
	}
	m.habitIdx = (m.habitIdx + dir + len(m.habits)) % len(m.habits)
	m.habit = m.habits[m.habitIdx]
	return m
}

func (m *Model) setHabitByID(id int) {
	for i, h := range m.habits {
		if h.ID == id {
			m.habitIdx = i
			m.habit = h
			return
		}
	}
}

func (m Model) handleConfirm(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		// Save the session
		completed := m.stashedCompleted
		habitID := m.habit.ID
		elapsed := m.elapsed
		overtime := m.overtime
		planned := m.plannedMinutes

		if err := m.client.CreateSession(habitID, planned, elapsed, overtime, completed); err != nil {
			m.lastErr = "Failed to save session"
		} else {
			m.lastErr = ""
		}

		endMsg := common.SessionEndMsg{
			HabitID:         habitID,
			PlannedMinutes:  planned,
			ActualSeconds:   elapsed,
			OvertimeSeconds: overtime,
			Completed:       completed,
		}

		m.state = Idle
		m.remaining = m.plannedMinutes * 60
		m.elapsed = 0
		m.overtime = 0
		return m, func() tea.Msg { return endMsg }

	case "x", "esc":
		// Discard - reset without saving
		m.state = Idle
		m.remaining = m.plannedMinutes * 60
		m.elapsed = 0
		m.overtime = 0
		return m, nil
	}
	return m, nil
}

func (m Model) handleStartPause() (Model, tea.Cmd) {
	if m.habit.ID == 0 {
		return m, nil
	}
	switch m.state {
	case Idle:
		m.state = Running
		m.elapsed = 0
		m.overtime = 0
		return m, tickCmd()
	case Running, Overtime:
		m.state = Paused
	case Paused:
		if m.remaining > 0 {
			m.state = Running
		} else {
			m.state = Overtime
		}
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) handleStop() (Model, tea.Cmd) {
	if m.state == Idle || m.state == Confirming {
		return m, nil
	}

	m.stashedCompleted = m.remaining <= 0
	m.confirmIsReset = false
	m.state = Confirming
	return m, nil
}

func (m Model) handleReset() (Model, tea.Cmd) {
	if m.state == Idle || m.state == Confirming {
		return m, nil
	}
	// If there's elapsed time, ask for confirmation
	if m.elapsed > 0 {
		m.stashedCompleted = m.remaining <= 0
		m.confirmIsReset = true
		m.state = Confirming
		return m, nil
	}
	// Nothing elapsed, just reset
	m.state = Idle
	m.remaining = m.plannedMinutes * 60
	m.elapsed = 0
	m.overtime = 0
	return m, nil
}

func (m Model) handleTick() (Model, tea.Cmd) {
	if m.state != Running && m.state != Overtime {
		return m, nil
	}

	m.elapsed++

	if m.state == Running {
		m.remaining--
		if m.remaining <= 0 {
			m.remaining = 0
			m.state = Overtime
			m.overtime++
			return m, tea.Batch(tickCmd(), playBells())
		}
	}

	if m.state == Overtime {
		m.overtime++
	}

	return m, tickCmd()
}

func playBells() tea.Cmd {
	return func() tea.Msg {
		for i := 0; i < 3; i++ {
			fmt.Print("\a")
			if i < 2 {
				time.Sleep(200 * time.Millisecond)
			}
		}
		return nil
	}
}

// SetHabit sets the current habit for the timer.
func (m *Model) SetHabit(h common.Habit) {
	m.setHabitByID(h.ID)
}

// RefreshHabits reloads the habit list from the store.
func (m *Model) RefreshHabits() {
	if habits, err := m.client.ListHabits(); err == nil {
		m.habits = habits
		// Keep current selection if still valid
		found := false
		for i, h := range m.habits {
			if h.ID == m.habit.ID {
				m.habitIdx = i
				found = true
				break
			}
		}
		if !found && len(m.habits) > 0 {
			m.habitIdx = 0
			m.habit = m.habits[0]
		}
	}
}

// IsRunning returns true if the timer is actively running or in overtime.
func (m Model) IsRunning() bool {
	return m.state == Running || m.state == Overtime
}

// IsConfirming returns true if waiting for save/discard confirmation.
func (m Model) IsConfirming() bool {
	return m.state == Confirming
}

// View renders the timer display.
func (m Model) View() string {
	var sections []string

	// Time display
	timeDisplay := m.renderBigTime()
	sections = append(sections, timeDisplay)

	// Spacer
	sections = append(sections, "")

	// Progress bar
	progressBar := m.renderProgress()
	sections = append(sections, progressBar)

	// Spacer
	sections = append(sections, "")

	// Habit name
	habitLine := m.renderHabit()
	sections = append(sections, habitLine)

	// Status
	statusLine := m.renderStatus()
	sections = append(sections, statusLine)

	// Error
	if m.lastErr != "" {
		sections = append(sections, common.OvertimeStyle.Render("⚠ "+m.lastErr))
	}

	// Spacer
	sections = append(sections, "")

	// Help
	helpLine := m.renderHelp()
	sections = append(sections, helpLine)

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)

	// Center in available space
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderBigTime() string {
	var minutes, seconds int
	isOvertime := m.state == Overtime

	if m.state == Confirming {
		// Show total elapsed time during confirmation
		minutes = m.elapsed / 60
		seconds = m.elapsed % 60
	} else if isOvertime {
		total := m.overtime
		minutes = total / 60
		seconds = total % 60
	} else {
		minutes = m.remaining / 60
		seconds = m.remaining % 60
	}

	// Clamp minutes to 2 digits for display
	if minutes > 99 {
		minutes = 99
	}

	d1 := minutes / 10
	d2 := minutes % 10
	d3 := seconds / 10
	d4 := seconds % 60 % 10

	// Build 3 lines of the big time display
	var lines [3]string
	for row := 0; row < 3; row++ {
		parts := []string{}
		if isOvertime {
			// Add minus sign
			switch row {
			case 0:
				parts = append(parts, "   ")
			case 1:
				parts = append(parts, "───")
			case 2:
				parts = append(parts, "   ")
			}
			parts = append(parts, " ")
		}
		parts = append(parts,
			bigDigits[d1][row], " ",
			bigDigits[d2][row], " ",
			bigColon[row], " ",
			bigDigits[d3][row], " ",
			bigDigits[d4][row],
		)
		lines[row] = strings.Join(parts, "")
	}

	// Style the time based on state
	var timeStyle lipgloss.Style
	switch m.state {
	case Overtime:
		timeStyle = lipgloss.NewStyle().Foreground(theme.ColorOvertime).Bold(true)
	case Confirming:
		timeStyle = lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true)
	default:
		timeStyle = lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true)
	}

	styled := make([]string, 3)
	for i, line := range lines {
		styled[i] = timeStyle.Render(line)
	}

	return lipgloss.JoinVertical(lipgloss.Center, styled...)
}

func (m Model) renderProgress() string {
	var pct float64
	if m.state == Overtime {
		pct = 1.0
	} else {
		total := float64(m.plannedMinutes * 60)
		if total > 0 {
			pct = 1.0 - float64(m.remaining)/total
		}
	}

	// Clamp
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	return m.progress.ViewAs(pct)
}

func (m Model) renderHabit() string {
	name := m.habit.Name
	if name == "" {
		name = "no habit selected"
		return common.MutedStyle.Render(name)
	}
	return common.AccentStyle.Render(name)
}

func (m Model) renderStatus() string {
	switch m.state {
	case Idle:
		return common.MutedStyle.Render("idle")
	case Running:
		return common.SuccessStyle.Render("running")
	case Paused:
		return common.WarningStyle.Render("paused")
	case Overtime:
		return common.OvertimeStyle.Render("overtime!")
	case Confirming:
		if m.confirmIsReset {
			return common.WarningStyle.Render("save before resetting?")
		}
		return common.WarningStyle.Render("save this session?")
	}
	return ""
}

func (m Model) renderHelp() string {
	var pairs []struct {
		key  string
		desc string
	}

	if m.state == Confirming {
		pairs = []struct {
			key  string
			desc string
		}{
			{"y", "save"},
			{"x", "discard"},
			{"esc", "discard"},
		}
	} else {
		pairs = []struct {
			key  string
			desc string
		}{
			{"space", "start/pause"},
			{"s", "stop"},
			{"j/k", "habit"},
			{"+/-", "adjust time"},
			{"r", "reset"},
		}
	}

	var parts []string
	for _, p := range pairs {
		k := common.HelpKeyStyle.Render(p.key)
		d := common.HelpDescStyle.Render(p.desc)
		parts = append(parts, k+" "+d)
	}

	return strings.Join(parts, common.MutedStyle.Render("  ·  "))
}
