package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var clockinCmd = &cobra.Command{
	Use:   "clockin [note]",
	Short: "Start a work session",
	Long:  `Clock in to start tracking your work hours. Optionally add a note or override time with -t.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetTodayProgress()
		if err != nil {
			return err
		}
		if progress.CurrentSessionID != "" {
			return fmt.Errorf("already clocked in! Use 'kairos clockout' first")
		}

		note := ""
		for _, arg := range args {
			if strings.Contains(arg, ":") {
				continue // Skip time-like args for note
			}
			if note != "" {
				note += " "
			}
			note += arg
		}

		timeStr, _ := cmd.Flags().GetString("time")
		session, err := trackerService.ClockInWithTime(note, timeStr)
		if err != nil {
			return err
		}

		fmt.Printf("Clocked in at %s\n", session.StartTime.Format("15:04"))
		if note != "" {
			fmt.Printf("Note: %s\n", note)
		}
		return nil
	},
}

var clockoutCmd = &cobra.Command{
	Use:   "clockout [break-minutes]",
	Short: "End current work session",
	Long:  `Clock out to end your current work session. Optional break time in minutes.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		breakMinutes := 0
		if len(args) > 0 {
			breakMinutes, _ = strconv.Atoi(args[0])
		}

		session, err := trackerService.GetActiveSession()
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("no active session found")
		}

		updated, err := trackerService.ClockOut(session.ID, breakMinutes, "")
		if err != nil {
			return err
		}

		duration := updated.EndTime.Sub(updated.StartTime)
		hours := duration.Hours() - float64(breakMinutes)/60.0
		fmt.Printf("Clocked out at %s\n", updated.EndTime.Format("15:04"))
		fmt.Printf("Duration: %.2f hours\n", hours)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show today's progress",
	Long:  `Display your work hours progress for today.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetTodayProgress()
		if err != nil {
			return err
		}

		fmt.Printf("Today: %s\n", progress.Date.Format("Monday, Jan 2"))
		fmt.Printf("Hours worked: %.2f\n", progress.TotalHours)

		active, err := trackerService.GetActiveSession()
		if err != nil {
			return err
		}
		if active != nil {
			fmt.Println("Status: Currently working")
			fmt.Printf("Started at: %s\n", active.StartTime.Format("15:04"))
		} else {
			fmt.Println("Status: Not clocked in")
		}

		return nil
	},
}

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show weekly summary",
	Long:  `Display your work hours summary for the current week.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		fmt.Printf("Week: %s - %s\n", progress.WeekStart.Format("Jan 2"), progress.WeekEnd.Format("Jan 2"))
		fmt.Printf("Total hours: %.2f / %.2f\n", progress.TotalHours, 38.5)
		fmt.Printf("Days worked: %d\n", progress.DaysWorkedCount)

		if progress.RemainingHours > 0 {
			fmt.Printf("Remaining: %.2f hours\n", progress.RemainingHours)
		} else {
			fmt.Printf("Overtime: +%.2f hours\n", -progress.RemainingHours)
		}

		fmt.Println("\nDaily breakdown:")
		for day, hours := range progress.DaysWorked {
			fmt.Printf("  %s: %.2f hrs\n", day, hours)
		}

		return nil
	},
}

var monthCmd = &cobra.Command{
	Use:   "month",
	Short: "Show monthly summary",
	Long:  `Display your work hours summary for the current month.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetMonthlyProgress()
		if err != nil {
			return err
		}

		fmt.Printf("Month: %s\n", progress.Month.Format("January 2006"))
		fmt.Printf("Total hours: %.2f\n", progress.TotalHours)
		fmt.Printf("Weeks tracked: %d\n", progress.WeekCount)
		fmt.Printf("Daily average: %.2f hrs\n", progress.DailyAverage)

		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit the current or last session",
	Long:  `Edit the current session, or a specific session by ID. Use without ID to edit today's session.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		breakMinutes, _ := cmd.Flags().GetInt("break")
		note, _ := cmd.Flags().GetString("note")
		timeStr, _ := cmd.Flags().GetString("time")

		id := ""
		if len(args) == 0 {
			progress, err := trackerService.GetTodayProgress()
			if err != nil {
				return err
			}
			if progress.CurrentSessionID == "" {
				return fmt.Errorf("no active session. Use: kairos edit <id>")
			}
			id = progress.CurrentSessionID
		} else {
			id = args[0]
		}

		return trackerService.EditSession(id, breakMinutes, note, timeStr)
	},
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List recent sessions",
	Long:  `Show your recent work sessions with IDs for editing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		fmt.Println("Recent Sessions:")
		fmt.Println("================")
		for _, s := range progress.Sessions {
			duration := "active"
			if s.EndTime != nil {
				d := s.EndTime.Sub(s.StartTime).Hours()
				duration = fmt.Sprintf("%.1fh", d)
			}
			note := ""
			if s.Note != "" {
				note = " - " + s.Note
			}
			status := ""
			if s.EndTime == nil {
				status = " [ACTIVE]"
			}
			fmt.Printf("%s: %s %s (%s)%s\n", s.ID[:8], s.Date.Format("Jan 02"), s.StartTime.Format("15:04"), duration, note+status)
		}
		return nil
	},
}

var askCmd = &cobra.Command{
	Use:   "ask \"your question\"",
	Short: "Ask AI about your work hours",
	Long:  `Ask an AI-powered question about your work hours. Requires Ollama running.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ollamaService.IsAvailable() {
			return fmt.Errorf("Ollama is not running. Start it with: ollama serve")
		}

		dayProgress, err := trackerService.GetTodayProgress()
		if err != nil {
			return err
		}

		weekProgress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		question := args[0]
		answer, err := ollamaService.Ask(question, dayProgress, weekProgress)
		if err != nil {
			return err
		}

		fmt.Println(answer)
		return nil
	},
}

var predictCmd = &cobra.Command{
	Use:   "predict",
	Short: "AI prediction for goal completion",
	Long:  `Get AI-powered predictions about when you'll reach your weekly goal.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ollamaService.IsAvailable() {
			return fmt.Errorf("Ollama is not running. Start it with: ollama serve")
		}

		weekProgress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		prediction, err := ollamaService.Predict(weekProgress)
		if err != nil {
			return err
		}

		fmt.Println(prediction)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Database: %s\n", cfg.DatabasePath)
		fmt.Printf("Weekly goal: %.2f hours\n", cfg.WeeklyGoal)
		fmt.Printf("Ollama URL: %s\n", cfg.OllamaURL)
		fmt.Printf("Ollama model: %s\n", cfg.OllamaModel)
		return nil
	},
}

func init() {
	editCmd.Flags().IntP("break", "b", 0, "Break time in minutes")
	editCmd.Flags().StringP("note", "n", "", "Add a note")
	editCmd.Flags().StringP("time", "t", "", "Override start time (HH:MM)")

	clockinCmd.Flags().StringP("time", "t", "", "Override start time (HH:MM)")
	clockoutCmd.Flags().StringP("time", "t", "", "Override end time (HH:MM)")
}

func overrideTime(flag string, defaultTime time.Time) time.Time {
	// Simplified - would parse flag in real implementation
	return defaultTime
}
