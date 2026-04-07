package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/habits"
	logview "github.com/MaximusBenjamin/terminal-pomodoro/internal/log"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/settings"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/stats"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/timer"
)

const numTabs = 5

// Model is the top-level Bubble Tea model that orchestrates all views.
type Model struct {
	activeTab common.Tab
	timer     timer.Model
	stats     stats.Model
	habits    habits.Model
	log       logview.Model
	settings  settings.Model
	client    *api.Client
	width     int
	height    int
	ready     bool
}

// New creates the top-level application model with all sub-models.
func New(c *api.Client) Model {
	h := habits.New(c)
	t := timer.New(c)
	se := settings.New(c)
	st := stats.New(c, se.Leeway())
	l := logview.New(c)

	// Set the first habit as the default for the timer.
	sel := h.SelectedHabit()
	if sel.ID != 0 {
		t.SetHabit(sel)
	}

	return Model{
		activeTab: common.TimerTab,
		timer:     t,
		stats:     st,
		habits:    h,
		log:       l,
		settings:  se,
		client:    c,
	}
}

// Init initializes all sub-models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.timer.Init(),
		m.stats.Init(),
		m.habits.Init(),
		m.log.Init(),
		m.settings.Init(),
	)
}

// Update handles messages and routes them to the appropriate sub-model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		inner := tea.WindowSizeMsg{
			Width:  max(0, msg.Width-8),
			Height: max(0, msg.Height-8),
		}
		var cmds []tea.Cmd
		var cmd tea.Cmd

		m.timer, cmd = m.timer.Update(inner)
		cmds = append(cmds, cmd)
		m.stats, cmd = m.stats.Update(inner)
		cmds = append(cmds, cmd)
		m.habits, cmd = m.habits.Update(inner)
		cmds = append(cmds, cmd)
		m.log, cmd = m.log.Update(inner)
		cmds = append(cmds, cmd)
		m.settings, cmd = m.settings.Update(inner)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			if m.activeTab == common.HabitsTab {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			if m.timer.IsConfirming() {
				break
			}
			return m, tea.Quit

		case "l":
			if m.timer.IsConfirming() {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			m.activeTab = (m.activeTab + 1) % numTabs
			return m, m.refreshTab()

		case "h":
			if m.timer.IsConfirming() {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			m.activeTab = (m.activeTab + numTabs - 1) % numTabs
			return m, m.refreshTab()
		}

	case settings.LeewayLoadedMsg:
		// Leeway loaded from Supabase on startup — update settings model then reload stats
		var settingsCmd tea.Cmd
		m.settings, settingsCmd = m.settings.Update(msg)
		m.stats.SetLeeway(m.settings.Leeway())
		var statsCmd tea.Cmd
		m.stats, statsCmd = m.stats.Update(common.StatsRefreshMsg{})
		return m, tea.Batch(settingsCmd, statsCmd)

	case common.HabitSelectedMsg:
		m.timer.RefreshHabits()
		h := common.Habit{
			ID:    msg.ID,
			Name:  msg.Name,
			Color: msg.Color,
		}
		m.timer.SetHabit(h)
		m.activeTab = common.TimerTab
		return m, nil

	case common.SessionEndMsg:
		// Session ended; refresh stats and log.
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.stats, cmd = m.stats.Update(common.StatsRefreshMsg{})
		cmds = append(cmds, cmd)
		m.log, cmd = m.log.Update(common.LogRefreshMsg{})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case common.StatsRefreshMsg:
		// Refresh both stats and log when stats refresh is triggered (e.g. from log edits)
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.stats, cmd = m.stats.Update(msg)
		cmds = append(cmds, cmd)
		m.log, cmd = m.log.Update(common.LogRefreshMsg{})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// For key messages, only forward to the active tab.
	if _, isKey := msg.(tea.KeyMsg); isKey {
		return m.updateActiveTab(msg)
	}

	// For async results, forward to all sub-models.
	return m.updateAll(msg)
}

// refreshTab sends a refresh message to the newly active tab so it picks up changes.
func (m Model) refreshTab() tea.Cmd {
	switch m.activeTab {
	case common.LogTab:
		return func() tea.Msg { return common.LogRefreshMsg{} }
	case common.StatsTab:
		return func() tea.Msg { return common.StatsRefreshMsg{} }
	}
	return nil
}

func (m Model) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case common.TimerTab:
		m.timer, cmd = m.timer.Update(msg)
	case common.StatsTab:
		m.stats, cmd = m.stats.Update(msg)
	case common.HabitsTab:
		m.habits, cmd = m.habits.Update(msg)
	case common.LogTab:
		m.log, cmd = m.log.Update(msg)
	case common.SettingsTab:
		prevLeeway := m.settings.Leeway()
		m.settings, cmd = m.settings.Update(msg)
		// Reload stats if leeway changed
		if m.settings.Leeway() != prevLeeway {
			m.stats.SetLeeway(m.settings.Leeway())
			var statsCmd tea.Cmd
			m.stats, statsCmd = m.stats.Update(common.StatsRefreshMsg{})
			cmd = tea.Batch(cmd, statsCmd)
		}
	}
	return m, cmd
}

func (m Model) updateAll(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.timer, cmd = m.timer.Update(msg)
	cmds = append(cmds, cmd)
	m.stats, cmd = m.stats.Update(msg)
	cmds = append(cmds, cmd)
	m.habits, cmd = m.habits.Update(msg)
	cmds = append(cmds, cmd)
	m.log, cmd = m.log.Update(msg)
	cmds = append(cmds, cmd)
	m.settings, cmd = m.settings.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the full application.
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Build tab bar.
	tabs := []struct {
		label string
		tab   common.Tab
	}{
		{"Timer", common.TimerTab},
		{"Stats", common.StatsTab},
		{"Habits", common.HabitsTab},
		{"Log", common.LogTab},
		{"Settings", common.SettingsTab},
	}

	var tabParts []string
	for _, t := range tabs {
		if t.tab == m.activeTab {
			tabParts = append(tabParts, common.ActiveTabStyle.Render(t.label))
		} else {
			tabParts = append(tabParts, common.InactiveTabStyle.Render(t.label))
		}
	}
	nav := common.MutedStyle.Render("  ← h  l →")
	tabParts = append(tabParts, nav)
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabParts...)

	// Separator line.
	innerWidth := max(0, m.width-6)
	separator := common.MutedStyle.Render(strings.Repeat("─", innerWidth))

	// Active tab content.
	var content string
	switch m.activeTab {
	case common.TimerTab:
		content = m.timer.View()
	case common.StatsTab:
		content = m.stats.View()
	case common.HabitsTab:
		content = m.habits.View()
	case common.LogTab:
		content = m.log.View()
	case common.SettingsTab:
		content = m.settings.View()
	}

	// Compose the inner layout.
	inner := lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		separator,
		content,
	)

	// Wrap in panel with border.
	panel := common.PanelStyle.
		Width(max(0, m.width-2)).
		Height(max(0, m.height-2)).
		Render(inner)

	return panel
}
