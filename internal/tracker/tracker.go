package tracker

import (
	"fmt"
	"time"

	"github.com/samaya/internal/storage"
)

type Tracker struct {
	db         *storage.Database
	weeklyGoal float64
}

func New(db *storage.Database, weeklyGoal float64) *Tracker {
	return &Tracker{
		db:         db,
		weeklyGoal: weeklyGoal,
	}
}

func (t *Tracker) ClockIn(note string) (*storage.WorkSession, error) {
	session := &storage.WorkSession{
		Date:         time.Now(),
		StartTime:    time.Now(),
		BreakMinutes: 0,
		Note:         note,
	}

	id, err := t.db.InsertSession(session)
	if err != nil {
		return nil, err
	}
	session.ID = id

	return session, nil
}

func (t *Tracker) ClockOut(id int64, breakMinutes int, note string) (*storage.WorkSession, error) {
	session, err := t.db.GetSessionByID(id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	now := time.Now()
	session.EndTime = &now
	session.BreakMinutes = breakMinutes
	session.Note = note

	if err := t.db.UpdateSession(session); err != nil {
		return nil, err
	}

	return session, nil
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
			progress.TotalHours -= float64(s.BreakMinutes) / 60.0
			progress.TotalHours += hours
		}
	}

	return progress, nil
}

func (t *Tracker) GetWeeklyProgress() (*WeekProgress, error) {
	now := time.Now()
	weekStart := getWeekStart(now)
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

func (t *Tracker) EditSession(id int64, breakMinutes int, note string) error {
	session, err := t.db.GetSessionByID(id)
	if err != nil {
		return err
	}
	session.BreakMinutes = breakMinutes
	session.Note = note
	return t.db.UpdateSession(session)
}

func (t *Tracker) DeleteSession(id int64) error {
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
	Date       time.Time
	Sessions   []storage.WorkSession
	TotalHours float64
}

type WeekProgress struct {
	WeekStart      time.Time
	WeekEnd        time.Time
	TotalHours     float64
	DaysWorked     map[string]float64
	DaysWorkedCount int
	RemainingHours float64
}

type MonthProgress struct {
	Month        time.Time
	TotalHours   float64
	DailyAverage float64
	WeekHours    map[int]float64
	WeekCount    int
}
