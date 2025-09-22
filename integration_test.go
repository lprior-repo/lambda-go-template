package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration from environment variables
var (
	helloEndpoint = getEnvOrDefault("HELLO_ENDPOINT", "https://sa5b99x1e8.execute-api.us-east-1.amazonaws.com/prod/hello")
	usersEndpoint = getEnvOrDefault("USERS_ENDPOINT", "https://sa5b99x1e8.execute-api.us-east-1.amazonaws.com/prod/users")
	apiBaseURL    = getEnvOrDefault("API_BASE_URL", "https://sa5b99x1e8.execute-api.us-east-1.amazonaws.com/prod")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Expected response structures
type HelloResponse struct {
	Message     string `json:"message"`
	Path        string `json:"path"`
	Timestamp   string `json:"timestamp"`
	Environment string `json:"environment"`
	RequestID   string `json:"requestId"`
	Version     string `json:"version"`
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

type UsersResponse struct {
	Users     []User `json:"users"`
	Count     int    `json:"count"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"requestId"`
	Version   string `json:"version"`
}

type SuccessWrapper struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
	Timestamp string      `json:"timestamp"`
}

type ErrorResponse struct {
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
	Path      string `json:"path,omitempty"`
}

// HTTP client with reasonable timeouts
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func TestHelloEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		expectedStatus int
		headers        map[string]string
		validateFunc   func(*testing.T, *http.Response, []byte)
	}{
		{
			name:           "GET /hello should return success",
			method:         "GET",
			endpoint:       helloEndpoint,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				// Note: CORS headers are set by API Gateway, not Lambda for v2 HTTP API
				validateCacheHeaders(t, resp)

				var wrapper SuccessWrapper
				var helloResp HelloResponse
				err := json.Unmarshal(body, &wrapper)
				require.NoError(t, err, "Response should be valid JSON")

				// Convert data to HelloResponse
				dataBytes, err := json.Marshal(wrapper.Data)
				require.NoError(t, err)
				err = json.Unmarshal(dataBytes, &helloResp)
				require.NoError(t, err)

				assert.Equal(t, "Hello from Lambda with observability!", helloResp.Message)
				assert.Contains(t, helloResp.Path, "/hello")
				assert.Equal(t, "dev", helloResp.Environment)
				assert.Equal(t, "1.0.0", helloResp.Version)
				assert.NotEmpty(t, helloResp.RequestID)
				assert.NotEmpty(t, helloResp.Timestamp)

				// Validate timestamp format
				_, err = time.Parse(time.RFC3339, helloResp.Timestamp)
				assert.NoError(t, err, "Timestamp should be valid RFC3339")

				// Validate request ID format
				assert.NotEmpty(t, wrapper.RequestID)
				assert.Equal(t, helloResp.RequestID, wrapper.RequestID)
			},
		},
		{
			name:           "POST /hello should return 404 (API Gateway behavior)",
			method:         "POST",
			endpoint:       helloEndpoint,
			expectedStatus: http.StatusNotFound,
			headers:        map[string]string{"Content-Type": "application/json"},
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				var errorResp map[string]interface{}
				err := json.Unmarshal(body, &errorResp)
				require.NoError(t, err, "Error response should be valid JSON")

				assert.Contains(t, errorResp["message"], "Not Found")
			},
		},
		{
			name:           "HEAD /hello should return 404 (API Gateway behavior)",
			method:         "HEAD",
			endpoint:       helloEndpoint,
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				assert.Empty(t, body, "HEAD request should have empty body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.endpoint, nil)
			require.NoError(t, err)

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Status code mismatch. Response body: %s", string(body))

			if tt.validateFunc != nil {
				tt.validateFunc(t, resp, body)
			}
		})
	}
}

func TestUsersEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		expectedStatus int
		headers        map[string]string
		validateFunc   func(*testing.T, *http.Response, []byte)
	}{
		{
			name:           "GET /users should return users list",
			method:         "GET",
			endpoint:       usersEndpoint,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				// Note: CORS headers are set by API Gateway, not Lambda for v2 HTTP API
				validateCacheHeaders(t, resp)

				var wrapper SuccessWrapper
				var usersResp UsersResponse
				err := json.Unmarshal(body, &wrapper)
				require.NoError(t, err, "Response should be valid JSON")

				// Convert data to UsersResponse
				dataBytes, err := json.Marshal(wrapper.Data)
				require.NoError(t, err)
				err = json.Unmarshal(dataBytes, &usersResp)
				require.NoError(t, err)

				assert.GreaterOrEqual(t, usersResp.Count, 1, "Should have at least one user")
				assert.Len(t, usersResp.Users, usersResp.Count, "Count should match users array length")
				assert.Equal(t, "1.0.0", usersResp.Version)
				assert.NotEmpty(t, usersResp.RequestID)
				assert.NotEmpty(t, usersResp.Timestamp)

				// Validate first user structure
				if len(usersResp.Users) > 0 {
					user := usersResp.Users[0]
					assert.NotEmpty(t, user.ID)
					assert.NotEmpty(t, user.Name)
					assert.NotEmpty(t, user.Email)
					assert.Contains(t, user.Email, "@")
					assert.NotEmpty(t, user.CreatedAt)

					// Validate timestamp format
					_, err = time.Parse(time.RFC3339, user.CreatedAt)
					assert.NoError(t, err, "User CreatedAt should be valid RFC3339")
				}

				// Validate timestamp format
				_, err = time.Parse(time.RFC3339, usersResp.Timestamp)
				assert.NoError(t, err, "Timestamp should be valid RFC3339")
			},
		},
		{
			name:           "POST /users should return method not allowed",
			method:         "POST",
			endpoint:       usersEndpoint,
			expectedStatus: http.StatusBadRequest,
			headers:        map[string]string{"Content-Type": "application/json"},
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				var errorResp ErrorResponse
				err := json.Unmarshal(body, &errorResp)
				require.NoError(t, err, "Error response should be valid JSON")

				assert.Contains(t, strings.ToLower(errorResp.Message), "get")
				assert.Contains(t, strings.ToLower(errorResp.Message), "allowed")
				assert.NotEmpty(t, errorResp.RequestID)
				assert.NotEmpty(t, errorResp.Timestamp)
			},
		},
		{
			name:           "PUT /users should return method not allowed",
			method:         "PUT",
			endpoint:       usersEndpoint,
			expectedStatus: http.StatusBadRequest,
			headers:        map[string]string{"Content-Type": "application/json"},
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				var errorResp ErrorResponse
				err := json.Unmarshal(body, &errorResp)
				require.NoError(t, err, "Error response should be valid JSON")

				assert.Contains(t, strings.ToLower(errorResp.Message), "method")
				assert.NotEmpty(t, errorResp.RequestID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody io.Reader
			if tt.method == "POST" || tt.method == "PUT" {
				reqBody = bytes.NewBufferString(`{"test": "data"}`)
			}

			req, err := http.NewRequest(tt.method, tt.endpoint, reqBody)
			require.NoError(t, err)

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Status code mismatch. Response body: %s", string(body))

			if tt.validateFunc != nil {
				tt.validateFunc(t, resp, body)
			}
		})
	}
}

func TestAPIGatewayBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		validateFunc   func(*testing.T, *http.Response, []byte)
	}{
		{
			name:           "Non-existent endpoint should return 404",
			endpoint:       apiBaseURL + "/nonexistent",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				var errorResp map[string]interface{}
				err := json.Unmarshal(body, &errorResp)
				require.NoError(t, err)
				assert.Contains(t, errorResp["message"], "Not Found")
			},
		},
		{
			name:           "Root path should return 404",
			endpoint:       apiBaseURL + "/",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
				var errorResp map[string]interface{}
				err := json.Unmarshal(body, &errorResp)
				require.NoError(t, err)
				assert.Contains(t, errorResp["message"], "Not Found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := httpClient.Get(tt.endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Status code mismatch. Response body: %s", string(body))

			if tt.validateFunc != nil {
				tt.validateFunc(t, resp, body)
			}
		})
	}
}

func TestConcurrentRequests(t *testing.T) {
	t.Parallel()

	const numRequests = 10
	const numWorkers = 5

	// Test concurrent hello requests
	t.Run("Concurrent hello requests", func(t *testing.T) {
		results := make(chan bool, numRequests)

		for i := 0; i < numWorkers; i++ {
			go func() {
				for j := 0; j < numRequests/numWorkers; j++ {
					resp, err := httpClient.Get(helloEndpoint)
					if err != nil {
						results <- false
						continue
					}

					success := resp.StatusCode == http.StatusOK
					resp.Body.Close()
					results <- success
				}
			}()
		}

		successCount := 0
		for i := 0; i < numRequests; i++ {
			if <-results {
				successCount++
			}
		}

		assert.GreaterOrEqual(t, successCount, numRequests*8/10,
			"At least 80%% of concurrent requests should succeed")
	})

	// Test concurrent users requests
	t.Run("Concurrent users requests", func(t *testing.T) {
		results := make(chan bool, numRequests)

		for i := 0; i < numWorkers; i++ {
			go func() {
				for j := 0; j < numRequests/numWorkers; j++ {
					resp, err := httpClient.Get(usersEndpoint)
					if err != nil {
						results <- false
						continue
					}

					success := resp.StatusCode == http.StatusOK
					resp.Body.Close()
					results <- success
				}
			}()
		}

		successCount := 0
		for i := 0; i < numRequests; i++ {
			if <-results {
				successCount++
			}
		}

		assert.GreaterOrEqual(t, successCount, numRequests*8/10,
			"At least 80%% of concurrent users requests should succeed")
	})
}

func TestResponseTimes(t *testing.T) {
	t.Parallel()

	endpoints := []struct {
		name     string
		endpoint string
	}{
		{"hello", helloEndpoint},
		{"users", usersEndpoint},
	}

	for _, ep := range endpoints {
		t.Run(fmt.Sprintf("%s response time", ep.name), func(t *testing.T) {
			const numSamples = 5
			var totalDuration time.Duration

			for i := 0; i < numSamples; i++ {
				start := time.Now()
				resp, err := httpClient.Get(ep.endpoint)
				duration := time.Since(start)

				require.NoError(t, err)
				resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				totalDuration += duration

				// Each request should complete within 30 seconds (Lambda timeout)
				assert.Less(t, duration, 30*time.Second,
					"Request should complete within 30 seconds")
			}

			avgDuration := totalDuration / numSamples
			t.Logf("%s average response time: %v", ep.name, avgDuration)

			// Average response time should be reasonable for a Lambda cold start
			assert.Less(t, avgDuration, 10*time.Second,
				"Average response time should be reasonable")
		})
	}
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           string
		headers        map[string]string
		expectedStatus int
		validateError  func(*testing.T, ErrorResponse)
	}{
		{
			name:           "Invalid JSON body should return 404 (API Gateway behavior)",
			method:         "POST",
			endpoint:       helloEndpoint,
			body:           `{"invalid": json}`,
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, resp ErrorResponse) {
				assert.Contains(t, strings.ToLower(resp.Message), "not found")
			},
		},
		{
			name:           "Missing content type for POST with body should return 404 (API Gateway behavior)",
			method:         "POST",
			endpoint:       helloEndpoint,
			body:           `{"test": "data"}`,
			headers:        map[string]string{},
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, resp ErrorResponse) {
				assert.Contains(t, strings.ToLower(resp.Message), "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}

			req, err := http.NewRequest(tt.method, tt.endpoint, reqBody)
			require.NoError(t, err)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusNotFound {
				// API Gateway 404 responses have different structure
				var errorResp map[string]interface{}
				err = json.Unmarshal(body, &errorResp)
				require.NoError(t, err, "Error response should be valid JSON")
				assert.NotEmpty(t, errorResp["message"])
			} else {
				var errorResp ErrorResponse
				err = json.Unmarshal(body, &errorResp)
				require.NoError(t, err, "Error response should be valid JSON")

				assert.NotEmpty(t, errorResp.Message)
				assert.NotEmpty(t, errorResp.RequestID)
				assert.NotEmpty(t, errorResp.Timestamp)
			}

			if tt.validateError != nil && tt.expectedStatus != http.StatusNotFound {
				var errorResp ErrorResponse
				json.Unmarshal(body, &errorResp)
				tt.validateError(t, errorResp)
			}
		})
	}
}

// Helper functions for validation

func validateCacheHeaders(t *testing.T, resp *http.Response) {
	cacheControl := resp.Header.Get("Cache-Control")
	assert.NotEmpty(t, cacheControl)
	assert.Contains(t, cacheControl, "max-age")
}

// Integration test setup and teardown
func TestMain(m *testing.M) {
	// Verify endpoints are accessible before running tests
	if !isEndpointAccessible(helloEndpoint) {
		fmt.Printf("Hello endpoint %s is not accessible\n", helloEndpoint)
		os.Exit(1)
	}

	if !isEndpointAccessible(usersEndpoint) {
		fmt.Printf("Users endpoint %s is not accessible\n", usersEndpoint)
		os.Exit(1)
	}

	fmt.Printf("Running integration tests against:\n")
	fmt.Printf("  Hello endpoint: %s\n", helloEndpoint)
	fmt.Printf("  Users endpoint: %s\n", usersEndpoint)

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func isEndpointAccessible(endpoint string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Accept any response that's not a connection error
	return resp.StatusCode > 0
}
