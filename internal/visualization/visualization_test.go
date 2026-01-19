package visualization

import (
	"strings"
	"testing"
	"time"

	"github.com/kairos/internal/tracker"
)

func TestGenerateWeekSVGBasics(t *testing.T) {
	v := New()
	weekStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday
	progress := &tracker.WeekProgress{
		WeekStart:  weekStart,
		WeekEnd:    weekStart.AddDate(0, 0, 6),
		TotalHours: 20.5,
		DaysWorked: map[string]float64{
			weekStart.Format("2006-01-02"): 2.5,
		},
	}

	svg := v.GenerateWeekSVG(progress)

	assertContains(t, svg, "<?xml")
	assertContains(t, svg, "Weekly Overview")
	assertContains(t, svg, "Total: 20.5/38.5h")
	assertContains(t, svg, ">Mon</text>")
	assertContains(t, svg, ">Sun</text>")

	rectCount := strings.Count(svg, "<rect")
	if rectCount != 8 {
		t.Fatalf("expected 8 rects (background + 7 bars), got %d", rectCount)
	}
}

func TestGenerateMonthSVGBasics(t *testing.T) {
	v := New()
	progress := &tracker.MonthProgress{
		Month:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		TotalHours:   80.0,
		DailyAverage: 4.0,
		WeekHours: map[int]float64{
			1: 20,
			2: 25,
			3: 15,
			4: 20,
		},
	}

	svg := v.GenerateMonthSVG(progress)

	assertContains(t, svg, "Monthly Overview")
	assertContains(t, svg, "January 2024")
	assertContains(t, svg, "Total: 80.0h | Daily Avg: 4.0h")

	rectCount := strings.Count(svg, "<rect")
	expectedRects := len(progress.WeekHours) + 1
	if rectCount != expectedRects {
		t.Fatalf("expected %d rects (background + %d bars), got %d", expectedRects, len(progress.WeekHours), rectCount)
	}
}

func TestGenerateHTMLReport(t *testing.T) {
	v := New()
	weekStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday
	weekProgress := &tracker.WeekProgress{
		WeekStart:       weekStart,
		WeekEnd:         weekStart.AddDate(0, 0, 6),
		TotalHours:      19.25,
		DaysWorkedCount: 3,
		DaysWorked: map[string]float64{
			weekStart.Format("2006-01-02"): 2.5,
		},
	}
	dayProgress := &tracker.DayProgress{
		TotalHours: 2.25,
	}

	html := v.GenerateHTMLReport(dayProgress, weekProgress)

	assertContains(t, html, "<!DOCTYPE html>")
	assertContains(t, html, "Kairos Report")
	assertContains(t, html, "Weekly Goal Progress")
	assertContains(t, html, "Generated on ")
	assertContains(t, html, "style=\"width: 50.0%\"")
	assertContains(t, html, "19.25 / 38.5 hours")
	assertContains(t, html, "<td>Monday</td><td>2.50 hours</td>")

	rowCount := strings.Count(html, "<tr><td>")
	if rowCount != 7 {
		t.Fatalf("expected 7 daily rows, got %d", rowCount)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q", needle)
	}
}
