package stats

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// dataLoadedMsg carries all stats data after loading.
type dataLoadedMsg struct {
	today        float64
	week         float64
	allTime      float64
	todayByHabit []api.HabitBreakdown
	dailyHours   []api.DailyHours
	weekByHabit  map[int]api.HabitWeekData
	loadErr      string
}

// Model is the stats sub-model.
type Model struct {
	client       *api.Client
	today        float64
	week         float64
	allTime      float64
	todayByHabit []api.HabitBreakdown
	dailyHours   []api.DailyHours
	weekByHabit  map[int]api.HabitWeekData
	width        int
	height       int
	loaded         bool
	loadErr        string
	scroll         int // vertical scroll offset
	contentHeight  int // total rendered lines (for clamping)
	weekOffset     int // 0 = current week, -1 = last week, etc.
}

// New creates a new stats model.
func New(c *api.Client) Model {
	return Model{
		client: c,
		width:  80,
	}
}

// Init returns a command that loads all stats data.
func (m Model) Init() tea.Cmd {
	return m.loadData()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.today = msg.today
		m.week = msg.week
		m.allTime = msg.allTime
		m.todayByHabit = msg.todayByHabit
		m.dailyHours = msg.dailyHours
		m.weekByHabit = msg.weekByHabit
		m.loaded = true
		m.loadErr = msg.loadErr
		m.contentHeight = m.computeContentHeight()
	case common.StatsRefreshMsg:
		return m, m.loadData()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll += 3
			maxScroll := m.contentHeight - m.height
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll > maxScroll {
				m.scroll = maxScroll
			}
		case "k", "up":
			m.scroll -= 3
			if m.scroll < 0 {
				m.scroll = 0
			}
		case "left":
			m.weekOffset--
			return m, m.loadData()
		case "right":
			if m.weekOffset < 0 {
				m.weekOffset++
				return m, m.loadData()
			}
		case "0":
			m.weekOffset = 0
			return m, m.loadData()
		}
	}
	return m, nil
}

// View renders the stats view.
func (m Model) View() string {
	if !m.loaded {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			common.MutedStyle.Render("Loading stats..."))
	}

	if m.loadErr != "" {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			common.OvertimeStyle.Render("⚠ "+m.loadErr))
	}

	center := func(s string) string {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(s)
	}

	content := m.renderContent()
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in viewport, center vertically, no scroll
	if totalLines <= m.height {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	// Clamp scroll
	scroll := m.scroll
	maxScroll := totalLines - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Slice the visible window
	end := scroll + m.height
	if end > totalLines {
		end = totalLines
	}
	visible := strings.Join(lines[scroll:end], "\n")

	// Scroll indicators
	if scroll > 0 {
		vlines := strings.Split(visible, "\n")
		vlines[0] = center(common.MutedStyle.Render("▲ scroll up (k)"))
		visible = strings.Join(vlines, "\n")
	}
	if end < totalLines {
		vlines := strings.Split(visible, "\n")
		vlines[len(vlines)-1] = center(common.MutedStyle.Render("▼ scroll down (j)"))
		visible = strings.Join(vlines, "\n")
	}

	return visible
}

func (m Model) renderTodayByHabit() string {
	title := common.TitleStyle.Render("Today")

	if len(m.todayByHabit) == 0 {
		return title + "\n" + common.MutedStyle.Render("  No sessions yet")
	}

	var lines []string
	lines = append(lines, title)
	for _, hb := range m.todayByHabit {
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(hb.Color))
		name := nameStyle.Render(fmt.Sprintf("  %-14s", hb.HabitName))
		hours := common.AccentStyle.Render(fmtDuration(hb.Hours))
		lines = append(lines, name+" "+hours)
	}
	return strings.Join(lines, "\n")
}

func (m Model) computeContentHeight() int {
	// Render content to count lines accurately
	content := m.renderContent()
	return strings.Count(content, "\n") + 1
}

func (m Model) renderContent() string {
	center := func(s string) string {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(s)
	}

	var sections []string
	sections = append(sections, RenderSummary(m.today, m.week, m.allTime, m.width))
	sections = append(sections, "")

	// Today breakdown + Habit Tracker side by side
	todayPanel := m.renderTodayByHabit()
	trackerPanel := RenderHabitTracker(m.weekByHabit)
	gap := "      "
	sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, todayPanel, gap, trackerPanel)
	sections = append(sections, center(sideBySide))
	sections = append(sections, "")

	sections = append(sections, center(RenderHeatmap(m.dailyHours, m.width)))

	// Week date range header
	monday := api.MondayOfWeek(m.weekOffset)
	sunday := monday.AddDate(0, 0, 6)
	weekLabel := monday.Format("Jan 2") + " – " + sunday.Format("Jan 2, 2006")
	if m.weekOffset == 0 {
		weekLabel += "  (this week)"
	}
	sections = append(sections, center(common.TitleStyle.Render(weekLabel)))
	sections = append(sections, center(RenderWeeklyByHabit(m.weekByHabit, m.width)))
	sections = append(sections, "")

	// Navigation hint
	hint := common.HelpKeyStyle.Render("←/→") + " " + common.HelpDescStyle.Render("prev/next week") +
		"  " + common.HelpKeyStyle.Render("0") + " " + common.HelpDescStyle.Render("this week")
	sections = append(sections, center(hint))

	return strings.Join(sections, "\n")
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		var d dataLoadedMsg
		var err error

		d.today, err = m.client.TodayHours()
		if err != nil {
			d.loadErr = "Failed to load stats"
			return d
		}
		d.week, _ = m.client.WeekHours()
		d.allTime, _ = m.client.AllTimeHours()
		d.todayByHabit, _ = m.client.TodayHoursByHabit()
		d.dailyHours, _ = m.client.DailyHoursRange(365)
		d.weekByHabit, _ = m.client.WeekDailyByHabit(m.weekOffset)

		return d
	}
}
