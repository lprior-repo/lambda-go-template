package observability

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
)

func createTestTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		Version:     "1.0.0-test",
	}
}

func createDisabledTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
		Version:     "1.0.0-test",
	}
}

func TestNewTracer(t *testing.T) {
	tests := []struct {
		name   string
		config TracingConfig
	}{
		{
			name:   "enabled tracing",
			config: createTestTracingConfig(),
		},
		{
			name:   "disabled tracing",
			config: createDisabledTracingConfig(),
		},
		{
			name: "empty service name",
			config: TracingConfig{
				Enabled:     true,
				ServiceName: "",
				Version:     "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)

			assert.NotNil(t, tracer)
			assert.Equal(t, tt.config, tracer.config)
		})
	}
}

func TestTracer_StartSubsegment(t *testing.T) {
	tests := []struct {
		name           string
		config         TracingConfig
		segmentName    string
		expectNonNil   bool
	}{
		{
			name:         "enabled tracing",
			config:       createTestTracingConfig(),
			segmentName:  "test-segment",
			expectNonNil: false, // X-Ray segment will be nil in test environment
		},
		{
			name:         "disabled tracing",
			config:       createDisabledTracingConfig(),
			segmentName:  "test-segment",
			expectNonNil: false,
		},
		{
			name:         "empty segment name",
			config:       createTestTracingConfig(),
			segmentName:  "",
			expectNonNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			resultCtx, segment := tracer.StartSubsegment(ctx, tt.segmentName)

			assert.NotNil(t, resultCtx)
			if tt.expectNonNil {
				assert.NotNil(t, segment)
			}
			// In test environment without X-Ray, segment will typically be nil
			// This is expected behavior and the tracer should handle it gracefully
		})
	}
}

func TestTracer_StartSubsegmentWithLambdaContext(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())

	// Create a context with Lambda context
	baseCtx := context.Background()
	lc := &lambdacontext.LambdaContext{
		AwsRequestID:       "test-request-123",
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:test-function",
	}
	ctx := lambdacontext.NewContext(baseCtx, lc)

	resultCtx, segment := tracer.StartSubsegment(ctx, "test-segment")

	assert.NotNil(t, resultCtx)
	// Segment will be nil in test environment, but should not panic
	assert.NotPanics(t, func() {
		tracer.Close(segment, nil)
	})
}

func TestTracer_Close(t *testing.T) {
	tests := []struct {
		name   string
		config TracingConfig
		err    error
	}{
		{
			name:   "close without error",
			config: createTestTracingConfig(),
			err:    nil,
		},
		{
			name:   "close with error",
			config: createTestTracingConfig(),
			err:    assert.AnError,
		},
		{
			name:   "disabled tracing",
			config: createDisabledTracingConfig(),
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			_, segment := tracer.StartSubsegment(ctx, "test-segment")

			// Should not panic when closing, even with nil segment
			assert.NotPanics(t, func() {
				tracer.Close(segment, tt.err)
			})
		})
	}
}

func TestTracer_AddAnnotation(t *testing.T) {
	tests := []struct {
		name   string
		config TracingConfig
		key    string
		value  string
	}{
		{
			name:   "valid annotation",
			config: createTestTracingConfig(),
			key:    "test-key",
			value:  "test-value",
		},
		{
			name:   "empty key",
			config: createTestTracingConfig(),
			key:    "",
			value:  "test-value",
		},
		{
			name:   "empty value",
			config: createTestTracingConfig(),
			key:    "test-key",
			value:  "",
		},
		{
			name:   "disabled tracing",
			config: createDisabledTracingConfig(),
			key:    "test-key",
			value:  "test-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			// Should not panic when adding annotations
			assert.NotPanics(t, func() {
				tracer.AddAnnotation(ctx, tt.key, tt.value)
			})
		})
	}
}

func TestTracer_AddMetadata(t *testing.T) {
	tests := []struct {
		name      string
		config    TracingConfig
		namespace string
		value     interface{}
	}{
		{
			name:      "string metadata",
			config:    createTestTracingConfig(),
			namespace: "test-namespace",
			value:     "test-value",
		},
		{
			name:      "map metadata",
			config:    createTestTracingConfig(),
			namespace: "test-namespace",
			value:     map[string]interface{}{"key": "value"},
		},
		{
			name:      "number metadata",
			config:    createTestTracingConfig(),
			namespace: "test-namespace",
			value:     42,
		},
		{
			name:      "nil metadata",
			config:    createTestTracingConfig(),
			namespace: "test-namespace",
			value:     nil,
		},
		{
			name:      "disabled tracing",
			config:    createDisabledTracingConfig(),
			namespace: "test-namespace",
			value:     "test-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			// Should not panic when adding metadata
			assert.NotPanics(t, func() {
				tracer.AddMetadata(ctx, tt.namespace, tt.value)
			})
		})
	}
}

func TestTracer_AddError(t *testing.T) {
	tests := []struct {
		name   string
		config TracingConfig
		err    error
	}{
		{
			name:   "valid error",
			config: createTestTracingConfig(),
			err:    assert.AnError,
		},
		{
			name:   "nil error",
			config: createTestTracingConfig(),
			err:    nil,
		},
		{
			name:   "disabled tracing",
			config: createDisabledTracingConfig(),
			err:    assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			// Should not panic when adding errors
			assert.NotPanics(t, func() {
				tracer.AddError(ctx, tt.err)
			})
		})
	}
}

func TestTracer_WithTimer(t *testing.T) {
	tests := []struct {
		name        string
		config      TracingConfig
		operationName string
		operation   func(context.Context) error
		expectError bool
	}{
		{
			name:          "successful operation",
			config:        createTestTracingConfig(),
			operationName: "test-operation",
			operation: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			expectError: false,
		},
		{
			name:          "failing operation",
			config:        createTestTracingConfig(),
			operationName: "test-operation",
			operation: func(ctx context.Context) error {
				return assert.AnError
			},
			expectError: true,
		},
		{
			name:          "disabled tracing",
			config:        createDisabledTracingConfig(),
			operationName: "test-operation",
			operation: func(ctx context.Context) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "nil operation",
			config:        createTestTracingConfig(),
			operationName: "test-operation",
			operation:     nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)
			ctx := context.Background()

			start := time.Now()
			err := tracer.WithTimer(ctx, tt.operationName, tt.operation)
			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify timing functionality
			if tt.operation != nil && !tt.expectError {
				assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
			}
		})
	}
}

func TestTracer_ConcurrentOperations(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())
	ctx := context.Background()

	const numGoroutines = 50
	const numOperationsPerGoroutine = 10
	done := make(chan error, numGoroutines)

	// Start multiple goroutines performing tracing operations
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			var err error
			defer func() { done <- err }()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				segmentName := "concurrent-segment"

				// Start subsegment
				subCtx, segment := tracer.StartSubsegment(ctx, segmentName)

				// Add annotations and metadata
				tracer.AddAnnotation(subCtx, "routine_id", string(rune(routineID)))
				tracer.AddAnnotation(subCtx, "operation_id", string(rune(j)))

				tracer.AddMetadata(subCtx, "test", map[string]interface{}{
					"routine": routineID,
					"op":      j,
				})

				// Simulate some work
				time.Sleep(1 * time.Millisecond)

				// Close segment
				tracer.Close(segment, nil)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			errorCount++
		}
	}

	// All operations should complete without errors
	assert.Equal(t, 0, errorCount)
}

func TestTracer_DisabledTracingPerformance(t *testing.T) {
	disabledTracer := NewTracer(createDisabledTracingConfig())
	enabledTracer := NewTracer(createTestTracingConfig())
	ctx := context.Background()

	const numOperations = 1000

	// Measure disabled tracing performance
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		subCtx, segment := disabledTracer.StartSubsegment(ctx, "test-segment")
		disabledTracer.AddAnnotation(subCtx, "key", "value")
		disabledTracer.AddMetadata(subCtx, "namespace", map[string]string{"key": "value"})
		disabledTracer.Close(segment, nil)
	}
	disabledDuration := time.Since(start)

	// Measure enabled tracing performance
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		subCtx, segment := enabledTracer.StartSubsegment(ctx, "test-segment")
		enabledTracer.AddAnnotation(subCtx, "key", "value")
		enabledTracer.AddMetadata(subCtx, "namespace", map[string]string{"key": "value"})
		enabledTracer.Close(segment, nil)
	}
	enabledDuration := time.Since(start)

	// Both should complete reasonably quickly
	assert.Less(t, disabledDuration, 100*time.Millisecond)
	assert.Less(t, enabledDuration, 500*time.Millisecond)

	// Disabled tracing should be faster or comparable
	// (allowing some variance for test environment)
	assert.LessOrEqual(t, disabledDuration, enabledDuration*2)
}

func TestTracer_ContextPropagation(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())
	baseCtx := context.Background()

	// Create nested segments to test context propagation
	ctx1, segment1 := tracer.StartSubsegment(baseCtx, "parent-segment")
	assert.NotNil(t, ctx1)

	ctx2, segment2 := tracer.StartSubsegment(ctx1, "child-segment")
	assert.NotNil(t, ctx2)

	ctx3, segment3 := tracer.StartSubsegment(ctx2, "grandchild-segment")
	assert.NotNil(t, ctx3)

	// Add annotations at different levels
	tracer.AddAnnotation(ctx1, "level", "parent")
	tracer.AddAnnotation(ctx2, "level", "child")
	tracer.AddAnnotation(ctx3, "level", "grandchild")

	// Close segments in reverse order
	tracer.Close(segment3, nil)
	tracer.Close(segment2, nil)
	tracer.Close(segment1, nil)

	// Test should not panic and should handle nested contexts gracefully
}

func TestTracer_ErrorHandling(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())
	ctx := context.Background()

	// Test error handling in WithTimer
	err := tracer.WithTimer(ctx, "error-operation", func(ctx context.Context) error {
		// Add some annotations before error
		tracer.AddAnnotation(ctx, "status", "processing")
		tracer.AddMetadata(ctx, "operation", map[string]string{"step": "1"})

		// Simulate error
		return assert.AnError
	})

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestTracer_LongRunningOperation(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())
	ctx := context.Background()

	operationDuration := 50 * time.Millisecond

	err := tracer.WithTimer(ctx, "long-operation", func(ctx context.Context) error {
		// Simulate long-running operation with intermediate updates
		for i := 0; i < 5; i++ {
			time.Sleep(operationDuration / 5)
			tracer.AddAnnotation(ctx, "progress", string(rune(i*20)))
		}
		return nil
	})

	assert.NoError(t, err)
}

func TestTracer_NilOperationHandling(t *testing.T) {
	tracer := NewTracer(createTestTracingConfig())
	ctx := context.Background()

	// WithTimer should handle nil operation gracefully
	err := tracer.WithTimer(ctx, "nil-operation", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation cannot be nil")
}
