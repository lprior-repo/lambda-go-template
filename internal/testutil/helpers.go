// Package testutil provides testing utilities and helpers for Lambda functions.
package testutil

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"lambda-go-template/pkg/config"
	"lambda-go-template/pkg/observability"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestConfig creates a test configuration with safe defaults.
func TestConfig() *config.Config {
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
		EnableTracing:   false, // Disable tracing in tests
		EnableMetrics:   false, // Disable metrics in tests
		CacheMaxAge:     300,
	}
}

// TestLogger creates a test logger that outputs to the test log.
func TestLogger(t *testing.T) *observability.Logger {
	zapLogger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
	return &observability.Logger{
		Logger: zapLogger,
	}
}

// TestTracer creates a test tracer with tracing disabled.
func TestTracer() *observability.Tracer {
	return observability.NewTracer(observability.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
		Version:     "1.0.0-test",
	})
}

// CreateTestContext creates a context with Lambda context for testing.
func CreateTestContext(requestID string) context.Context {
	ctx := context.Background()
	lc := &lambdacontext.LambdaContext{
		AwsRequestID:       requestID,
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:test-function",
	}
	return lambdacontext.NewContext(ctx, lc)
}

// CreateTestAPIGatewayRequest creates a test API Gateway request.
func CreateTestAPIGatewayRequest(method, path string) events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		HTTPMethod: method,
		Path:       path,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "test-agent/1.0",
		},
		QueryStringParameters: make(map[string]string),
		PathParameters:        make(map[string]string),
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-id",
			Identity: events.APIGatewayRequestIdentity{
				SourceIP:  "127.0.0.1",
				UserAgent: "test-agent/1.0",
			},
			Stage: "test",
		},
	}
}

// CreateTestAPIGatewayRequestWithBody creates a test API Gateway request with JSON body.
func CreateTestAPIGatewayRequestWithBody(method, path string, body interface{}) events.APIGatewayProxyRequest {
	request := CreateTestAPIGatewayRequest(method, path)

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		request.Body = string(bodyBytes)
	}

	return request
}

// CreateTestAPIGatewayRequestWithHeaders creates a test API Gateway request with custom headers.
func CreateTestAPIGatewayRequestWithHeaders(method, path string, headers map[string]string) events.APIGatewayProxyRequest {
	request := CreateTestAPIGatewayRequest(method, path)

	for key, value := range headers {
		request.Headers[key] = value
	}

	return request
}

// CreateTestAPIGatewayRequestWithQuery creates a test API Gateway request with query parameters.
func CreateTestAPIGatewayRequestWithQuery(method, path string, queryParams map[string]string) events.APIGatewayProxyRequest {
	request := CreateTestAPIGatewayRequest(method, path)
	request.QueryStringParameters = queryParams
	return request
}

// CreateTestAPIGatewayRequestWithPath creates a test API Gateway request with path parameters.
func CreateTestAPIGatewayRequestWithPath(method, path string, pathParams map[string]string) events.APIGatewayProxyRequest {
	request := CreateTestAPIGatewayRequest(method, path)
	request.PathParameters = pathParams
	return request
}

// CreateTestAPIGatewayV2Request creates a test API Gateway v2 HTTP request.
func CreateTestAPIGatewayV2Request(method, path string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		Version:               "2.0",
		RouteKey:              method + " " + path,
		RawPath:               path,
		RawQueryString:        "",
		Headers: map[string]string{
			"content-type": "application/json",
			"user-agent":   "test-agent/1.0",
		},
		QueryStringParameters: make(map[string]string),
		PathParameters:        make(map[string]string),
		StageVariables:        make(map[string]string),
		Body:                  "",
		IsBase64Encoded:       false,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			AccountID:    "123456789012",
			APIID:        "test-api-id",
			DomainName:   "test.execute-api.us-east-1.amazonaws.com",
			DomainPrefix: "test",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:    method,
				Path:      path,
				Protocol:  "HTTP/1.1",
				SourceIP:  "127.0.0.1",
				UserAgent: "test-agent/1.0",
			},
			RequestID: "test-request-123",
			RouteKey:  method + " " + path,
			Stage:     "test",
			Time:      "22/Sep/2025:01:00:00 +0000",
			TimeEpoch: time.Now().Unix(),
		},
	}
}

// CreateTestAPIGatewayV2RequestWithBody creates a test API Gateway v2 HTTP request with JSON body.
func CreateTestAPIGatewayV2RequestWithBody(method, path string, body interface{}) events.APIGatewayV2HTTPRequest {
	request := CreateTestAPIGatewayV2Request(method, path)

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		request.Body = string(bodyBytes)
	}

	return request
}

// CreateTestAPIGatewayV2RequestWithHeaders creates a test API Gateway v2 HTTP request with custom headers.
func CreateTestAPIGatewayV2RequestWithHeaders(method, path string, headers map[string]string) events.APIGatewayV2HTTPRequest {
	request := CreateTestAPIGatewayV2Request(method, path)

	for key, value := range headers {
		request.Headers[key] = value
	}

	return request
}

// CreateTestAPIGatewayV2RequestWithQuery creates a test API Gateway v2 HTTP request with query parameters.
func CreateTestAPIGatewayV2RequestWithQuery(method, path string, queryParams map[string]string) events.APIGatewayV2HTTPRequest {
	request := CreateTestAPIGatewayV2Request(method, path)
	request.QueryStringParameters = queryParams
	return request
}

// CreateTestAPIGatewayV2RequestWithPath creates a test API Gateway v2 HTTP request with path parameters.
func CreateTestAPIGatewayV2RequestWithPath(method, path string, pathParams map[string]string) events.APIGatewayV2HTTPRequest {
	request := CreateTestAPIGatewayV2Request(method, path)
	request.PathParameters = pathParams
	return request
}

// SetupTestEnvironment sets up environment variables for testing.
func SetupTestEnvironment(t *testing.T, envVars map[string]string) func() {
	originalEnv := make(map[string]string)

	// Save original values
	for key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Set test values
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// Return cleanup function
	return func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// AssertJSONResponse asserts that a response body can be unmarshaled into the expected structure.
func AssertJSONResponse(t *testing.T, responseBody string, expected interface{}) ([]byte, error) {
	data := []byte(responseBody)
	err := json.Unmarshal(data, expected)
	assert.NoError(t, err, "Response body should be valid JSON")
	return data, err
}

// AssertValidJSONResponse asserts that a response body is valid JSON.
func AssertValidJSONResponse(t *testing.T, responseBody string) {
	var result interface{}
	err := json.Unmarshal([]byte(responseBody), &result)
	assert.NoError(t, err, "Response body should be valid JSON")
}

// AssertResponseHeaders asserts that required headers are present in the response.
func AssertResponseHeaders(t *testing.T, headers map[string]string, requiredHeaders ...string) {
	for _, header := range requiredHeaders {
		assert.Contains(t, headers, header, "Response should contain header: %s", header)
		assert.NotEmpty(t, headers[header], "Header %s should not be empty", header)
	}
}

// AssertCORSHeaders asserts that CORS headers are properly set.
func AssertCORSHeaders(t *testing.T, headers map[string]string) {
	assert.Equal(t, "*", headers["Access-Control-Allow-Origin"])
	assert.Contains(t, headers, "Access-Control-Allow-Headers")
	assert.Contains(t, headers, "Access-Control-Allow-Methods")
}

// AssertErrorResponse asserts that a response is a proper error response.
func AssertErrorResponse(t *testing.T, responseBody string, expectedMessage string) {
	var errorResponse struct {
		Message   string `json:"message"`
		Error     string `json:"error,omitempty"`
		RequestID string `json:"requestId,omitempty"`
		Timestamp string `json:"timestamp"`
		Path      string `json:"path,omitempty"`
	}

	err := json.Unmarshal([]byte(responseBody), &errorResponse)
	require.NoError(t, err, "Error response should be valid JSON")

	assert.Equal(t, expectedMessage, errorResponse.Message)
	assert.NotEmpty(t, errorResponse.Timestamp)

	// Validate timestamp format
	_, timeErr := time.Parse(time.RFC3339, errorResponse.Timestamp)
	assert.NoError(t, timeErr, "Timestamp should be in RFC3339 format")
}

// AssertSuccessResponse asserts that a response is a proper success response.
func AssertSuccessResponse(t *testing.T, responseBody string, expectedData interface{}) {
	var successResponse struct {
		Data      interface{} `json:"data"`
		RequestID string      `json:"requestId,omitempty"`
		Timestamp string      `json:"timestamp"`
	}

	err := json.Unmarshal([]byte(responseBody), &successResponse)
	require.NoError(t, err, "Success response should be valid JSON")

	assert.NotEmpty(t, successResponse.Timestamp)

	// Validate timestamp format
	_, timeErr := time.Parse(time.RFC3339, successResponse.Timestamp)
	assert.NoError(t, timeErr, "Timestamp should be in RFC3339 format")

	// Compare data if provided
	if expectedData != nil {
		dataBytes, _ := json.Marshal(successResponse.Data)
		expectedBytes, _ := json.Marshal(expectedData)
		assert.JSONEq(t, string(expectedBytes), string(dataBytes))
	}
}

// MockTime mocks the current time for testing.
type MockTime struct {
	currentTime time.Time
}

// NewMockTime creates a new mock time with the specified time.
func NewMockTime(t time.Time) *MockTime {
	return &MockTime{currentTime: t}
}

// Now returns the mocked current time.
func (mt *MockTime) Now() time.Time {
	return mt.currentTime
}

// Advance advances the mocked time by the specified duration.
func (mt *MockTime) Advance(d time.Duration) {
	mt.currentTime = mt.currentTime.Add(d)
}

// TestTableEntry represents a single test case in a table-driven test.
type TestTableEntry struct {
	Name          string
	Input         interface{}
	Expected      interface{}
	ExpectedError bool
	ErrorMessage  string
	Setup         func()
	Cleanup       func()
}

// RunTableTest runs a table-driven test.
func RunTableTest(t *testing.T, tests []TestTableEntry, testFunc func(t *testing.T, input interface{}) (interface{}, error)) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if tt.Setup != nil {
				tt.Setup()
			}

			if tt.Cleanup != nil {
				defer tt.Cleanup()
			}

			result, err := testFunc(t, tt.Input)

			if tt.ExpectedError {
				assert.Error(t, err)
				if tt.ErrorMessage != "" {
					assert.Contains(t, err.Error(), tt.ErrorMessage)
				}
			} else {
				assert.NoError(t, err)
				if tt.Expected != nil {
					assert.Equal(t, tt.Expected, result)
				}
			}
		})
	}
}

// CreateTempFile creates a temporary file for testing.
func CreateTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.tmp")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// Clean up file after test
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

// AssertTimestampRecent asserts that a timestamp is within the last few seconds.
func AssertTimestampRecent(t *testing.T, timestamp string, tolerance time.Duration) {
	ts, err := time.Parse(time.RFC3339, timestamp)
	require.NoError(t, err, "Timestamp should be valid RFC3339")

	now := time.Now()
	diff := now.Sub(ts)

	assert.True(t, diff >= 0, "Timestamp should not be in the future")
	assert.True(t, diff <= tolerance, "Timestamp should be recent (within %v)", tolerance)
}

// AssertRequestIDFormat asserts that a request ID has the expected format.
func AssertRequestIDFormat(t *testing.T, requestID string) {
	assert.NotEmpty(t, requestID, "Request ID should not be empty")
	// Add more specific format validation if needed
}
