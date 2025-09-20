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
			requestID: "users-request-123",
			expected: map[string]string{
				"Content-Type":                     "application/json",
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				"Access-Control-Allow-Methods":     "OPTIONS,POST,GET",
				"X-Request-ID":                     "users-request-123",
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

func TestGetUsersFromDatabase(t *testing.T) {
	tests := []struct {
		name          string
		expectedCount int
	}{
		{
			name:          "should return predefined users",
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			users, err := getUsersFromDatabase(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, users)
			assert.Len(t, users, tt.expectedCount)

			// Validate user structure
			for _, user := range users {
				assert.NotEmpty(t, user.ID)
				assert.NotEmpty(t, user.Name)
				assert.NotEmpty(t, user.Email)
				assert.NotEmpty(t, user.CreatedAt)

				// Validate timestamp format
				_, timeErr := time.Parse("2006-01-02T15:04:05Z", user.CreatedAt)
				assert.NoError(t, timeErr, "CreatedAt should be in ISO format")
			}

			// Validate specific users
			expectedUsers := map[string]User{
				"1": {
					ID:        "1",
					Name:      "John Doe",
					Email:     "john@example.com",
					CreatedAt: "2024-01-15T10:30:00Z",
				},
				"2": {
					ID:        "2",
					Name:      "Jane Smith",
					Email:     "jane@example.com",
					CreatedAt: "2024-01-16T14:45:00Z",
				},
				"3": {
					ID:        "3",
					Name:      "Alice Johnson",
					Email:     "alice@example.com",
					CreatedAt: "2024-01-17T09:15:00Z",
				},
			}

			for _, user := range users {
				expected, exists := expectedUsers[user.ID]
				assert.True(t, exists, "User ID %s should exist in expected users", user.ID)
				assert.Equal(t, expected, user)
			}
		})
	}
}

func TestProcessUsersRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
	}{
		{
			name: "should process valid users request",
			request: events.APIGatewayProxyRequest{
				Path:       "/users",
				HTTPMethod: "GET",
			},
			expectError: false,
		},
		{
			name: "should process POST request",
			request: events.APIGatewayProxyRequest{
				Path:       "/users",
				HTTPMethod: "POST",
			},
			expectError: false,
		},
		{
			name: "should process request with different path",
			request: events.APIGatewayProxyRequest{
				Path:       "/api/users",
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
				AwsRequestID: "users-test-request-123",
			}
			ctx = lambdacontext.NewContext(ctx, lc)

			// Call the function
			result, err := processUsersRequest(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Validate response structure
				assert.Equal(t, 3, result.Count)
				assert.Len(t, result.Users, 3)
				assert.Equal(t, "users-test-request-123", result.RequestID)

				// Validate timestamp format
				_, timeErr := time.Parse(time.RFC3339, result.Timestamp)
				assert.NoError(t, timeErr, "Timestamp should be in RFC3339 format")

				// Validate users data
				for _, user := range result.Users {
					assert.NotEmpty(t, user.ID)
					assert.NotEmpty(t, user.Name)
					assert.NotEmpty(t, user.Email)
					assert.NotEmpty(t, user.CreatedAt)
				}
			}
		})
	}
}

func TestHandler(t *testing.T) {
	// Set environment variables for testing
	originalFuncName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	originalFuncVersion := os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")

	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "users-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "1")

	defer func() {
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
				Path:       "/users",
				HTTPMethod: "GET",
			},
			expectedStatus: 200,
		},
		{
			name: "should return 200 for valid POST request",
			request: events.APIGatewayProxyRequest{
				Path:       "/users",
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
				AwsRequestID: "users-test-request-456",
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
			assert.Equal(t, "users-test-request-456", response.Headers["X-Request-ID"])

			// Validate response body can be unmarshaled
			var responseData UsersResponse
			err = json.Unmarshal([]byte(response.Body), &responseData)
			assert.NoError(t, err)
			assert.Equal(t, 3, responseData.Count)
			assert.Len(t, responseData.Users, 3)
			assert.Equal(t, "users-test-request-456", responseData.RequestID)
		})
	}
}

func TestUserJSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		user User
	}{
		{
			name: "should serialize complete user",
			user: User{
				ID:        "test-id",
				Name:      "Test User",
				Email:     "test@example.com",
				CreatedAt: "2024-01-01T12:00:00Z",
			},
		},
		{
			name: "should serialize user with empty fields",
			user: User{
				ID:        "",
				Name:      "",
				Email:     "",
				CreatedAt: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tt.user)
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Unmarshal back to struct
			var unmarshaled User
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.user, unmarshaled)
		})
	}
}

func TestUsersResponseJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response UsersResponse
	}{
		{
			name: "should serialize complete response",
			response: UsersResponse{
				Users: []User{
					{
						ID:        "1",
						Name:      "John Doe",
						Email:     "john@example.com",
						CreatedAt: "2024-01-01T12:00:00Z",
					},
				},
				Count:     1,
				Timestamp: "2024-01-01T12:00:00Z",
				RequestID: "req-123",
			},
		},
		{
			name: "should serialize response with empty users",
			response: UsersResponse{
				Users:     []User{},
				Count:     0,
				Timestamp: "2024-01-01T12:00:00Z",
				RequestID: "req-456",
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
			var unmarshaled UsersResponse
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.response, unmarshaled)
		})
	}
}