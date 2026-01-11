package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Model settings
	Model struct {
		Default string `yaml:"default"`
	} `yaml:"model"`

	// VAD settings
	VAD struct {
		Enabled      bool    `yaml:"enabled"`
		Threshold    float64 `yaml:"threshold"`
		SilenceDelay float64 `yaml:"silence_delay"`
	} `yaml:"vad"`

	// Output settings
	Output struct {
		Format string `yaml:"format"`
		File   string `yaml:"file"`
	} `yaml:"output"`

	// Audio settings
	Audio struct {
		Device string `yaml:"device"`
	} `yaml:"audio"`

	// Server settings (for future use)
	Server struct {
		Mode      string `yaml:"mode"`
		Port      int    `yaml:"port"`
		Host      string `yaml:"host"`
		EnableTLS bool   `yaml:"enable_tls"`
		CertFile  string `yaml:"cert_file"`
		KeyFile   string `yaml:"key_file"`
	} `yaml:"server"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	cfg := &Config{}

	// Model defaults
	cfg.Model.Default = ""

	// VAD defaults
	cfg.VAD.Enabled = true
	cfg.VAD.Threshold = 0.01
	cfg.VAD.SilenceDelay = 5.0

	// Output defaults
	cfg.Output.Format = "json"
	cfg.Output.File = ""

	// Audio defaults
	cfg.Audio.Device = ""

	// Server defaults
	cfg.Server.Mode = "cli"
	cfg.Server.Port = 8080
	cfg.Server.Host = "localhost"
	cfg.Server.EnableTLS = false

	return cfg
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadWithFallback attempts to load configuration from multiple locations
// Priority: explicit path > ~/.voxrc > /etc/vox/config.yaml
func LoadWithFallback(explicitPath string) (*Config, error) {
	// If explicit path is provided, use it
	if explicitPath != "" {
		return Load(explicitPath)
	}

	// Try user config (~/.voxrc)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".voxrc")
		if _, err := os.Stat(userConfigPath); err == nil {
			cfg, err := Load(userConfigPath)
			if err == nil {
				return cfg, nil
			}
		}
	}

	// Try system config (/etc/vox/config.yaml)
	systemConfigPath := "/etc/vox/config.yaml"
	if _, err := os.Stat(systemConfigPath); err == nil {
		cfg, err := Load(systemConfigPath)
		if err == nil {
			return cfg, nil
		}
	}

	// No config file found, return defaults
	return DefaultConfig(), nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
