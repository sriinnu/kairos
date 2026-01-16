package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kairos/internal/ai"
	"github.com/kairos/internal/archive"
	"github.com/kairos/internal/config"
	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	cfg            *config.Config
	db             *storage.Database
	trackerService *tracker.Tracker
	aiService      *ai.AIService
	dataQuerier    *ai.DataQuerier
)

var rootCmd = &cobra.Command{
	Use:   "kairos",
	Short: "AI-powered time tracking with insights",
	Long:  `Kairos helps you track your working hours and provides AI-powered insights about your schedule.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}
		db, err = storage.New(cfg.DatabasePath)
		if err != nil {
			return err
		}
		trackerService = tracker.NewWithDefaults(db)
		aiService = ai.NewAIService(cfg)
		aiService.Initialize()
		historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
		dataQuerier = ai.NewDataQuerierWithHistory(db, historyPath)

		// Auto-archive past months (silent, non-blocking)
		go func() {
			historyPath := filepath.Join(filepath.Dir(cfg.DatabasePath), "history")
			archiver := archive.New(db, historyPath)
			archived, _ := archiver.AutoArchivePastMonths()
			if len(archived) > 0 {
				fmt.Printf("Auto-archived %d month(s) to %s\n", len(archived), historyPath)
			}
		}()

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clockinCmd)
	rootCmd.AddCommand(clockoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(weekCmd)
	rootCmd.AddCommand(monthCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(historyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
