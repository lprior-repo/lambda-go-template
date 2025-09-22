package main

import (
	"encoding/json"
	"testing"
	"time"

	"lambda-go-template/internal/testutil"
	"lambda-go-template/pkg/lambda"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelloService_ProcessHelloRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
		validate    func(*testing.T, *HelloResponse)
	}{
		{
			name:        "should process valid GET request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/hello"),
			expectError: false,
			validate: func(t *testing.T, response *HelloResponse) {
				assert.Equal(t, "Hello from Lambda with observability!", response.Message)
				assert.Equal(t, "/hello", response.Path)
				assert.Equal(t, "test", response.Environment)
				assert.Equal(t, "1.0.0-test", response.Version)
				assert.Equal(t, "test-request-123", response.RequestID)

				// Validate timestamp format
				_, err := time.Parse(time.RFC3339, response.Timestamp)
				assert.NoError(t, err, "Timestamp should be in RFC3339 format")

				// Validate timestamp is recent
				testutil.AssertTimestampRecent(t, response.Timestamp, 5*time.Second)
			},
		},
		{
			name:        "should process valid POST request",
			request:     testutil.CreateTestAPIGatewayRequest("POST", "/hello"),
			expectError: false,
			validate: func(t *testing.T, response *HelloResponse) {
				assert.Equal(t, "Hello from Lambda with observability!", response.Message)
				assert.Equal(t, "/hello", response.Path)
				assert.NotEmpty(t, response.Timestamp)
			},
		},
		{
			name:        "should process request with different path",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/api/hello"),
			expectError: false,
			validate: func(t *testing.T, response *HelloResponse) {
				assert.Equal(t, "/api/hello", response.Path)
				assert.Equal(t, "Hello from Lambda with observability!", response.Message)
			},
		},
		{
			name:        "should handle request with query parameters",
			request:     testutil.CreateTestAPIGatewayRequestWithQuery("GET", "/hello", map[string]string{"name": "world"}),
			expectError: false,
			validate: func(t *testing.T, response *HelloResponse) {
				assert.Equal(t, "/hello", response.Path)
				assert.Equal(t, "Hello from Lambda with observability!", response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cfg := testutil.TestConfig()
			logger := testutil.TestLogger(t)
			tracer := testutil.TestTracer()

			// Create service
			service := NewHelloService(cfg, logger, tracer)

			// Create test context
			ctx := testutil.CreateTestContext("test-request-123")

			// Execute test
			result, err := service.ProcessHelloRequest(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)

				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestCreateHandler(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
	}{
		{
			name:        "should handle valid request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/hello"),
			expectError: false,
		},
		{
			name:        "should handle POST request",
			request:     testutil.CreateTestAPIGatewayRequest("POST", "/hello"),
			expectError: false,
		},
		{
			name:        "should handle request with body",
			request:     testutil.CreateTestAPIGatewayRequestWithBody("POST", "/hello", map[string]string{"test": "data"}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cfg := testutil.TestConfig()
			logger := testutil.TestLogger(t)
			tracer := testutil.TestTracer()

			// Create handler
			handler := CreateHandler(cfg, logger, tracer)

			// Create test context
			ctx := testutil.CreateTestContext("test-request-456")

			// Execute test
			result, err := handler(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Validate that result is a HelloResponse
				response, ok := result.(*HelloResponse)
				require.True(t, ok, "Result should be a HelloResponse")

				assert.Equal(t, "Hello from Lambda with observability!", response.Message)
				assert.Equal(t, tt.request.Path, response.Path)
				assert.Equal(t, "test", response.Environment)
				assert.Equal(t, "test-request-456", response.RequestID)
			}
		})
	}
}

func TestMainIntegration(t *testing.T) {
	// Test the complete Lambda handler with middleware
	tests := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedStatusCode int
		validateResponse   func(*testing.T, string)
	}{
		{
			name:               "should return 200 for valid GET request",
			request:            testutil.CreateTestAPIGatewayRequest("GET", "/hello"),
			expectedStatusCode: 200,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertSuccessResponse(t, body, nil)
			},
		},
		{
			name:               "should return 200 for valid POST request",
			request:            testutil.CreateTestAPIGatewayRequest("POST", "/hello"),
			expectedStatusCode: 200,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertSuccessResponse(t, body, nil)
			},
		},
		{
			name:               "should return 400 for invalid HTTP method",
			request:            testutil.CreateTestAPIGatewayRequest("INVALID", "/hello"),
			expectedStatusCode: 400,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertErrorResponse(t, body, "HTTP method INVALID is not allowed")
			},
		},
		{
			name: "should return 400 for invalid content type",
			request: func() events.APIGatewayProxyRequest {
				req := testutil.CreateTestAPIGatewayRequestWithHeaders("POST", "/hello", map[string]string{
					"Content-Type": "text/plain",
				})
				req.Body = `{"test": "data"}`
				return req
			}(),
			expectedStatusCode: 400,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertErrorResponse(t, body, "Content-Type must be application/json for requests with body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cleanup := testutil.SetupTestEnvironment(t, map[string]string{
				"ENVIRONMENT":     "test",
				"SERVICE_NAME":    "hello-service",
				"SERVICE_VERSION": "1.0.0-test",
				"LOG_LEVEL":       "debug",
				"ENABLE_TRACING":  "false",
			})
			defer cleanup()

			// Setup test dependencies
			cfg := testutil.TestConfig()
			logger := testutil.TestLogger(t)
			tracer := testutil.TestTracer()

			// Create handler with middleware
			handler := lambda.NewHandler(cfg, logger, tracer)
			businessHandler := CreateHandler(cfg, logger, tracer)

			wrappedHandler := handler.Wrap(
				businessHandler,
				handler.ValidationMiddleware(),
				handler.LoggingMiddleware(),
				handler.TracingMiddleware(),
			)

			// Create test context
			ctx := testutil.CreateTestContext("test-integration-request")

			// Execute test
			response, err := wrappedHandler(ctx, tt.request)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatusCode, response.StatusCode)

			// Validate headers
			testutil.AssertResponseHeaders(t, response.Headers, "Content-Type", "X-Request-ID")
			testutil.AssertCORSHeaders(t, response.Headers)
			assert.Equal(t, "application/json", response.Headers["Content-Type"])

			// Validate response body
			if tt.validateResponse != nil {
				tt.validateResponse(t, response.Body)
			}
		})
	}
}

func TestHelloResponseJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response HelloResponse
	}{
		{
			name: "should serialize complete response",
			response: HelloResponse{
				Message:     "Hello World",
				Path:        "/test",
				Timestamp:   "2024-01-01T12:00:00Z",
				Environment: "test",
				RequestID:   "req-123",
				Version:     "1.0.0",
			},
		},
		{
			name: "should serialize response with empty fields",
			response: HelloResponse{
				Message:     "",
				Path:        "",
				Timestamp:   "",
				Environment: "",
				RequestID:   "",
				Version:     "",
			},
		},
		{
			name: "should serialize response with special characters",
			response: HelloResponse{
				Message:     "Hello 世界! 🌍",
				Path:        "/api/v1/hello",
				Timestamp:   "2024-01-01T12:00:00Z",
				Environment: "production",
				RequestID:   "req-abc-123-xyz",
				Version:     "2.1.0-beta.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tt.response)
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Unmarshal back to struct
			var unmarshaled HelloResponse
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.response, unmarshaled)
		})
	}
}

func BenchmarkHelloService_ProcessHelloRequest(b *testing.B) {
	// Setup
	cfg := testutil.TestConfig()
	logger := testutil.TestLogger(&testing.T{}) // Use testing.T for benchmark
	tracer := testutil.TestTracer()
	service := NewHelloService(cfg, logger, tracer)

	request := testutil.CreateTestAPIGatewayRequest("GET", "/hello")
	ctx := testutil.CreateTestContext("bench-request")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.ProcessHelloRequest(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateHandler(b *testing.B) {
	// Setup
	cfg := testutil.TestConfig()
	logger := testutil.TestLogger(&testing.T{}) // Use testing.T for benchmark
	tracer := testutil.TestTracer()
	handler := CreateHandler(cfg, logger, tracer)

	request := testutil.CreateTestAPIGatewayRequest("GET", "/hello")
	ctx := testutil.CreateTestContext("bench-request")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}
