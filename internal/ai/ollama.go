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
			Timeout: 30 * time.Second,
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

func (o *Ollama) Ask(question string, dayProgress *tracker.DayProgress, weekProgress *tracker.WeekProgress) (string, error) {
	prompt := o.buildPrompt(question, dayProgress, weekProgress)

	reqBody := ChatRequest{
		Model:    o.model,
		Prompt:   prompt,
		Stream:   false,
		Format:   "json",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(o.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (o *Ollama) Predict(weekProgress *tracker.WeekProgress) (string, error) {
	prompt := fmt.Sprintf(`Based on the following work week data:
- Total hours worked so far: %.2f
- Weekly goal: %.2f hours
- Days worked: %d
- Remaining hours to goal: %.2f

Predict:
1. When will I reach my weekly goal?
2. What should be my average hours per remaining day?
3. Am I on track to meet my goal?

Respond in a friendly, concise manner.`, weekProgress.TotalHours, 38.5, weekProgress.DaysWorkedCount, weekProgress.RemainingHours)

	reqBody := ChatRequest{
		Model:    o.model,
		Prompt:   prompt,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(o.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (o *Ollama) buildPrompt(question string, dayProgress *tracker.DayProgress, weekProgress *tracker.WeekProgress) string {
	prompt := fmt.Sprintf(`You are a helpful work hours assistant. Current user data:
- Today: %.2f hours worked
- This week: %.2f hours worked
- Weekly goal: 38.5 hours
- Days worked this week: %d

User question: "%s"

Provide a helpful, concise answer.`, dayProgress.TotalHours, weekProgress.TotalHours, weekProgress.DaysWorkedCount, question)
	return prompt
}

type ChatRequest struct {
	Model    string  `json:"model"`
	Prompt   string  `json:"prompt"`
	Stream   bool    `json:"stream,omitempty"`
	Format   string  `json:"format,omitempty"`
	Options  Options `json:"options,omitempty"`
}

type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
}

type ChatResponse struct {
	Response string `json:"response"`
}
