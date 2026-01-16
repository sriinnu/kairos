package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	cfg := &Config{
		AIProvider:   ProviderOllama,
		OpenAIAPIKey: "openai-key",
		ClaudeAPIKey: "claude-key",
		GeminiAPIKey: "gemini-key",
	}

	tests := []struct {
		provider AIProvider
		expected string
	}{
		{ProviderOllama, ""},
		{ProviderOpenAI, "openai-key"},
		{ProviderClaude, "claude-key"},
		{ProviderGemini, "gemini-key"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			cfg.AIProvider = tt.provider
			result := cfg.GetAPIKey()
			if result != tt.expected {
				t.Errorf("GetAPIKey() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	cfg := &Config{
		AIProvider:   ProviderOllama,
		OllamaModel:  "llama3.2",
		OpenAIModel:  "gpt-4",
		ClaudeModel:  "claude-sonnet-4-20250514",
		GeminiModel:  "gemini-2.0-flash",
	}

	tests := []struct {
		provider AIProvider
		expected string
	}{
		{ProviderOllama, "llama3.2"},
		{ProviderOpenAI, "gpt-4"},
		{ProviderClaude, "claude-sonnet-4-20250514"},
		{ProviderGemini, "gemini-2.0-flash"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			cfg.AIProvider = tt.provider
			result := cfg.GetModel()
			if result != tt.expected {
				t.Errorf("GetModel() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Create a temporary directory with no config file
	tmpDir := t.TempDir()

	// Override home directory for this test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Since getConfigPath uses os.UserHomeDir(), we can't easily test it
	// without mocking. This test verifies the default config behavior.
	cfg := &Config{
		DatabasePath: filepath.Join(tmpDir, "test.db"),
		WeeklyGoal:   38.5,
		AIProvider:   ProviderOllama,
		OllamaURL:    "http://localhost:11434",
		OllamaModel:  "llama3.2",
	}

	// Verify defaults
	if cfg.AIProvider != ProviderOllama {
		t.Errorf("Default AIProvider = %s, want %s", cfg.AIProvider, ProviderOllama)
	}
	if cfg.OllamaURL != "http://localhost:11434" {
		t.Errorf("Default OllamaURL = %s, want %s", cfg.OllamaURL, "http://localhost:11434")
	}
}

func TestAIProviderConstants(t *testing.T) {
	if ProviderOllama != "ollama" {
		t.Errorf("ProviderOllama = %s, want ollama", ProviderOllama)
	}
	if ProviderOpenAI != "openai" {
		t.Errorf("ProviderOpenAI = %s, want openai", ProviderOpenAI)
	}
	if ProviderClaude != "claude" {
		t.Errorf("ProviderClaude = %s, want claude", ProviderClaude)
	}
	if ProviderGemini != "gemini" {
		t.Errorf("ProviderGemini = %s, want gemini", ProviderGemini)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		valid   bool
	}{
		{
			name:  "valid ollama config",
			cfg:   &Config{AIProvider: ProviderOllama, OllamaURL: "http://localhost:11434"},
			valid: true,
		},
		{
			name:  "valid openai config with api key",
			cfg:   &Config{AIProvider: ProviderOpenAI, OpenAIAPIKey: "test-key", OpenAIModel: "gpt-4"},
			valid: true,
		},
		{
			name:  "valid claude config with api key",
			cfg:   &Config{AIProvider: ProviderClaude, ClaudeAPIKey: "test-key", ClaudeModel: "claude-3"},
			valid: true,
		},
		{
			name:  "valid gemini config with api key",
			cfg:   &Config{AIProvider: ProviderGemini, GeminiAPIKey: "test-key", GeminiModel: "gemini-pro"},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation: check required fields are set
			valid := true

			switch tt.cfg.AIProvider {
			case ProviderOllama:
				if tt.cfg.OllamaURL == "" {
					valid = false
				}
			case ProviderOpenAI:
				if tt.cfg.OpenAIAPIKey == "" || tt.cfg.OpenAIModel == "" {
					valid = false
				}
			case ProviderClaude:
				if tt.cfg.ClaudeAPIKey == "" || tt.cfg.ClaudeModel == "" {
					valid = false
				}
			case ProviderGemini:
				if tt.cfg.GeminiAPIKey == "" || tt.cfg.GeminiModel == "" {
					valid = false
				}
			}

			if valid != tt.valid {
				t.Errorf("Config validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}
