package main

import (
	"context"
	"time"

	"lambda-go-template/pkg/config"
	"lambda-go-template/pkg/lambda"
	"lambda-go-template/pkg/observability"

	"github.com/aws/aws-lambda-go/events"
	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// User represents a user entity.
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// UsersResponse represents the response structure for the users endpoint.
type UsersResponse struct {
	Users     []User `json:"users"`
	Count     int    `json:"count"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"requestId"`
	Version   string `json:"version"`
}

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetUsers(ctx context.Context) ([]User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

// MockUserRepository provides a mock implementation for testing and development.
type MockUserRepository struct {
	users []User
}

// NewMockUserRepository creates a new mock user repository with sample data.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
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
			{
				ID:        "3",
				Name:      "Alice Johnson",
				Email:     "alice@example.com",
				CreatedAt: "2024-01-17T09:15:00Z",
			},
		},
	}
}

// GetUsers retrieves all users from the mock repository.
func (r *MockUserRepository) GetUsers(ctx context.Context) ([]User, error) {
	// Simulate database latency
	time.Sleep(50 * time.Millisecond)
	return r.users, nil
}

// GetUserByID retrieves a user by ID from the mock repository.
func (r *MockUserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	// Simulate database latency
	time.Sleep(25 * time.Millisecond)

	for _, user := range r.users {
		if user.ID == id {
			return &user, nil
		}
	}

	return nil, lambda.NewResourceNotFoundError("user", id, "user not found")
}

// UsersService handles the business logic for user operations.
type UsersService struct {
	config     *config.Config
	logger     *observability.Logger
	tracer     *observability.Tracer
	repository UserRepository
}

// NewUsersService creates a new users service instance.
func NewUsersService(cfg *config.Config, logger *observability.Logger, tracer *observability.Tracer, repo UserRepository) *UsersService {
	return &UsersService{
		config:     cfg,
		logger:     logger,
		tracer:     tracer,
		repository: repo,
	}
}

// ProcessUsersRequest processes a users list request and returns the response data.
func (s *UsersService) ProcessUsersRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*UsersResponse, error) {
	// Add business logic tracing
	ctx, seg := s.tracer.StartSubsegment(ctx, "processUsersRequest")
	defer s.tracer.Close(seg, nil)

	// Get Lambda context for request ID
	lc, _ := lambdacontext.FromContext(ctx)
	requestID := ""
	if lc != nil {
		requestID = lc.AwsRequestID
	}

	// Add tracing annotations
	s.tracer.AddAnnotation(ctx, "path", request.RawPath)
	s.tracer.AddAnnotation(ctx, "httpMethod", request.RequestContext.HTTP.Method)

	// Log structured information about the request processing
	s.logger.WithFields(map[string]interface{}{
		"path":       request.RawPath,
		"httpMethod": request.RequestContext.HTTP.Method,
		"requestId":  requestID,
	}).Info("Processing users request")

	// Check if this is a request for a specific user
	userID := request.PathParameters["id"]
	if userID != "" {
		return s.processSingleUserRequest(ctx, userID, requestID)
	}

	// Fetch all users data
	var allUsers []User
	err := s.tracer.WithTimer(ctx, "getUsersFromDatabase", func(ctx context.Context) error {
		var fetchErr error
		allUsers, fetchErr = s.repository.GetUsers(ctx)
		if fetchErr != nil {
			return lambda.NewInternalErrorWithOperation("database query", "failed to fetch users", fetchErr)
		}

		// Add tracing annotation for user count
		s.tracer.AddAnnotation(ctx, "userCount", len(allUsers))

		s.logger.WithFields(map[string]interface{}{
			"userCount": len(allUsers),
		}).Info("Users retrieved from database")

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Create response
	response := &UsersResponse{
		Users:     allUsers,
		Count:     len(allUsers),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Version:   s.config.ServiceVersion,
	}

	// Add response metadata to tracing
	s.tracer.AddMetadata(ctx, "response", map[string]interface{}{
		"userCount":    response.Count,
		"responseSize": len(allUsers) * 100, // Approximate size
	})

	s.logger.WithFields(map[string]interface{}{
		"requestId": requestID,
		"userCount": response.Count,
		"version":   response.Version,
	}).Info("Users request processed successfully")

	return response, nil
}

// processSingleUserRequest handles requests for a specific user by ID.
func (s *UsersService) processSingleUserRequest(ctx context.Context, userID, requestID string) (*UsersResponse, error) {
	ctx, seg := s.tracer.StartSubsegment(ctx, "processSingleUserRequest")
	defer s.tracer.Close(seg, nil)

	s.tracer.AddAnnotation(ctx, "userId", userID)

	s.logger.WithFields(map[string]interface{}{
		"userId":    userID,
		"requestId": requestID,
	}).Info("Processing single user request")

	// Validate user ID format
	if userID == "" {
		return nil, lambda.NewValidationError("user ID cannot be empty", "id", userID)
	}

	// Fetch specific user
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		// Check if it's a not found error and pass it through
		if lambda.IsNotFoundError(err) {
			return nil, err
		}
		return nil, lambda.NewInternalErrorWithOperation("user retrieval", "failed to get user from repository", err)
	}

	// Create response with single user
	response := &UsersResponse{
		Users:     []User{*user},
		Count:     1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Version:   s.config.ServiceVersion,
	}

	s.logger.WithFields(map[string]interface{}{
		"requestId": requestID,
		"userId":    userID,
		"userName":  user.Name,
	}).Info("Single user request processed successfully")

	return response, nil
}

// ValidateUsersRequest validates the incoming request for users operations.
func (s *UsersService) ValidateUsersRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) error {
	// Validate HTTP method
	if request.RequestContext.HTTP.Method != "GET" {
		return lambda.NewValidationError("only GET method is allowed for users endpoint", "httpMethod", request.RequestContext.HTTP.Method)
	}

	// Validate path parameters if present
	if userID, exists := request.PathParameters["id"]; exists {
		if userID == "" {
			return lambda.NewValidationError("user ID cannot be empty", "id", userID)
		}
	}

	return nil
}

// CreateHandler creates the Lambda handler function.
func CreateHandler(cfg *config.Config, logger *observability.Logger, tracer *observability.Tracer) func(context.Context, events.APIGatewayV2HTTPRequest) (interface{}, error) {
	// Initialize repository (in production, this might be a DynamoDB repository)
	repository := NewMockUserRepository()
	service := NewUsersService(cfg, logger, tracer, repository)

	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
		// Validate request first
		if err := service.ValidateUsersRequest(ctx, request); err != nil {
			return nil, err
		}

		return service.ProcessUsersRequest(ctx, request)
	}
}

// CustomValidationMiddleware provides users-specific validation.
func CustomValidationMiddleware(cfg *config.Config) func(lambda.HandlerFuncV2) lambda.HandlerFuncV2 {
	return func(next lambda.HandlerFuncV2) lambda.HandlerFuncV2 {
		return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
			// Only allow GET requests for users endpoint
			if request.RequestContext.HTTP.Method != "GET" {
				return nil, lambda.NewValidationError("only GET method is allowed for users endpoint", "httpMethod", request.RequestContext.HTTP.Method)
			}

			return next(ctx, request)
		}
	}
}

func main() {
	// Load configuration
	cfg := config.MustLoad()

	// Initialize logger
	logger := observability.MustNewLogger(cfg)
	defer logger.Close()

	// Set global logger for packages that need it
	observability.SetGlobalLogger(logger)

	// Initialize tracer
	tracer := observability.NewTracer(observability.TracingConfig{
		Enabled:     cfg.EnableTracing,
		ServiceName: cfg.ServiceName,
		Version:     cfg.ServiceVersion,
	})

	// Create Lambda handler with middleware
	handler := lambda.NewHandler(cfg, logger, tracer)

	// Create the business logic handler
	businessHandler := CreateHandler(cfg, logger, tracer)

	// Wrap with middleware (including custom validation)
	// Wrap with middleware
	wrappedHandler := handler.WrapV2(
		businessHandler,
		CustomValidationMiddleware(cfg),
		handler.ValidationMiddlewareV2(),
		handler.LoggingMiddlewareV2(),
		handler.TracingMiddlewareV2(),
		handler.TimeoutMiddlewareV2(),
	)

	logger.WithFields(map[string]interface{}{
		"service":     cfg.ServiceName,
		"version":     cfg.ServiceVersion,
		"environment": cfg.Environment,
		"tracing":     cfg.EnableTracing,
		"metrics":     cfg.EnableMetrics,
	}).Info("Starting users Lambda function")

	// Start Lambda
	awslambda.Start(wrappedHandler)
}
