package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kairos/internal/config"
	"github.com/kairos/internal/tracker"
	"github.com/kairos/internal/work"
)

// Provider interface for AI services
type Provider interface {
	Name() string
	IsAvailable() bool
	Ask(question string, ctx *WorkContext) (string, error)
	Predict(weekProgress *tracker.WeekProgress) (string, error)
	Analyze(dq *DataQuerier) (string, error)
}

// AIService manages AI providers
type AIService struct {
	provider Provider
	cfg      *config.Config
}

func (s *AIService) now() time.Time {
	if s.cfg != nil {
		return s.cfg.Now()
	}
	return time.Now()
}

// NewAIService creates an AI service with the configured provider
func NewAIService(cfg *config.Config) *AIService {
	return &AIService{
		cfg: cfg,
	}
}

// Initialize sets up the provider based on config
func (s *AIService) Initialize() error {
	switch s.cfg.AIProvider {
	case config.ProviderOllama:
		s.provider = NewOllamaProvider(s.cfg.OllamaURL, s.cfg.OllamaModel, s.cfg.GetLocation())
	case config.ProviderOpenAI:
		s.provider = NewOpenAIProvider(s.cfg.OpenAIModel, s.cfg.OpenAIAPIKey, s.cfg.GetLocation())
	case config.ProviderClaude:
		s.provider = NewClaudeProvider(s.cfg.ClaudeModel, s.cfg.ClaudeAPIKey, s.cfg.GetLocation())
	case config.ProviderGemini:
		s.provider = NewGeminiProvider(s.cfg.GeminiModel, s.cfg.GeminiAPIKey, s.cfg.GetLocation())
	default:
		return fmt.Errorf("unknown AI provider: %s", s.cfg.AIProvider)
	}
	return nil
}

// IsAvailable checks if the current provider is available
func (s *AIService) IsAvailable() bool {
	if s.provider == nil {
		return false
	}
	return s.provider.IsAvailable()
}

// Name returns the current provider name
func (s *AIService) Name() string {
	if s.provider == nil {
		return "none"
	}
	return s.provider.Name()
}

// Ask sends a question to the AI
func (s *AIService) Ask(question string, ctx *WorkContext) (string, error) {
	if s.provider == nil {
		return s.offlineAsk(question, ctx), nil
	}

	if !s.provider.IsAvailable() {
		return s.offlineAsk(question, ctx), nil
	}

	return s.provider.Ask(question, ctx)
}

// Predict generates predictions
func (s *AIService) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	if s.provider == nil || !s.provider.IsAvailable() {
		return s.offlinePredict(weekProgress), nil
	}
	return s.provider.Predict(weekProgress)
}

// Analyze provides work pattern analysis
func (s *AIService) Analyze(dq *DataQuerier) (string, error) {
	if s.provider == nil || !s.provider.IsAvailable() {
		return s.offlineAnalyze(dq), nil
	}
	return s.provider.Analyze(dq)
}

// offlineAsk provides rule-based responses when AI is unavailable
func (s *AIService) offlineAsk(question string, ctx *WorkContext) string {
	question = strings.ToLower(question)

	// Check for common questions and provide helpful responses
	if strings.Contains(question, "leave") || strings.Contains(question, "done") || strings.Contains(question, "go home") {
		if ctx.RemainingHours > 0 {
			return fmt.Sprintf("You need %.2f more hours to reach your weekly goal. At your current pace, plan to work about %.2f hours per remaining day.", ctx.RemainingHours, ctx.DailyTarget)
		}
		return "You've exceeded your weekly goal! You're done for the week. Great work!"
	}

	if strings.Contains(question, "hours today") || strings.Contains(question, "worked today") {
		return fmt.Sprintf("You've worked %.2f hours today.", ctx.TodayHours)
	}

	if strings.Contains(question, "hours left") || strings.Contains(question, "remaining") {
		if ctx.RemainingHours > 0 {
			return fmt.Sprintf("You have %.2f hours remaining to reach your weekly goal (%.2f hours/day over %d days).", ctx.RemainingHours, ctx.DailyTarget, ctx.RemainingDays)
		}
		return "You've reached your weekly goal! No more hours needed."
	}

	if strings.Contains(question, "behind") || strings.Contains(question, "track") {
		if ctx.RemainingDays > 0 && ctx.DailyTarget > 0 {
			return fmt.Sprintf("You need %.2f hours/day to hit your goal. You have %d days left.", ctx.DailyTarget, ctx.RemainingDays)
		}
		return "You're on track or ahead of schedule!"
	}

	// Default helpful response
	return fmt.Sprintf("Current status: %.2f/%.2f hours this week (%.2f remaining). You need %.2f hours/day over %d days. AI is unavailable - install Ollama for smart insights!", ctx.WeekHours, ctx.WeeklyGoal, ctx.RemainingHours, ctx.DailyTarget, ctx.RemainingDays)
}

// offlinePredict provides rule-based predictions
func (s *AIService) offlinePredict(weekProgress *tracker.WeekProgress) string {
	remainingDays := work.RemainingWorkDaysInWeek(s.now())
	goal := weeklyGoalFromProgress(weekProgress)

	if weekProgress.RemainingHours <= 0 {
		return "You've already reached your weekly goal! Well done!"
	}

	if remainingDays <= 0 {
		return fmt.Sprintf("No work days left this week. You're %.2f hours short of your goal.", weekProgress.RemainingHours)
	}

	dailyTarget := weekProgress.RemainingHours / float64(remainingDays)

	return fmt.Sprintf("Prediction: You need %.2f more hours to reach your %.2fh weekly goal. That's %.2f hours/day over %d remaining work days. AI unavailable - install Ollama for detailed analysis!", weekProgress.RemainingHours, goal, dailyTarget, remainingDays)
}

// offlineAnalyze provides basic analysis without AI
func (s *AIService) offlineAnalyze(dq *DataQuerier) string {
	// Get basic stats
	week, _ := dq.GetWeekHours()
	goal := dq.weeklyGoal()

	if week >= goal {
		return "You've already hit your weekly goal! Great consistency this week."
	}

	remaining := goal - week
	daysLeft := work.RemainingWorkDaysInWeek(s.now())

	if daysLeft > 0 {
		daily := remaining / float64(daysLeft)
		return fmt.Sprintf("Analysis: %.2f/%.2f hours this week (%.2f remaining). At %.2fh/day over %d days, you can still reach your goal.", week, goal, remaining, daily, daysLeft)
	}

	return fmt.Sprintf("Analysis: You've logged %.2f/%.2f hours this week with no days left. Install Ollama for smarter insights!", week, goal)
}

// WorkContext contains all the work data for AI queries
type WorkContext struct {
	TodayHours          float64
	WeekHours           float64
	MonthHours          float64
	WeeklyGoal          float64
	RemainingHours      float64
	DaysWorked          int
	RemainingDays       int
	DailyTarget         float64
	IsWorking           bool
	CurrentSessionStart string
	DailyBreakdown      map[string]float64
}

// BuildWorkContext creates a comprehensive context from tracker data
func BuildWorkContext(t *tracker.Tracker) (*WorkContext, error) {
	dayProgress, err := t.GetTodayProgress()
	if err != nil {
		return nil, err
	}

	weekProgress, err := t.GetWeeklyProgress()
	if err != nil {
		return nil, err
	}

	monthProgress, err := t.GetMonthlyProgress()
	if err != nil {
		return nil, err
	}

	activeSession, _ := t.GetActiveSession()

	ctx := &WorkContext{
		TodayHours:     dayProgress.TotalHours,
		WeekHours:      weekProgress.TotalHours,
		MonthHours:     monthProgress.TotalHours,
		WeeklyGoal:     t.WeeklyGoal(),
		RemainingHours: weekProgress.RemainingHours,
		DaysWorked:     weekProgress.DaysWorkedCount,
		RemainingDays:  work.RemainingWorkDaysInWeek(t.Now()),
		DailyBreakdown: weekProgress.DaysWorked,
		IsWorking:      activeSession != nil,
	}

	if ctx.RemainingDays > 0 && ctx.RemainingHours > 0 {
		ctx.DailyTarget = ctx.RemainingHours / float64(ctx.RemainingDays)
	}

	if activeSession != nil {
		ctx.CurrentSessionStart = activeSession.StartTime.Format("15:04")
	}

	return ctx, nil
}

// ==================== Ollama Provider ====================

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
	loc     *time.Location
}

func NewOllamaProvider(baseURL, model string, loc *time.Location) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		loc: loc,
	}
}

func (o *OllamaProvider) now() time.Time {
	if o.loc != nil {
		return time.Now().In(o.loc)
	}
	return time.Now()
}

func (o *OllamaProvider) Name() string {
	return "Ollama (" + o.model + ")"
}

func (o *OllamaProvider) IsAvailable() bool {
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

func (o *OllamaProvider) Ask(question string, ctx *WorkContext) (string, error) {
	prompt := o.buildPrompt(question, ctx)
	return o.query(prompt)
}

func (o *OllamaProvider) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	remainingDays := work.RemainingWorkDaysInWeek(o.now())
	dailyTarget := 0.0
	if remainingDays > 0 && weekProgress.RemainingHours > 0 {
		dailyTarget = weekProgress.RemainingHours / float64(remainingDays)
	}

	weeklyGoal := weeklyGoalFromProgress(weekProgress)
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
		weeklyGoal,
		weekProgress.DaysWorkedCount,
		weekProgress.RemainingHours,
		remainingDays,
		dailyTarget)

	return o.query(prompt)
}

func (o *OllamaProvider) Analyze(dq *DataQuerier) (string, error) {
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

func (o *OllamaProvider) query(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reqBody := map[string]interface{}{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OllamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Response, nil
}

func (o *OllamaProvider) buildPrompt(question string, ctx *WorkContext) string {
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

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// ==================== OpenAI Provider ====================

type OpenAIProvider struct {
	model  string
	apiKey string
	client *http.Client
	loc    *time.Location
}

func NewOpenAIProvider(model, apiKey string, loc *time.Location) *OpenAIProvider {
	return &OpenAIProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		loc: loc,
	}
}

func (o *OpenAIProvider) now() time.Time {
	if o.loc != nil {
		return time.Now().In(o.loc)
	}
	return time.Now()
}

func (o *OpenAIProvider) Name() string {
	return "OpenAI (" + o.model + ")"
}

func (o *OpenAIProvider) IsAvailable() bool {
	if o.apiKey == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func (o *OpenAIProvider) Ask(question string, ctx *WorkContext) (string, error) {
	return o.chatCompletion(o.buildMessages(question, ctx))
}

func (o *OpenAIProvider) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	remainingDays := work.RemainingWorkDaysInWeek(o.now())
	dailyTarget := 0.0
	if remainingDays > 0 && weekProgress.RemainingHours > 0 {
		dailyTarget = weekProgress.RemainingHours / float64(remainingDays)
	}

	weeklyGoal := weeklyGoalFromProgress(weekProgress)
	systemMsg := "You are a helpful work hours assistant. Provide concise predictions."
	userMsg := fmt.Sprintf(`Based on my work data:
- Hours worked: %.2f
- Weekly goal: %.2f hours
- Days worked: %d
- Remaining: %.2f hours
- Remaining days: %d
- Daily target: %.2f hours

Predict when I'll reach my goal and if I'm on track.`,
		weekProgress.TotalHours, weeklyGoal, weekProgress.DaysWorkedCount,
		weekProgress.RemainingHours, remainingDays, dailyTarget)

	return o.chatCompletion([]OpenAIMessage{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: userMsg},
	})
}

func (o *OpenAIProvider) Analyze(dq *DataQuerier) (string, error) {
	dataContext, err := dq.BuildDataContext()
	if err != nil {
		return "", err
	}

	systemMsg := "You are a work hours analyst. Keep responses concise (3-4 sentences)."
	userMsg := "Analyze my work patterns and give one actionable suggestion:\n\n" + dataContext

	return o.chatCompletion([]OpenAIMessage{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: userMsg},
	})
}

func (o *OpenAIProvider) buildMessages(question string, ctx *WorkContext) []OpenAIMessage {
	status := "Not working"
	if ctx.IsWorking {
		status = fmt.Sprintf("Working since %s", ctx.CurrentSessionStart)
	}

	systemMsg := fmt.Sprintf(`You are a helpful work hours assistant. Current user data:
- Today: %.2f hours
- This week: %.2f hours (goal: %.2f hours)
- This month: %.2f hours
- Status: %s
- Days worked: %d
- Remaining: %.2f hours over %d days (%.2f h/day)`,
		ctx.TodayHours, ctx.WeekHours, ctx.WeeklyGoal,
		ctx.MonthHours, status, ctx.DaysWorked, ctx.RemainingHours, ctx.RemainingDays, ctx.DailyTarget)

	return []OpenAIMessage{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: question},
	}
}

func (o *OpenAIProvider) chatCompletion(messages []OpenAIMessage) (string, error) {
	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not set. Use: kairos config --openai-key YOUR_KEY")
	}

	reqBody := OpenAIRequest{
		Model:    o.model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI error: %s", string(body))
	}

	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
}

// ==================== Claude Provider ====================

type ClaudeProvider struct {
	model  string
	apiKey string
	client *http.Client
	loc    *time.Location
}

func NewClaudeProvider(model, apiKey string, loc *time.Location) *ClaudeProvider {
	return &ClaudeProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		loc: loc,
	}
}

func (c *ClaudeProvider) now() time.Time {
	if c.loc != nil {
		return time.Now().In(c.loc)
	}
	return time.Now()
}

func (c *ClaudeProvider) Name() string {
	return "Claude (" + c.model + ")"
}

func (c *ClaudeProvider) IsAvailable() bool {
	if c.apiKey == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/messages", nil)
	if err != nil {
		return false
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200 || resp.StatusCode == 401 // 401 = valid key, wrong endpoint
}

func (c *ClaudeProvider) Ask(question string, ctx *WorkContext) (string, error) {
	return c.claudeMessage(c.buildPrompt(question, ctx))
}

func (c *ClaudeProvider) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	remainingDays := work.RemainingWorkDaysInWeek(c.now())
	dailyTarget := 0.0
	if remainingDays > 0 && weekProgress.RemainingHours > 0 {
		dailyTarget = weekProgress.RemainingHours / float64(remainingDays)
	}

	weeklyGoal := weeklyGoalFromProgress(weekProgress)
	prompt := fmt.Sprintf(`You are a work hours assistant. Based on this data:
- Hours worked: %.2f / %.2f goal
- Days worked: %d
- Remaining: %.2f hours over %d days (%.2f h/day)

Predict when I'll reach my goal and if I'm on track. Keep response concise.`,
		weekProgress.TotalHours, weeklyGoal, weekProgress.DaysWorkedCount,
		weekProgress.RemainingHours, remainingDays, dailyTarget)

	return c.claudeMessage(prompt)
}

func (c *ClaudeProvider) Analyze(dq *DataQuerier) (string, error) {
	dataContext, err := dq.BuildDataContext()
	if err != nil {
		return "", err
	}

	prompt := "Analyze my work patterns and give one actionable suggestion:\n\n" + dataContext
	return c.claudeMessage(prompt)
}

func (c *ClaudeProvider) buildPrompt(question string, ctx *WorkContext) string {
	status := "Not working"
	if ctx.IsWorking {
		status = fmt.Sprintf("Working since %s", ctx.CurrentSessionStart)
	}

	return fmt.Sprintf(`You are a helpful work hours assistant. Current data:
- Today: %.2f hours | Week: %.2f/%.2f | Month: %.2f hours
- Status: %s | Days: %d | Remaining: %.2fh (%d days, %.2fh/day)

Question: %s`,
		ctx.TodayHours, ctx.WeekHours, ctx.WeeklyGoal, ctx.MonthHours,
		status, ctx.DaysWorked, ctx.RemainingHours, ctx.RemainingDays, ctx.DailyTarget,
		question)
}

func (c *ClaudeProvider) claudeMessage(prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("Claude API key not set. Use: kairos config --claude-key YOUR_KEY")
	}

	reqBody := ClaudeRequest{
		Model: c.model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 1024,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Claude request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude error: %s", string(body))
	}

	var result ClaudeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Content[0].Text, nil
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []ClaudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// ==================== Gemini Provider ====================

type GeminiProvider struct {
	model  string
	apiKey string
	client *http.Client
	loc    *time.Location
}

func NewGeminiProvider(model, apiKey string, loc *time.Location) *GeminiProvider {
	return &GeminiProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		loc: loc,
	}
}

func (g *GeminiProvider) now() time.Time {
	if g.loc != nil {
		return time.Now().In(g.loc)
	}
	return time.Now()
}

func (g *GeminiProvider) Name() string {
	return "Gemini (" + g.model + ")"
}

func (g *GeminiProvider) IsAvailable() bool {
	if g.apiKey == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func (g *GeminiProvider) Ask(question string, ctx *WorkContext) (string, error) {
	return g.geminiGenerate(g.buildPrompt(question, ctx))
}

func (g *GeminiProvider) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	remainingDays := work.RemainingWorkDaysInWeek(g.now())
	weeklyGoal := weeklyGoalFromProgress(weekProgress)

	prompt := fmt.Sprintf(`Work hours prediction: %.2f/%.2f hours, %d days worked, %.2f remaining over %d days.
Predict when I'll reach my goal. Be concise.`,
		weekProgress.TotalHours, weeklyGoal, weekProgress.DaysWorkedCount,
		weekProgress.RemainingHours, remainingDays)

	return g.geminiGenerate(prompt)
}

func (g *GeminiProvider) Analyze(dq *DataQuerier) (string, error) {
	dataContext, err := dq.BuildDataContext()
	if err != nil {
		return "", err
	}

	prompt := "Analyze work patterns, give one suggestion:\n\n" + dataContext
	return g.geminiGenerate(prompt)
}

func (g *GeminiProvider) buildPrompt(question string, ctx *WorkContext) string {
	status := "Not working"
	if ctx.IsWorking {
		status = fmt.Sprintf("Working since %s", ctx.CurrentSessionStart)
	}

	return fmt.Sprintf(`Work hours assistant. Current: Today=%.2fh, Week=%.2f/%.2f, Month=%.2f, Status=%s, Days=%d, Remaining=%.2fh/%dd@%.2fh/day. Question: %s`,
		ctx.TodayHours, ctx.WeekHours, ctx.WeeklyGoal, ctx.MonthHours,
		status, ctx.DaysWorked, ctx.RemainingHours, ctx.RemainingDays, ctx.DailyTarget,
		question)
}

func (g *GeminiProvider) geminiGenerate(prompt string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("Gemini API key not set. Use: kairos config --gemini-key YOUR_KEY")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini error: %s", string(body))
	}

	var result GeminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content GeminiContent `json:"content"`
	} `json:"candidates"`
}

func weeklyGoalFromProgress(weekProgress *tracker.WeekProgress) float64 {
	if weekProgress == nil {
		return work.WeeklyGoalHours
	}
	goal := weekProgress.TotalHours + weekProgress.RemainingHours
	if goal <= 0 {
		return work.WeeklyGoalHours
	}
	return goal
}
