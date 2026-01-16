package work

import (
	"testing"
	"time"
)

func TestGetBreakMinutesForDay(t *testing.T) {
	tests := []struct {
		name     string
		weekday  time.Weekday
		expected int
	}{
		{"Monday", time.Monday, DefaultBreakMinutes},
		{"Tuesday", time.Tuesday, DefaultBreakMinutes},
		{"Wednesday", time.Wednesday, DefaultBreakMinutes},
		{"Thursday", time.Thursday, DefaultBreakMinutes},
		{"Friday", time.Friday, FridayBreakMinutes},
		{"Saturday", time.Saturday, DefaultBreakMinutes},
		{"Sunday", time.Sunday, DefaultBreakMinutes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday
			// Adjust to the target weekday
			daysToAdd := (int(tt.weekday) - int(date.Weekday()) + 7) % 7
			targetDate := date.AddDate(0, 0, daysToAdd)

			result := GetBreakMinutesForDay(targetDate)
			if result != tt.expected {
				t.Errorf("GetBreakMinutesForDay(%v) = %d, want %d", tt.weekday, result, tt.expected)
			}
		})
	}
}

func TestIsWorkDay(t *testing.T) {
	monday := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Monday Jan 1, 2024
	friday := monday.AddDate(0, 0, 4)
	saturday := monday.AddDate(0, 0, 5)
	sunday := monday.AddDate(0, 0, 6)

	if !IsWorkDay(monday) {
		t.Error("Monday should be a work day")
	}
	if !IsWorkDay(friday) {
		t.Error("Friday should be a work day")
	}
	if IsWorkDay(saturday) {
		t.Error("Saturday should not be a work day")
	}
	if IsWorkDay(sunday) {
		t.Error("Sunday should not be a work day")
	}
}

func TestRemainingWorkDaysInWeek(t *testing.T) {
	monday := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		weekday  time.Weekday
		expected int
	}{
		{"Monday", time.Monday, 5},
		{"Tuesday", time.Tuesday, 4},
		{"Wednesday", time.Wednesday, 3},
		{"Thursday", time.Thursday, 2},
		{"Friday", time.Friday, 1},
		{"Saturday", time.Saturday, 0},
		{"Sunday", time.Sunday, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daysToAdd := (int(tt.weekday) - int(monday.Weekday()) + 7) % 7
			targetDate := monday.AddDate(0, 0, daysToAdd)

			result := RemainingWorkDaysInWeek(targetDate)
			if result != tt.expected {
				t.Errorf("RemainingWorkDaysInWeek(%v) = %d, want %d", tt.weekday, result, tt.expected)
			}
		})
	}
}

func TestCalculateRequiredDailyHours(t *testing.T) {
	tests := []struct {
		name          string
		hoursWorked   float64
		remainingDays int
		expected      float64
	}{
		{"0 hours worked, 5 days left", 0, 5, WeeklyGoalHours / 5},
		{"Half goal met, 5 days left", WeeklyGoalHours / 2, 5, (WeeklyGoalHours / 2) / 5},
		{"Goal exceeded", 50, 5, 0},
		{"No days remaining", 20, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRequiredDailyHours(tt.hoursWorked, tt.remainingDays)
			if result != tt.expected {
				t.Errorf("CalculateRequiredDailyHours(%f, %d) = %f, want %f",
					tt.hoursWorked, tt.remainingDays, result, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify daily target calculation is correct
	expectedDailyTarget := WeeklyGoalHours / WorkDaysPerWeek
	if DailyTargetHours != expectedDailyTarget {
		t.Errorf("DailyTargetHours = %f, expected %f", DailyTargetHours, expectedDailyTarget)
	}
}
