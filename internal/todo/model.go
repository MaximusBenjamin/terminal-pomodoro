package todo

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

type viewMode int

const (
	modeNormal viewMode = iota
	modeAdd
	modeEdit
	modeConfirmDelete
)

type todosLoadedMsg struct {
	todos []common.Todo
	err   string
}

// Model is the Bubble Tea sub-model for the todo view.
type Model struct {
	client      *api.Client
	todos       []common.Todo
	cursor      int
	mode        viewMode
	input       textinput.Model
	editID      int
	viewingDate time.Time // date-only; result of api.EffectiveDate
	width       int
	height      int
	lastErr     string
}

// New creates a new todo Model. Loads today's todos eagerly so the view has
// data before Init runs (matches the habits package pattern).
func New(c *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. review chapter 3"
	ti.CharLimit = 200
	ti.Width = 50

	m := Model{
		client:      c,
		input:       ti,
		viewingDate: api.EffectiveDate(time.Now().Local()),
	}

	if todos, err := c.ListTodos(m.viewingDate); err == nil {
		m.todos = todos
	}
	return m
}

// Init returns a command that loads today's todos.
func (m Model) Init() tea.Cmd {
	return m.loadTodos
}

func (m Model) loadTodos() tea.Msg {
	todos, err := m.client.ListTodos(m.viewingDate)
	if err != nil {
		return todosLoadedMsg{err: "Failed to load todos"}
	}
	return todosLoadedMsg{todos: todos}
}

// IsEditing returns true when the input is focused (add or edit).
func (m Model) IsEditing() bool {
	return m.mode == modeAdd || m.mode == modeEdit
}

// IsConfirming returns true when the delete confirm prompt is open.
func (m Model) IsConfirming() bool {
	return m.mode == modeConfirmDelete
}

func (m Model) isToday() bool {
	return m.viewingDate.Equal(api.EffectiveDate(time.Now().Local()))
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case todosLoadedMsg:
		if msg.err != "" {
			m.lastErr = msg.err
			return m, nil
		}
		m.lastErr = ""
		m.todos = msg.todos
		if m.cursor >= len(m.todos) {
			m.cursor = max(0, len(m.todos)-1)
		}
		return m, nil

	case common.TodoRefreshMsg:
		return m, m.loadTodos

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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
		if len(m.todos) > 0 {
			m.cursor = (m.cursor + 1) % len(m.todos)
		}
		return m, nil

	case "k", "up":
		if len(m.todos) > 0 {
			m.cursor = (m.cursor - 1 + len(m.todos)) % len(m.todos)
		}
		return m, nil

	case "left":
		m.viewingDate = m.viewingDate.AddDate(0, 0, -1)
		m.cursor = 0
		return m, m.loadTodos

	case "right":
		today := api.EffectiveDate(time.Now().Local())
		if m.viewingDate.Before(today) {
			m.viewingDate = m.viewingDate.AddDate(0, 0, 1)
			m.cursor = 0
			return m, m.loadTodos
		}
		return m, nil

	case "0":
		today := api.EffectiveDate(time.Now().Local())
		if !m.viewingDate.Equal(today) {
			m.viewingDate = today
			m.cursor = 0
			return m, m.loadTodos
		}
		return m, nil
	}

	// Past-day view is read-only. The keys below only apply on today.
	if !m.isToday() {
		return m, nil
	}

	switch msg.String() {
	case "a":
		m.mode = modeAdd
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink

	case " ":
		if len(m.todos) > 0 {
			t := m.todos[m.cursor]
			newCompleted := !t.Completed
			id := t.ID
			return m, func() tea.Msg {
				if err := m.client.ToggleTodo(id, newCompleted); err != nil {
					return todosLoadedMsg{todos: m.todos, err: "Failed to toggle"}
				}
				return m.loadTodos()
			}
		}
		return m, nil

	case "e":
		if len(m.todos) > 0 {
			t := m.todos[m.cursor]
			m.mode = modeEdit
			m.editID = t.ID
			m.input.Reset()
			m.input.SetValue(t.Text)
			m.input.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "d":
		if len(m.todos) > 0 {
			m.mode = modeConfirmDelete
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg, isEdit bool) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			// No-op; stay in input mode.
			return m, nil
		}
		if isEdit {
			id := m.editID
			m.mode = modeNormal
			m.input.Reset()
			m.input.Blur()
			return m, func() tea.Msg {
				if err := m.client.EditTodo(id, text); err != nil {
					return todosLoadedMsg{todos: m.todos, err: "Failed to edit"}
				}
				return m.loadTodos()
			}
		}
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		return m, func() tea.Msg {
			if _, err := m.client.AddTodo(text); err != nil {
				return todosLoadedMsg{todos: m.todos, err: "Failed to add"}
			}
			return m.loadTodos()
		}

	case "esc":
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if len(m.todos) > 0 {
			id := m.todos[m.cursor].ID
			m.mode = modeNormal
			return m, func() tea.Msg {
				if err := m.client.DeleteTodo(id); err != nil {
					return todosLoadedMsg{todos: m.todos, err: "Failed to delete"}
				}
				return m.loadTodos()
			}
		}
		m.mode = modeNormal
		return m, nil

	case "x", "esc", "n":
		m.mode = modeNormal
		return m, nil
	}
	return m, nil
}

// View renders the todo view.
func (m Model) View() string {
	var b strings.Builder

	// Header.
	b.WriteString(common.TitleStyle.Render("Todo"))
	b.WriteString("  ")
	if m.isToday() {
		b.WriteString(common.MutedStyle.Render("Today"))
	} else {
		label := m.viewingDate.Format("Mon Jan 2")
		b.WriteString(common.MutedStyle.Render(label + "  (press 0 for today)"))
	}
	b.WriteString("\n\n")

	// List.
	if len(m.todos) == 0 {
		if m.isToday() {
			b.WriteString(common.MutedStyle.Render("  No todos yet. Press [a] to add one."))
		} else {
			b.WriteString(common.MutedStyle.Render("  No todos this day."))
		}
		b.WriteString("\n")
	} else {
		for i, t := range m.todos {
			cursor := "  "
			if i == m.cursor {
				cursor = common.AccentStyle.Render("▸ ")
			}
			box := "[ ]"
			if t.Completed {
				box = "[✓]"
			}
			textStyle := lipgloss.NewStyle()
			if t.Completed {
				textStyle = common.MutedStyle.Strikethrough(true)
			}
			line := fmt.Sprintf("%s%s %s", cursor, box, textStyle.Render(t.Text))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Input / confirm.
	if m.mode == modeAdd {
		b.WriteString("\n  ")
		b.WriteString(common.MutedStyle.Render("New: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else if m.mode == modeEdit {
		b.WriteString("\n  ")
		b.WriteString(common.MutedStyle.Render("Edit: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else if m.mode == modeConfirmDelete && len(m.todos) > 0 {
		t := m.todos[m.cursor]
		b.WriteString("\n")
		b.WriteString(common.WarningStyle.Render(fmt.Sprintf("  Delete \"%s\"?", t.Text)))
		b.WriteString("\n")
		b.WriteString("  " + helpEntry("y", "delete") + "  " + helpEntry("x", "cancel"))
		b.WriteString("\n")
	}

	if m.lastErr != "" {
		b.WriteString("\n")
		b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.lastErr))
		b.WriteString("\n")
	}

	// Help line.
	b.WriteString("\n")
	if m.mode == modeAdd || m.mode == modeEdit {
		b.WriteString("  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel"))
	} else if m.mode == modeConfirmDelete {
		// help already rendered above
	} else if m.isToday() {
		b.WriteString("  " +
			helpEntry("j/k", "navigate") + "  " +
			helpEntry("a", "add") + "  " +
			helpEntry("space", "toggle") + "  " +
			helpEntry("e", "edit") + "  " +
			helpEntry("d", "delete") + "  " +
			helpEntry("←/→", "day") + "  " +
			helpEntry("0", "today"))
	} else {
		b.WriteString("  " +
			helpEntry("j/k", "navigate") + "  " +
			helpEntry("←/→", "day") + "  " +
			helpEntry("0", "today"))
	}

	return b.String()
}

func helpEntry(key, desc string) string {
	return common.HelpKeyStyle.Render("["+key+"]") + " " + common.HelpDescStyle.Render(desc)
}
