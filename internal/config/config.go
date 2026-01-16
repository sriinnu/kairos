package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AIProvider types
type AIProvider string

const (
	ProviderOllama  AIProvider = "ollama"
	ProviderOpenAI  AIProvider = "openai"
	ProviderClaude  AIProvider = "claude"
	ProviderGemini  AIProvider = "gemini"
)

type Config struct {
	DatabasePath string     `yaml:"DatabasePath"`
	WeeklyGoal   float64    `yaml:"WeeklyGoal"`
	AIProvider   AIProvider `yaml:"AIProvider"`
	TimeZone     string     `yaml:"TimeZone"`

	// Ollama settings
	OllamaURL   string `yaml:"OllamaURL"`
	OllamaModel string `yaml:"OllamaModel"`

	// OpenAI settings
	OpenAIModel  string `yaml:"OpenAIModel"`
	OpenAIAPIKey string `yaml:"OpenAIAPIKey"`

	// Claude settings
	ClaudeModel  string `yaml:"ClaudeModel"`
	ClaudeAPIKey string `yaml:"ClaudeAPIKey"`

	// Gemini settings
	GeminiModel  string `yaml:"GeminiModel"`
	GeminiAPIKey string `yaml:"GeminiAPIKey"`

	// Auto-clockout settings
	AutoClockoutMinutes int  `yaml:"AutoClockoutMinutes"`
	AutoArchive         bool `yaml:"AutoArchive"`
}

func Load() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults for missing values
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.OllamaModel == "" {
		cfg.OllamaModel = "llama3.2"
	}
	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = "gpt-4"
	}
	if cfg.ClaudeModel == "" {
		cfg.ClaudeModel = "claude-sonnet-4-20250514"
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "gemini-2.0-flash"
	}
	if cfg.AIProvider == "" {
		cfg.AIProvider = ProviderOllama
	}

	// Expand ~ in database path
	if strings.HasPrefix(cfg.DatabasePath, "~/") {
		home, _ := os.UserHomeDir()
		cfg.DatabasePath = filepath.Join(home, cfg.DatabasePath[2:])
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	configPath := getConfigPath()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kairos.yaml")
}

func getDefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		DatabasePath:       filepath.Join(home, ".kairos", "data.db"),
		WeeklyGoal:         38.5,
		AIProvider:         ProviderOllama,
		TimeZone:           time.Local.String(),
		OllamaURL:          "http://localhost:11434",
		OllamaModel:        "llama3.2",
		OpenAIModel:        "gpt-4",
		ClaudeModel:        "claude-sonnet-4-20250514",
		GeminiModel:        "gemini-2.0-flash",
		AutoClockoutMinutes: 0, // 0 = disabled
		AutoArchive:         true,
	}
}

// GetLocation returns the time.Location for this config
func (c *Config) GetLocation() *time.Location {
	if c.TimeZone == "" || c.TimeZone == "local" {
		return time.Local
	}
	loc, err := time.LoadLocation(c.TimeZone)
	if err != nil {
		return time.Local
	}
	return loc
}

// Now returns the current time in the configured timezone
func (c *Config) Now() time.Time {
	return time.Now().In(c.GetLocation())
}

// GetAPIKey returns the API key for the current provider
func (c *Config) GetAPIKey() string {
	switch c.AIProvider {
	case ProviderOpenAI:
		return c.OpenAIAPIKey
	case ProviderClaude:
		return c.ClaudeAPIKey
	case ProviderGemini:
		return c.GeminiAPIKey
	default:
		return ""
	}
}

// GetModel returns the model name for the current provider
func (c *Config) GetModel() string {
	switch c.AIProvider {
	case ProviderOllama:
		return c.OllamaModel
	case ProviderOpenAI:
		return c.OpenAIModel
	case ProviderClaude:
		return c.ClaudeModel
	case ProviderGemini:
		return c.GeminiModel
	default:
		return ""
	}
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s - %s", e.Field, e.Message)
}

// Validate checks the configuration for common issues
func (c *Config) Validate() error {
	// Check for missing required fields based on provider
	switch c.AIProvider {
	case ProviderOllama:
		if c.OllamaURL == "" {
			return &ValidationError{Field: "OllamaURL", Message: "Ollama URL is required"}
		}
	case ProviderOpenAI:
		if c.OpenAIModel == "" {
			return &ValidationError{Field: "OpenAIModel", Message: "OpenAI model is required"}
		}
		if c.OpenAIAPIKey == "" {
			return &ValidationError{Field: "OpenAIAPIKey", Message: "OpenAI API key is required (set OPENAI_API_KEY env var)"}
		}
	case ProviderClaude:
		if c.ClaudeModel == "" {
			return &ValidationError{Field: "ClaudeModel", Message: "Claude model is required"}
		}
		if c.ClaudeAPIKey == "" {
			return &ValidationError{Field: "ClaudeAPIKey", Message: "Claude API key is required (set ANTHROPIC_API_KEY env var)"}
		}
	case ProviderGemini:
		if c.GeminiModel == "" {
			return &ValidationError{Field: "GeminiModel", Message: "Gemini model is required"}
		}
		if c.GeminiAPIKey == "" {
			return &ValidationError{Field: "GeminiAPIKey", Message: "Gemini API key is required (set GEMINI_API_KEY env var)"}
		}
	}

	// Validate weekly goal is positive
	if c.WeeklyGoal <= 0 {
		return &ValidationError{Field: "WeeklyGoal", Message: "Weekly goal must be positive"}
	}

	// Validate database path is set
	if c.DatabasePath == "" {
		return &ValidationError{Field: "DatabasePath", Message: "Database path is required"}
	}

	return nil
}

// IsConfigured returns true if the current provider appears to be properly configured
func (c *Config) IsConfigured() bool {
	switch c.AIProvider {
	case ProviderOllama:
		return c.OllamaURL != ""
	case ProviderOpenAI:
		return c.OpenAIAPIKey != ""
	case ProviderClaude:
		return c.ClaudeAPIKey != ""
	case ProviderGemini:
		return c.GeminiAPIKey != ""
	}
	return false
}
