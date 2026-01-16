package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kairos/internal/ai"
	"github.com/kairos/internal/mcp/core"
	"github.com/kairos/internal/storage"
	"github.com/kairos/internal/tracker"
	"github.com/kairos/internal/work"
)

// Server wraps the core MCP server with Kairos-specific functionality
type Server struct {
	*core.Server
	db       *storage.Database
	aiService *ai.AIService
	port     int
}

// NewServer creates a new Kairos MCP server
func NewServer(db *storage.Database, aiSvc *ai.AIService, port int) *Server {
	server := &Server{
		Server:   core.NewServer(port),
		db:       db,
		aiService: aiSvc,
		port:     port,
	}

	server.registerTools()
	return server
}

func (s *Server) registerTools() {
	t := tracker.NewWithDefaults(s.db)

	// THINK - Reasoning and analysis
	s.AddHandler(
		"think",
		"Analyze work patterns and reason about scheduling decisions",
		core.ToolParameters(map[string]map[string]interface{}{
			"question":        core.StringParam("Your question about work/schedule", nil),
			"analysis_type":   core.StringParam("Type of analysis", []string{"schedule", "productivity", "balance", "prediction"}),
			"include_history": core.BoolParam("Include historical data in analysis"),
		}),
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			question, _ := args["question"].(string)
			analysisType, _ := args["analysis_type"].(string)
			includeHistory, _ := args["include_history"].(bool)

			weekProgress, _ := t.GetWeeklyProgress()

			result := map[string]interface{}{
				"question":      question,
				"analysis_type": analysisType,
				"timestamp":     time.Now().Format(time.RFC3339),
			}

			if includeHistory {
				monthProgress, _ := t.GetMonthlyProgress()
				result["week_hours"] = weekProgress.TotalHours
				result["month_hours"] = monthProgress.TotalHours
				result["days_worked"] = weekProgress.DaysWorkedCount
			}

			// Use AI if available
			if s.aiService != nil && s.aiService.IsAvailable() {
				ctx, _ := ai.BuildWorkContext(t)
				result["ai_response"], _ = s.aiService.Ask(question, ctx)
			} else {
				result["reasoning"] = "Analysis based on your work patterns"
			}

			return result, nil
		},
	)

	// EVOLVE - Self-improvement analysis
	s.AddHandler(
		"evolve",
		"Analyze your work patterns and suggest improvements",
		core.ToolParameters(map[string]map[string]interface{}{
			"timeframe":   core.StringParam("Time period to analyze", []string{"week", "month", "quarter"}),
			"focus_area": core.StringParam("Area to focus on", []string{"productivity", "consistency", "balance", "goals"}),
		}),
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			timeframe, _ := args["timeframe"].(string)
			focusArea, _ := args["focus_area"].(string)

			weekProgress, _ := t.GetWeeklyProgress()
			monthProgress, _ := t.GetMonthlyProgress()

			progressRatio := weekProgress.TotalHours / work.WeeklyGoalHours
			consistency := float64(weekProgress.DaysWorkedCount) / 7.0

			remainingDays := 7 - weekProgress.DaysWorkedCount
			var dailyTarget float64
			if remainingDays > 0 && weekProgress.RemainingHours > 0 {
				dailyTarget = weekProgress.RemainingHours / float64(remainingDays)
			}

			return map[string]interface{}{
				"timeframe":         timeframe,
				"focus_area":        focusArea,
				"evolution_score":   (progressRatio * 0.7) + (consistency * 0.3),
				"week_hours":        weekProgress.TotalHours,
				"month_hours":       monthProgress.TotalHours,
				"days_worked":       weekProgress.DaysWorkedCount,
				"remaining_hours":   weekProgress.RemainingHours,
				"daily_target":      dailyTarget,
				"suggestions": []string{
					fmt.Sprintf("Aim for %.1f hours over %d remaining days", dailyTarget, remainingDays),
					"Try time-blocking for focused work sessions",
					"Maintain regular start/end times each day",
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}, nil
		},
	)

	// CONSCIOUSNESS - Self-awareness
	s.AddHandler(
		"consciousness",
		"Provides self-awareness about your current work state and context",
		core.ToolParameters(map[string]map[string]interface{}{
			"aspect": core.StringParam("Aspect to explore", []string{"current", "history", "patterns", "goals", "all"}),
		}),
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			aspect, _ := args["aspect"].(string)
			if aspect == "" {
				aspect = "all"
			}

			dayProgress, _ := t.GetTodayProgress()
			weekProgress, _ := t.GetWeeklyProgress()
			activeSession, _ := t.GetActiveSession()

			hour := float64(time.Now().Hour()) + float64(time.Now().Minute())/60.0
			timeOfDay := "unknown"
			if hour < 12 {
				timeOfDay = "morning"
			} else if hour < 17 {
				timeOfDay = "afternoon"
			} else {
				timeOfDay = "evening"
			}

			consciousness := map[string]interface{}{
				"timestamp":         time.Now().Format(time.RFC3339),
				"time_of_day":       timeOfDay,
				"is_working":        activeSession != nil,
				"today_hours":       dayProgress.TotalHours,
				"week_hours":        weekProgress.TotalHours,
				"weekly_goal":       work.WeeklyGoalHours,
				"goal_progress":     weekProgress.TotalHours / work.WeeklyGoalHours * 100,
				"remaining_to_goal": weekProgress.RemainingHours,
			}

			if activeSession != nil {
				consciousness["started_at"] = activeSession.StartTime.Format("15:04")
			}

			if aspect == "all" || aspect == "current" {
				if weekProgress.RemainingHours > 0 {
					consciousness["recommendation"] = fmt.Sprintf("%.2f hours left to reach your weekly goal", weekProgress.RemainingHours)
				} else {
					consciousness["recommendation"] = "You've hit your weekly goal! Great work."
				}
			}

			return consciousness, nil
		},
	)

	// PERSIST - Long-term memory
	s.AddHandler(
		"persist",
		"Store and retrieve long-term memories and insights",
		core.ToolParameters(map[string]map[string]interface{}{
			"action":   core.StringParam("Action to perform", []string{"store", "retrieve", "search", "list", "delete"}),
			"key":      core.StringParam("Memory key/identifier", nil),
			"value":    core.StringParam("Value to store", nil),
			"category": core.StringParam("Category for organization", nil),
			"tags":     core.ArrayParam("Tags for search", core.StringParam("tag", nil)),
			"query":    core.StringParam("Search query", nil),
		}),
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return handlePersist(s.db, args)
		},
	)
}

// RunServer starts the MCP server
func RunServer(db *storage.Database, aiSvc *ai.AIService, port int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	server := NewServer(db, aiSvc, port)
	return server.Start(ctx)
}
