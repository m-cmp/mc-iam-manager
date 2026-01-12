package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the swagger-to-actions configuration
type Config struct {
	Output     string      `yaml:"output"`
	Frameworks []Framework `yaml:"frameworks"`
	Verbose    bool        `yaml:"verbose"`
	Timeout    int         `yaml:"timeout"`
}

// Framework represents a framework configuration
type Framework struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
	Swagger    string `yaml:"swagger"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Output:  "./service-actions.yaml",
		Timeout: 30,
		Verbose: false,
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Resolve relative paths based on config file location
	configDir := filepath.Dir(path)
	cfg.ResolvePaths(configDir)

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Output == "" {
		return fmt.Errorf("output path is required")
	}

	if len(c.Frameworks) == 0 {
		return fmt.Errorf("at least one framework is required")
	}

	for i, fw := range c.Frameworks {
		if fw.Name == "" {
			return fmt.Errorf("framework[%d]: name is required", i)
		}
		if fw.Swagger == "" {
			return fmt.Errorf("framework[%d]: swagger path is required", i)
		}
	}

	if c.Timeout <= 0 {
		c.Timeout = 30
	}

	return nil
}

// ResolvePaths resolves relative paths based on the config file location
func (c *Config) ResolvePaths(configDir string) {
	// Resolve output path
	if !filepath.IsAbs(c.Output) && !IsURL(c.Output) {
		c.Output = filepath.Join(configDir, c.Output)
	}

	// Resolve swagger paths for each framework
	for i := range c.Frameworks {
		swaggerPath := c.Frameworks[i].Swagger
		if !filepath.IsAbs(swaggerPath) && !IsURL(swaggerPath) {
			c.Frameworks[i].Swagger = filepath.Join(configDir, swaggerPath)
		}
	}
}
