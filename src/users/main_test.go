package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"lambda-go-template/internal/testutil"
	"lambda-go-template/pkg/lambda"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUserRepository for testing
type TestUserRepository struct {
	users   []User
	failGet bool
}

func NewTestUserRepository() *TestUserRepository {
	return &TestUserRepository{
		users: []User{
			{
				ID:        "1",
				Name:      "John Doe",
				Email:     "john@example.com",
				CreatedAt: "2024-01-15T10:30:00Z",
			},
			{
				ID:        "2",
				Name:      "Jane Smith",
				Email:     "jane@example.com",
				CreatedAt: "2024-01-16T14:45:00Z",
			},
		},
	}
}

func (r *TestUserRepository) GetUsers(ctx context.Context) ([]User, error) {
	if r.failGet {
		return nil, assert.AnError
	}
	return r.users, nil
}

func (r *TestUserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	if r.failGet {
		return nil, assert.AnError
	}

	for _, user := range r.users {
		if user.ID == id {
			return &user, nil
		}
	}

	return nil, lambda.NewResourceNotFoundError("user", id, "user not found")
}

func TestUsersService_ProcessUsersRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
		validate    func(*testing.T, *UsersResponse)
		setupRepo   func(*TestUserRepository)
	}{
		{
			name:        "should process valid users list request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: false,
			validate: func(t *testing.T, response *UsersResponse) {
				assert.Equal(t, 2, response.Count)
				assert.Len(t, response.Users, 2)
				assert.Equal(t, "test", response.RequestID)
				assert.Equal(t, "1.0.0-test", response.Version)

				// Validate timestamp format
				_, err := time.Parse(time.RFC3339, response.Timestamp)
				assert.NoError(t, err, "Timestamp should be in RFC3339 format")

				// Validate timestamp is recent
				testutil.AssertTimestampRecent(t, response.Timestamp, 5*time.Second)

				// Validate users structure
				for _, user := range response.Users {
					assert.NotEmpty(t, user.ID)
					assert.NotEmpty(t, user.Name)
					assert.NotEmpty(t, user.Email)
					assert.NotEmpty(t, user.CreatedAt)
				}
			},
		},
		{
			name: "should process single user request",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/1", map[string]string{
				"id": "1",
			}),
			expectError: false,
			validate: func(t *testing.T, response *UsersResponse) {
				assert.Equal(t, 1, response.Count)
				assert.Len(t, response.Users, 1)
				assert.Equal(t, "1", response.Users[0].ID)
				assert.Equal(t, "John Doe", response.Users[0].Name)
				assert.Equal(t, "john@example.com", response.Users[0].Email)
			},
		},
		{
			name: "should return error for non-existent user",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/999", map[string]string{
				"id": "999",
			}),
			expectError: true,
		},
		{
			name: "should handle request without user ID as list request",
			request: testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: false,
			validate: func(t *testing.T, response *UsersResponse) {
				assert.Equal(t, 2, response.Count)
				assert.Len(t, response.Users, 2)
			},
		},
		{
			name:        "should handle repository error",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: true,
			setupRepo: func(repo *TestUserRepository) {
				repo.failGet = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cfg := testutil.TestConfig()
			logger := testutil.TestLogger(t)
			tracer := testutil.TestTracer()
			repo := NewTestUserRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			// Create service
			service := NewUsersService(cfg, logger, tracer, repo)

			// Create test context
			ctx := testutil.CreateTestContext("test")

			// Execute test
			result, err := service.ProcessUsersRequest(ctx, tt.request)

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

func TestUsersService_ValidateUsersRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
		errorType   string
	}{
		{
			name:        "should validate GET request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: false,
		},
		{
			name:        "should reject POST request",
			request:     testutil.CreateTestAPIGatewayRequest("POST", "/users"),
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name:        "should reject PUT request",
			request:     testutil.CreateTestAPIGatewayRequest("PUT", "/users"),
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name:        "should reject DELETE request",
			request:     testutil.CreateTestAPIGatewayRequest("DELETE", "/users"),
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "should reject empty user ID in path",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/", map[string]string{
				"id": "",
			}),
			expectError: true,
			errorType:   "ValidationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cfg := testutil.TestConfig()
			logger := testutil.TestLogger(t)
			tracer := testutil.TestTracer()
			repo := NewTestUserRepository()

			// Create service
			service := NewUsersService(cfg, logger, tracer, repo)

			// Create test context
			ctx := testutil.CreateTestContext("test-validation")

			// Execute test
			err := service.ValidateUsersRequest(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != "" {
					switch tt.errorType {
					case "ValidationError":
						assert.True(t, lambda.IsValidationError(err))
					}
				}
			} else {
				assert.NoError(t, err)
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
			name:        "should handle valid users list request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: false,
		},
		{
			name: "should handle valid single user request",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/1", map[string]string{
				"id": "1",
			}),
			expectError: false,
		},
		{
			name:        "should reject invalid HTTP method",
			request:     testutil.CreateTestAPIGatewayRequest("POST", "/users"),
			expectError: true,
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
			ctx := testutil.CreateTestContext("test-handler")

			// Execute test
			result, err := handler(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Validate that result is a UsersResponse
				response, ok := result.(*UsersResponse)
				require.True(t, ok, "Result should be a UsersResponse")

				assert.NotEmpty(t, response.Timestamp)
				assert.Equal(t, "test-handler", response.RequestID)
				assert.Equal(t, "1.0.0-test", response.Version)
			}
		})
	}
}

func TestCustomValidationMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
	}{
		{
			name:        "should allow GET request",
			request:     testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectError: false,
		},
		{
			name:        "should reject POST request",
			request:     testutil.CreateTestAPIGatewayRequest("POST", "/users"),
			expectError: true,
		},
		{
			name: "should allow normal user ID",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/123", map[string]string{
				"id": "123",
			}),
			expectError: false,
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cfg := testutil.TestConfig()

			// Create mock handler
			mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
				return "success", nil
			}

			// Apply middleware
			middleware := CustomValidationMiddleware(cfg)
			wrappedHandler := middleware(mockHandler)

			// Create test context
			ctx := testutil.CreateTestContext("test-middleware")

			// Execute test
			result, err := wrappedHandler(ctx, tt.request)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, lambda.IsValidationError(err))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "success", result)
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
			request:            testutil.CreateTestAPIGatewayRequest("GET", "/users"),
			expectedStatusCode: 200,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertSuccessResponse(t, body, nil)

				// Additional validation for users response
				var successResponse struct {
					Data struct {
						Users     []User `json:"users"`
						Count     int    `json:"count"`
						Timestamp string `json:"timestamp"`
						RequestID string `json:"requestId"`
						Version   string `json:"version"`
					} `json:"data"`
				}
				err := json.Unmarshal([]byte(body), &successResponse)
				require.NoError(t, err)

				assert.Equal(t, 3, successResponse.Data.Count) // MockUserRepository has 3 users
				assert.Len(t, successResponse.Data.Users, 3)
				assert.Equal(t, "1.0.0-test", successResponse.Data.Version)
			},
		},
		{
			name: "should return 200 for single user request",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/1", map[string]string{
				"id": "1",
			}),
			expectedStatusCode: 200,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertSuccessResponse(t, body, nil)
			},
		},
		{
			name: "should return 404 for non-existent user",
			request: testutil.CreateTestAPIGatewayRequestWithPath("GET", "/users/999", map[string]string{
				"id": "999",
			}),
			expectedStatusCode: 404,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertErrorResponse(t, body, "user not found")
			},
		},
		{
			name:               "should return 400 for invalid HTTP method",
			request:            testutil.CreateTestAPIGatewayRequest("POST", "/users"),
			expectedStatusCode: 400,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertErrorResponse(t, body, "only GET method is allowed for users endpoint")
			},
		},
		{
			name:               "should return 400 for unsupported HTTP method",
			request:            testutil.CreateTestAPIGatewayRequest("PATCH", "/users"),
			expectedStatusCode: 400,
			validateResponse: func(t *testing.T, body string) {
				testutil.AssertValidJSONResponse(t, body)
				testutil.AssertErrorResponse(t, body, "only GET method is allowed for users endpoint")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cleanup := testutil.SetupTestEnvironment(t, map[string]string{
				"ENVIRONMENT":     "test",
				"SERVICE_NAME":    "users-service",
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
				CustomValidationMiddleware(cfg),
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
		{
			name: "should serialize user with special characters",
			user: User{
				ID:        "user-123",
				Name:      "João da Silva",
				Email:     "joão@example.com",
				CreatedAt: "2024-01-01T12:00:00Z",
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
				Version:   "1.0.0",
			},
		},
		{
			name: "should serialize response with empty users",
			response: UsersResponse{
				Users:     []User{},
				Count:     0,
				Timestamp: "2024-01-01T12:00:00Z",
				RequestID: "req-456",
				Version:   "1.0.0",
			},
		},
		{
			name: "should serialize response with multiple users",
			response: UsersResponse{
				Users: []User{
					{
						ID:        "1",
						Name:      "John Doe",
						Email:     "john@example.com",
						CreatedAt: "2024-01-01T12:00:00Z",
					},
					{
						ID:        "2",
						Name:      "Jane Smith",
						Email:     "jane@example.com",
						CreatedAt: "2024-01-02T12:00:00Z",
					},
				},
				Count:     2,
				Timestamp: "2024-01-01T12:00:00Z",
				RequestID: "req-789",
				Version:   "2.0.0",
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

func TestMockUserRepository(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	t.Run("GetUsers should return all users", func(t *testing.T) {
		users, err := repo.GetUsers(ctx)
		assert.NoError(t, err)
		assert.Len(t, users, 3)

		// Validate user data
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

	t.Run("GetUserByID should return specific user", func(t *testing.T) {
		user, err := repo.GetUserByID(ctx, "1")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "1", user.ID)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)
	})

	t.Run("GetUserByID should return error for non-existent user", func(t *testing.T) {
		user, err := repo.GetUserByID(ctx, "999")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, lambda.IsNotFoundError(err))
	})
}

func BenchmarkUsersService_ProcessUsersRequest(b *testing.B) {
	// Setup
	cfg := testutil.TestConfig()
	logger := testutil.TestLogger(&testing.T{}) // Use testing.T for benchmark
	tracer := testutil.TestTracer()
	repo := NewMockUserRepository()
	service := NewUsersService(cfg, logger, tracer, repo)

	request := testutil.CreateTestAPIGatewayRequest("GET", "/users")
	ctx := testutil.CreateTestContext("bench-request")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.ProcessUsersRequest(ctx, request)
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

	request := testutil.CreateTestAPIGatewayRequest("GET", "/users")
	ctx := testutil.CreateTestContext("bench-request")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}
