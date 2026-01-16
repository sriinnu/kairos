package work

import "time"

// =============================================================================
// WORK RULES CONFIGURATION
// =============================================================================
// Edit these values to match your local work regulations.
// Current defaults: Austrian work rules
//
// To customize for your country/company:
// 1. Change WeeklyGoalHours to your standard work week
// 2. Change DefaultBreakMinutes to your standard break duration
// 3. Change FridayBreakMinutes if Friday has different break rules
// =============================================================================

const (
	// WeeklyGoalHours - your standard weekly working hours
	// Austria: 38.5 | Germany: 40 | France: 35 | US: 40
	WeeklyGoalHours = 38.5

	// DefaultBreakMinutes - standard break duration for Mon-Thu
	// Austria: 30 minutes mandatory for 6+ hour shifts
	DefaultBreakMinutes = 30

	// FridayBreakMinutes - break duration on Fridays
	// Set to 0 if your workplace has no Friday breaks (common in Austria)
	FridayBreakMinutes = 0

	// WorkDaysPerWeek - standard work week (typically 5)
	WorkDaysPerWeek = 5

	// DailyTargetHours - calculated average daily target
	DailyTargetHours = WeeklyGoalHours / WorkDaysPerWeek
)

// GetBreakMinutesForDay returns the appropriate break time based on the day of week
func GetBreakMinutesForDay(t time.Time) int {
	if t.Weekday() == time.Friday {
		return FridayBreakMinutes
	}
	return DefaultBreakMinutes
}

// GetBreakMinutesForToday returns break minutes for today
func GetBreakMinutesForToday() int {
	return GetBreakMinutesForDay(time.Now())
}

// IsWorkDay returns true if the given day is a standard work day (Mon-Fri)
func IsWorkDay(t time.Time) bool {
	day := t.Weekday()
	return day >= time.Monday && day <= time.Friday
}

// RemainingWorkDaysInWeek returns how many work days are left in the current week
func RemainingWorkDaysInWeek(t time.Time) int {
	day := t.Weekday()
	if day == time.Saturday || day == time.Sunday {
		return 0
	}
	// Monday=1, Friday=5, so remaining = 5 - current + 1 (including today)
	return int(time.Friday) - int(day) + 1
}

// CalculateRequiredDailyHours calculates hours needed per remaining day to meet goal
func CalculateRequiredDailyHours(hoursWorked float64, remainingDays int) float64 {
	if remainingDays <= 0 {
		return 0
	}
	remaining := WeeklyGoalHours - hoursWorked
	if remaining <= 0 {
		return 0
	}
	return remaining / float64(remainingDays)
}
