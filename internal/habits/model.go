package habits

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// habitsLoadedMsg is sent after habits are loaded from the store.
type habitsLoadedMsg struct {
	habits []common.Habit
}

// Default colors assigned to new habits in rotation.
var defaultColors = []string{
	"#7aa2f7", "#bb9af7", "#9ece6a", "#e0af68", "#7dcfff", "#ff9e64",
}

// Model is the Bubble Tea sub-model for the habits view.
type Model struct {
	client     *api.Client
	habits     []common.Habit
	cursor     int
	adding     bool
	confirming bool // true when confirming delete
	input      textinput.Model
	width      int
	height     int
	selected   int // currently selected habit ID
	lastErr    string
}

// New creates a new habits Model with initial data loaded from the API.
func New(c *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "habit name..."
	ti.CharLimit = 40
	ti.Width = 30

	m := Model{
		client: c,
		input:  ti,
	}

	// Eagerly load habits so we have data before Init runs.
	if habits, err := c.ListHabits(); err == nil {
		m.habits = habits
		if len(habits) > 0 {
			m.selected = habits[0].ID
		}
	}

	return m
}

// Init returns a command that loads habits from the store.
func (m Model) Init() tea.Cmd {
	return m.loadHabits
}

func (m Model) loadHabits() tea.Msg {
	habits, err := m.client.ListHabits()
	if err != nil {
		return habitsLoadedMsg{habits: nil}
	}
	return habitsLoadedMsg{habits: habits}
}

// Update handles messages for the habits view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case habitsLoadedMsg:
		m.habits = msg.habits
		if m.cursor >= len(m.habits) {
			m.cursor = max(0, len(m.habits)-1)
		}
		if len(m.habits) > 0 {
			m.selected = m.habits[m.cursor].ID
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirming {
			return m.updateConfirming(msg)
		}
		if m.adding {
			return m.updateAdding(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

func (m Model) updateAdding(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name != "" {
			color := defaultColors[len(m.habits)%len(defaultColors)]
			if _, err := m.client.AddHabit(name, color); err != nil {
				m.lastErr = "Failed to add habit"
			} else {
				m.lastErr = ""
			}
		}
		m.adding = false
		m.input.Reset()
		m.input.Blur()
		return m, m.loadHabits

	case "esc":
		m.adding = false
		m.input.Reset()
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirming(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if len(m.habits) > 0 {
			h := m.habits[m.cursor]
			if err := m.client.DeleteHabit(h.ID); err != nil {
				m.lastErr = "Failed to delete habit"
			} else {
				m.lastErr = ""
			}
		}
		m.confirming = false
		return m, m.loadHabits
	case "x", "esc", "n":
		m.confirming = false
		return m, nil
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if len(m.habits) > 0 {
			m.cursor = (m.cursor + 1) % len(m.habits)
		}
		return m, nil

	case "k", "up":
		if len(m.habits) > 0 {
			m.cursor = (m.cursor - 1 + len(m.habits)) % len(m.habits)
		}
		return m, nil

	case "enter":
		if len(m.habits) > 0 {
			h := m.habits[m.cursor]
			m.selected = h.ID
			return m, func() tea.Msg {
				return common.HabitSelectedMsg{
					ID:    h.ID,
					Name:  h.Name,
					Color: h.Color,
				}
			}
		}
		return m, nil

	case "a":
		m.adding = true
		m.input.Focus()
		return m, textinput.Blink

	case "d":
		if len(m.habits) > 0 {
			m.confirming = true
		}
		return m, nil
	}

	return m, nil
}

// View renders the habits view.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(common.TitleStyle.Render("Habits"))
	b.WriteString("\n\n")

	if len(m.habits) == 0 {
		b.WriteString(common.MutedStyle.Render("  No habits yet. Press [a] to add one."))
		b.WriteString("\n")
	}

	for i, h := range m.habits {
		cursor := "  "
		if i == m.cursor {
			cursor = common.AccentStyle.Render("▸ ")
		}

		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(h.Color))
		if i == m.cursor {
			nameStyle = nameStyle.Background(lipgloss.Color("#24283b"))
		}

		line := fmt.Sprintf("%s%s", cursor, nameStyle.Render(h.Name))
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.confirming && len(m.habits) > 0 {
		h := m.habits[m.cursor]
		b.WriteString("\n")
		b.WriteString(common.WarningStyle.Render(fmt.Sprintf("  Delete \"%s\"? This removes all its data.", h.Name)))
		b.WriteString("\n")
		b.WriteString("  " + helpEntry("y", "delete") + "  " + helpEntry("x", "cancel"))
		b.WriteString("\n")
	} else if m.adding {
		b.WriteString("\n")
		b.WriteString(common.MutedStyle.Render("  New: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	}

	if m.lastErr != "" {
		b.WriteString("\n")
		b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.lastErr))
	}

	b.WriteString("\n")
	help := helpEntry("enter", "select") + "  " +
		helpEntry("a", "add") + "  " +
		helpEntry("d", "delete") + "  " +
		helpEntry("esc", "back")
	b.WriteString(help)

	return b.String()
}

func helpEntry(key, desc string) string {
	return common.HelpKeyStyle.Render("["+key+"]") + " " + common.HelpDescStyle.Render(desc)
}

// SelectedHabit returns the currently selected habit.
func (m Model) SelectedHabit() common.Habit {
	for _, h := range m.habits {
		if h.ID == m.selected {
			return h
		}
	}
	if len(m.habits) > 0 {
		return m.habits[0]
	}
	return common.Habit{}
}
