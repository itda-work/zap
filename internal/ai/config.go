package ai

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the AI module configuration.
type Config struct {
	// Default is the default provider (auto, claude, codex, gemini)
	Default string `yaml:"default"`

	// Claude CLI configuration
	Claude ProviderConfig `yaml:"claude"`

	// Codex CLI configuration
	Codex ProviderConfig `yaml:"codex"`

	// Gemini CLI configuration
	Gemini ProviderConfig `yaml:"gemini"`

	// Timeout for CLI execution
	Timeout time.Duration `yaml:"timeout"`

	// TemplatesDir is the custom prompt templates directory
	TemplatesDir string `yaml:"templates_dir"`
}

// ProviderConfig holds CLI provider specific configuration.
type ProviderConfig struct {
	Model string `yaml:"model"` // Model name (optional)
	Bin   string `yaml:"bin"`   // Custom binary path (optional)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Default: "auto",
		Claude: ProviderConfig{
			Bin: "claude",
		},
		Codex: ProviderConfig{
			Bin: "codex",
		},
		Gemini: ProviderConfig{
			Bin: "gemini",
		},
		Timeout: 60 * time.Second,
	}
}

// LoadConfig loads the AI configuration from the default path.
func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	// Start with defaults
	cfg := DefaultConfig()

	// Try to load from file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Expand ~ in templates_dir
	if cfg.TemplatesDir != "" {
		cfg.TemplatesDir = expandPath(cfg.TemplatesDir)
	}

	return cfg, nil
}

// getConfigPath returns the default config file path.
func getConfigPath() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "zap", "ai.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zap", "ai.yaml")
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
