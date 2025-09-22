// Package observability provides structured logging and distributed tracing utilities.
package observability

import (
	"context"
	"fmt"
	"os"

	"lambda-go-template/pkg/config"

	"github.com/aws/aws-xray-sdk-go/xray"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with additional context-aware functionality.
type Logger struct {
	*zap.Logger
	serviceName string
	version     string
}

// NewLogger creates a new structured logger based on configuration.
func NewLogger(cfg *config.Config) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.LogFormat == "console" || cfg.IsDevelopment() {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.LogLevel, err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Configure output paths
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}

	// Add service metadata to all logs
	zapConfig.InitialFields = map[string]interface{}{
		"service": cfg.ServiceName,
		"version": cfg.ServiceVersion,
		"env":     cfg.Environment,
	}

	// Add AWS Lambda context if available
	if cfg.FunctionName != "" {
		zapConfig.InitialFields["function_name"] = cfg.FunctionName
	}
	if cfg.FunctionVersion != "" {
		zapConfig.InitialFields["function_version"] = cfg.FunctionVersion
	}
	if cfg.Region != "" {
		zapConfig.InitialFields["region"] = cfg.Region
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{
		Logger:      zapLogger,
		serviceName: cfg.ServiceName,
		version:     cfg.ServiceVersion,
	}, nil
}

// MustNewLogger creates a new logger and panics if it fails.
func MustNewLogger(cfg *config.Config) *Logger {
	logger, err := NewLogger(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	return logger
}

// WithContext returns a logger with context-specific fields.
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	logger := l.Logger

	// Add tracing information if available
	if seg := xray.GetSegment(ctx); seg != nil {
		logger = logger.With(
			zap.String("trace_id", seg.TraceID),
			zap.String("segment_id", seg.ID),
		)
	}

	// Add request ID from Lambda context if available
	if requestID := GetRequestID(ctx); requestID != "" {
		logger = logger.With(zap.String("request_id", requestID))
	}

	return logger
}

// WithRequestID adds request ID to the logger context.
func (l *Logger) WithRequestID(requestID string) *zap.Logger {
	return l.Logger.With(zap.String("request_id", requestID))
}

// WithError adds error information to the logger context.
func (l *Logger) WithError(err error) *zap.Logger {
	if err == nil {
		return l.Logger
	}
	return l.Logger.With(zap.Error(err))
}

// WithFields adds multiple fields to the logger context.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return &Logger{
		Logger:      l.Logger.With(zapFields...),
		serviceName: l.serviceName,
		version:     l.version,
	}
}

// LogHTTPRequest logs HTTP request information.
func (l *Logger) LogHTTPRequest(ctx context.Context, method, path string, statusCode int, duration int64) {
	l.WithContext(ctx).Info("HTTP request completed",
		zap.String("http_method", method),
		zap.String("http_path", path),
		zap.Int("http_status", statusCode),
		zap.Int64("duration_ms", duration),
	)
}

// LogLambdaStart logs Lambda function invocation start.
func (l *Logger) LogLambdaStart(ctx context.Context, functionName, functionVersion string, remainingTime int64) {
	l.WithContext(ctx).Info("Lambda function invocation started",
		zap.String("function_name", functionName),
		zap.String("function_version", functionVersion),
		zap.Int64("remaining_time_ms", remainingTime),
	)
}

// LogLambdaEnd logs Lambda function invocation completion.
func (l *Logger) LogLambdaEnd(ctx context.Context, duration int64) {
	l.WithContext(ctx).Info("Lambda function invocation completed",
		zap.Int64("duration_ms", duration),
	)
}

// LogLambdaError logs Lambda function errors.
func (l *Logger) LogLambdaError(ctx context.Context, err error, msg string) {
	l.WithContext(ctx).Error(msg,
		zap.Error(err),
		zap.String("error_type", fmt.Sprintf("%T", err)),
	)
}

// Close flushes any buffered log entries.
func (l *Logger) Close() error {
	return l.Logger.Sync()
}

// SetGlobalLogger sets the global logger for the application.
// This is useful for packages that need to log but don't have access to the logger instance.
func SetGlobalLogger(logger *Logger) {
	zap.ReplaceGlobals(logger.Logger)
}

// GetGlobalLogger returns the global logger.
func GetGlobalLogger() *zap.Logger {
	return zap.L()
}

// GetRequestID extracts request ID from context.
// This function attempts to get the request ID from various sources in the context.
func GetRequestID(ctx context.Context) string {
	// Try to get from Lambda context
	if lc := GetLambdaContext(ctx); lc != nil {
		return lc.AwsRequestID
	}

	// Try to get from context value (for custom request IDs)
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}

	return ""
}

// Fatal logs a fatal error and exits the application.
// This should only be used for unrecoverable errors during application startup.
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
	os.Exit(1)
}

// Panic logs a panic-level message and panics.
func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
}
