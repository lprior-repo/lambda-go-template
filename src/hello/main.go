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

type HelloResponse struct {
	Message     string `json:"message"`
	Path        string `json:"path"`
	Timestamp   string `json:"timestamp"`
	Environment string `json:"environment"`
	RequestID   string `json:"requestId"`
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
		"service": "hello-service",
		"version": "1.0.0",
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
}

func processHelloRequest(ctx context.Context, request events.APIGatewayProxyRequest) (*HelloResponse, error) {
	// Get Lambda context for request ID
	lc, _ := lambdacontext.FromContext(ctx)

	// Add tracing subsegment
	_, seg := xray.BeginSubsegment(ctx, "processHelloRequest")
	defer seg.Close(nil)

	// Add tracing annotations
	seg.AddAnnotation("path", request.Path)
	seg.AddAnnotation("httpMethod", request.HTTPMethod)

	// Log structured information
	logger.Info("Processing hello request",
		zap.String("path", request.Path),
		zap.String("httpMethod", request.HTTPMethod),
		zap.String("requestId", lc.AwsRequestID),
		zap.String("functionName", lc.FunctionName),
		zap.String("functionVersion", lc.FunctionVersion),
	)

	// Simulate business processing
	time.Sleep(50 * time.Millisecond)

	response := &HelloResponse{
		Message:     "Hello from Lambda with observability!",
		Path:        request.Path,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Environment: os.Getenv("ENVIRONMENT"),
		RequestID:   lc.AwsRequestID,
	}

	// Add response to tracing metadata
	responseData, _ := json.Marshal(response)
	seg.AddMetadata("response", string(responseData))

	logger.Info("Hello request processed successfully",
		zap.String("requestId", lc.AwsRequestID),
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

	// Create tracing segment
	ctx, seg := xray.BeginSegment(ctx, "hello-lambda")
	defer seg.Close(nil)

	// Add correlation ID for tracing
	seg.AddAnnotation("correlationId", lc.AwsRequestID)

	logger.Info("Lambda invocation started",
		zap.String("requestId", lc.AwsRequestID),
		zap.String("functionName", lc.FunctionName),
		zap.String("functionVersion", lc.FunctionVersion),
		zap.Int64("remainingTimeMs", lc.RemainingTimeInMillis()),
	)

	// Process the request
	responseData, err := processHelloRequest(ctx, request)
	if err != nil {
		logger.Error("Failed to process hello request",
			zap.Error(err),
			zap.String("requestId", lc.AwsRequestID),
		)

		// Add error to tracing
		seg.AddError(err)

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