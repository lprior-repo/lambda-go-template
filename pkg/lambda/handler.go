// Package lambda provides common utilities and middleware for AWS Lambda functions.
package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lambda-go-template/pkg/config"
	"lambda-go-template/pkg/http"
	"lambda-go-template/pkg/observability"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// Context key types to avoid collisions
type contextKey string

const (
	contextKeyParsedBody contextKey = "parsed_body"
	contextKeyRequestID  contextKey = "request_id"
	contextKeyTimestamp  contextKey = "timestamp"
)

// Handler represents a Lambda function handler with observability and error handling.
type Handler struct {
	config *config.Config
	logger *observability.Logger
	tracer *observability.Tracer
}

// HandlerFunc represents a Lambda function that processes API Gateway requests.
type HandlerFunc func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error)

// HandlerFuncV2 represents a Lambda function that processes API Gateway v2 HTTP requests.
type HandlerFuncV2 func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error)

// MiddlewareFunc represents middleware that can wrap handlers.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// MiddlewareFuncV2 represents middleware that can wrap v2 handlers.
type MiddlewareFuncV2 func(HandlerFuncV2) HandlerFuncV2

// NewHandler creates a new Lambda handler with observability.
func NewHandler(cfg *config.Config, logger *observability.Logger, tracer *observability.Tracer) *Handler {
	return &Handler{
		config: cfg,
		logger: logger,
		tracer: tracer,
	}
}

// Wrap wraps a handler function with common Lambda functionality including:
// - Request/response logging
// - Distributed tracing
// - Error handling
// - Response formatting
func (h *Handler) Wrap(handlerFunc HandlerFunc, middlewares ...MiddlewareFunc) func(context.Context, events.APIGatewayProxyRequest) (http.Response, error) {
	// Apply middlewares in reverse order
	wrapped := handlerFunc
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return func(ctx context.Context, request events.APIGatewayProxyRequest) (http.Response, error) {
		start := time.Now()

		// Get Lambda context
		lc, _ := lambdacontext.FromContext(ctx)
		requestID := ""
		if lc != nil {
			requestID = lc.AwsRequestID
		}

		// Create tracing segment
		ctx, seg := h.tracer.StartSegment(ctx, h.config.ServiceName)
		defer h.tracer.Close(seg, nil)

		// Add request metadata to tracing
		h.tracer.AddAnnotation(ctx, "http_method", request.HTTPMethod)
		h.tracer.AddAnnotation(ctx, "http_path", request.Path)
		h.tracer.AddAnnotation(ctx, "request_id", requestID)

		// Log Lambda invocation start
		if lc != nil {
			h.logger.LogLambdaStart(ctx, "", "", 0)
		}

		// Create response builder
		responseBuilder := http.NewResponseBuilder().
			WithRequestID(requestID).
			WithPath(request.Path).
			WithCORS().
			WithCacheControl(h.config.CacheMaxAge)

		// Process the request
		data, err := wrapped(ctx, request)
		duration := time.Since(start).Milliseconds()

		if err != nil {
			// Log error
			h.logger.LogLambdaError(ctx, err, "Handler execution failed")

			// Add error to tracing
			h.tracer.AddError(ctx, err)
			h.tracer.AddAnnotation(ctx, "error", true)

			// Determine error type and create appropriate response
			var response http.Response
			switch e := err.(type) {
			case *ValidationError:
				response = responseBuilder.BadRequest(e.Message, e.Err)
			case *NotFoundError:
				response = responseBuilder.NotFound(e.Message)
			case *ConflictError:
				response = responseBuilder.Conflict(e.Message, e.Err)
			case *UnauthorizedError:
				response = responseBuilder.Unauthorized(e.Message)
			case *ForbiddenError:
				response = responseBuilder.Forbidden(e.Message)
			default:
				response = responseBuilder.InternalServerError("Internal server error", err)
			}

			// Log HTTP response
			h.logger.LogHTTPRequest(ctx, request.HTTPMethod, request.Path, response.StatusCode, duration)
			h.tracer.AddAnnotation(ctx, "http_status", response.StatusCode)
			h.tracer.AddAnnotation(ctx, "duration_ms", duration)

			return response, nil
		}

		// Create success response
		response := responseBuilder.OK(data)

		// Add response metadata to tracing
		h.tracer.AddAnnotation(ctx, "http_status", response.StatusCode)
		h.tracer.AddAnnotation(ctx, "duration_ms", duration)
		h.tracer.AddAnnotation(ctx, "success", true)

		// Add response size to metadata
		responseSize := len(response.Body)
		h.tracer.AddMetadata(ctx, "response", map[string]interface{}{
			"size_bytes": responseSize,
		})

		// Log successful completion
		h.logger.LogHTTPRequest(ctx, request.HTTPMethod, request.Path, response.StatusCode, duration)
		h.logger.LogLambdaEnd(ctx, duration)

		return response, nil
	}
}

// WrapV2 wraps a handler function for API Gateway v2 HTTP API with common Lambda functionality.
func (h *Handler) WrapV2(handlerFunc HandlerFuncV2, middlewares ...MiddlewareFuncV2) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// Apply middlewares in reverse order
	wrapped := handlerFunc
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		start := time.Now()

		// Get Lambda context
		lc, _ := lambdacontext.FromContext(ctx)
		requestID := ""
		if lc != nil {
			requestID = lc.AwsRequestID
		}

		// Create tracing segment
		ctx, seg := h.tracer.StartSegment(ctx, h.config.ServiceName)
		defer h.tracer.Close(seg, nil)

		// Add request metadata to tracing
		h.tracer.AddAnnotation(ctx, "http_method", request.RequestContext.HTTP.Method)
		h.tracer.AddAnnotation(ctx, "http_path", request.RawPath)
		h.tracer.AddAnnotation(ctx, "request_id", requestID)

		// Log Lambda invocation start
		if lc != nil {
			h.logger.LogLambdaStart(ctx, "", "", 0)
		}

		// Create response builder
		responseBuilder := http.NewResponseBuilder().
			WithRequestID(requestID).
			WithPath(request.RawPath).
			WithCORS().
			WithCacheControl(h.config.CacheMaxAge)

		// Process the request
		data, err := wrapped(ctx, request)
		duration := time.Since(start).Milliseconds()

		if err != nil {
			// Log error
			h.logger.LogLambdaError(ctx, err, "Handler execution failed")

			// Add error to tracing
			h.tracer.AddError(ctx, err)
			h.tracer.AddAnnotation(ctx, "error", true)

			// Determine error type and create appropriate response
			var response http.Response
			switch e := err.(type) {
			case *ValidationError:
				response = responseBuilder.BadRequest(e.Message, e.Err)
			case *NotFoundError:
				response = responseBuilder.NotFound(e.Message)
			case *ConflictError:
				response = responseBuilder.Conflict(e.Message, e.Err)
			case *UnauthorizedError:
				response = responseBuilder.Unauthorized(e.Message)
			case *ForbiddenError:
				response = responseBuilder.Forbidden(e.Message)
			default:
				response = responseBuilder.InternalServerError("Internal server error", err)
			}

			// Log HTTP response
			h.logger.LogHTTPRequest(ctx, request.RequestContext.HTTP.Method, request.RawPath, response.StatusCode, duration)
			h.tracer.AddAnnotation(ctx, "http_status", response.StatusCode)
			h.tracer.AddAnnotation(ctx, "duration_ms", duration)

			// Convert to v2 response
			return events.APIGatewayV2HTTPResponse{
				StatusCode: response.StatusCode,
				Headers:    response.Headers,
				Body:       response.Body,
			}, nil
		}

		// Create success response
		response := responseBuilder.OK(data)

		// Add response metadata to tracing
		h.tracer.AddAnnotation(ctx, "http_status", response.StatusCode)
		h.tracer.AddAnnotation(ctx, "duration_ms", duration)
		h.tracer.AddAnnotation(ctx, "success", true)

		// Add response size to metadata
		responseSize := len(response.Body)
		h.tracer.AddMetadata(ctx, "response", map[string]interface{}{
			"size_bytes": responseSize,
		})

		// Log successful completion
		h.logger.LogHTTPRequest(ctx, request.RequestContext.HTTP.Method, request.RawPath, response.StatusCode, duration)
		h.logger.LogLambdaEnd(ctx, duration)

		// Convert to v2 response
		return events.APIGatewayV2HTTPResponse{
			StatusCode: response.StatusCode,
			Headers:    response.Headers,
			Body:       response.Body,
		}, nil
	}
}

// LoggingMiddleware adds request/response logging.
func (h *Handler) LoggingMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			// Log request details
			h.logger.WithFields(map[string]interface{}{
				"method":      request.HTTPMethod,
				"path":        request.Path,
				"query":       request.QueryStringParameters,
				"headers":     request.Headers,
				"user_agent":  request.Headers["User-Agent"],
				"source_ip":   request.RequestContext.Identity.SourceIP,
			}).Info("Processing request")

			return next(ctx, request)
		}
	}
}

// LoggingMiddlewareV2 adds request/response logging for v2 HTTP API.
func (h *Handler) LoggingMiddlewareV2() MiddlewareFuncV2 {
	return func(next HandlerFuncV2) HandlerFuncV2 {
		return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
			// Log request details
			h.logger.WithFields(map[string]interface{}{
				"method":      request.RequestContext.HTTP.Method,
				"path":        request.RawPath,
				"query":       request.QueryStringParameters,
				"headers":     request.Headers,
				"user_agent":  request.RequestContext.HTTP.UserAgent,
				"source_ip":   request.RequestContext.HTTP.SourceIP,
			}).Info("Processing request")

			return next(ctx, request)
		}
	}
}

// TracingMiddleware adds detailed tracing information.
func (h *Handler) TracingMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			// Add request details to tracing
			requestData := map[string]interface{}{
				"headers":         request.Headers,
				"query_parameters": request.QueryStringParameters,
				"path_parameters":  request.PathParameters,
			}
			if request.Body != "" {
				requestData["body_size"] = len(request.Body)
			}
			h.tracer.AddMetadata(ctx, "request", requestData)

			return next(ctx, request)
		}
	}
}

// TracingMiddlewareV2 adds detailed tracing information for v2 HTTP API.
func (h *Handler) TracingMiddlewareV2() MiddlewareFuncV2 {
	return func(next HandlerFuncV2) HandlerFuncV2 {
		return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
			// Add request details to tracing
			requestData := map[string]interface{}{
				"headers":         request.Headers,
				"query_parameters": request.QueryStringParameters,
				"path_parameters":  request.PathParameters,
			}
			if request.Body != "" {
				requestData["body_size"] = len(request.Body)
			}
			h.tracer.AddMetadata(ctx, "request", requestData)

			return next(ctx, request)
		}
	}
}

// TimeoutMiddleware adds timeout handling to requests.
func (h *Handler) TimeoutMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, h.config.ResponseTimeout)
			defer cancel()

			// Channel to receive result
			resultChan := make(chan struct {
				data interface{}
				err  error
			}, 1)

			// Execute handler in goroutine
			go func() {
				data, err := next(timeoutCtx, request)
				resultChan <- struct {
					data interface{}
					err  error
				}{data, err}
			}()

			// Wait for result or timeout
			select {
			case result := <-resultChan:
				return result.data, result.err
			case <-timeoutCtx.Done():
				h.tracer.AddAnnotation(ctx, "timeout", true)
				return nil, &TimeoutError{
					Message: fmt.Sprintf("Request timeout after %v", h.config.ResponseTimeout),
					Timeout: h.config.ResponseTimeout,
				}
			}
		}
	}
}

// TimeoutMiddlewareV2 adds timeout handling to v2 HTTP API requests.
func (h *Handler) TimeoutMiddlewareV2() MiddlewareFuncV2 {
	return func(next HandlerFuncV2) HandlerFuncV2 {
		return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, h.config.ResponseTimeout)
			defer cancel()

			// Channel to receive result
			resultChan := make(chan struct {
				data interface{}
				err  error
			}, 1)

			// Execute handler in goroutine
			go func() {
				data, err := next(timeoutCtx, request)
				resultChan <- struct {
					data interface{}
					err  error
				}{data, err}
			}()

			// Wait for result or timeout
			select {
			case result := <-resultChan:
				return result.data, result.err
			case <-timeoutCtx.Done():
				h.tracer.AddAnnotation(ctx, "timeout", true)
				return nil, &TimeoutError{
					Message: fmt.Sprintf("Request timeout after %v", h.config.ResponseTimeout),
					Timeout: h.config.ResponseTimeout,
				}
			}
		}
	}
}

// ValidationMiddleware validates common request parameters.
func (h *Handler) ValidationMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			// Validate HTTP method
			allowedMethods := map[string]bool{
				"GET":     true,
				"POST":    true,
				"PUT":     true,
				"DELETE":  true,
				"PATCH":   true,
				"OPTIONS": true,
			}

			if !allowedMethods[request.HTTPMethod] {
				return nil, &ValidationError{
					Message: fmt.Sprintf("HTTP method %s is not allowed", request.HTTPMethod),
					Field:   "httpMethod",
					Value:   request.HTTPMethod,
				}
			}

			// Validate content type for POST/PUT/PATCH requests with body
			if (request.HTTPMethod == "POST" || request.HTTPMethod == "PUT" || request.HTTPMethod == "PATCH") && request.Body != "" {
				contentType := request.Headers["Content-Type"]
				if contentType == "" {
					contentType = request.Headers["content-type"] // Try lowercase
				}

				if contentType != "application/json" {
					return nil, &ValidationError{
						Message: "Content-Type must be application/json for requests with body",
						Field:   "content-type",
						Value:   contentType,
					}
				}
			}

			return next(ctx, request)
		}
	}
}

// ValidationMiddlewareV2 validates common request parameters for v2 HTTP API.
func (h *Handler) ValidationMiddlewareV2() MiddlewareFuncV2 {
	return func(next HandlerFuncV2) HandlerFuncV2 {
		return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (interface{}, error) {
			// Validate HTTP method
			allowedMethods := map[string]bool{
				"GET":     true,
				"POST":    true,
				"PUT":     true,
				"DELETE":  true,
				"PATCH":   true,
				"OPTIONS": true,
			}

			httpMethod := request.RequestContext.HTTP.Method
			if !allowedMethods[httpMethod] {
				return nil, &ValidationError{
					Message: fmt.Sprintf("HTTP method %s is not allowed", httpMethod),
					Field:   "httpMethod",
					Value:   httpMethod,
				}
			}

			// Validate content type for POST/PUT/PATCH requests with body
			if (httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH") && request.Body != "" {
				contentType := request.Headers["Content-Type"]
				if contentType == "" {
					contentType = request.Headers["content-type"] // Try lowercase
				}

				if contentType != "application/json" {
					return nil, &ValidationError{
						Message: "Content-Type must be application/json for requests with body",
						Field:   "content-type",
						Value:   contentType,
					}
				}
			}

			return next(ctx, request)
		}
	}
}

// JSONParsingMiddleware parses JSON request bodies.
func (h *Handler) JSONParsingMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (interface{}, error) {
			// Add parsed body to context if it's JSON
			if request.Body != "" {
				var parsedBody interface{}
				if err := json.Unmarshal([]byte(request.Body), &parsedBody); err != nil {
					return nil, &ValidationError{
						Message: "Invalid JSON in request body",
						Field:   "body",
						Err:     err,
					}
				}

				// Add parsed body to context
				ctx = context.WithValue(ctx, contextKeyParsedBody, parsedBody)
			}

			return next(ctx, request)
		}
	}
}

// GetParsedBody retrieves the parsed JSON body from context.
func GetParsedBody(ctx context.Context) (interface{}, bool) {
	body := ctx.Value(contextKeyParsedBody)
	return body, body != nil
}

// GetRequestID retrieves the request ID from Lambda context.
func GetRequestID(ctx context.Context) string {
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		return lc.AwsRequestID
	}
	return ""
}

// GetLambdaContext retrieves the Lambda context from the Go context.
func GetLambdaContext(ctx context.Context) (*lambdacontext.LambdaContext, bool) {
	return lambdacontext.FromContext(ctx)
}

// CreateContext creates a new context with common values.
func CreateContext(baseCtx context.Context, requestID string) context.Context {
	ctx := context.WithValue(baseCtx, contextKeyRequestID, requestID)
	ctx = context.WithValue(ctx, contextKeyTimestamp, time.Now())
	return ctx
}
