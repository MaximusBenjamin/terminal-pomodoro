package common

import "time"

// SessionEndMsg is sent when a timer session ends (user stops or timer completes).
type SessionEndMsg struct {
	HabitID         int
	PlannedMinutes  int
	ActualSeconds   int
	OvertimeSeconds int
	Completed       bool
}

// HabitSelectedMsg is sent when the user selects a habit.
type HabitSelectedMsg struct {
	ID    int
	Name  string
	Color string
}

// StatsRefreshMsg triggers a stats data reload.
type StatsRefreshMsg struct{}

// Habit represents a study habit.
type Habit struct {
	ID       int
	Name     string
	Color    string
	Archived bool
}

// LogRefreshMsg triggers a log data reload.
type LogRefreshMsg struct{}

// Todo represents a single daily-todo item.
type Todo struct {
	ID            int
	Text          string
	Completed     bool
	EffectiveDate time.Time  // date-only (00:00:00 in local TZ); the day this todo belongs to
	CreatedAt     time.Time
	CompletedAt   *time.Time // nil when not completed
}

// TodoRefreshMsg triggers a todo data reload for the currently-viewed date.
type TodoRefreshMsg struct{}

// Tab identifiers
type Tab int

const (
	TimerTab Tab = iota
	TodoTab
	StatsTab
	HabitsTab
	LogTab
	SettingsTab
)
