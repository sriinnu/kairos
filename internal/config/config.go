package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DatabasePath string  `yaml:"DatabasePath"`
	WeeklyGoal   float64 `yaml:"WeeklyGoal"`
	OllamaURL    string  `yaml:"OllamaURL"`
	OllamaModel  string  `yaml:"OllamaModel"`
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
	return filepath.Join(home, ".samaya.yaml")
}

func getDefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		DatabasePath: filepath.Join(home, ".samaya", "data.db"),
		WeeklyGoal:   38.5,
		OllamaURL:    "http://localhost:11434",
		OllamaModel:  "llama3.2",
	}
}
