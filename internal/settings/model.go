package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// SavedMsg is sent after settings are saved.
type SavedMsg struct{}

// Model is the Bubble Tea sub-model for the settings view.
type Model struct {
	settings Settings
	lastErr  string
	width    int
	height   int
}

// New creates a new settings model, loading from disk.
func New() Model {
	s, _ := Load()
	return Model{settings: s}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles key messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "+", "=", "right", "l":
			if m.settings.LeewayDaysPerWeek < 7 {
				m.settings.LeewayDaysPerWeek++
				m.save()
			}
		case "-", "left", "h":
			if m.settings.LeewayDaysPerWeek > 0 {
				m.settings.LeewayDaysPerWeek--
				m.save()
			}
		}
	}
	return m, nil
}

func (m *Model) save() {
	if err := Save(m.settings); err != nil {
		m.lastErr = "Failed to save settings"
	} else {
		m.lastErr = ""
	}
}

// Leeway returns the current leeway setting (for use by other models).
func (m Model) Leeway() int {
	return m.settings.LeewayDaysPerWeek
}

// View renders the settings page.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(common.TitleStyle.Render("Settings"))
	b.WriteString("\n\n")

	// Leeway setting
	b.WriteString(common.AccentStyle.Render("  Streak leeway days per week"))
	b.WriteString("\n")
	b.WriteString(common.MutedStyle.Render("  Days you can miss each week without breaking your streak."))
	b.WriteString("\n\n")

	// Value display with +/- controls
	leeway := m.settings.LeewayDaysPerWeek
	days := 7 - leeway
	b.WriteString(fmt.Sprintf("  %s  %s  %s",
		common.HelpKeyStyle.Render("[-]"),
		common.AccentStyle.Render(fmt.Sprintf("%d", leeway)),
		common.HelpKeyStyle.Render("[+]"),
	))
	b.WriteString("\n\n")

	// Explanation
	if leeway == 0 {
		b.WriteString(common.MutedStyle.Render("  You must complete all habits every day."))
	} else if leeway == 7 {
		b.WriteString(common.MutedStyle.Render("  No streak tracking (always leeway)."))
	} else {
		b.WriteString(common.MutedStyle.Render(fmt.Sprintf("  Target: all habits %d/7 days per week.", days)))
	}
	b.WriteString("\n\n")

	if m.lastErr != "" {
		b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.lastErr))
		b.WriteString("\n\n")
	}

	b.WriteString("  " + helpEntry("-/+", "adjust leeway"))

	return b.String()
}

func helpEntry(key, desc string) string {
	return common.HelpKeyStyle.Render("["+key+"]") + " " + common.HelpDescStyle.Render(desc)
}
