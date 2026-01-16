package visualization

import (
	"fmt"
	"strings"
	"time"

	"github.com/kairos/internal/tracker"
)

type Visualizer struct{}

func New() *Visualizer {
	return &Visualizer{}
}

func (v *Visualizer) GenerateWeekSVG(progress *tracker.WeekProgress) string {
	width := 600
	height := 300
	padding := 40
	barWidth := float64((width - 2*padding) / 7)
	maxHours := 12.0 // Max hours per day to display

	var days []string
	var hours []float64
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	for i := 0; i < 7; i++ {
		day := progress.WeekStart.AddDate(0, 0, i)
		dayKey := day.Format("2006-01-02")
		days = append(days, dayNames[i])
		hours = append(hours, progress.DaysWorked[dayKey])
	}

	var bars strings.Builder
	for i, h := range hours {
		barHeight := (h / maxHours) * float64(height-2*padding)
		if barHeight > float64(height-2*padding) {
			barHeight = float64(height - 2*padding)
		}

		x := float64(padding) + float64(i)*barWidth + 5
		y := float64(height) - float64(padding) - barHeight

		color := "#4CAF50"
		if h > 10 {
			color = "#FF9800"
		}
		if h > 12 {
			color = "#F44336"
		}

		bars.WriteString(fmt.Sprintf(`<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="%s" rx="4"/>
    <text x="%.0f" y="%d" text-anchor="middle" font-size="12" fill="#333">%.1fh</text>`,
			x, y, barWidth-10, barHeight, color,
			x+barWidth/2-5, int(y)-5, h))
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#f5f7fa"/>
      <stop offset="100%%" style="stop-color:#e4e8ec"/>
    </linearGradient>
  </defs>
  <rect width="%d" height="%d" fill="url(#bgGrad)" rx="10"/>
  <text x="%d" y="30" text-anchor="middle" font-size="18" font-weight="bold" fill="#2c3e50">Weekly Overview</text>
  <text x="%d" y="55" text-anchor="middle" font-size="12" fill="#7f8c8d">%s - %s | Total: %.1f/38.5h</text>

  <!-- Goal line -->
  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#E74C3C" stroke-width="2" stroke-dasharray="5,5"/>
  <text x="%d" y="%d" font-size="10" fill="#E74C3C">Daily Avg Goal</text>

  <!-- Bars -->
  %s

  <!-- X-axis labels -->
  %s

  <!-- Grid lines -->
  %s
</svg>`,
		width, height, width, height,
		width, height,
		width/2,
		width/2, progress.WeekStart.Format("Jan 2"), progress.WeekEnd.Format("Jan 2"), progress.TotalHours,
		padding, height-padding-30, width-padding, height-padding-30,
		width-padding+10, height-padding-35,
		bars.String(),
		v.generateXLabels(days, float64(padding), barWidth, float64(height-padding)),
		v.generateGridLines(maxHours, height, padding, width),
	)
}

func (v *Visualizer) GenerateMonthSVG(progress *tracker.MonthProgress) string {
	width := 600
	height := 400
	padding := 50
	cellSize := float64(width-2*padding) / 4 // 4 weeks

	var weeks []string
	var weekHours []float64
	for week, hours := range progress.WeekHours {
		weeks = append(weeks, fmt.Sprintf("W%d", week))
		weekHours = append(weekHours, hours)
	}

	var bars strings.Builder
	for i, h := range weekHours {
		barHeight := (h / 40) * float64(height-2*padding)
		x := float64(padding) + float64(i)*cellSize + 10
		y := float64(height) - float64(padding) - barHeight

		bars.WriteString(fmt.Sprintf(`<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="#3498DB" rx="4"/>
    <text x="%.0f" y="%d" text-anchor="middle" font-size="12" fill="#333">%.1fh</text>`,
			x, y, cellSize-20, barHeight,
			x+cellSize/2-10, int(y)-5, h))
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#f5f7fa"/>
      <stop offset="100%%" style="stop-color:#e4e8ec"/>
    </linearGradient>
  </defs>
  <rect width="%d" height="%d" fill="url(#bgGrad)" rx="10"/>
  <text x="%d" y="30" text-anchor="middle" font-size="18" font-weight="bold" fill="#2c3e50">Monthly Overview</text>
  <text x="%d" y="55" text-anchor="middle" font-size="12" fill="#7f8c8d">%s | Total: %.1fh | Daily Avg: %.1fh</text>

  <!-- Progress ring -->
  <circle cx="%d" cy="%d" r="60" fill="none" stroke="#E0E0E0" stroke-width="10"/>
  <circle cx="%d" cy="%d" r="60" fill="none" stroke="#4CAF50" stroke-width="10"
    stroke-dasharray="%.0f %.0f" transform="rotate(-90 %d %d)"/>
  <text x="%d" y="%d" text-anchor="middle" font-size="14" fill="#333">%.0f%%</text>

  <!-- Bars -->
  %s
</svg>`,
		width, height, width, height,
		width, height,
		width/2,
		width/2, progress.Month.Format("January 2006"), progress.TotalHours, progress.DailyAverage,
		width-100, 120,
		width-100, 120,
		2*3.14*60*progress.TotalHours/154, 2*3.14*60,
		width-100, 120,
		width-100, 125,
		progress.TotalHours/154*100, // percentage of monthly goal (4 weeks * 38.5)
		bars.String(),
	)
}

func (v *Visualizer) GenerateHTMLReport(dayProgress *tracker.DayProgress, weekProgress *tracker.WeekProgress) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Kairos - Work Hours Report</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; background: #f5f7fa; }
    .container { max-width: 800px; margin: 0 auto; }
    .card { background: white; border-radius: 10px; padding: 24px; margin-bottom: 20px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    h1 { color: #2c3e50; margin-bottom: 8px; }
    h2 { color: #34495e; font-size: 18px; margin-bottom: 16px; }
    .subtitle { color: #7f8c8d; margin-bottom: 30px; }
    .stat { display: inline-block; text-align: center; padding: 20px; margin: 10px; background: #f8f9fa; border-radius: 8px; min-width: 120px; }
    .stat-value { font-size: 32px; font-weight: bold; color: #3498DB; }
    .stat-label { font-size: 12px; color: #7f8c8d; margin-top: 4px; }
    .progress-bar { height: 24px; background: #E0E0E0; border-radius: 12px; overflow: hidden; margin: 16px 0; }
    .progress-fill { height: 100%%; background: linear-gradient(90deg, #4CAF50, #8BC34A); border-radius: 12px; transition: width 0.3s; }
    .status { padding: 12px 20px; border-radius: 8px; margin: 10px 0; }
    .status.working { background: #E8F5E9; color: #2E7D32; }
    .status.not-working { background: #FFEBEE; color: #C62828; }
    table { width: 100%%; border-collapse: collapse; margin-top: 16px; }
    th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
    th { color: #7f8c8d; font-weight: 500; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Kairos Report</h1>
    <p class="subtitle">Generated on %s</p>

    <div class="card">
      <h2>Today's Progress</h2>
      <div class="stat">
        <div class="stat-value">%.2fh</div>
        <div class="stat-label">Hours Today</div>
      </div>
      <div class="stat">
        <div class="stat-value">%.2fh</div>
        <div class="stat-label">Weekly Total</div>
      </div>
      <div class="stat">
        <div class="stat-value">%.1f</div>
        <div class="stat-label">Days Worked</div>
      </div>
    </div>

    <div class="card">
      <h2>Weekly Goal Progress</h2>
      <div class="progress-bar">
        <div class="progress-fill" style="width: %.1f%%"></div>
      </div>
      <p style="color: #7f8c8d; text-align: center;">%.2f / 38.5 hours</p>
    </div>

    <div class="card">
      <h2>Daily Breakdown</h2>
      <table>
        <tr><th>Day</th><th>Hours</th></tr>
        %s
      </table>
    </div>
  </div>
</body>
</html>`,
		time.Now().Format("Monday, January 2, 2006"),
		dayProgress.TotalHours,
		weekProgress.TotalHours,
		float64(weekProgress.DaysWorkedCount),
		(weekProgress.TotalHours/38.5)*100,
		weekProgress.TotalHours,
		v.formatDailyRows(weekProgress),
	)
}

func (v *Visualizer) formatDailyRows(progress *tracker.WeekProgress) string {
	var rows []string
	dayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	for i := 0; i < 7; i++ {
		day := progress.WeekStart.AddDate(0, 0, i)
		dayKey := day.Format("2006-01-02")
		hours := progress.DaysWorked[dayKey]
		rows = append(rows, fmt.Sprintf("<tr><td>%s</td><td>%.2f hours</td></tr>", dayNames[i], hours))
	}

	return strings.Join(rows, "\n")
}

func (v *Visualizer) generateXLabels(days []string, padding float64, barWidth float64, y float64) string {
	var labels strings.Builder
	for i, day := range days {
		x := padding + float64(i)*barWidth + barWidth/2 - 5
		labels.WriteString(fmt.Sprintf(`<text x="%.0f" y="%d" text-anchor="middle" font-size="12" fill="#7f8c8d">%s</text>`,
			x, int(y)+20, day))
	}
	return labels.String()
}

func (v *Visualizer) generateGridLines(maxHours float64, height int, padding int, width int) string {
	var lines strings.Builder
	for i := 1; i <= 4; i++ {
		y := float64(height) - float64(padding) - (float64(i)/4.0)*float64(height-2*padding)
		lines.WriteString(fmt.Sprintf(`<line x1="%d" y1="%.0f" x2="%d" y2="%.0f" stroke="#E0E0E0"/>`,
			padding, y, width-padding, y))
	}
	return lines.String()
}
