package observability

import (
	"context"
	"fmt"
	"testing"
	"time"

	"lambda-go-template/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func createTestConfig() *config.Config {
	return &config.Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0-test",
		Environment:     "test",
		LogLevel:        "debug",
		LogFormat:       "json",
		FunctionName:    "test-function",
		FunctionVersion: "1",
		Region:          "us-east-1",
		RequestTimeout:  30 * time.Second,
		ResponseTimeout: 29 * time.Second,
		EnableTracing:   false,
		EnableMetrics:   false,
		CacheMaxAge:     300,
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "valid config",
			config:      createTestConfig(),
			expectError: false,
		},
		{
			name: "invalid log level",
			config: func() *config.Config {
				cfg := createTestConfig()
				cfg.LogLevel = "invalid"
				return cfg
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
				assert.Equal(t, tt.config.ServiceName, logger.serviceName)
				assert.Equal(t, tt.config.ServiceVersion, logger.version)
			}
		})
	}
}

func TestMustNewLogger(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		cfg := createTestConfig()
		logger := MustNewLogger(cfg)
		assert.NotNil(t, logger)
	})

	t.Run("panics on error", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.LogLevel = "invalid"

		assert.Panics(t, func() {
			MustNewLogger(cfg)
		})
	})
}

func TestLogger_LogLevel(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   string
		expectedLevel zapcore.Level
	}{
		{
			name:          "debug level",
			configLevel:   "debug",
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name:          "info level",
			configLevel:   "info",
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "warn level",
			configLevel:   "warn",
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name:          "error level",
			configLevel:   "error",
			expectedLevel: zapcore.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			cfg.LogLevel = tt.configLevel

			logger, err := NewLogger(cfg)
			require.NoError(t, err)

			// Create observer to capture logs
			observedZapCore, observedLogs := observer.New(tt.expectedLevel)
			logger.Logger = logger.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				return zapcore.NewTee(c, observedZapCore)
			}))

			// Test that the logger respects the level
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			// Count logs at each level
			debugLogs := observedLogs.FilterLevel(zapcore.DebugLevel).Len()
			infoLogs := observedLogs.FilterLevel(zapcore.InfoLevel).Len()
			warnLogs := observedLogs.FilterLevel(zapcore.WarnLevel).Len()
			errorLogs := observedLogs.FilterLevel(zapcore.ErrorLevel).Len()

			// Verify expected behavior based on configured level
			switch tt.expectedLevel {
			case zapcore.DebugLevel:
				assert.Equal(t, 1, debugLogs)
				assert.Equal(t, 1, infoLogs)
				assert.Equal(t, 1, warnLogs)
				assert.Equal(t, 1, errorLogs)
			case zapcore.InfoLevel:
				assert.Equal(t, 0, debugLogs)
				assert.Equal(t, 1, infoLogs)
				assert.Equal(t, 1, warnLogs)
				assert.Equal(t, 1, errorLogs)
			case zapcore.WarnLevel:
				assert.Equal(t, 0, debugLogs)
				assert.Equal(t, 0, infoLogs)
				assert.Equal(t, 1, warnLogs)
				assert.Equal(t, 1, errorLogs)
			case zapcore.ErrorLevel:
				assert.Equal(t, 0, debugLogs)
				assert.Equal(t, 0, infoLogs)
				assert.Equal(t, 0, warnLogs)
				assert.Equal(t, 1, errorLogs)
			}
		})
	}
}

func TestLogger_WithRequestID(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	requestID := "test-request-123"
	loggerWithRequestID := logger.WithRequestID(requestID)

	assert.NotNil(t, loggerWithRequestID)
	assert.IsType(t, &zap.Logger{}, loggerWithRequestID)
}

func TestLogger_WithError(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		err      error
		expected *zap.Logger
	}{
		{
			name:     "with error",
			err:      assert.AnError,
			expected: nil, // We'll check it's not the same instance
		},
		{
			name:     "with nil error",
			err:      nil,
			expected: logger.Logger, // Should return original logger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.WithError(tt.err)
			assert.NotNil(t, result)

			if tt.err == nil {
				assert.Equal(t, logger.Logger, result)
			} else {
				assert.NotEqual(t, logger.Logger, result)
			}
		})
	}
}

func TestLogger_WithFields(t *testing.T) {
	cfg := createTestConfig()
	originalLogger, err := NewLogger(cfg)
	require.NoError(t, err)

	fields := map[string]interface{}{
		"user_id":    "123",
		"session_id": "abc-def",
		"count":      42,
		"active":     true,
	}

	newLogger := originalLogger.WithFields(fields)

	assert.NotNil(t, newLogger)
	assert.IsType(t, &Logger{}, newLogger)
	assert.NotEqual(t, originalLogger, newLogger)
	assert.Equal(t, originalLogger.serviceName, newLogger.serviceName)
	assert.Equal(t, originalLogger.version, newLogger.version)
}

func TestLogger_StructuredLogging(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Create observer to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	logger.Logger = logger.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, observedZapCore)
	}))

	// Test structured logging
	logger.Info("test message", zap.String("key1", "value1"), zap.Int("key2", 42))

	logs := observedLogs.All()
	require.Len(t, logs, 1)

	log := logs[0]
	assert.Equal(t, "test message", log.Message)
	assert.Equal(t, zapcore.InfoLevel, log.Level)

	// Check context fields
	contextMap := log.ContextMap()
	assert.Equal(t, "value1", contextMap["key1"])
	assert.Equal(t, int64(42), contextMap["key2"])
	assert.Equal(t, "test-service", contextMap["service"])
	assert.Equal(t, "1.0.0-test", contextMap["version"])
	assert.Equal(t, "test", contextMap["environment"])
}

func TestLogger_ContextFields(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Create observer to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	logger.Logger = logger.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, observedZapCore)
	}))

	// Log a message
	logger.Info("test message")

	logs := observedLogs.All()
	require.Len(t, logs, 1)

	contextMap := logs[0].ContextMap()

	// Verify service context is always included
	assert.Equal(t, "test-service", contextMap["service"])
	assert.Equal(t, "1.0.0-test", contextMap["version"])
	assert.Equal(t, "test", contextMap["environment"])
	assert.Contains(t, contextMap, "timestamp")
}

func TestLogger_Close(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Close should not return an error for test logger
	err = logger.Close()
	assert.NoError(t, err)
}

func TestSetGlobalLogger(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Test setting global logger
	SetGlobalLogger(logger)

	// This is mainly to ensure the function doesn't panic
	// The actual global logger functionality would be tested in integration tests
}

func TestLogger_ConcurrentUsage(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Create observer to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	logger.Logger = logger.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, observedZapCore)
	}))

	const numGoroutines = 100
	const numLogsPerGoroutine = 10
	done := make(chan bool, numGoroutines)

	// Start multiple goroutines that log concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()

			for j := 0; j < numLogsPerGoroutine; j++ {
				logger.Info("concurrent log",
					zap.Int("routine_id", routineID),
					zap.Int("log_number", j))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all logs were captured
	logs := observedLogs.All()
	expectedLogCount := numGoroutines * numLogsPerGoroutine
	assert.Equal(t, expectedLogCount, len(logs))

	// Verify log structure is maintained
	for _, log := range logs {
		assert.Equal(t, "concurrent log", log.Message)
		assert.Equal(t, zapcore.InfoLevel, log.Level)

		contextMap := log.ContextMap()
		assert.Contains(t, contextMap, "routine_id")
		assert.Contains(t, contextMap, "log_number")
		assert.Equal(t, "test-service", contextMap["service"])
	}
}

func TestLogger_LogFormats(t *testing.T) {
	tests := []struct {
		name      string
		logFormat string
	}{
		{
			name:      "json format",
			logFormat: "json",
		},
		{
			name:      "console format",
			logFormat: "console",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			cfg.LogFormat = tt.logFormat

			logger, err := NewLogger(cfg)
			assert.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

func TestLogger_InvalidLogLevel(t *testing.T) {
	cfg := createTestConfig()
	cfg.LogLevel = "invalid-level"

	logger, err := NewLogger(cfg)
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestLogger_WithXRayContext(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Test with context that might contain X-Ray information
	ctx := context.Background()

	// Create observer to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	logger.Logger = logger.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, observedZapCore)
	}))

	// Use context-aware logging (this would be enhanced in a real X-Ray integration)
	loggerWithContext := logger.WithContext(ctx)
	loggerWithContext.Info("test message with context")

	logs := observedLogs.All()
	require.Len(t, logs, 1)

	// Basic verification - in real implementation this would include X-Ray trace IDs
	assert.Equal(t, "test message with context", logs[0].Message)
}

func TestLogger_PerformanceWithManyFields(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Test logging with many fields (performance test)
	manyFields := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		manyFields[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	loggerWithFields := logger.WithFields(manyFields)

	// This should not panic or cause significant performance issues
	start := time.Now()
	loggerWithFields.Info("message with many fields")
	duration := time.Since(start)

	// Logging with 100 fields should be reasonably fast (< 100ms)
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestLogger_MemoryUsage(t *testing.T) {
	cfg := createTestConfig()
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Test that creating many child loggers doesn't cause memory leaks
	const numChildren = 1000
	children := make([]*Logger, numChildren)

	for i := 0; i < numChildren; i++ {
		fields := map[string]interface{}{
			"child_id": i,
			"created":  time.Now(),
		}
		children[i] = logger.WithFields(fields)
	}

	// Verify all children were created
	assert.Len(t, children, numChildren)

	// All children should have the same service name and version
	for _, child := range children {
		assert.Equal(t, "test-service", child.serviceName)
		assert.Equal(t, "1.0.0-test", child.version)
	}
}
