package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-xray-sdk-go/xray"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

var logger *zap.Logger

func init() {
	// Configure structured logging
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Add service name to all logs
	config.InitialFields = map[string]interface{}{
		"service": "users-service",
		"version": "1.0.0",
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
}

func getUsersFromDatabase(ctx context.Context) ([]User, error) {
	// Add tracing subsegment (only in AWS Lambda environment)
	var seg *xray.Segment
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		_, seg = xray.BeginSubsegment(ctx, "getUsersFromDatabase")
		defer seg.Close(nil)
	}

	// Simulate database latency
	time.Sleep(50 * time.Millisecond)

	// Mock users data - in real implementation, you'd fetch from DynamoDB
	users := []User{
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
	}

	// Add tracing annotation (only if tracing is enabled)
	if seg != nil {
		seg.AddAnnotation("userCount", len(users))
	}

	logger.Info("Users retrieved from database",
		zap.Int("userCount", len(users)),
	)

	return users, nil
}

func processUsersRequest(ctx context.Context, request events.APIGatewayProxyRequest) (*UsersResponse, error) {
	// Get Lambda context for request ID
	lc, _ := lambdacontext.FromContext(ctx)

	// Add tracing subsegment (only in AWS Lambda environment)
	var seg *xray.Segment
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		_, seg = xray.BeginSubsegment(ctx, "processUsersRequest")
		defer seg.Close(nil)

		// Add tracing annotations
		seg.AddAnnotation("path", request.Path)
		seg.AddAnnotation("httpMethod", request.HTTPMethod)
	}

	// Log structured information
	logger.Info("Processing users request",
		zap.String("path", request.Path),
		zap.String("httpMethod", request.HTTPMethod),
		zap.String("requestId", lc.AwsRequestID),
		zap.String("functionName", os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
		zap.String("functionVersion", os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")),
	)

	// Fetch users data
	users, err := getUsersFromDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users from database: %w", err)
	}

	response := &UsersResponse{
		Users:     users,
		Count:     len(users),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: lc.AwsRequestID,
	}

	// Add response to tracing metadata (only if tracing is enabled)
	responseData, _ := json.Marshal(response)
	if seg != nil {
		seg.AddMetadata("response", string(responseData))
	}

	logger.Info("Users request processed successfully",
		zap.String("requestId", lc.AwsRequestID),
		zap.Int("userCount", len(users)),
		zap.Int("responseSize", len(responseData)),
	)

	return response, nil
}

func createResponseHeaders(requestID string) map[string]string {
	return map[string]string{
		"Content-Type":                     "application/json",
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		"Access-Control-Allow-Methods":     "OPTIONS,POST,GET",
		"X-Request-ID":                     requestID,
		"Cache-Control":                    "max-age=300",
	}
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	// Get Lambda context
	lc, _ := lambdacontext.FromContext(ctx)

	// Create tracing segment (only in AWS Lambda environment)
	var seg *xray.Segment
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		ctx, seg = xray.BeginSegment(ctx, "users-lambda")
		defer seg.Close(nil)

		// Add correlation ID for tracing
		seg.AddAnnotation("correlationId", lc.AwsRequestID)
	}

	logger.Info("Lambda invocation started",
		zap.String("requestId", lc.AwsRequestID),
		zap.String("functionName", os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
		zap.String("functionVersion", os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")),
	)

	// Process the request
	responseData, err := processUsersRequest(ctx, request)
	if err != nil {
		logger.Error("Failed to process users request",
			zap.Error(err),
			zap.String("requestId", lc.AwsRequestID),
		)

		// Add error to tracing (only if tracing is enabled)
		if seg != nil {
			seg.AddError(err)
		}

		// Create error response
		errorResponse := map[string]interface{}{
			"message":   "Internal server error",
			"requestId": lc.AwsRequestID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		errorBody, _ := json.Marshal(errorResponse)
		return Response{
			StatusCode: 500,
			Headers:    createResponseHeaders(lc.AwsRequestID),
			Body:       string(errorBody),
		}, nil
	}

	// Marshal response
	responseBody, err := json.Marshal(responseData)
	if err != nil {
		logger.Error("Failed to marshal response",
			zap.Error(err),
			zap.String("requestId", lc.AwsRequestID),
		)

		return Response{
			StatusCode: 500,
			Headers:    createResponseHeaders(lc.AwsRequestID),
			Body:       `{"error": "Internal server error"}`,
		}, nil
	}

	response := Response{
		StatusCode: 200,
		Headers:    createResponseHeaders(lc.AwsRequestID),
		Body:       string(responseBody),
	}

	logger.Info("Lambda invocation completed successfully",
		zap.String("requestId", lc.AwsRequestID),
	)

	return response, nil
}

func main() {
	lambda.Start(handler)
}