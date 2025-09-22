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

// HelloResponse represents the response structure for the hello endpoint.
type HelloResponse struct {
	Message     string `json:"message"`
	Path        string `json:"path"`
	Timestamp   string `json:"timestamp"`
	Environment string `json:"environment"`
	RequestID   string `json:"requestId"`
	Version     string `json:"version"`
}

// HelloService handles the business logic for hello operations.
type HelloService struct {
	config *config.Config
	logger *observability.Logger
	tracer *observability.Tracer
}

// NewHelloService creates a new hello service instance.
func NewHelloService(cfg *config.Config, logger *observability.Logger, tracer *observability.Tracer) *HelloService {
	return &HelloService{
		config: cfg,
		logger: logger,
		tracer: tracer,
	}
}

// ProcessHelloRequest processes a hello request and returns the response data.
func (s *HelloService) ProcessHelloRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*HelloResponse, error) {
	// Add business logic tracing
	ctx, seg := s.tracer.StartSubsegment(ctx, "processHelloRequest")
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
	}).Info("Processing hello request")

	// Simulate business processing (in real world, this might be database calls, etc.)
	err := s.tracer.WithTimer(ctx, "simulate_processing", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if err != nil {
		return nil, lambda.NewInternalErrorWithOperation("hello processing", "failed to simulate processing", err)
	}

	// Create response
	response := &HelloResponse{
		Message:     "Hello from Lambda with observability!",
		Path:        request.RawPath,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Environment: s.config.Environment,
		RequestID:   requestID,
		Version:     s.config.ServiceVersion,
	}

	// Add response metadata to tracing
	s.tracer.AddMetadata(ctx, "response", map[string]interface{}{
		"message":     response.Message,
		"environment": response.Environment,
	})

	s.logger.WithFields(map[string]interface{}{
		"requestId":   requestID,
		"environment": response.Environment,
		"version":     response.Version,
	}).Info("Hello request processed successfully")

	return response, nil
}

// CreateHandler creates the Lambda handler function.
func CreateHandler(cfg *config.Config, logger *observability.Logger, tracer *observability.Tracer) func(context.Context, events.APIGatewayV2HTTPRequest) (interface{}, error) {
	service := NewHelloService(cfg, logger, tracer)

	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
		return service.ProcessHelloRequest(ctx, request)
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

	// Wrap with middleware
	wrappedHandler := handler.WrapV2(
		businessHandler,
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
	}).Info("Starting hello Lambda function")

	// Start Lambda
	awslambda.Start(wrappedHandler)
}
