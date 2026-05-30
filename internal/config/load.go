package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse unmarshals raw YAML into a Config without applying env overrides,
// defaults, or validation. Useful in tests.
func Parse(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Load reads the YAML config at path and returns a fully resolved Config:
// parsed, overlaid with MEMPUNK_* env vars, defaulted, and validated.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if err := c.applyEnv(); err != nil {
		return nil, err
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return c, nil
}
