package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kairos/internal/ai"
	"github.com/kairos/internal/tracker"
	"github.com/kairos/internal/work"
	"github.com/spf13/cobra"
)

var clockinCmd = &cobra.Command{
	Use:     "clockin [note]",
	Aliases: []string{"in", "ci"},
	Short:   "Start a work session",
	Long:    `Clock in to start tracking your work hours. Optionally add a note or override time with -t.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetTodayProgress()
		if err != nil {
			return err
		}
		if progress.CurrentSessionID != "" {
			return fmt.Errorf("already clocked in! Use 'kairos clockout' first")
		}

		// Join all args as note (don't skip colons - notes can contain them)
		note := strings.Join(args, " ")

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
	Use:     "clockout [break-minutes]",
	Aliases: []string{"out", "co"},
	Short:   "End current work session",
	Long: `Clock out to end your current work session.
Break time defaults based on day (30 min Mon-Thu, 0 on Friday).
Override with argument or use -b flag.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := trackerService.GetActiveSession()
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("no active session found")
		}

		// Default break based on day of week
		breakMinutes := work.GetBreakMinutesForToday()

		// Override from flag first
		if cmd.Flags().Changed("break") {
			breakMinutes, _ = cmd.Flags().GetInt("break")
		} else if len(args) > 0 {
			// Or from positional argument
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid break minutes: %s", args[0])
			}
			breakMinutes = parsed
		}

		// Handle time override
		timeStr, _ := cmd.Flags().GetString("time")
		updated, err := trackerService.ClockOutWithTime(session.ID, breakMinutes, "", timeStr)
		if err != nil {
			return err
		}

		duration := updated.EndTime.Sub(updated.StartTime)
		hours := duration.Hours() - float64(breakMinutes)/60.0
		fmt.Printf("Clocked out: %s | Duration: %.2fh | Break: %dmin\n", updated.EndTime.Format("15:04"), hours, breakMinutes)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st", "today"},
	Short:   "Show today's progress",
	Long:    `Display your work hours progress for today.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetTodayProgress()
		if err != nil {
			return err
		}

		active, err := trackerService.GetActiveSession()
		if err != nil {
			return err
		}

		if active != nil {
			elapsed := time.Since(active.StartTime)
			h := int(elapsed.Hours())
			m := int(elapsed.Minutes()) % 60
			fmt.Printf("Today: %s | Hours worked: %.2f | Status: Currently working | Clocked in: %s (%dh %dm elapsed)\n",
				progress.Date.Format("Monday, Jan 2"), progress.TotalHours, active.StartTime.Format("15:04"), h, m)
		} else {
			fmt.Printf("Today: %s | Hours worked: %.2f | Status: Not clocked in\n",
				progress.Date.Format("Monday, Jan 2"), progress.TotalHours)
		}

		return nil
	},
}

var weekCmd = &cobra.Command{
	Use:     "week [last|date]",
	Aliases: []string{"w"},
	Short:   "Show weekly summary",
	Long:    `Display your work hours summary for the current week. Use "last" for previous week or a date (YYYY-MM-DD) for that week's summary.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var progress *tracker.WeekProgress
		var err error

		if len(args) == 0 {
			progress, err = trackerService.GetWeeklyProgress()
		} else if args[0] == "last" {
			progress, err = trackerService.GetLastWeekProgress()
		} else {
			// Try parsing as date
			t, parseErr := time.Parse("2006-01-02", args[0])
			if parseErr != nil {
				// Try YYYY-MM-DD format variations
				for _, fmt := range []string{"2006-01-02", "Jan 2", "Jan 02", "1/2"} {
					if t, parseErr = time.Parse(fmt, args[0]); parseErr == nil {
						break
					}
				}
				if parseErr != nil {
					return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", args[0])
				}
			}
			progress, err = trackerService.GetWeekProgressForDate(t)
		}

		if err != nil {
			return err
		}

		// Summary row
		var summary string
		if progress.RemainingHours > 0 {
			summary = fmt.Sprintf("Remaining: %.2fh", progress.RemainingHours)
		} else {
			summary = fmt.Sprintf("Overtime: +%.2fh", -progress.RemainingHours)
		}
		fmt.Printf("Week: %s - %s | Total: %.2f/%gh | %s\n",
			progress.WeekStart.Format("Jan 2"), progress.WeekEnd.Format("Jan 2"),
			progress.TotalHours, work.WeeklyGoalHours, summary)

		// One row per day
		dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		for i := 0; i < 7; i++ {
			dayDate := progress.WeekStart.AddDate(0, 0, i)
			dayKey := dayDate.Format("2006-01-02")
			hours := progress.DaysWorked[dayKey]
			dayName := dayNames[i]
			// Highlight today
			if i == 6 { // Sunday
				dayName = "Sun"
			}
			if dayDate.Format("2006-01-02") == time.Now().Format("2006-01-02") {
				fmt.Printf("  %s %s: %.2fh *\n", dayDate.Format("01/02"), dayName, hours)
			} else {
				fmt.Printf("  %s %s: %.2fh\n", dayDate.Format("01/02"), dayName, hours)
			}
		}

		return nil
	},
}

var monthCmd = &cobra.Command{
	Use:     "month",
	Aliases: []string{"m"},
	Short:   "Show monthly summary",
	Long:    `Display your work hours summary for the current month.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetMonthlyProgress()
		if err != nil {
			return err
		}

		fmt.Printf("Month: %s | Total hours: %.2f | Weeks tracked: %d | Daily avg: %.2f hrs\n",
			progress.Month.Format("January 2006"), progress.TotalHours, progress.WeekCount, progress.DailyAverage)

		return nil
	},
}

var editCmd = &cobra.Command{
	Use:     "edit [id]",
	Aliases: []string{"e", "update"},
	Short:   "Edit the current or last session",
	Long:    `Edit the current session, or a specific session by ID. Use without ID to edit today's session.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		breakMinutes, _ := cmd.Flags().GetInt("break")
		note, _ := cmd.Flags().GetString("note")
		timeStr, _ := cmd.Flags().GetString("time")
		endTimeStr, _ := cmd.Flags().GetString("end")

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

		// Only update fields that were explicitly set
		breakChanged := cmd.Flags().Changed("break")
		noteChanged := cmd.Flags().Changed("note")

		err := trackerService.EditSessionSelective(id, breakMinutes, breakChanged, note, noteChanged, timeStr, endTimeStr)
		if err != nil {
			return err
		}

		fmt.Printf("Session %s updated\n", id[:8])
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"del", "rm", "remove"},
	Short:   "Delete a session",
	Long:    `Delete a work session by its ID. Use 'sessions' to see IDs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete session %s? This cannot be undone. Use --force to confirm.\n", id)
			return nil
		}

		err := trackerService.DeleteSession(id)
		if err != nil {
			return err
		}

		fmt.Printf("Session %s deleted\n", id[:8])
		return nil
	},
}

var sessionsCmd = &cobra.Command{
	Use:     "sessions",
	Aliases: []string{"ls", "list"},
	Short:   "List recent sessions",
	Long:    `Show your recent work sessions with IDs for editing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		progress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		if len(progress.Sessions) == 0 {
			fmt.Println("No sessions this week")
			return nil
		}

		var lines []string
		for _, s := range progress.Sessions {
			duration := "active"
			if s.EndTime != nil {
				d := s.EndTime.Sub(s.StartTime).Hours() - float64(s.BreakMinutes)/60.0
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
			lines = append(lines, fmt.Sprintf("%s %s %s (%s)%s%s", s.ID[:8], s.Date.Format("Jan 02"), s.StartTime.Format("15:04"), duration, note, status))
		}
		fmt.Printf("Sessions: %s\n", strings.Join(lines, " | "))
		return nil
	},
}

var askCmd = &cobra.Command{
	Use:     "ask \"your question\"",
	Aliases: []string{"a", "ai"},
	Short:   "Ask AI about your work hours",
	Long:    `Ask an AI-powered question about your work hours. Configure provider with: kairos config --provider ollama|openai|claude|gemini`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !aiService.IsAvailable() {
			return fmt.Errorf("%s is not available. Configure with: kairos config", aiService.Name())
		}

		question := strings.Join(args, " ")

		// Build work context
		ctx, err := ai.BuildWorkContext(trackerService)
		if err != nil {
			return err
		}

		answer, err := aiService.Ask(question, ctx)
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
		if !aiService.IsAvailable() {
			return fmt.Errorf("%s is not available. Configure with: kairos config", aiService.Name())
		}

		weekProgress, err := trackerService.GetWeeklyProgress()
		if err != nil {
			return err
		}

		prediction, err := aiService.Predict(weekProgress)
		if err != nil {
			return err
		}

		fmt.Println(prediction)
		return nil
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "AI analysis of work patterns",
	Long:  `Get AI-powered analysis of your work patterns and suggestions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !aiService.IsAvailable() {
			return fmt.Errorf("%s is not available. Configure with: kairos config", aiService.Name())
		}

		analysis, err := aiService.Analyze(dataQuerier)
		if err != nil {
			return err
		}

		fmt.Println(analysis)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings and work rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Config: DB=%s | Ollama=%s (%s)\n", cfg.DatabasePath, cfg.OllamaURL, cfg.OllamaModel)
		fmt.Printf("Rules: Weekly: %.2fh | Daily: %.2fh | Break: %dmin (Fri: %dmin)\n",
			work.WeeklyGoalHours, work.DailyTargetHours, work.DefaultBreakMinutes, work.FridayBreakMinutes)
		return nil
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for kairos.

To load completions:

Bash:
  $ source <(kairos completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kairos completion bash > /etc/bash_completion.d/kairos
  # macOS:
  $ kairos completion bash > /usr/local/etc/bash_completion.d/kairos

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kairos completion zsh > "${fpath[1]}/_kairos"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ kairos completion fish > ~/.config/fish/completions/kairos.fish

PowerShell:
  PS> kairos completion powershell > kairos.ps1
  # To load completions for every new session, run:
  PS> . kairos.ps1
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
		return nil
	},
}

var batchCmd = &cobra.Command{
	Use:     "batch <command>",
	Aliases: []string{"bulk", "batchedit"},
	Short:   "Batch edit sessions",
	Long: `Batch edit sessions matching criteria.

Examples:
  kairos batch edit --ids a1b2c3d4,e5f6g7h8 --note "Team meeting"
  kairos batch delete --date 2024-01-15 --force

Use --dry-run to preview changes without applying them.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Batch operations for editing multiple sessions
		idsStr, _ := cmd.Flags().GetString("ids")
		dateStr, _ := cmd.Flags().GetString("date")
		note, _ := cmd.Flags().GetString("note")
		breakMinutes, _ := cmd.Flags().GetInt("break")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		fmt.Printf("Batch command: %s\n", args[0])
		fmt.Printf("  IDs: %s\n", idsStr)
		fmt.Printf("  Date: %s\n", dateStr)
		fmt.Printf("  Note: %s\n", note)
		fmt.Printf("  Break: %d min\n", breakMinutes)
		fmt.Printf("  Dry run: %t\n", dryRun)

		if args[0] == "edit" && idsStr != "" {
			fmt.Println("Batch edit mode - feature coming soon")
		} else if args[0] == "delete" {
			fmt.Println("Batch delete mode - feature coming soon")
		}
		return nil
	},
}

func init() {
	editCmd.Flags().IntP("break", "b", 0, "Break time in minutes")
	editCmd.Flags().StringP("note", "n", "", "Add a note")
	editCmd.Flags().StringP("time", "t", "", "Override start time (HH:MM)")
	editCmd.Flags().StringP("end", "e", "", "Override end time (HH:MM)")

	deleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	clockinCmd.Flags().StringP("time", "t", "", "Override start time (HH:MM)")

	clockoutCmd.Flags().StringP("time", "t", "", "Override end time (HH:MM)")
	clockoutCmd.Flags().IntP("break", "b", -1, "Override break time in minutes")

	// Batch command flags
	batchCmd.Flags().String("ids", "", "Comma-separated session IDs")
	batchCmd.Flags().String("date", "", "Filter by date (YYYY-MM-DD)")
	batchCmd.Flags().StringP("note", "n", "", "Note to set")
	batchCmd.Flags().IntP("break", "b", 0, "Break time in minutes")
	batchCmd.Flags().Bool("dry-run", false, "Preview changes without applying")
}
