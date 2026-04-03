package log

import (
	"testing"
	"time"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

var testHabits = []common.Habit{
	{ID: 1, Name: "programming", Color: "#7aa2f7"},
	{ID: 2, Name: "mathematics", Color: "#bb9af7"},
	{ID: 3, Name: "finance", Color: "#9ece6a"},
	{ID: 4, Name: "japanese", Color: "#e0af68"},
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int
		wantErr  bool
	}{
		{"30m math", 30 * 60, false},
		{"2h programming", 2 * 3600, false},
		{"1.5h finance", 90 * 60, false},
		{"1h 30m math", 90 * 60, false},
		{"45 minutes finance", 45 * 60, false},
		{"2 hours programming", 2 * 3600, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSessionInput(tt.input, testHabits)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.DurationSecs != tt.wantSecs {
				t.Errorf("duration = %d, want %d", result.DurationSecs, tt.wantSecs)
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int
		wantErr  bool
	}{
		{"1pm to 2pm programming", 3600, false},
		{"1:30pm - 2:30pm math", 3600, false},
		{"9am to 11am programming", 2 * 3600, false},
		{"13:00 to 14:00 finance", 3600, false},
		{"2pm to 1pm math", 0, true}, // end before start
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSessionInput(tt.input, testHabits)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.DurationSecs != tt.wantSecs {
				t.Errorf("duration = %d, want %d", result.DurationSecs, tt.wantSecs)
			}
		})
	}
}

func TestParseHabitMatching(t *testing.T) {
	tests := []struct {
		input     string
		wantHabit string
		wantErr   bool
	}{
		{"30m programming", "programming", false},
		{"30m math", "mathematics", false},   // prefix match
		{"30m prog", "programming", false},   // prefix match
		{"30m fin", "finance", false},         // prefix match
		{"30m jap", "japanese", false},        // prefix match
		{"30m nonexistent", "", true},         // no match
		{"30m", "", true},                     // no habit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSessionInput(tt.input, testHabits)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.HabitName != tt.wantHabit {
				t.Errorf("habit = %q, want %q", result.HabitName, tt.wantHabit)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := ParseSessionInput("", testHabits)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseInvalidFormat(t *testing.T) {
	_, err := ParseSessionInput("just some words", testHabits)
	if err == nil {
		t.Fatal("expected error for unparseable input")
	}
}

func TestMatchHabit(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{"programming", "programming", false},
		{"PROGRAMMING", "programming", false}, // case insensitive
		{"math", "mathematics", false},         // prefix
		{"", "", true},                          // empty
		{"xyz", "", true},                       // no match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := matchHabit(tt.name, testHabits)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if h.Name != tt.want {
				t.Errorf("got %q, want %q", h.Name, tt.want)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"1pm", 13, 0, false},
		{"1:30pm", 13, 30, false},
		{"1:30 pm", 13, 30, false},
		{"12am", 0, 0, false},
		{"12pm", 12, 0, false},
		{"13:00", 13, 0, false},
		{"9am", 9, 0, false},
		{"abc", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pt, err := parseTimeOfDay(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pt.hour != tt.wantHour || pt.min != tt.wantMin {
				t.Errorf("got %d:%02d, want %d:%02d", pt.hour, pt.min, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestResolveDate(t *testing.T) {
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		input string
		want  time.Time
	}{
		{"", today},
		{"today", today},
		{"yesterday", today.AddDate(0, 0, -1)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveDate(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	// Weekday resolves to the most recent occurrence (not today)
	monday := resolveDate("monday")
	if monday.Weekday() != time.Monday {
		t.Errorf("monday resolved to %v", monday.Weekday())
	}
	if !monday.Before(today) {
		t.Error("monday should be before today (resolves to past week)")
	}
}

func TestParseDateModifiers(t *testing.T) {
	// "30m math yesterday" should produce a session on yesterday's date
	result, err := ParseSessionInput("30m math yesterday", testHabits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().Local()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	if result.StartTime.Year() != yesterday.Year() || result.StartTime.Month() != yesterday.Month() || result.StartTime.Day() != yesterday.Day() {
		t.Errorf("expected yesterday's date, got %v", result.StartTime)
	}
}
