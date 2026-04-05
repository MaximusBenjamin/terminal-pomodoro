package common

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

// Tab identifiers
type Tab int

const (
	TimerTab Tab = iota
	StatsTab
	HabitsTab
	LogTab
	SettingsTab
)
