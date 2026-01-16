package tracker

import (
	"fmt"
	"time"

	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/work"
)

type Tracker struct {
	db         *storage.Database
	weeklyGoal float64
}

func New(db *storage.Database, weeklyGoal float64) *Tracker {
	if weeklyGoal <= 0 {
		weeklyGoal = work.WeeklyGoalHours
	}
	return &Tracker{
		db:         db,
		weeklyGoal: weeklyGoal,
	}
}

// NewWithDefaults creates a tracker with default work rules
func NewWithDefaults(db *storage.Database) *Tracker {
	return &Tracker{
		db:         db,
		weeklyGoal: work.WeeklyGoalHours,
	}
}

func (t *Tracker) ClockIn(note string) (*storage.WorkSession, error) {
	session := &storage.WorkSession{
		Date:         time.Now(),
		StartTime:    time.Now(),
		BreakMinutes: 0,
		Note:         note,
	}

	if err := t.db.InsertSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (t *Tracker) ClockInWithTime(note, timeStr string) (*storage.WorkSession, error) {
	session := &storage.WorkSession{
		Date:         time.Now(),
		StartTime:    time.Now(),
		BreakMinutes: 0,
		Note:         note,
	}

	// Parse time override if provided
	if timeStr != "" {
		startTime, err := parseTime(timeStr)
		if err == nil {
			session.StartTime = startTime
			// Adjust date if time is from previous day (e.g., 8:45 AM when it's evening)
			if startTime.After(time.Now()) {
				session.Date = session.Date.AddDate(0, 0, -1)
			}
		}
	}

	if err := t.db.InsertSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (t *Tracker) ClockOut(id string, breakMinutes int, note string) (*storage.WorkSession, error) {
	return t.ClockOutWithTime(id, breakMinutes, note, "")
}

func (t *Tracker) ClockOutWithTime(id string, breakMinutes int, note string, timeStr string) (*storage.WorkSession, error) {
	session, err := t.db.GetSessionByID(id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	endTime := time.Now()
	if timeStr != "" {
		parsed, err := parseTime(timeStr)
		if err == nil {
			endTime = parsed
		}
	}

	session.EndTime = &endTime
	session.BreakMinutes = breakMinutes
	if note != "" {
		session.Note = note
	}

	if err := t.db.UpdateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func parseTime(s string) (time.Time, error) {
	now := time.Now()
	for _, format := range []string{"15:04", "3:04", "15:04:05", "3:04:05"} {
		if t, err := time.Parse(format, s); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location()), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", s)
}

func (t *Tracker) GetTodayProgress() (*DayProgress, error) {
	sessions, err := t.db.GetTodaySessions()
	if err != nil {
		return nil, err
	}

	progress := &DayProgress{
		Date:       time.Now(),
		Sessions:   sessions,
		TotalHours: 0,
	}

	for _, s := range sessions {
		if s.EndTime != nil {
			hours := s.EndTime.Sub(s.StartTime).Hours()
			hours -= float64(s.BreakMinutes) / 60.0
			progress.TotalHours += hours
		} else {
			// This is the current open session
			progress.CurrentSessionID = s.ID
		}
	}

	return progress, nil
}

func (t *Tracker) GetWeeklyProgress() (*WeekProgress, error) {
	return t.GetWeekProgressForDate(time.Now())
}

func (t *Tracker) GetLastWeekProgress() (*WeekProgress, error) {
	lastWeekStart := getWeekStart(time.Now()).AddDate(0, 0, -7)
	return t.computeWeekProgress(lastWeekStart)
}

func (t *Tracker) GetWeekProgressForDate(date time.Time) (*WeekProgress, error) {
	weekStart := getWeekStart(date)
	return t.computeWeekProgress(weekStart)
}

func (t *Tracker) computeWeekProgress(weekStart time.Time) (*WeekProgress, error) {
	weekEnd := weekStart.AddDate(0, 0, 6)

	sessions, err := t.db.GetSessionsInRange(weekStart, weekEnd)
	if err != nil {
		return nil, err
	}

	progress := &WeekProgress{
		WeekStart:  weekStart,
		WeekEnd:    weekEnd,
		TotalHours: 0,
		DaysWorked: make(map[string]float64),
		Sessions:   sessions,
	}

	for _, s := range sessions {
		if s.EndTime != nil {
			hours := s.EndTime.Sub(s.StartTime).Hours()
			hours -= float64(s.BreakMinutes) / 60.0
			progress.TotalHours += hours
			dayKey := s.Date.Format("2006-01-02")
			progress.DaysWorked[dayKey] += hours
		}
	}

	progress.DaysWorkedCount = len(progress.DaysWorked)
	progress.RemainingHours = t.weeklyGoal - progress.TotalHours

	return progress, nil
}

func (t *Tracker) GetMonthlyProgress() (*MonthProgress, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	sessions, err := t.db.GetSessionsInRange(monthStart, now)
	if err != nil {
		return nil, err
	}

	progress := &MonthProgress{
		Month:      monthStart,
		TotalHours: 0,
		WeekHours:  make(map[int]float64),
	}

	for _, s := range sessions {
		if s.EndTime != nil {
			hours := s.EndTime.Sub(s.StartTime).Hours()
			hours -= float64(s.BreakMinutes) / 60.0
			progress.TotalHours += hours
			_, week := s.Date.ISOWeek()
			progress.WeekHours[week] += hours
		}
	}

	progress.WeekCount = len(progress.WeekHours)
	progress.DailyAverage = progress.TotalHours / float64(now.Day())

	return progress, nil
}

func (t *Tracker) GetActiveSession() (*storage.WorkSession, error) {
	return t.db.GetActiveSession()
}

func (t *Tracker) EditSession(id string, breakMinutes int, note string, timeStr string) error {
	return t.EditSessionSelective(id, breakMinutes, true, note, true, timeStr, "")
}

// EditSessionSelective updates only the fields that are explicitly changed
func (t *Tracker) EditSessionSelective(id string, breakMinutes int, breakChanged bool, note string, noteChanged bool, startTimeStr string, endTimeStr string) error {
	session, err := t.db.GetSessionByID(id)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", id)
	}

	// Only update break if explicitly changed
	if breakChanged {
		session.BreakMinutes = breakMinutes
	}

	// Only update note if explicitly changed
	if noteChanged {
		session.Note = note
	}

	// Update start time if provided
	if startTimeStr != "" {
		newTime, err := parseTime(startTimeStr)
		if err == nil {
			session.StartTime = newTime
		}
	}

	// Update end time if provided
	if endTimeStr != "" {
		newTime, err := parseTime(endTimeStr)
		if err == nil {
			session.EndTime = &newTime
		}
	}

	return t.db.UpdateSession(session)
}

func (t *Tracker) DeleteSession(id string) error {
	return t.db.DeleteSession(id)
}

func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -weekday+1)
}

type DayProgress struct {
	Date            time.Time
	Sessions        []storage.WorkSession
	TotalHours      float64
	CurrentSessionID string
}

type WeekProgress struct {
	WeekStart      time.Time
	WeekEnd        time.Time
	TotalHours     float64
	DaysWorked     map[string]float64
	DaysWorkedCount int
	RemainingHours float64
	Sessions       []storage.WorkSession
}

type MonthProgress struct {
	Month        time.Time
	TotalHours   float64
	DailyAverage float64
	WeekHours    map[int]float64
	WeekCount    int
}
