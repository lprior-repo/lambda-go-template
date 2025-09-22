package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"SERVICE_NAME", "SERVICE_VERSION", "ENVIRONMENT", "LOG_LEVEL", "LOG_FORMAT",
		"AWS_LAMBDA_FUNCTION_NAME", "AWS_LAMBDA_FUNCTION_VERSION", "AWS_REGION",
		"REQUEST_TIMEOUT", "RESPONSE_TIMEOUT", "ENABLE_TRACING", "ENABLE_METRICS",
		"CACHE_MAX_AGE",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	// Cleanup function to restore environment
	cleanup := func() {
		for _, env := range envVars {
			if originalValue, exists := originalEnv[env]; exists && originalValue != "" {
				os.Setenv(env, originalValue)
			} else {
				os.Unsetenv(env)
			}
		}
	}
	defer cleanup()

	tests := []struct {
		name          string
		envVars       map[string]string
		expectedError bool
		validateFunc  func(*testing.T, *Config)
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				// Clear all environment variables to test defaults
			},
			expectedError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "lambda-service", cfg.ServiceName)
				assert.Equal(t, "1.0.0", cfg.ServiceVersion)
				assert.Equal(t, "development", cfg.Environment)
				assert.Equal(t, "info", cfg.LogLevel)
				assert.Equal(t, "json", cfg.LogFormat)
				assert.Equal(t, "us-east-1", cfg.Region)
				assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
				assert.Equal(t, 29*time.Second, cfg.ResponseTimeout)
				assert.True(t, cfg.EnableTracing)
				assert.True(t, cfg.EnableMetrics)
				assert.Equal(t, 300, cfg.CacheMaxAge)
			},
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"SERVICE_NAME":                 "my-service",
				"SERVICE_VERSION":              "2.1.0",
				"ENVIRONMENT":                  "production",
				"LOG_LEVEL":                    "warn",
				"LOG_FORMAT":                   "console",
				"AWS_LAMBDA_FUNCTION_NAME":     "my-lambda",
				"AWS_LAMBDA_FUNCTION_VERSION":  "$LATEST",
				"AWS_REGION":                   "us-west-2",
				"REQUEST_TIMEOUT":              "45s",
				"RESPONSE_TIMEOUT":             "40s",
				"ENABLE_TRACING":               "false",
				"ENABLE_METRICS":               "false",
				"CACHE_MAX_AGE":                "600",
			},
			expectedError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "my-service", cfg.ServiceName)
				assert.Equal(t, "2.1.0", cfg.ServiceVersion)
				assert.Equal(t, "production", cfg.Environment)
				assert.Equal(t, "warn", cfg.LogLevel)
				assert.Equal(t, "console", cfg.LogFormat)
				assert.Equal(t, "my-lambda", cfg.FunctionName)
				assert.Equal(t, "$LATEST", cfg.FunctionVersion)
				assert.Equal(t, "us-west-2", cfg.Region)
				assert.Equal(t, 45*time.Second, cfg.RequestTimeout)
				assert.Equal(t, 40*time.Second, cfg.ResponseTimeout)
				assert.False(t, cfg.EnableTracing)
				assert.False(t, cfg.EnableMetrics)
				assert.Equal(t, 600, cfg.CacheMaxAge)
			},
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"LOG_LEVEL": "invalid",
			},
			expectedError: true,
		},
		{
			name: "invalid log format",
			envVars: map[string]string{
				"LOG_FORMAT": "xml",
			},
			expectedError: true,
		},
		{
			name: "invalid timeout configuration",
			envVars: map[string]string{
				"REQUEST_TIMEOUT":  "10s",
				"RESPONSE_TIMEOUT": "15s", // greater than request timeout
			},
			expectedError: true,
		},
		{
			name: "negative cache max age",
			envVars: map[string]string{
				"CACHE_MAX_AGE": "-1",
			},
			expectedError: true,
		},
		{
			name: "zero request timeout",
			envVars: map[string]string{
				"REQUEST_TIMEOUT": "0s",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for _, env := range envVars {
				os.Unsetenv(env)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Test Load function
			cfg, err := Load()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validateFunc != nil {
					tt.validateFunc(t, cfg)
				}
			}
		})
	}
}

func TestMustLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"LOG_LEVEL"}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	defer func() {
		for _, env := range envVars {
			if originalValue, exists := originalEnv[env]; exists && originalValue != "" {
				os.Setenv(env, originalValue)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	t.Run("success case", func(t *testing.T) {
		os.Unsetenv("LOG_LEVEL") // Use default
		cfg := MustLoad()
		assert.NotNil(t, cfg)
		assert.Equal(t, "info", cfg.LogLevel)
	})

	t.Run("panic case", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "invalid")
		assert.Panics(t, func() {
			MustLoad()
		})
	})
}

func TestConfigMethods(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expectedProd bool
		expectedDev  bool
		expectedTest bool
	}{
		{
			name:         "production environment",
			environment:  "production",
			expectedProd: true,
			expectedDev:  false,
			expectedTest: false,
		},
		{
			name:         "prod environment",
			environment:  "prod",
			expectedProd: true,
			expectedDev:  false,
			expectedTest: false,
		},
		{
			name:         "development environment",
			environment:  "development",
			expectedProd: false,
			expectedDev:  true,
			expectedTest: false,
		},
		{
			name:         "dev environment",
			environment:  "dev",
			expectedProd: false,
			expectedDev:  true,
			expectedTest: false,
		},
		{
			name:         "test environment",
			environment:  "test",
			expectedProd: false,
			expectedDev:  false,
			expectedTest: true,
		},
		{
			name:         "testing environment",
			environment:  "testing",
			expectedProd: false,
			expectedDev:  false,
			expectedTest: true,
		},
		{
			name:         "unknown environment",
			environment:  "staging",
			expectedProd: false,
			expectedDev:  false,
			expectedTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}

			assert.Equal(t, tt.expectedProd, cfg.IsProduction())
			assert.Equal(t, tt.expectedDev, cfg.IsDevelopment())
			assert.Equal(t, tt.expectedTest, cfg.IsTest())
		})
	}
}

func TestGetCacheMaxAgeSeconds(t *testing.T) {
	tests := []struct {
		name         string
		cacheMaxAge  int
		expectedDuration time.Duration
	}{
		{
			name:             "300 seconds",
			cacheMaxAge:      300,
			expectedDuration: 300 * time.Second,
		},
		{
			name:             "0 seconds",
			cacheMaxAge:      0,
			expectedDuration: 0 * time.Second,
		},
		{
			name:             "3600 seconds",
			cacheMaxAge:      3600,
			expectedDuration: 3600 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{CacheMaxAge: tt.cacheMaxAge}
			assert.Equal(t, tt.expectedDuration, cfg.GetCacheMaxAgeSeconds())
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError bool
		errorContains string
	}{
		{
			name: "valid configuration",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: false,
		},
		{
			name: "empty service name",
			config: Config{
				ServiceName:     "",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "service name cannot be empty",
		},
		{
			name: "empty service version",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "service version cannot be empty",
		},
		{
			name: "invalid log level",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "invalid",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "invalid log level",
		},
		{
			name: "invalid log format",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "xml",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "invalid log format",
		},
		{
			name: "zero request timeout",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  0,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "request timeout must be positive",
		},
		{
			name: "zero response timeout",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 0,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "response timeout must be positive",
		},
		{
			name: "response timeout greater than request timeout",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  25 * time.Second,
				ResponseTimeout: 30 * time.Second,
				CacheMaxAge:     300,
			},
			expectedError: true,
			errorContains: "response timeout must be less than request timeout",
		},
		{
			name: "negative cache max age",
			config: Config{
				ServiceName:     "test-service",
				ServiceVersion:  "1.0.0",
				LogLevel:        "info",
				LogFormat:       "json",
				RequestTimeout:  30 * time.Second,
				ResponseTimeout: 25 * time.Second,
				CacheMaxAge:     -1,
			},
			expectedError: true,
			errorContains: "cache max age cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
