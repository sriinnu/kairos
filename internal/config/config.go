package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AIProvider types
type AIProvider string

const (
	ProviderOllama AIProvider = "ollama"
	ProviderOpenAI AIProvider = "openai"
	ProviderClaude AIProvider = "claude"
	ProviderGemini AIProvider = "gemini"
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

	cfg := getDefaultConfig()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		applyConfigMap(cfg, raw)
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
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = getDefaultConfig().DatabasePath
	}

	// Expand ~ in database path
	if strings.HasPrefix(cfg.DatabasePath, "~/") {
		home, _ := os.UserHomeDir()
		cfg.DatabasePath = filepath.Join(home, cfg.DatabasePath[2:])
	}
	if !filepath.IsAbs(cfg.DatabasePath) {
		cfg.DatabasePath = filepath.Join(getProjectRoot(), cfg.DatabasePath)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	return filepath.Join(getProjectRoot(), ".kairos", "config.yaml")
}

func getDefaultConfig() *Config {
	dataDir := filepath.Join(getProjectRoot(), ".kairos")
	return &Config{
		DatabasePath:        filepath.Join(dataDir, "data.db"),
		WeeklyGoal:          38.5,
		AIProvider:          ProviderOllama,
		TimeZone:            time.Local.String(),
		OllamaURL:           "http://localhost:11434",
		OllamaModel:         "llama3.2",
		OpenAIModel:         "gpt-4",
		ClaudeModel:         "claude-sonnet-4-20250514",
		GeminiModel:         "gemini-2.0-flash",
		AutoClockoutMinutes: 0, // 0 = disabled
		AutoArchive:         false,
	}
}

func getProjectRoot() string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}

	for _, start := range candidates {
		if root, ok := findProjectRoot(start); ok {
			return root
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "."
}

func findProjectRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		if _, err := os.Stat(filepath.Join(dir, ".kairos", "config.yaml")); err == nil {
			return dir, true
		}
		if _, err := os.Stat(filepath.Join(dir, ".kairos")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// GetLocation returns the time.Location for this config
func (c *Config) GetLocation() *time.Location {
	if c.TimeZone == "" || c.TimeZone == "local" {
		return time.Local
	}
	if loc, ok := parseTimezone(c.TimeZone); ok {
		return loc
	}
	return time.Local
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

func applyConfigMap(cfg *Config, raw map[string]interface{}) {
	for key, value := range raw {
		normalized := normalizeKey(key)
		switch normalized {
		case "databasepath", "database", "db":
			if s, ok := asString(value); ok && s != "" {
				cfg.DatabasePath = s
			}
		case "weeklygoal", "weeklyhours":
			if f, ok := asFloat(value); ok {
				cfg.WeeklyGoal = f
			}
		case "aiprovider", "provider":
			if s, ok := asString(value); ok && s != "" {
				cfg.AIProvider = AIProvider(strings.ToLower(s))
			}
		case "timezone", "timezonelocal", "tz", "time_zone":
			if s, ok := asString(value); ok && s != "" {
				cfg.TimeZone = s
			}
		case "ollamaurl":
			if s, ok := asString(value); ok && s != "" {
				cfg.OllamaURL = s
			}
		case "ollamamodel":
			if s, ok := asString(value); ok && s != "" {
				cfg.OllamaModel = s
			}
		case "openaimodel":
			if s, ok := asString(value); ok && s != "" {
				cfg.OpenAIModel = s
			}
		case "openaiapikey":
			if s, ok := asString(value); ok && s != "" {
				cfg.OpenAIAPIKey = s
			}
		case "claudemodel":
			if s, ok := asString(value); ok && s != "" {
				cfg.ClaudeModel = s
			}
		case "claudeapikey":
			if s, ok := asString(value); ok && s != "" {
				cfg.ClaudeAPIKey = s
			}
		case "geminimodel":
			if s, ok := asString(value); ok && s != "" {
				cfg.GeminiModel = s
			}
		case "geminiapikey":
			if s, ok := asString(value); ok && s != "" {
				cfg.GeminiAPIKey = s
			}
		case "autoclockoutminutes":
			if i, ok := asInt(value); ok {
				cfg.AutoClockoutMinutes = i
			}
		case "autoarchive":
			if b, ok := asBool(value); ok {
				cfg.AutoArchive = b
			}
		}
	}
}

func normalizeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func asString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case []byte:
		return strings.TrimSpace(string(v)), true
	default:
		return "", false
	}
}

func asFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func asInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func asBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return b, true
	default:
		return false, false
	}
}

func parseTimezone(value string) (*time.Location, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false
	}
	upper := strings.ToUpper(trimmed)
	if upper == "UTC" || upper == "GMT" || upper == "Z" {
		return time.UTC, true
	}
	if strings.HasPrefix(upper, "UTC") || strings.HasPrefix(upper, "GMT") {
		offsetPart := strings.TrimSpace(trimmed[3:])
		if offsetPart == "" || strings.EqualFold(offsetPart, "Z") {
			return time.UTC, true
		}
		if loc, ok := parseOffsetLocation(offsetPart); ok {
			return loc, true
		}
	}
	if strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "-") {
		if loc, ok := parseOffsetLocation(trimmed); ok {
			return loc, true
		}
	}
	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, false
	}
	return loc, true
}

func parseOffsetLocation(offset string) (*time.Location, bool) {
	if offset == "" {
		return nil, false
	}
	sign := 1
	switch offset[0] {
	case '+':
		offset = offset[1:]
	case '-':
		sign = -1
		offset = offset[1:]
	default:
		return nil, false
	}
	offset = strings.TrimSpace(offset)
	if offset == "" {
		return nil, false
	}

	hours := 0
	minutes := 0

	if strings.Contains(offset, ":") {
		parts := strings.SplitN(offset, ":", 2)
		if len(parts) != 2 {
			return nil, false
		}
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, false
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, false
		}
		hours = h
		minutes = m
	} else {
		switch len(offset) {
		case 1, 2:
			h, err := strconv.Atoi(offset)
			if err != nil {
				return nil, false
			}
			hours = h
		case 3:
			h, err := strconv.Atoi(offset[:1])
			if err != nil {
				return nil, false
			}
			m, err := strconv.Atoi(offset[1:])
			if err != nil {
				return nil, false
			}
			hours = h
			minutes = m
		case 4:
			h, err := strconv.Atoi(offset[:2])
			if err != nil {
				return nil, false
			}
			m, err := strconv.Atoi(offset[2:])
			if err != nil {
				return nil, false
			}
			hours = h
			minutes = m
		default:
			return nil, false
		}
	}

	if minutes < 0 || minutes >= 60 {
		return nil, false
	}

	seconds := sign * ((hours * 60 * 60) + (minutes * 60))
	name := fmt.Sprintf("UTC%+03d:%02d", sign*hours, minutes)
	return time.FixedZone(name, seconds), true
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
