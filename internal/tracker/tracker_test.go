package tracker

import (
	"testing"
	"time"
)

func TestGetWeekStart(t *testing.T) {
	// Monday Jan 1, 2024
	monday := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    time.Time
		expected time.Weekday
	}{
		{"Monday", monday, time.Monday},
		{"Tuesday", monday.AddDate(0, 0, 1), time.Monday},
		{"Wednesday", monday.AddDate(0, 0, 2), time.Monday},
		{"Thursday", monday.AddDate(0, 0, 3), time.Monday},
		{"Friday", monday.AddDate(0, 0, 4), time.Monday},
		{"Saturday", monday.AddDate(0, 0, 5), time.Monday},
		{"Sunday", monday.AddDate(0, 0, 6), time.Monday},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeekStart(tt.input)
			if result.Weekday() != tt.expected {
				t.Errorf("getWeekStart(%v) = %v (weekday: %v), want %v",
					tt.input, result, result.Weekday(), tt.expected)
			}
			// Verify the result is at the same time as input (not forced to midnight)
			if result.Hour() != tt.input.Hour() || result.Minute() != tt.input.Minute() {
				t.Errorf("getWeekStart should preserve time of day, got %v, want hour %d",
					result, tt.input.Hour())
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{"09:00", time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()), false},
		{"9:00", time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()), false},
		{"17:30", time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, now.Location()), false},
		{"5:30", time.Date(now.Year(), now.Month(), now.Day(), 5, 30, 0, 0, now.Location()), false},
		{"14:00:00", time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location()), false},
		{"invalid", time.Time{}, true},
		{"", time.Time{}, true},
		{"25:00", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseTime(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseTime(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseTime(%q) unexpected error: %v", tt.input, err)
				}
				if result.Hour() != tt.expected.Hour() || result.Minute() != tt.expected.Minute() {
					t.Errorf("parseTime(%q) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestNewTrackerWithDefaults(t *testing.T) {
	tracker := NewWithDefaults(nil)
	if tracker == nil {
		t.Fatal("NewWithDefaults returned nil")
	}
	if tracker.weeklyGoal <= 0 {
		t.Errorf("weeklyGoal should be positive, got %f", tracker.weeklyGoal)
	}
}

func TestNewTrackerCustomGoal(t *testing.T) {
	tracker := New(nil, 40.0)
	if tracker.weeklyGoal != 40.0 {
		t.Errorf("weeklyGoal = %f, want 40.0", tracker.weeklyGoal)
	}
}

func TestNewTrackerInvalidGoal(t *testing.T) {
	tracker := New(nil, 0)
	if tracker.weeklyGoal <= 0 {
		t.Errorf("weeklyGoal should use default for non-positive value, got %f", tracker.weeklyGoal)
	}
}

func TestNewTrackerNegativeGoal(t *testing.T) {
	tracker := New(nil, -10)
	if tracker.weeklyGoal <= 0 {
		t.Errorf("weeklyGoal should use default for negative value, got %f", tracker.weeklyGoal)
	}
}

func TestDayProgressCalculation(t *testing.T) {
	progress := &DayProgress{
		Date:            time.Now(),
		TotalHours:      0,
		CurrentSessionID: "",
		Sessions:        nil,
	}

	if progress.TotalHours != 0 {
		t.Errorf("Initial TotalHours should be 0, got %f", progress.TotalHours)
	}
}

func TestWeekProgressFields(t *testing.T) {
	progress := &WeekProgress{
		WeekStart:       time.Now(),
		WeekEnd:         time.Now().AddDate(0, 0, 6),
		TotalHours:      0,
		DaysWorked:      make(map[string]float64),
		DaysWorkedCount: 0,
		RemainingHours:  0,
		Sessions:        nil,
	}

	if progress.WeekStart.After(progress.WeekEnd) {
		t.Error("WeekStart should not be after WeekEnd")
	}
	if progress.DaysWorked == nil {
		t.Error("DaysWorked should not be nil")
	}
}

func TestMonthProgressFields(t *testing.T) {
	progress := &MonthProgress{
		Month:        time.Now(),
		TotalHours:   0,
		DailyAverage: 0,
		WeekHours:    make(map[int]float64),
		WeekCount:    0,
	}

	if progress.WeekHours == nil {
		t.Error("WeekHours should not be nil")
	}
}
