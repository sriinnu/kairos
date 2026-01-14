package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type WorkSession struct {
	ID            int64     `json:"id"`
	Date          time.Time `json:"date"`
	StartTime     time.Time `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	BreakMinutes  int       `json:"break_minutes"`
	Note          string    `json:"note,omitempty"`
}

type DailySummary struct {
	Date         time.Time `json:"date"`
	TotalHours   float64   `json:"total_hours"`
	SessionCount int       `json:"session_count"`
}

type WeeklySummary struct {
	WeekStart    time.Time `json:"week_start"`
	WeekEnd      time.Time `json:"week_end"`
	TotalHours   float64   `json:"total_hours"`
	GoalHours    float64   `json:"goal_hours"`
	DaysWorked   int       `json:"days_worked"`
}

type MonthlySummary struct {
	Month       time.Time `json:"month"`
	TotalHours  float64   `json:"total_hours"`
	GoalHours   float64   `json:"goal_hours"`
	DaysWorked  int       `json:"days_worked"`
	WeekCount   int       `json:"week_count"`
}

type Database struct {
	db *sql.DB
}

func New(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS work_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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

func (d *Database) InsertSession(session *WorkSession) (int64, error) {
	result, err := d.db.Exec(
		`INSERT INTO work_sessions (date, start_time, end_time, break_minutes, note)
		 VALUES (?, ?, ?, ?, ?)`,
		session.Date.Format("2006-01-02"),
		session.StartTime.Format("2006-01-02T15:04:05"),
		session.EndTime,
		session.BreakMinutes,
		session.Note,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) UpdateSession(session *WorkSession) error {
	_, err := d.db.Exec(
		`UPDATE work_sessions SET end_time = ?, break_minutes = ?, note = ? WHERE id = ?`,
		session.EndTime,
		session.BreakMinutes,
		session.Note,
		session.ID,
	)
	return err
}

func (d *Database) GetSessionByID(id int64) (*WorkSession, error) {
	var session WorkSession
	var endTime sql.NullString

	err := d.db.QueryRow(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE id = ?`,
		id,
	).Scan(&session.ID, &session.Date, &session.StartTime, &endTime, &session.BreakMinutes, &session.Note)

	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		t, _ := time.Parse("2006-01-02T15:04:05", endTime.String)
		session.EndTime = &t
	}

	return &session, nil
}

func (d *Database) GetActiveSession() (*WorkSession, error) {
	var session WorkSession
	var endTime sql.NullString

	err := d.db.QueryRow(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE end_time IS NULL ORDER BY start_time DESC LIMIT 1`,
	).Scan(&session.ID, &session.Date, &session.StartTime, &endTime, &session.BreakMinutes, &session.Note)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (d *Database) GetSessionsInRange(start, end time.Time) ([]WorkSession, error) {
	rows, err := d.db.Query(
		`SELECT id, date, start_time, end_time, break_minutes, note
		 FROM work_sessions WHERE date >= ? AND date <= ?
		 ORDER BY start_time ASC`,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []WorkSession
	for rows.Next() {
		var session WorkSession
		var endTime sql.NullString

		if err := rows.Scan(&session.ID, &session.Date, &session.StartTime, &endTime, &session.BreakMinutes, &session.Note); err != nil {
			return nil, err
		}

		if endTime.Valid {
			t, _ := time.Parse("2006-01-02T15:04:05", endTime.String)
			session.EndTime = &t
		}

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (d *Database) GetTodaySessions() ([]WorkSession, error) {
	today := time.Now().Format("2006-01-02")
	return d.GetSessionsInRange(time.Now(), time.Now())
}

func (d *Database) DeleteSession(id int64) error {
	_, err := d.db.Exec("DELETE FROM work_sessions WHERE id = ?", id)
	return err
}
