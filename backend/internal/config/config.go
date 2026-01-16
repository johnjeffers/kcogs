package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Clusters ClustersConfig `yaml:"clusters"`
	Pricing  PricingConfig  `yaml:"pricing"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port int `yaml:"port"`
}

// ClustersConfig holds cluster discovery and connection settings
type ClustersConfig struct {
	AutoDiscover AutoDiscoverConfig `yaml:"autoDiscover"`
	Manual       []ClusterConfig    `yaml:"manual"`
}

// AutoDiscoverConfig settings for automatic EKS cluster discovery
type AutoDiscoverConfig struct {
	Enabled bool     `yaml:"enabled"`
	Regions []string `yaml:"regions"`
}

// ClusterConfig defines how to connect to a specific cluster
type ClusterConfig struct {
	Name           string `yaml:"name"`
	Region         string `yaml:"region"`
	KubeconfigPath string `yaml:"kubeconfigPath,omitempty"`
	KubeconfigCtx  string `yaml:"kubeconfigContext,omitempty"`
	EKSClusterName string `yaml:"eksClusterName,omitempty"`
	RoleARN        string `yaml:"roleArn,omitempty"`
}

// PricingConfig holds AWS pricing settings
type PricingConfig struct {
	RefreshIntervalMinutes int  `yaml:"refreshIntervalMinutes"`
	CacheEnabled           bool `yaml:"cacheEnabled"`
}

// LogConfig holds logging settings
type LogConfig struct {
	Level string `yaml:"level"`
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Clusters: ClustersConfig{
			AutoDiscover: AutoDiscoverConfig{
				Enabled: true,
				Regions: []string{"us-east-1"},
			},
		},
		Pricing: PricingConfig{
			RefreshIntervalMinutes: 60,
			CacheEnabled:           true,
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// loadFromEnv overrides config values from environment variables
func (c *Config) loadFromEnv() {
	if port := os.Getenv("KCOGS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}

	if level := os.Getenv("KCOGS_LOG_LEVEL"); level != "" {
		c.Log.Level = level
	}

	if regions := os.Getenv("KCOGS_DISCOVER_REGIONS"); regions != "" {
		c.Clusters.AutoDiscover.Regions = strings.Split(regions, ",")
	}

	if enabled := os.Getenv("KCOGS_AUTO_DISCOVER"); enabled != "" {
		c.Clusters.AutoDiscover.Enabled = enabled == "true" || enabled == "1"
	}

	if interval := os.Getenv("KCOGS_PRICING_REFRESH_MINUTES"); interval != "" {
		if i, err := strconv.Atoi(interval); err == nil {
			c.Pricing.RefreshIntervalMinutes = i
		}
	}
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Pricing.RefreshIntervalMinutes < 1 {
		return fmt.Errorf("pricing refresh interval must be at least 1 minute")
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	return nil
}
