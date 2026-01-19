package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type WorkSession struct {
	ID           string     `json:"id"`
	Date         time.Time  `json:"date"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	BreakMinutes int        `json:"break_minutes"`
	Note         string     `json:"note,omitempty"`
}

type DailySummary struct {
	Date         time.Time `json:"date"`
	TotalHours   float64   `json:"total_hours"`
	SessionCount int       `json:"session_count"`
}

type WeeklySummary struct {
	WeekStart  time.Time `json:"week_start"`
	WeekEnd    time.Time `json:"week_end"`
	TotalHours float64   `json:"total_hours"`
	GoalHours  float64   `json:"goal_hours"`
	DaysWorked int       `json:"days_worked"`
}

type MonthlySummary struct {
	Month      time.Time `json:"month"`
	TotalHours float64   `json:"total_hours"`
	GoalHours  float64   `json:"goal_hours"`
	DaysWorked int       `json:"days_worked"`
	WeekCount  int       `json:"week_count"`
}

type Database struct {
	db  *sql.DB
	loc *time.Location
}

func New(path string, loc *time.Location) (*Database, error) {
	if loc == nil {
		loc = time.Local
	}
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &Database{db: db, loc: loc}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS work_sessions (
			id TEXT PRIMARY KEY,
			date TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT,
			break_minutes INTEGER DEFAULT 0,
			note TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS daily_summary (
			date TEXT PRIMARY KEY,
			total_hours REAL,
			session_count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS weekly_summary (
			week_start TEXT PRIMARY KEY,
			week_end TEXT,
			total_hours REAL,
			goal_hours REAL,
			days_worked INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS monthly_summary (
			month TEXT PRIMARY KEY,
			total_hours REAL,
			goal_hours REAL,
			days_worked INTEGER,
			week_count INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_date ON work_sessions(date)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_start ON work_sessions(start_time)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Location() *time.Location {
	if d.loc != nil {
		return d.loc
	}
	return time.Local
}

func (d *Database) InsertSession(session *WorkSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	var endTimeStr interface{}
	if session.EndTime != nil {
		endTimeStr = session.EndTime.UTC().Format("2006-01-02T15:04:05")
	}
	dateValue := session.Date
	if !session.StartTime.IsZero() {
		dateValue = session.StartTime
	}

	_, err := d.db.Exec(
		`INSERT INTO work_sessions (id, date, start_time, end_time, break_minutes, note)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID,
		dateValue.UTC().Format("2006-01-02"),
		session.StartTime.UTC().Format("2006-01-02T15:04:05"),
		endTimeStr,
		session.BreakMinutes,
		session.Note,
	)
	return err
}

func (d *Database) UpdateSession(session *WorkSession) error {
	var endTimeStr interface{}
	if session.EndTime != nil {
		endTimeStr = session.EndTime.UTC().Format("2006-01-02T15:04:05")
	}
	dateValue := session.Date
	if !session.StartTime.IsZero() {
		dateValue = session.StartTime
	}

	_, err := d.db.Exec(
		`UPDATE work_sessions SET date = ?, start_time = ?, end_time = ?, break_minutes = ?, note = ? WHERE id = ?`,
		dateValue.UTC().Format("2006-01-02"),
		session.StartTime.UTC().Format("2006-01-02T15:04:05"),
		endTimeStr,
		session.BreakMinutes,
		session.Note,
		session.ID,
	)
	return err
}

func (d *Database) GetSessionByID(id string) (*WorkSession, error) {
	var session WorkSession
	var dateStr, startTimeStr, endTime sql.NullString

	// Try exact match first
	err := d.db.QueryRow(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE id = ?`,
		id,
	).Scan(&session.ID, &dateStr, &startTimeStr, &endTime, &session.BreakMinutes, &session.Note)

	if err == sql.ErrNoRows && len(id) >= 8 {
		// Try prefix match
		return d.GetSessionByPrefix(id[:8])
	}

	if err != nil {
		return nil, err
	}

	d.populateSessionTimes(&session, dateStr, startTimeStr, endTime)

	return &session, nil
}

func (d *Database) GetSessionByPrefix(prefix string) (*WorkSession, error) {
	var session WorkSession
	var dateStr, startTimeStr, endTime sql.NullString

	err := d.db.QueryRow(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE id LIKE ?`,
		prefix+"%",
	).Scan(&session.ID, &dateStr, &startTimeStr, &endTime, &session.BreakMinutes, &session.Note)

	if err != nil {
		return nil, err
	}

	d.populateSessionTimes(&session, dateStr, startTimeStr, endTime)

	return &session, nil
}

func (d *Database) GetActiveSession() (*WorkSession, error) {
	var session WorkSession
	var dateStr, startTimeStr sql.NullString

	err := d.db.QueryRow(
		`SELECT id, date, start_time, break_minutes, note
		 FROM work_sessions WHERE end_time IS NULL ORDER BY start_time DESC LIMIT 1`,
	).Scan(&session.ID, &dateStr, &startTimeStr, &session.BreakMinutes, &session.Note)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	d.populateSessionTimes(&session, dateStr, startTimeStr, sql.NullString{})

	return &session, nil
}

func (d *Database) GetSessionsInRange(start, end time.Time) ([]WorkSession, error) {
	rangeStart, rangeEnd := d.normalizeRange(start, end)
	rows, err := d.db.Query(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE start_time <= ? AND (end_time IS NULL OR end_time >= ?)
		 ORDER BY start_time ASC`,
		rangeEnd.Format("2006-01-02T15:04:05"),
		rangeStart.Format("2006-01-02T15:04:05"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []WorkSession
	for rows.Next() {
		var session WorkSession
		var dateStr, startTimeStr, endTime sql.NullString

		if err := rows.Scan(&session.ID, &dateStr, &startTimeStr, &endTime, &session.BreakMinutes, &session.Note); err != nil {
			return nil, err
		}

		d.populateSessionTimes(&session, dateStr, startTimeStr, endTime)

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (d *Database) GetTodaySessions() ([]WorkSession, error) {
	now := time.Now().In(d.Location())
	return d.GetSessionsInRange(now, now)
}

func (d *Database) DeleteSession(id string) error {
	_, err := d.db.Exec("DELETE FROM work_sessions WHERE id = ?", id)
	return err
}

// DeleteSessionsInRange removes all sessions within a date range
func (d *Database) DeleteSessionsInRange(start, end time.Time) error {
	rangeStart, rangeEnd := d.normalizeRange(start, end)
	_, err := d.db.Exec(
		"DELETE FROM work_sessions WHERE start_time <= ? AND (end_time IS NULL OR end_time >= ?)",
		rangeEnd.Format("2006-01-02T15:04:05"),
		rangeStart.Format("2006-01-02T15:04:05"),
	)
	return err
}

// GetOldestSessionDate returns the date of the oldest session
func (d *Database) GetOldestSessionDate() (*time.Time, error) {
	var dateStr sql.NullString
	err := d.db.QueryRow("SELECT MIN(start_time) FROM work_sessions").Scan(&dateStr)
	if err != nil {
		return nil, err
	}
	if !dateStr.Valid {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", dateStr.String, time.UTC)
	if err != nil {
		return nil, err
	}
	local := t.In(d.Location())
	return &local, nil
}

func (d *Database) normalizeRange(start, end time.Time) (time.Time, time.Time) {
	loc := d.Location()
	rangeStart := start.In(loc)
	rangeEnd := end.In(loc)
	rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, loc)
	rangeEnd = time.Date(rangeEnd.Year(), rangeEnd.Month(), rangeEnd.Day(), 23, 59, 59, 0, loc)
	return rangeStart.UTC(), rangeEnd.UTC()
}

func (d *Database) populateSessionTimes(session *WorkSession, dateStr, startTimeStr, endTime sql.NullString) {
	loc := d.Location()
	if startTimeStr.Valid {
		t, _ := time.ParseInLocation("2006-01-02T15:04:05", startTimeStr.String, time.UTC)
		localStart := t.In(loc)
		session.StartTime = localStart
		session.Date = time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc)
	}
	if endTime.Valid {
		t, _ := time.ParseInLocation("2006-01-02T15:04:05", endTime.String, time.UTC)
		localEnd := t.In(loc)
		session.EndTime = &localEnd
	}
	if session.Date.IsZero() && dateStr.Valid {
		t, _ := time.ParseInLocation("2006-01-02", dateStr.String, loc)
		session.Date = t
	}
}

// Exec executes a raw SQL query (for MCP tools)
func (d *Database) Exec(query string, args ...interface{}) error {
	_, err := d.db.Exec(query, args...)
	return err
}

// Query executes a raw SQL query and returns rows
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

// QueryRow executes a raw SQL query and returns a single row
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}
