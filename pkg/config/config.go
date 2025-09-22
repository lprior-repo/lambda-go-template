// Package config provides application configuration management with environment variable support.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration settings.
type Config struct {
	// Service information
	ServiceName    string `envconfig:"SERVICE_NAME" default:"lambda-service"`
	ServiceVersion string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	Environment    string `envconfig:"ENVIRONMENT" default:"development"`

	// Logging configuration
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"` // json or console

	// AWS Lambda specific
	FunctionName    string `envconfig:"AWS_LAMBDA_FUNCTION_NAME"`
	FunctionVersion string `envconfig:"AWS_LAMBDA_FUNCTION_VERSION"`
	Region          string `envconfig:"AWS_REGION" default:"us-east-1"`

	// HTTP configuration
	RequestTimeout  time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	ResponseTimeout time.Duration `envconfig:"RESPONSE_TIMEOUT" default:"29s"`

	// Observability
	EnableTracing bool `envconfig:"ENABLE_TRACING" default:"true"`
	EnableMetrics bool `envconfig:"ENABLE_METRICS" default:"true"`

	// Cache configuration
	CacheMaxAge int `envconfig:"CACHE_MAX_AGE" default:"300"` // seconds
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// MustLoad loads configuration and panics if it fails.
// This is typically used in main functions where configuration
// failure should halt the application.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
	return cfg
}

// validate ensures the configuration is valid.
func (c *Config) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if c.ServiceVersion == "" {
		return fmt.Errorf("service version cannot be empty")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if c.ResponseTimeout <= 0 {
		return fmt.Errorf("response timeout must be positive")
	}

	if c.ResponseTimeout >= c.RequestTimeout {
		return fmt.Errorf("response timeout must be less than request timeout")
	}

	if c.CacheMaxAge < 0 {
		return fmt.Errorf("cache max age cannot be negative")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json":    true,
		"console": true,
	}

	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s", c.LogFormat)
	}

	return nil
}

// IsProduction returns true if the environment is production.
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// IsDevelopment returns true if the environment is development.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// IsTest returns true if the environment is for testing.
func (c *Config) IsTest() bool {
	return c.Environment == "test" || c.Environment == "testing"
}

// GetCacheMaxAgeSeconds returns cache max age as time.Duration.
func (c *Config) GetCacheMaxAgeSeconds() time.Duration {
	return time.Duration(c.CacheMaxAge) * time.Second
}
