package linter

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads lint configuration from a YAML reader.
func LoadConfig(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if len(cfg.Extends) == 0 {
		cfg.Extends = []string{"all"}
	}
	if cfg.Categories == nil {
		cfg.Categories = make(map[string]CategoryConfig)
	}
	if cfg.Rules == nil {
		cfg.Rules = []RuleEntry{}
	}
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = OutputFormatText
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadConfigFromFile loads lint configuration from a YAML file.
func LoadConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	return LoadConfig(f)
}
