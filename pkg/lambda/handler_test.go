package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"lambda-go-template/pkg/config"
	"lambda-go-template/pkg/observability"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

// Test helper functions
func createTestConfig() *config.Config {
	return &config.Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0-test",
		Environment:     "test",
		LogLevel:        "debug",
		LogFormat:       "console",
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

func createTestLogger(t *testing.T) *observability.Logger {
	logger := observability.MustNewLogger(createTestConfig())
	return logger
}

func createTestTracer() *observability.Tracer {
	return observability.NewTracer(observability.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
		Version:     "1.0.0-test",
	})
}

func TestNewHandler(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()

	handler := NewHandler(cfg, logger, tracer)

	assert.NotNil(t, handler)
	assert.Equal(t, cfg, handler.config)
	assert.Equal(t, logger, handler.logger)
	assert.Equal(t, tracer, handler.tracer)
}

func TestHandler_Wrap(t *testing.T) {
	tests := []struct {
		name           string
		handlerFunc    HandlerFunc
		request        events.APIGatewayProxyRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful handler",
			handlerFunc: func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
				return map[string]string{"message": "success"}, nil
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/test",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name: "handler returns error",
			handlerFunc: func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
				return nil, errors.New("test error")
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/test",
			},
			expectedStatus: 500,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			logger := createTestLogger(t)
			tracer := createTestTracer()
			handler := NewHandler(cfg, logger, tracer)

			wrappedHandler := handler.Wrap(tt.handlerFunc)

			ctx := context.Background()
			response, err := wrappedHandler(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, response.StatusCode)
				assert.Contains(t, response.Headers, "Content-Type")
				assert.Equal(t, "application/json", response.Headers["Content-Type"])
			}
		})
	}
}

func TestHandler_WrapV2(t *testing.T) {
	tests := []struct {
		name           string
		handlerFunc    HandlerFuncV2
		request        events.APIGatewayV2HTTPRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful handler",
			handlerFunc: func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
				return map[string]string{"message": "success"}, nil
			},
			request: events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/test",
					},
				},
				Headers: map[string]string{"Content-Type": "application/json"},
			},
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name: "handler returns validation error",
			handlerFunc: func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
				return nil, NewValidationError("test validation error", "field", "value")
			},
			request: events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/test",
					},
				},
			},
			expectedStatus: 400,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			logger := createTestLogger(t)
			tracer := createTestTracer()
			handler := NewHandler(cfg, logger, tracer)

			wrappedHandler := handler.WrapV2(tt.handlerFunc)

			ctx := context.Background()
			response, err := wrappedHandler(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, response.StatusCode)
				assert.Contains(t, response.Headers, "content-type")
				assert.Equal(t, "application/json", response.Headers["content-type"])
			}
		})
	}
}

func TestHandler_ValidationMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		expectedStatus int
		shouldContinue bool
	}{
		{
			name: "valid GET request",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/test",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
			expectedStatus: 200,
			shouldContinue: true,
		},
		{
			name: "POST with valid content type",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/test",
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"test": "data"}`,
			},
			expectedStatus: 200,
			shouldContinue: true,
		},
		{
			name: "POST with invalid content type",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/test",
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       `{"test": "data"}`,
			},
			expectedStatus: 400,
			shouldContinue: false,
		},
		{
			name: "unsupported HTTP method",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "PATCH",
				Path:       "/test",
			},
			expectedStatus: 400,
			shouldContinue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			logger := createTestLogger(t)
			tracer := createTestTracer()
			handler := NewHandler(cfg, logger, tracer)

			middleware := handler.ValidationMiddleware()

			handlerCalled := false
			testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
				handlerCalled = true
				return map[string]string{"message": "success"}, nil
			}

			wrappedHandler := middleware(testHandler)
			ctx := context.Background()

			result, err := wrappedHandler(ctx, tt.request)

			assert.Equal(t, tt.shouldContinue, handlerCalled)

			if !tt.shouldContinue {
				assert.Error(t, err)
				validationErr, ok := err.(*ValidationError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, validationErr.StatusCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestHandler_ValidationMiddlewareV2(t *testing.T) {
	tests := []struct {
		name           string
		request        events.APIGatewayV2HTTPRequest
		expectedStatus int
		shouldContinue bool
	}{
		{
			name: "valid GET request",
			request: events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/test",
					},
				},
				Headers: map[string]string{"content-type": "application/json"},
			},
			expectedStatus: 200,
			shouldContinue: true,
		},
		{
			name: "POST with valid JSON body",
			request: events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "POST",
						Path:   "/test",
					},
				},
				Headers: map[string]string{"content-type": "application/json"},
				Body:    `{"test": "data"}`,
			},
			expectedStatus: 200,
			shouldContinue: true,
		},
		{
			name: "POST with invalid JSON",
			request: events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "POST",
						Path:   "/test",
					},
				},
				Headers: map[string]string{"content-type": "application/json"},
				Body:    `{invalid json}`,
			},
			expectedStatus: 400,
			shouldContinue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			logger := createTestLogger(t)
			tracer := createTestTracer()
			handler := NewHandler(cfg, logger, tracer)

			middleware := handler.ValidationMiddlewareV2()

			handlerCalled := false
			testHandler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
				handlerCalled = true
				return map[string]string{"message": "success"}, nil
			}

			wrappedHandler := middleware(testHandler)
			ctx := context.Background()

			result, err := wrappedHandler(ctx, tt.request)

			assert.Equal(t, tt.shouldContinue, handlerCalled)

			if !tt.shouldContinue {
				assert.Error(t, err)
				validationErr, ok := err.(*ValidationError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, validationErr.StatusCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestGetParsedBody(t *testing.T) {
	tests := []struct {
		name        string
		contextData interface{}
		expectFound bool
	}{
		{
			name:        "valid parsed body",
			contextData: map[string]interface{}{"key": "value"},
			expectFound: true,
		},
		{
			name:        "nil context data",
			contextData: nil,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.contextData != nil {
				ctx = context.WithValue(ctx, contextKeyParsedBody, tt.contextData)
			}

			body, found := GetParsedBody(ctx)

			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				assert.Equal(t, tt.contextData, body)
			}
		})
	}
}

func TestCreateContext(t *testing.T) {
	baseCtx := context.Background()
	requestID := "test-request-123"

	ctx := CreateContext(baseCtx, requestID)

	assert.NotNil(t, ctx)

	// Verify request ID is set
	reqID := ctx.Value(contextKeyRequestID)
	assert.Equal(t, requestID, reqID)

	// Verify timestamp is set
	timestamp := ctx.Value(contextKeyTimestamp)
	assert.NotNil(t, timestamp)
	assert.IsType(t, time.Time{}, timestamp)
}

func TestHandler_LoggingMiddleware(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	middleware := handler.LoggingMiddleware()

	testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
		return map[string]string{"message": "success"}, nil
	}

	wrappedHandler := middleware(testHandler)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		Headers:    map[string]string{"User-Agent": "test-agent"},
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-123",
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "127.0.0.1",
			},
		},
	}

	ctx := context.Background()
	result, err := wrappedHandler(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandler_TracingMiddleware(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	middleware := handler.TracingMiddleware()

	testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
		return map[string]string{"message": "success"}, nil
	}

	wrappedHandler := middleware(testHandler)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
	}

	ctx := context.Background()
	result, err := wrappedHandler(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandler_TimeoutMiddleware(t *testing.T) {
	cfg := createTestConfig()
	cfg.RequestTimeout = 100 * time.Millisecond // Short timeout for testing
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	middleware := handler.TimeoutMiddleware()

	t.Run("handler completes within timeout", func(t *testing.T) {
		testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			return map[string]string{"message": "success"}, nil
		}

		wrappedHandler := middleware(testHandler)

		request := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/test",
		}

		ctx := context.Background()
		result, err := wrappedHandler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("handler times out", func(t *testing.T) {
		testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return map[string]string{"message": "success"}, nil
		}

		wrappedHandler := middleware(testHandler)

		request := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/test",
		}

		ctx := context.Background()
		_, err := wrappedHandler(ctx, request)

		assert.Error(t, err)
		timeoutErr, ok := err.(*TimeoutError)
		assert.True(t, ok)
		assert.Equal(t, 408, timeoutErr.StatusCode)
		assert.Contains(t, timeoutErr.Message, "timeout")
	})
}

func TestErrorHandling(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	tests := []struct {
		name           string
		handlerError   error
		expectedStatus int
	}{
		{
			name:           "validation error",
			handlerError:   NewValidationError("test validation", "field", "value"),
			expectedStatus: 400,
		},
		{
			name:           "not found error",
			handlerError:   NewNotFoundError("resource", "123"),
			expectedStatus: 404,
		},
		{
			name:           "timeout error",
			handlerError:   NewTimeoutError("operation"),
			expectedStatus: 408,
		},
		{
			name:           "internal error",
			handlerError:   NewInternalError("operation failed"),
			expectedStatus: 500,
		},
		{
			name:           "generic error",
			handlerError:   errors.New("generic error"),
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
				return nil, tt.handlerError
			}

			wrappedHandler := handler.Wrap(testHandler)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/test",
			}

			ctx := context.Background()
			response, err := wrappedHandler(ctx, request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, response.StatusCode)

			var errorResponse map[string]interface{}
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse, "message")
			assert.Contains(t, errorResponse, "timestamp")
		})
	}
}

func TestResponseHeaders(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
		return map[string]string{"message": "success"}, nil
	}

	wrappedHandler := handler.Wrap(testHandler)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-123",
		},
	}

	ctx := context.Background()
	response, err := wrappedHandler(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	// Check required headers
	assert.Equal(t, "application/json", response.Headers["Content-Type"])
	assert.Equal(t, "test-request-123", response.Headers["X-Request-ID"])
	assert.Contains(t, response.Headers, "Cache-Control")

	// Check CORS headers
	assert.Equal(t, "*", response.Headers["Access-Control-Allow-Origin"])
	assert.Contains(t, response.Headers, "Access-Control-Allow-Headers")
	assert.Contains(t, response.Headers, "Access-Control-Allow-Methods")
}

func TestJSONResponseFormatting(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	testData := map[string]interface{}{
		"message": "test message",
		"count":   42,
		"items":   []string{"item1", "item2"},
	}

	testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
		return testData, nil
	}

	wrappedHandler := handler.Wrap(testHandler)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-123",
		},
	}

	ctx := context.Background()
	response, err := wrappedHandler(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	var responseData map[string]interface{}
	err = json.Unmarshal([]byte(response.Body), &responseData)
	assert.NoError(t, err)

	// Check response structure
	assert.Contains(t, responseData, "data")
	assert.Contains(t, responseData, "requestId")
	assert.Contains(t, responseData, "timestamp")

	// Check data content
	data, ok := responseData["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test message", data["message"])
	assert.Equal(t, float64(42), data["count"]) // JSON unmarshals numbers as float64
}

func TestMultipleMiddleware(t *testing.T) {
	cfg := createTestConfig()
	logger := createTestLogger(t)
	tracer := createTestTracer()
	handler := NewHandler(cfg, logger, tracer)

	testHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
		return map[string]string{"message": "success"}, nil
	}

	// Apply multiple middleware
	wrappedHandler := handler.Wrap(
		testHandler,
		handler.ValidationMiddleware(),
		handler.LoggingMiddleware(),
		handler.TracingMiddleware(),
		handler.TimeoutMiddleware(),
	)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		Headers:    map[string]string{"Content-Type": "application/json"},
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-123",
		},
	}

	ctx := context.Background()
	response, err := wrappedHandler(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "application/json", response.Headers["Content-Type"])
}
