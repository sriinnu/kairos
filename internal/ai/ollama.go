package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kairos/internal/tracker"
	"github.com/kairos/internal/work"
)

type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

func New(baseURL, model string) *Ollama {
	return &Ollama{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (o *Ollama) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Ask queries the AI with work context - handles nil dayProgress gracefully
func (o *Ollama) Ask(question string, dayProgress *tracker.DayProgress, weekProgress *tracker.WeekProgress) (string, error) {
	prompt := o.buildPrompt(question, dayProgress, weekProgress)
	return o.query(prompt)
}

// AskWithContext queries the AI with full work context
func (o *Ollama) AskWithContext(question string, ctx *WorkContext) (string, error) {
	prompt := o.buildContextPrompt(question, ctx)
	return o.query(prompt)
}

// Predict generates predictions based on weekly progress
func (o *Ollama) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	remainingDays := work.RemainingWorkDaysInWeek(time.Now())
	dailyTarget := 0.0
	if remainingDays > 0 && weekProgress.RemainingHours > 0 {
		dailyTarget = weekProgress.RemainingHours / float64(remainingDays)
	}

	prompt := fmt.Sprintf(`Based on the following work week data:
- Total hours worked so far: %.2f
- Weekly goal: %.2f hours
- Days worked: %d
- Remaining hours to goal: %.2f
- Remaining work days: %d
- Required daily average: %.2f hours

Predict:
1. When will I reach my weekly goal?
2. What should be my average hours per remaining day?
3. Am I on track to meet my goal?

Respond in a friendly, concise manner.`,
		weekProgress.TotalHours,
		work.WeeklyGoalHours,
		weekProgress.DaysWorkedCount,
		weekProgress.RemainingHours,
		remainingDays,
		dailyTarget)

	return o.query(prompt)
}

func (o *Ollama) query(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reqBody := ChatRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Response, nil
}

func (o *Ollama) buildPrompt(question string, dayProgress *tracker.DayProgress, weekProgress *tracker.WeekProgress) string {
	todayHours := 0.0
	if dayProgress != nil {
		todayHours = dayProgress.TotalHours
	}

	weekHours := 0.0
	daysWorked := 0
	if weekProgress != nil {
		weekHours = weekProgress.TotalHours
		daysWorked = weekProgress.DaysWorkedCount
	}

	return fmt.Sprintf(`You are a helpful work hours assistant. Current user data:
- Today: %.2f hours worked
- This week: %.2f hours worked
- Weekly goal: %.2f hours
- Days worked this week: %d
- Standard break: %d minutes (no break on Fridays)

User question: "%s"

Provide a helpful, concise answer based on the data above.`,
		todayHours,
		weekHours,
		work.WeeklyGoalHours,
		daysWorked,
		work.DefaultBreakMinutes,
		question)
}

func (o *Ollama) buildContextPrompt(question string, ctx *WorkContext) string {
	workingStatus := "Not currently working"
	if ctx.IsWorking {
		workingStatus = fmt.Sprintf("Currently working (started at %s)", ctx.CurrentSessionStart)
	}

	return fmt.Sprintf(`You are a helpful work hours assistant with access to the user's time tracking data.

Current Status:
- %s
- Today: %.2f hours worked
- This week: %.2f hours worked (goal: %.2f hours)
- This month: %.2f hours worked
- Days worked this week: %d
- Remaining to weekly goal: %.2f hours
- Remaining work days: %d
- Required daily average: %.2f hours
- Standard break: %d minutes (no break on Fridays)

User question: "%s"

Answer based on the data above. Be concise and helpful.`,
		workingStatus,
		ctx.TodayHours,
		ctx.WeekHours,
		ctx.WeeklyGoal,
		ctx.MonthHours,
		ctx.DaysWorked,
		ctx.RemainingHours,
		ctx.RemainingDays,
		ctx.DailyTarget,
		work.DefaultBreakMinutes,
		question)
}

type ChatRequest struct {
	Model   string  `json:"model"`
	Prompt  string  `json:"prompt"`
	Stream  bool    `json:"stream,omitempty"`
	Format  string  `json:"format,omitempty"`
	Options Options `json:"options,omitempty"`
}

type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// AskWithDataQuery queries the AI with full data context from the database
func (o *Ollama) AskWithDataQuery(question string, dq *DataQuerier) (string, error) {
	dataContext, err := dq.BuildDataContext()
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a helpful work hours assistant with access to the user's time tracking database.

%s
User question: "%s"

Answer based on the data above. Be concise and helpful. If the question requires calculations, show your work.`,
		dataContext, question)

	return o.query(prompt)
}

// AnalyzeWorkPatterns analyzes work patterns and provides insights
func (o *Ollama) AnalyzeWorkPatterns(dq *DataQuerier) (string, error) {
	dataContext, err := dq.BuildDataContext()
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a work hours analyst. Based on this data:

%s
Provide a brief analysis of:
1. Current pace towards weekly goal
2. Any patterns you notice
3. One actionable suggestion

Keep it concise (3-4 sentences max).`, dataContext)

	return o.query(prompt)
}
