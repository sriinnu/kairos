package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kairos/internal/archive"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive old months to markdown",
	Long: `Archive past months' data to markdown files in ~/.samaya/history/
This keeps SQLite lean while preserving historical data.`,
}

var archiveAutoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Auto-archive all past months",
	Long:  `Automatically archive all complete months before the current month.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		archiver := archive.New(db, historyPath)

		archived, err := archiver.AutoArchivePastMonths()
		if err != nil {
			return err
		}

		if len(archived) == 0 {
			fmt.Println("No months to archive (current month or already archived)")
			return nil
		}

		fmt.Printf("Archived %d month(s):\n", len(archived))
		for _, f := range archived {
			fmt.Printf("  - %s\n", f)
		}
		return nil
	},
}

var archiveMonthCmd = &cobra.Command{
	Use:   "month <YYYY-MM>",
	Short: "Archive a specific month",
	Long:  `Archive a specific month to markdown. Use format YYYY-MM (e.g., 2025-01).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse YYYY-MM
		t, err := time.Parse("2006-01", args[0])
		if err != nil {
			return fmt.Errorf("invalid format, use YYYY-MM (e.g., 2025-01)")
		}

		clean, _ := cmd.Flags().GetBool("clean")
		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		archiver := archive.New(db, historyPath)

		err = archiver.ArchiveMonth(t.Year(), t.Month(), clean)
		if err != nil {
			return err
		}

		fmt.Printf("Archived %s to %s/%d-%02d.md\n", t.Format("January 2006"), historyPath, t.Year(), t.Month())
		if clean {
			fmt.Println("Database cleaned for this month")
		}
		return nil
	},
}

var archiveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List archived months",
	RunE: func(cmd *cobra.Command, args []string) error {
		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		archiver := archive.New(db, historyPath)

		archives, err := archiver.ListArchives()
		if err != nil {
			return err
		}

		if len(archives) == 0 {
			fmt.Println("No archives found")
			return nil
		}

		fmt.Println("Archived months:")
		for _, a := range archives {
			fmt.Printf("  %s\n", a)
		}
		return nil
	},
}

var archiveShowCmd = &cobra.Command{
	Use:   "show <YYYY-MM>",
	Short: "Show archived month data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := time.Parse("2006-01", args[0])
		if err != nil {
			return fmt.Errorf("invalid format, use YYYY-MM (e.g., 2025-01)")
		}

		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		archiver := archive.New(db, historyPath)

		content, err := archiver.ReadArchive(t.Year(), t.Month())
		if err != nil {
			return err
		}

		fmt.Println(content)
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history [months]",
	Short: "Show historical summary",
	Long:  `Show summary of archived months. Optionally specify how many months back.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		monthsBack := 3
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err == nil && n > 0 {
				monthsBack = n
			}
		}

		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		archiver := archive.New(db, historyPath)

		context, err := archiver.GetHistoryContext(monthsBack)
		if err != nil {
			return err
		}

		if context == "" {
			fmt.Println("No historical data found. Run 'kairos archive auto' first.")
			return nil
		}

		fmt.Println(context)
		return nil
	},
}

func init() {
	archiveCmd.AddCommand(archiveAutoCmd)
	archiveCmd.AddCommand(archiveMonthCmd)
	archiveCmd.AddCommand(archiveListCmd)
	archiveCmd.AddCommand(archiveShowCmd)

	archiveMonthCmd.Flags().Bool("clean", false, "Remove archived data from database")
}
