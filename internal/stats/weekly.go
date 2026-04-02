package stats

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/store"
)

const barMaxHeight = 8

var weekLabels = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Bar column width: each bar is 5 block chars + 1 space gap = 6 chars per slot.
const barWidth = 5
const colWidth = 6

// RenderWeeklyByHabit renders a stacked bar chart colored by habit, Mon-Sun.
func RenderWeeklyByHabit(weekByHabit map[int]store.HabitWeekData, width int) string {
	title := common.TitleStyle.Render("This Week ─ Per Category")

	if len(weekByHabit) == 0 {
		return title + "\n" + common.MutedStyle.Render("  No data")
	}

	type habitInfo struct {
		name  string
		color string
		daily [7]float64
	}
	var habits []habitInfo
	for _, hw := range weekByHabit {
		habits = append(habits, habitInfo{
			name:  hw.HabitName,
			color: hw.Color,
			daily: hw.Daily,
		})
	}
	sort.Slice(habits, func(i, j int) bool {
		return habits[i].name < habits[j].name
	})

	// Totals per day
	totals := make([]float64, 7)
	for _, h := range habits {
		for i := 0; i < 7; i++ {
			totals[i] += h.daily[i]
		}
	}
	maxVal := 0.0
	for _, t := range totals {
		if t > maxVal {
			maxVal = t
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	yStep := maxVal / float64(barMaxHeight)

	for row := barMaxHeight; row >= 1; row-- {
		threshold := float64(row) * yStep
		sb.WriteString(yLabel(row, barMaxHeight, maxVal, yStep))

		for day := 0; day < 7; day++ {
			if totals[day] >= threshold {
				// Find which habit owns this segment
				accumulated := 0.0
				rendered := false
				for _, h := range habits {
					accumulated += h.daily[day]
					if accumulated >= threshold && h.daily[day] > 0 {
						style := lipgloss.NewStyle().Foreground(lipgloss.Color(h.color))
						sb.WriteString(style.Render(strings.Repeat("█", barWidth)))
						rendered = true
						break
					}
				}
				if !rendered {
					sb.WriteString(common.AccentStyle.Render(strings.Repeat("█", barWidth)))
				}
			} else if row == 1 && totals[day] > 0 {
				// Show a half-block so small values still touch the axis
				accumulated := 0.0
				rendered := false
				for _, h := range habits {
					accumulated += h.daily[day]
					if h.daily[day] > 0 {
						style := lipgloss.NewStyle().Foreground(lipgloss.Color(h.color))
						sb.WriteString(style.Render(strings.Repeat("▄", barWidth)))
						rendered = true
						break
					}
				}
				if !rendered {
					sb.WriteString(common.AccentStyle.Render(strings.Repeat("▄", barWidth)))
				}
			} else {
				sb.WriteString(strings.Repeat(" ", barWidth))
			}
			sb.WriteString(" ")
		}
		sb.WriteString("\n")
	}

	writeAxis(&sb, 7)
	writeDayLabels(&sb, weekLabels[:])
	writeValues(&sb, totals)

	// Legend
	sb.WriteString("\n")
	sb.WriteString("     ")
	var legend []string
	for _, h := range habits {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(h.color))
		legend = append(legend, style.Render("██")+" "+common.MutedStyle.Render(h.name))
	}
	sb.WriteString(strings.Join(legend, "  "))

	return sb.String()
}

// yLabel renders the Y-axis label for a given row.
func yLabel(row, maxRow int, maxVal, yStep float64) string {
	if row == maxRow {
		return common.MutedStyle.Render(fmt.Sprintf("%4.1f│", maxVal))
	}
	if row == 1 {
		return common.MutedStyle.Render(fmt.Sprintf("%4.1f│", yStep))
	}
	return common.MutedStyle.Render("    │")
}

// writeAxis writes the X-axis separator line.
func writeAxis(sb *strings.Builder, cols int) {
	sb.WriteString(common.MutedStyle.Render("    └"))
	sb.WriteString(common.MutedStyle.Render(strings.Repeat("─", cols*colWidth)))
	sb.WriteString("\n")
}

// writeDayLabels writes evenly spaced day labels.
func writeDayLabels(sb *strings.Builder, labels []string) {
	sb.WriteString("     ")
	for _, label := range labels {
		sb.WriteString(common.MutedStyle.Render(fmt.Sprintf("%-6s", label)))
	}
	sb.WriteString("\n")
}

// writeValues writes hour values under each bar.
func writeValues(sb *strings.Builder, values []float64) {
	sb.WriteString("     ")
	for _, v := range values {
		if v > 0 {
			sb.WriteString(common.AccentStyle.Render(fmt.Sprintf("%-6s", formatHours(v))))
		} else {
			sb.WriteString(common.MutedStyle.Render("─     "))
		}
	}
	sb.WriteString("\n")
}

func formatHours(h float64) string {
	if h == 0 {
		return "0"
	}
	if h < 0.1 {
		m := int(math.Round(h * 60))
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%.1fh", h)
}
