package main

import (
	"os"

	"github.com/kairos/internal/ai"
	"github.com/kairos/internal/config"
	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	cfg           *config.Config
	db            *storage.Database
	trackerService *tracker.Tracker
	ollamaService  *ai.Ollama
)

var rootCmd = &cobra.Command{
	Use:   "samaya",
	Short: "Working hours tracker with AI insights",
	Long:  `Samaya helps you track your working hours and provides AI-powered insights about your schedule.`,
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
		trackerService = tracker.New(db, cfg.WeeklyGoal)
		ollamaService = ai.New(cfg.OllamaURL, cfg.OllamaModel)
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
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(visualizeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
