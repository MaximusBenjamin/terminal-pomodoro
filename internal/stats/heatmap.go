package stats

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/store"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/theme"
)

// intensityLevel returns 0-4 based on hours.
func intensityLevel(hours float64) int {
	switch {
	case hours <= 0:
		return 0
	case hours < 0.5:
		return 1
	case hours < 1:
		return 2
	case hours < 2:
		return 3
	default:
		return 4
	}
}

// RenderHeatmap renders a GitHub-style heatmap for the current year,
// split into two rows: Jan-Jun on top, Jul-Dec on bottom.
// Empty days show a dim box so the grid structure is always visible.
func RenderHeatmap(data []store.DailyHours, width int) string {
	title := common.TitleStyle.Render("Activity")

	// Build lookup map.
	hoursMap := make(map[string]float64, len(data))
	for _, d := range data {
		hoursMap[d.Date.Format("2006-01-02")] = d.Hours
	}

	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	year := today.Year()

	// First half: Jan 1 - Jun 30
	h1Start := time.Date(year, 1, 1, 0, 0, 0, 0, today.Location())
	h1End := time.Date(year, 6, 30, 0, 0, 0, 0, today.Location())

	// Second half: Jul 1 - Dec 31
	h2Start := time.Date(year, 7, 1, 0, 0, 0, 0, today.Location())
	h2End := time.Date(year, 12, 31, 0, 0, 0, 0, today.Location())

	top := renderHalfYear(hoursMap, h1Start, h1End, today)
	bottom := renderHalfYear(hoursMap, h2Start, h2End, today)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(top)
	sb.WriteString("\n")
	sb.WriteString(bottom)

	return sb.String()
}

type cell struct {
	hours  float64
	date   time.Time
	valid  bool   // within the half-year range
	future bool   // after today
}

func renderHalfYear(hoursMap map[string]float64, start, end, today time.Time) string {
	// Align start to Monday.
	alignedStart := start
	for alignedStart.Weekday() != time.Monday {
		alignedStart = alignedStart.AddDate(0, 0, -1)
	}

	// Build weekly columns.
	var weeks [][]cell
	d := alignedStart
	for !d.After(end) {
		var week []cell
		for dow := 0; dow < 7; dow++ {
			inRange := !d.Before(start) && !d.After(end)
			c := cell{
				date:   d,
				valid:  inRange,
				future: d.After(today),
			}
			if inRange && !c.future {
				c.hours = hoursMap[d.Format("2006-01-02")]
			}
			week = append(week, c)
			d = d.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	// Styles
	emptyStyle := lipgloss.NewStyle().Foreground(theme.ColorBorder)
	futureStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2b3d"))

	levelStyles := make([]lipgloss.Style, len(theme.HeatmapColors))
	for i, c := range theme.HeatmapColors {
		levelStyles[i] = lipgloss.NewStyle().Foreground(c)
	}

	// Month labels.
	monthRow := "    " // space for day labels
	prevMonth := time.Month(0)
	for _, week := range weeks {
		// Use first valid day of the week for month detection.
		var refDate time.Time
		for _, c := range week {
			if c.valid {
				refDate = c.date
				break
			}
		}
		if refDate.IsZero() {
			monthRow += "  "
			continue
		}
		m := refDate.Month()
		if m != prevMonth {
			label := refDate.Format("Jan")
			monthRow += label
			if len(label) < 2 {
				monthRow += " "
			}
			prevMonth = m
		} else {
			monthRow += "  "
		}
	}

	dayLabels := []string{" ", "M", " ", "W", " ", "F", " "}

	var sb strings.Builder
	sb.WriteString(common.MutedStyle.Render(monthRow))
	sb.WriteString("\n")

	for row := 0; row < 7; row++ {
		sb.WriteString(common.MutedStyle.Render(dayLabels[row] + "  "))
		for _, week := range weeks {
			if row >= len(week) {
				sb.WriteString("  ")
				continue
			}
			c := week[row]
			if !c.valid {
				sb.WriteString("  ")
			} else if c.future {
				sb.WriteString(futureStyle.Render("□") + " ")
			} else if c.hours <= 0 {
				sb.WriteString(emptyStyle.Render("□") + " ")
			} else {
				level := intensityLevel(c.hours)
				sb.WriteString(levelStyles[level].Render("■") + " ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
