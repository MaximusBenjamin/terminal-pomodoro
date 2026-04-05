package log

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// ParsedSession is the result of parsing a natural language session input.
type ParsedSession struct {
	DurationSecs int
	HabitID      int
	HabitName    string
	StartTime    time.Time
	EndTime      time.Time
}

// Time pattern: "1pm", "1:30pm", "1:30 pm", "13:00", "9 am"
var timePattern = `\d{1,2}(?::\d{2})?\s*(?:am|pm)?`

// Standalone time: matches a single time expression (for "30m math 1pm" style)
var standaloneTimeRe = regexp.MustCompile(`(?i)\b(\d{1,2}(?::\d{2})?)\s*(am|pm)\b`)

// Duration pattern: "30m", "2h", "1h30m", "1.5h", "45 minutes", "2 hours"
// Word boundary \b after unit prevents matching "25 m" from "25 march"
var durationRe = regexp.MustCompile(
	`(?i)(\d+(?:\.\d+)?)\s*(m|min|mins|minutes?|h|hrs?|hours?)\b(?:\s*(\d+)\s*(m|min|mins|minutes?)\b)?`)

// Time range: extracts two time expressions separated by -, –, to, til, till, until
var timeRangeRe = regexp.MustCompile(
	`(?i)(` + timePattern + `)\s*(?:[-–]|to|til|till|until)\s*(` + timePattern + `)`)

var dateWords = map[string]bool{
	"today": true, "yesterday": true,
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true,
	"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true,
}

// Matches date-like tokens: 01/04/2026, 2026-04-01, 1/4, 01-04
var dateTokenRe = regexp.MustCompile(`^\d{1,4}[/\-.]\d{1,2}(?:[/\-.]\d{2,4})?$`)

var monthNames = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

func isDateToken(w string) bool {
	lower := strings.ToLower(w)
	if dateWords[lower] {
		return true
	}
	if dateTokenRe.MatchString(w) {
		return true
	}
	if _, ok := monthNames[lower]; ok {
		return true
	}
	return false
}

func isMonthName(w string) bool {
	_, ok := monthNames[strings.ToLower(w)]
	return ok
}

func isDayNumber(w string) bool {
	w = strings.TrimRight(w, "stndrdth")
	n, err := strconv.Atoi(w)
	return err == nil && n >= 1 && n <= 31
}

// ParseSessionInput parses flexible natural language session input.
// Accepts many formats:
//
//	"30m math", "2h programming", "1.5h finance"
//	"1pm to 2pm programming", "1:30pm - 2:30 pm math"
//	"from 1pm to 2pm programming yesterday"
//	"programming 30m yesterday"
//	"math 1pm to 1:30pm"
//
// The parser extracts time ranges, durations, habit names, and date modifiers
// regardless of their order in the input.
func ParseSessionInput(input string, habits []common.Habit) (ParsedSession, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return ParsedSession{}, fmt.Errorf("empty input")
	}

	// Normalize: strip "from" prefix
	cleaned := regexp.MustCompile(`(?i)\bfrom\b`).ReplaceAllString(input, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ") // collapse whitespace

	// Strip date tokens before time/duration matching to avoid
	// e.g. "2026-04-01" being parsed as a time range "2026" - "04"
	strippedForMatch := cleaned
	for _, w := range strings.Fields(cleaned) {
		if dateTokenRe.MatchString(w) {
			strippedForMatch = strings.Replace(strippedForMatch, w, " ", 1)
		}
	}
	strippedForMatch = strings.Join(strings.Fields(strippedForMatch), " ")

	// 1. Try to extract a time range (from input with date tokens removed)
	if m := timeRangeRe.FindStringSubmatch(strippedForMatch); m != nil {
		remaining := strings.TrimSpace(timeRangeRe.ReplaceAllString(cleaned, " "))
		habit, dateStr := extractHabitAndDate(remaining, habits)
		if habit.ID == 0 {
			return ParsedSession{}, habitError(remaining, habits)
		}
		return buildTimeRange(m[1], m[2], habit, dateStr)
	}

	// 2. Try to extract a duration (from input with date tokens removed)
	if m := durationRe.FindStringSubmatch(strippedForMatch); m != nil {
		remaining := strings.TrimSpace(durationRe.ReplaceAllString(cleaned, " "))
		secs := parseDurationMatch(m)
		if secs <= 0 {
			return ParsedSession{}, fmt.Errorf("duration must be positive")
		}

		// Check if there's a standalone start time in the remaining text (e.g. "programming 1pm")
		if tm := standaloneTimeRe.FindStringSubmatch(remaining); tm != nil {
			afterTimeRemoved := strings.TrimSpace(standaloneTimeRe.ReplaceAllString(remaining, " "))
			habit, dateStr := extractHabitAndDate(afterTimeRemoved, habits)
			if habit.ID == 0 {
				return ParsedSession{}, habitError(afterTimeRemoved, habits)
			}
			startStr := tm[1] + tm[2] // e.g. "1pm" or "1:30pm"
			return buildDurationFromStart(secs, startStr, habit, dateStr)
		}

		// Also check for 24-hour time like "13:00"
		time24Re := regexp.MustCompile(`\b(\d{1,2}:\d{2})\b`)
		if tm := time24Re.FindStringSubmatch(remaining); tm != nil {
			// Make sure it's not part of a date
			candidate := tm[1]
			pt, err := parseTimeOfDay(candidate)
			if err == nil && pt.hour <= 23 && pt.min <= 59 {
				afterTimeRemoved := strings.TrimSpace(time24Re.ReplaceAllString(remaining, " "))
				habit, dateStr := extractHabitAndDate(afterTimeRemoved, habits)
				if habit.ID == 0 {
					return ParsedSession{}, habitError(afterTimeRemoved, habits)
				}
				return buildDurationFromStart(secs, candidate, habit, dateStr)
			}
		}

		habit, dateStr := extractHabitAndDate(remaining, habits)
		if habit.ID == 0 {
			return ParsedSession{}, habitError(remaining, habits)
		}
		return buildDuration(secs, habit, dateStr)
	}

	return ParsedSession{}, fmt.Errorf("could not parse. Examples: \"30m math\", \"1pm to 2pm programming\", \"1:30pm-2:30pm math yesterday\"")
}

// extractHabitAndDate takes the leftover words (after removing time/duration)
// and finds the habit name and optional date modifier.
func extractHabitAndDate(remaining string, habits []common.Habit) (common.Habit, string) {
	words := strings.Fields(remaining)
	var dateParts []string
	var habitWords []string

	for i := 0; i < len(words); i++ {
		w := words[i]
		lower := strings.ToLower(w)

		if dateWords[lower] || dateTokenRe.MatchString(w) {
			dateParts = append(dateParts, w)
		} else if isMonthName(w) {
			// Grab "april 1" or "1 april" patterns
			dateParts = append(dateParts, w)
			if i+1 < len(words) && isDayNumber(words[i+1]) {
				dateParts = append(dateParts, words[i+1])
				i++
			}
		} else if isDayNumber(w) && i+1 < len(words) && isMonthName(words[i+1]) {
			// "1 april" pattern
			dateParts = append(dateParts, w, words[i+1])
			i++
		} else if lower != "" {
			habitWords = append(habitWords, w)
		}
	}

	dateStr := strings.Join(dateParts, " ")

	if len(habitWords) == 0 {
		return common.Habit{}, dateStr
	}

	// Try matching the full remaining string first, then individual words
	fullName := strings.Join(habitWords, " ")
	if h, err := matchHabit(fullName, habits); err == nil {
		return h, dateStr
	}

	// Try each word individually
	for _, w := range habitWords {
		if h, err := matchHabit(w, habits); err == nil {
			return h, dateStr
		}
	}

	return common.Habit{}, dateStr
}

func habitError(remaining string, habits []common.Habit) error {
	words := strings.Fields(remaining)
	var habitPart string
	for _, w := range words {
		if !dateWords[strings.ToLower(w)] {
			habitPart = w
			break
		}
	}
	names := make([]string, len(habits))
	for i, h := range habits {
		names[i] = h.Name
	}
	if habitPart == "" {
		return fmt.Errorf("no habit specified. Available: %s", strings.Join(names, ", "))
	}
	return fmt.Errorf("no habit %q. Available: %s", habitPart, strings.Join(names, ", "))
}

func parseDurationMatch(m []string) int {
	amount, _ := strconv.ParseFloat(m[1], 64)
	unit := strings.ToLower(m[2][:1])

	var secs int
	switch unit {
	case "h":
		secs = int(amount * 3600)
	case "m":
		secs = int(amount * 60)
	}

	// Handle compound like "1h30m"
	if m[3] != "" {
		extra, _ := strconv.Atoi(m[3])
		secs += extra * 60
	}

	return secs
}

func buildTimeRange(startStr, endStr string, habit common.Habit, dateStr string) (ParsedSession, error) {
	baseDate := resolveDate(dateStr)

	startTime, err := parseTimeOfDay(startStr)
	if err != nil {
		return ParsedSession{}, fmt.Errorf("invalid start time %q", startStr)
	}
	endTime, err := parseTimeOfDay(endStr)
	if err != nil {
		return ParsedSession{}, fmt.Errorf("invalid end time %q", endStr)
	}

	start := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
		startTime.hour, startTime.min, 0, 0, time.Local)
	end := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
		endTime.hour, endTime.min, 0, 0, time.Local)

	// If end is before start, assume it crosses midnight (e.g. 23:36-0:09)
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}

	dur := int(end.Sub(start).Seconds())
	return ParsedSession{
		DurationSecs: dur,
		HabitID:      habit.ID,
		HabitName:    habit.Name,
		StartTime:    start,
		EndTime:      end,
	}, nil
}

func buildDurationFromStart(secs int, startStr string, habit common.Habit, dateStr string) (ParsedSession, error) {
	baseDate := resolveDate(dateStr)
	st, err := parseTimeOfDay(startStr)
	if err != nil {
		return ParsedSession{}, fmt.Errorf("invalid start time %q", startStr)
	}

	start := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
		st.hour, st.min, 0, 0, time.Local)
	end := start.Add(time.Duration(secs) * time.Second)

	return ParsedSession{
		DurationSecs: secs,
		HabitID:      habit.ID,
		HabitName:    habit.Name,
		StartTime:    start,
		EndTime:      end,
	}, nil
}

func buildDuration(secs int, habit common.Habit, dateStr string) (ParsedSession, error) {
	baseDate := resolveDate(dateStr)
	now := time.Now().Local()

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	base := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local)

	var end time.Time
	if base.Equal(today) {
		end = now
	} else {
		end = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 12, 0, 0, 0, time.Local)
	}
	start := end.Add(-time.Duration(secs) * time.Second)

	return ParsedSession{
		DurationSecs: secs,
		HabitID:      habit.ID,
		HabitName:    habit.Name,
		StartTime:    start,
		EndTime:      end,
	}, nil
}

type parsedTime struct {
	hour int
	min  int
}

// parseTimeOfDay parses flexible time formats into hour and minute.
func parseTimeOfDay(s string) (parsedTime, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Normalize spaces: "1:30 pm" → "1:30pm"
	s = strings.ReplaceAll(s, " am", "am")
	s = strings.ReplaceAll(s, " pm", "pm")
	s = strings.TrimSpace(s)

	// Try 12-hour with minutes: "1:30pm"
	if t, err := time.Parse("3:04pm", s); err == nil {
		return parsedTime{t.Hour(), t.Minute()}, nil
	}
	// Try 12-hour no minutes: "1pm"
	if t, err := time.Parse("3pm", s); err == nil {
		return parsedTime{t.Hour(), 0}, nil
	}
	// Try 24-hour: "13:00"
	if t, err := time.Parse("15:04", s); err == nil {
		return parsedTime{t.Hour(), t.Minute()}, nil
	}
	// Try bare number as hour: "13"
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 23 {
		return parsedTime{n, 0}, nil
	}

	return parsedTime{}, fmt.Errorf("cannot parse time %q", s)
}

// matchHabit finds a habit by case-insensitive exact or prefix match.
func matchHabit(name string, habits []common.Habit) (common.Habit, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return common.Habit{}, fmt.Errorf("empty habit name")
	}

	// Exact match first
	for _, h := range habits {
		if strings.ToLower(h.Name) == lower {
			return h, nil
		}
	}

	// Prefix match
	var matches []common.Habit
	for _, h := range habits {
		if strings.HasPrefix(strings.ToLower(h.Name), lower) {
			matches = append(matches, h)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return common.Habit{}, fmt.Errorf("ambiguous: %q matches %s", name, strings.Join(names, ", "))
	}

	names := make([]string, len(habits))
	for i, h := range habits {
		names[i] = h.Name
	}
	return common.Habit{}, fmt.Errorf("no habit %q. Available: %s", name, strings.Join(names, ", "))
}

// resolveDate turns date expressions into a time.Time.
// Supports: "yesterday", "monday", "01/04/2026", "2026-04-01", "1/4", "april 1", "1 april"
func resolveDate(s string) time.Time {
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "today" {
		return today
	}
	if s == "yesterday" {
		return today.AddDate(0, 0, -1)
	}

	// Weekday names
	weekdays := map[string]time.Weekday{
		"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
		"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday, "sunday": time.Sunday,
		"mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
		"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday, "sun": time.Sunday,
	}
	if target, ok := weekdays[s]; ok {
		d := today
		for i := 0; i < 7; i++ {
			d = d.AddDate(0, 0, -1)
			if d.Weekday() == target {
				return d
			}
		}
	}

	// Numeric date formats: DD/MM/YYYY, DD-MM-YYYY, DD.MM.YYYY, DD/MM, YYYY-MM-DD
	dateFormats := []string{
		"02/01/2006", "02-01-2006", "02.01.2006", // DD/MM/YYYY
		"2/1/2006", "2-1-2006",                    // D/M/YYYY
		"2006-01-02",                               // YYYY-MM-DD (ISO)
		"02/01", "2/1",                             // DD/MM (current year)
		"02-01", "2-1",                             // DD-MM (current year)
	}
	for _, fmt := range dateFormats {
		if t, err := time.Parse(fmt, s); err == nil {
			y := t.Year()
			if y == 0 {
				y = today.Year()
			}
			return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
		}
	}

	// "april 1", "1 april", "apr 1", "1 apr"
	words := strings.Fields(s)
	if len(words) == 2 {
		// Try month + day
		if m, ok := monthNames[words[0]]; ok {
			day := strings.TrimRight(words[1], "stndrdth")
			if d, err := strconv.Atoi(day); err == nil && d >= 1 && d <= 31 {
				return time.Date(today.Year(), m, d, 0, 0, 0, 0, now.Location())
			}
		}
		// Try day + month
		if m, ok := monthNames[words[1]]; ok {
			day := strings.TrimRight(words[0], "stndrdth")
			if d, err := strconv.Atoi(day); err == nil && d >= 1 && d <= 31 {
				return time.Date(today.Year(), m, d, 0, 0, 0, 0, now.Location())
			}
		}
	}
	// Single month name (means 1st of that month)
	if m, ok := monthNames[s]; ok {
		return time.Date(today.Year(), m, 1, 0, 0, 0, 0, now.Location())
	}

	return today
}
