package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/work"
)

// Archiver handles monthly data archival to markdown
type Archiver struct {
	db          *storage.Database
	historyPath string
}

// New creates a new Archiver
func New(db *storage.Database, historyPath string) *Archiver {
	return &Archiver{
		db:          db,
		historyPath: historyPath,
	}
}

// MonthSummary contains archived month data
type MonthSummary struct {
	Month        time.Time
	TotalHours   float64
	DaysWorked   int
	WeeklyGoal   float64
	Sessions     []SessionRecord
	WeekBreakdown map[int]float64
}

// SessionRecord is a simplified session for archive
type SessionRecord struct {
	Date         string
	StartTime    string
	EndTime      string
	Hours        float64
	BreakMinutes int
	Note         string
}

// ArchiveMonth exports a month's data to markdown and optionally cleans DB
func (a *Archiver) ArchiveMonth(year int, month time.Month, cleanDB bool) error {
	// Get month boundaries
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	// Get sessions for the month
	sessions, err := a.db.GetSessionsInRange(monthStart, monthEnd)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no sessions found for %s %d", month, year)
	}

	// Build summary
	summary := a.buildSummary(monthStart, sessions)

	// Generate markdown
	markdown := a.generateMarkdown(summary)

	// Ensure history directory exists
	if err := os.MkdirAll(a.historyPath, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%d-%02d.md", year, month)
	filePath := filepath.Join(a.historyPath, filename)

	if err := os.WriteFile(filePath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	// Clean DB if requested
	if cleanDB {
		if err := a.cleanMonth(year, month); err != nil {
			return fmt.Errorf("failed to clean database: %w", err)
		}
	}

	return nil
}

func (a *Archiver) buildSummary(monthStart time.Time, sessions []storage.WorkSession) *MonthSummary {
	summary := &MonthSummary{
		Month:         monthStart,
		WeeklyGoal:    work.WeeklyGoalHours,
		Sessions:      make([]SessionRecord, 0, len(sessions)),
		WeekBreakdown: make(map[int]float64),
	}

	daysWorked := make(map[string]bool)

	for _, s := range sessions {
		if s.EndTime == nil {
			continue // Skip incomplete sessions
		}

		hours := s.EndTime.Sub(s.StartTime).Hours() - float64(s.BreakMinutes)/60.0
		summary.TotalHours += hours

		dayKey := s.Date.Format("2006-01-02")
		daysWorked[dayKey] = true

		_, week := s.Date.ISOWeek()
		summary.WeekBreakdown[week] += hours

		summary.Sessions = append(summary.Sessions, SessionRecord{
			Date:         s.Date.Format("2006-01-02"),
			StartTime:    s.StartTime.Format("15:04"),
			EndTime:      s.EndTime.Format("15:04"),
			Hours:        hours,
			BreakMinutes: s.BreakMinutes,
			Note:         s.Note,
		})
	}

	summary.DaysWorked = len(daysWorked)
	return summary
}

func (a *Archiver) generateMarkdown(summary *MonthSummary) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# %s\n\n", summary.Month.Format("January 2006")))

	// Summary stats
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Total Hours | %.2f |\n", summary.TotalHours))
	sb.WriteString(fmt.Sprintf("| Days Worked | %d |\n", summary.DaysWorked))
	sb.WriteString(fmt.Sprintf("| Daily Average | %.2f |\n", summary.TotalHours/float64(max(summary.DaysWorked, 1))))
	sb.WriteString(fmt.Sprintf("| Weekly Goal | %.2f |\n", summary.WeeklyGoal))
	sb.WriteString("\n")

	// Week breakdown
	sb.WriteString("## Weekly Breakdown\n\n")
	sb.WriteString("| Week | Hours |\n")
	sb.WriteString("|------|-------|\n")

	weeks := make([]int, 0, len(summary.WeekBreakdown))
	for w := range summary.WeekBreakdown {
		weeks = append(weeks, w)
	}
	sort.Ints(weeks)

	for _, w := range weeks {
		sb.WriteString(fmt.Sprintf("| W%d | %.2f |\n", w, summary.WeekBreakdown[w]))
	}
	sb.WriteString("\n")

	// Session details
	sb.WriteString("## Sessions\n\n")
	sb.WriteString("| Date | Start | End | Hours | Break | Note |\n")
	sb.WriteString("|------|-------|-----|-------|-------|------|\n")

	for _, s := range summary.Sessions {
		note := s.Note
		if len(note) > 30 {
			note = note[:27] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.2f | %dm | %s |\n",
			s.Date, s.StartTime, s.EndTime, s.Hours, s.BreakMinutes, note))
	}
	sb.WriteString("\n")

	// Footer
	sb.WriteString(fmt.Sprintf("---\n*Archived: %s*\n", time.Now().Format("2006-01-02 15:04")))

	return sb.String()
}

func (a *Archiver) cleanMonth(year int, month time.Month) error {
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	return a.db.DeleteSessionsInRange(monthStart, monthEnd)
}

// AutoArchivePastMonths archives all complete months older than current
func (a *Archiver) AutoArchivePastMonths() ([]string, error) {
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	// Get oldest session date
	oldestDate, err := a.db.GetOldestSessionDate()
	if err != nil {
		return nil, err
	}
	if oldestDate == nil {
		return nil, nil // No sessions
	}

	var archived []string
	monthStart := time.Date(oldestDate.Year(), oldestDate.Month(), 1, 0, 0, 0, 0, time.Local)

	// Archive each month before current
	for monthStart.Before(currentMonth) {
		filename := fmt.Sprintf("%d-%02d.md", monthStart.Year(), monthStart.Month())
		filePath := filepath.Join(a.historyPath, filename)

		// Skip if already archived
		if _, err := os.Stat(filePath); err == nil {
			monthStart = monthStart.AddDate(0, 1, 0)
			continue
		}

		err := a.ArchiveMonth(monthStart.Year(), monthStart.Month(), true)
		if err != nil {
			// Skip months with no data
			if strings.Contains(err.Error(), "no sessions found") {
				monthStart = monthStart.AddDate(0, 1, 0)
				continue
			}
			return archived, err
		}

		archived = append(archived, filename)
		monthStart = monthStart.AddDate(0, 1, 0)
	}

	return archived, nil
}

// ListArchives returns list of archived months
func (a *Archiver) ListArchives() ([]string, error) {
	entries, err := os.ReadDir(a.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var archives []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			archives = append(archives, e.Name())
		}
	}

	sort.Strings(archives)
	return archives, nil
}

// ReadArchive reads a specific month's archive
func (a *Archiver) ReadArchive(year int, month time.Month) (string, error) {
	filename := fmt.Sprintf("%d-%02d.md", year, month)
	filePath := filepath.Join(a.historyPath, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("archive not found: %s", filename)
	}

	return string(data), nil
}

// GetHistoryContext returns summarized history for AI context
func (a *Archiver) GetHistoryContext(monthsBack int) (string, error) {
	archives, err := a.ListArchives()
	if err != nil {
		return "", err
	}

	if len(archives) == 0 {
		return "", nil
	}

	// Get last N archives
	start := len(archives) - monthsBack
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	sb.WriteString("HISTORICAL DATA:\n")

	for _, archive := range archives[start:] {
		content, err := os.ReadFile(filepath.Join(a.historyPath, archive))
		if err != nil {
			continue
		}

		// Extract just the summary section
		lines := strings.Split(string(content), "\n")
		inSummary := false
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				sb.WriteString(fmt.Sprintf("\n%s:\n", strings.TrimPrefix(line, "# ")))
			}
			if strings.HasPrefix(line, "## Summary") {
				inSummary = true
				continue
			}
			if strings.HasPrefix(line, "## Weekly") {
				inSummary = false
			}
			if inSummary && strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| Metric") && !strings.HasPrefix(line, "|--") {
				sb.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}

	return sb.String(), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
