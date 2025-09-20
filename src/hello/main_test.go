package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
)

func TestCreateResponseHeaders(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		expected  map[string]string
	}{
		{
			name:      "should create response headers with request ID",
			requestID: "test-request-123",
			expected: map[string]string{
				"Content-Type":                     "application/json",
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "OPTIONS,POST,GET",
				"X-Request-ID":                     "test-request-123",
				"Cache-Control":                    "max-age=300",
			},
		},
		{
			name:      "should handle empty request ID",
			requestID: "",
			expected: map[string]string{
				"Content-Type":                     "application/json",
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "OPTIONS,POST,GET",
				"X-Request-ID":                     "",
				"Cache-Control":                    "max-age=300",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createResponseHeaders(tt.requestID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessHelloRequest(t *testing.T) {
	// Set environment variable for testing
	originalEnv := os.Getenv("ENVIRONMENT")
	os.Setenv("ENVIRONMENT", "test")
	defer os.Setenv("ENVIRONMENT", originalEnv)

	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
	}{
		{
			name: "should process valid hello request",
			request: events.APIGatewayProxyRequest{
				Path:       "/hello",
				HTTPMethod: "GET",
			},
			expectError: false,
		},
		{
			name: "should process POST request",
			request: events.APIGatewayProxyRequest{
				Path:       "/hello",
				HTTPMethod: "POST",
			},
			expectError: false,
		},
		{
			name: "should process request with different path",
			request: events.APIGatewayProxyRequest{
				Path:       "/api/hello",
				HTTPMethod: "GET",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with Lambda context
			ctx := context.Background()
			lc := &lambdacontext.LambdaContext{
				AwsRequestID: "test-request-123",
			}
			ctx = lambdacontext.NewContext(ctx, lc)

			// Call the function
			result, err := processHelloRequest(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Validate response structure
				assert.Equal(t, "Hello from Lambda with observability!", result.Message)
				assert.Equal(t, tt.request.Path, result.Path)
				assert.Equal(t, "test", result.Environment)
				assert.Equal(t, "test-request-123", result.RequestID)

				// Validate timestamp format
				_, timeErr := time.Parse(time.RFC3339, result.Timestamp)
				assert.NoError(t, timeErr, "Timestamp should be in RFC3339 format")
			}
		})
	}
}

func TestHandler(t *testing.T) {
	// Set environment variables for testing
	originalEnv := os.Getenv("ENVIRONMENT")
	originalFuncName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	originalFuncVersion := os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")

	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "hello-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "1")

	defer func() {
		os.Setenv("ENVIRONMENT", originalEnv)
		os.Setenv("AWS_LAMBDA_FUNCTION_NAME", originalFuncName)
		os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", originalFuncVersion)
	}()

	tests := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		expectedStatus int
	}{
		{
			name: "should return 200 for valid GET request",
			request: events.APIGatewayProxyRequest{
				Path:       "/hello",
				HTTPMethod: "GET",
			},
			expectedStatus: 200,
		},
		{
			name: "should return 200 for valid POST request",
			request: events.APIGatewayProxyRequest{
				Path:       "/hello",
				HTTPMethod: "POST",
			},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with Lambda context
			ctx := context.Background()
			lc := &lambdacontext.LambdaContext{
				AwsRequestID: "test-request-456",
			}
			ctx = lambdacontext.NewContext(ctx, lc)

			// Call the handler
			response, err := handler(ctx, tt.request)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, response.StatusCode)
			assert.NotEmpty(t, response.Body)

			// Validate headers
			assert.Contains(t, response.Headers, "Content-Type")
			assert.Equal(t, "application/json", response.Headers["Content-Type"])
			assert.Contains(t, response.Headers, "X-Request-ID")
			assert.Equal(t, "test-request-456", response.Headers["X-Request-ID"])

			// Validate response body can be unmarshaled
			var responseData HelloResponse
			err = json.Unmarshal([]byte(response.Body), &responseData)
			assert.NoError(t, err)
			assert.Equal(t, "Hello from Lambda with observability!", responseData.Message)
			assert.Equal(t, tt.request.Path, responseData.Path)
			assert.Equal(t, "test-request-456", responseData.RequestID)
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