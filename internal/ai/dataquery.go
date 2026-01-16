package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/tracker"
	"github.com/kairos/internal/work"
)

// DataQuerier wraps storage for AI-friendly data queries
type DataQuerier struct {
	db          *storage.Database
	tracker     *tracker.Tracker
	historyPath string
}

// NewDataQuerier creates a new data querier
func NewDataQuerier(db *storage.Database) *DataQuerier {
	return &DataQuerier{
		db:          db,
		tracker:     tracker.NewWithDefaults(db),
		historyPath: "", // Set via SetHistoryPath if needed
	}
}

// NewDataQuerierWithHistory creates a data querier with history support
func NewDataQuerierWithHistory(db *storage.Database, historyPath string) *DataQuerier {
	return &DataQuerier{
		db:          db,
		tracker:     tracker.NewWithDefaults(db),
		historyPath: historyPath,
	}
}

// SetHistoryPath sets the path to archived history files
func (dq *DataQuerier) SetHistoryPath(path string) {
	dq.historyPath = path
}

// QueryResult contains structured data for AI consumption
type QueryResult struct {
	QueryType   string                 `json:"query_type"`
	Data        map[string]interface{} `json:"data"`
	Summary     string                 `json:"summary"`
	Timestamp   time.Time              `json:"timestamp"`
}

// GetTodaySummary returns today's work data
func (dq *DataQuerier) GetTodaySummary() (*QueryResult, error) {
	progress, err := dq.tracker.GetTodayProgress()
	if err != nil {
		return nil, err
	}

	active, _ := dq.tracker.GetActiveSession()

	data := map[string]interface{}{
		"date":          progress.Date.Format("2006-01-02"),
		"day_of_week":   progress.Date.Weekday().String(),
		"hours_worked":  progress.TotalHours,
		"session_count": len(progress.Sessions),
		"is_working":    active != nil,
	}

	if active != nil {
		data["current_session_start"] = active.StartTime.Format("15:04")
		data["current_session_hours"] = time.Since(active.StartTime).Hours()
	}

	summary := fmt.Sprintf("Today (%s): %.2f hours worked",
		progress.Date.Format("Monday"), progress.TotalHours)
	if active != nil {
		summary += fmt.Sprintf(", currently working since %s", active.StartTime.Format("15:04"))
	}

	return &QueryResult{
		QueryType: "today_summary",
		Data:      data,
		Summary:   summary,
		Timestamp: time.Now(),
	}, nil
}

// GetWeekSummary returns weekly work data
func (dq *DataQuerier) GetWeekSummary() (*QueryResult, error) {
	progress, err := dq.tracker.GetWeeklyProgress()
	if err != nil {
		return nil, err
	}

	remainingDays := work.RemainingWorkDaysInWeek(time.Now())
	dailyTarget := 0.0
	if remainingDays > 0 && progress.RemainingHours > 0 {
		dailyTarget = progress.RemainingHours / float64(remainingDays)
	}

	data := map[string]interface{}{
		"week_start":      progress.WeekStart.Format("2006-01-02"),
		"week_end":        progress.WeekEnd.Format("2006-01-02"),
		"total_hours":     progress.TotalHours,
		"weekly_goal":     work.WeeklyGoalHours,
		"remaining_hours": progress.RemainingHours,
		"days_worked":     progress.DaysWorkedCount,
		"remaining_days":  remainingDays,
		"daily_target":    dailyTarget,
		"progress_pct":    (progress.TotalHours / work.WeeklyGoalHours) * 100,
		"daily_breakdown": progress.DaysWorked,
	}

	summary := fmt.Sprintf("Week: %.2f/%.2f hours (%.1f%%), %d days worked, %.2f hours remaining",
		progress.TotalHours, work.WeeklyGoalHours,
		(progress.TotalHours/work.WeeklyGoalHours)*100,
		progress.DaysWorkedCount, progress.RemainingHours)

	return &QueryResult{
		QueryType: "week_summary",
		Data:      data,
		Summary:   summary,
		Timestamp: time.Now(),
	}, nil
}

// GetMonthSummary returns monthly work data
func (dq *DataQuerier) GetMonthSummary() (*QueryResult, error) {
	progress, err := dq.tracker.GetMonthlyProgress()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"month":         progress.Month.Format("January 2006"),
		"total_hours":   progress.TotalHours,
		"week_count":    progress.WeekCount,
		"daily_average": progress.DailyAverage,
		"week_hours":    progress.WeekHours,
	}

	summary := fmt.Sprintf("Month %s: %.2f hours total, %.2f daily average across %d weeks",
		progress.Month.Format("January"), progress.TotalHours, progress.DailyAverage, progress.WeekCount)

	return &QueryResult{
		QueryType: "month_summary",
		Data:      data,
		Summary:   summary,
		Timestamp: time.Now(),
	}, nil
}

// GetRecentSessions returns recent work sessions
func (dq *DataQuerier) GetRecentSessions(limit int) (*QueryResult, error) {
	if limit <= 0 {
		limit = 10
	}

	progress, err := dq.tracker.GetWeeklyProgress()
	if err != nil {
		return nil, err
	}

	sessions := progress.Sessions
	if len(sessions) > limit {
		sessions = sessions[len(sessions)-limit:]
	}

	sessionData := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		sess := map[string]interface{}{
			"id":         s.ID[:8],
			"date":       s.Date.Format("2006-01-02"),
			"start_time": s.StartTime.Format("15:04"),
			"break_min":  s.BreakMinutes,
			"note":       s.Note,
		}
		if s.EndTime != nil {
			sess["end_time"] = s.EndTime.Format("15:04")
			hours := s.EndTime.Sub(s.StartTime).Hours() - float64(s.BreakMinutes)/60.0
			sess["hours"] = hours
			sess["is_active"] = false
		} else {
			sess["is_active"] = true
			sess["hours"] = time.Since(s.StartTime).Hours()
		}
		sessionData = append(sessionData, sess)
	}

	data := map[string]interface{}{
		"sessions": sessionData,
		"count":    len(sessionData),
	}

	summary := fmt.Sprintf("%d recent sessions", len(sessionData))

	return &QueryResult{
		QueryType: "recent_sessions",
		Data:      data,
		Summary:   summary,
		Timestamp: time.Now(),
	}, nil
}

// GetWorkStatus returns current work status
func (dq *DataQuerier) GetWorkStatus() (*QueryResult, error) {
	active, err := dq.tracker.GetActiveSession()
	if err != nil {
		return nil, err
	}

	dayProgress, err := dq.tracker.GetTodayProgress()
	if err != nil {
		return nil, err
	}

	weekProgress, err := dq.tracker.GetWeeklyProgress()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"is_working":       active != nil,
		"today_hours":      dayProgress.TotalHours,
		"week_hours":       weekProgress.TotalHours,
		"weekly_goal":      work.WeeklyGoalHours,
		"remaining_hours":  weekProgress.RemainingHours,
		"remaining_days":   work.RemainingWorkDaysInWeek(time.Now()),
		"time_now":         time.Now().Format("15:04"),
		"day_of_week":      time.Now().Weekday().String(),
		"default_break":    work.GetBreakMinutesForToday(),
	}

	if active != nil {
		data["session_start"] = active.StartTime.Format("15:04")
		data["running_hours"] = time.Since(active.StartTime).Hours()
	}

	var summary string
	if active != nil {
		summary = fmt.Sprintf("Currently working since %s (%.1f hrs). Today: %.2f hrs, Week: %.2f/%.2f hrs",
			active.StartTime.Format("15:04"),
			time.Since(active.StartTime).Hours(),
			dayProgress.TotalHours,
			weekProgress.TotalHours, work.WeeklyGoalHours)
	} else {
		summary = fmt.Sprintf("Not currently working. Today: %.2f hrs, Week: %.2f/%.2f hrs",
			dayProgress.TotalHours, weekProgress.TotalHours, work.WeeklyGoalHours)
	}

	return &QueryResult{
		QueryType: "work_status",
		Data:      data,
		Summary:   summary,
		Timestamp: time.Now(),
	}, nil
}

// BuildDataContext creates a full context string for AI prompts
func (dq *DataQuerier) BuildDataContext() (string, error) {
	status, err := dq.GetWorkStatus()
	if err != nil {
		return "", err
	}

	week, err := dq.GetWeekSummary()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("CURRENT WORK DATA:\n")
	sb.WriteString(fmt.Sprintf("- %s\n", status.Summary))
	sb.WriteString(fmt.Sprintf("- %s\n", week.Summary))
	sb.WriteString(fmt.Sprintf("- Standard break: %d min (0 on Fridays)\n", work.DefaultBreakMinutes))

	if data, ok := week.Data["daily_breakdown"].(map[string]float64); ok && len(data) > 0 {
		sb.WriteString("- Daily breakdown: ")
		parts := make([]string, 0, len(data))
		for day, hrs := range data {
			parts = append(parts, fmt.Sprintf("%s=%.1fh", day, hrs))
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("\n")
	}

	// Include historical context if available
	if dq.historyPath != "" {
		history := dq.getHistorySummary(3)
		if history != "" {
			sb.WriteString("\n")
			sb.WriteString(history)
		}
	}

	return sb.String(), nil
}

// getHistorySummary reads archived months and returns a summary
func (dq *DataQuerier) getHistorySummary(monthsBack int) string {
	if dq.historyPath == "" {
		return ""
	}

	entries, err := os.ReadDir(dq.historyPath)
	if err != nil {
		return ""
	}

	var archives []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			archives = append(archives, e.Name())
		}
	}

	if len(archives) == 0 {
		return ""
	}

	// Get last N archives
	start := len(archives) - monthsBack
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	sb.WriteString("HISTORICAL DATA (archived months):\n")

	for _, archive := range archives[start:] {
		content, err := os.ReadFile(filepath.Join(dq.historyPath, archive))
		if err != nil {
			continue
		}

		// Extract summary from markdown
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				sb.WriteString(fmt.Sprintf("- %s: ", strings.TrimPrefix(line, "# ")))
			}
			if strings.HasPrefix(line, "| Total Hours |") {
				parts := strings.Split(line, "|")
				if len(parts) >= 3 {
					sb.WriteString(fmt.Sprintf("%s hours\n", strings.TrimSpace(parts[2])))
				}
			}
		}
	}

	return sb.String()
}

// GetTodayHours returns hours worked today (for offline fallback)
func (dq *DataQuerier) GetTodayHours() (float64, error) {
	progress, err := dq.tracker.GetTodayProgress()
	if err != nil {
		return 0, err
	}
	return progress.TotalHours, nil
}

// GetWeekHours returns hours worked this week (for offline fallback)
func (dq *DataQuerier) GetWeekHours() (float64, error) {
	progress, err := dq.tracker.GetWeeklyProgress()
	if err != nil {
		return 0, err
	}
	return progress.TotalHours, nil
}

// GetHoursInRange returns total hours worked in a date range
func (dq *DataQuerier) GetHoursInRange(start, end time.Time) (float64, error) {
	sessions, err := dq.db.GetSessionsInRange(start, end)
	if err != nil {
		return 0, err
	}

	total := 0.0
	for _, s := range sessions {
		if s.EndTime != nil {
			hours := s.EndTime.Sub(s.StartTime).Hours()
			hours -= float64(s.BreakMinutes) / 60.0
			total += hours
		}
	}
	return total, nil
}

// GetSessionsInRange returns sessions in a date range
func (dq *DataQuerier) GetSessionsInRange(start, end time.Time) ([]storage.WorkSession, error) {
	return dq.db.GetSessionsInRange(start, end)
}
