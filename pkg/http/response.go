// Package http provides HTTP response utilities for Lambda functions.
package http

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Response represents a standard HTTP response for API Gateway.
type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ErrorResponse represents a standard error response structure.
type ErrorResponse struct {
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Timestamp string `json:"timestamp"`
	Path      string `json:"path,omitempty"`
}

// SuccessResponse represents a standard success response wrapper.
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ResponseBuilder helps build HTTP responses with consistent structure.
type ResponseBuilder struct {
	requestID string
	path      string
	headers   map[string]string
}

// NewResponseBuilder creates a new response builder.
func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		headers: make(map[string]string),
	}
}

// WithRequestID sets the request ID for responses.
func (rb *ResponseBuilder) WithRequestID(requestID string) *ResponseBuilder {
	rb.requestID = requestID
	return rb
}

// WithPath sets the request path for error responses.
func (rb *ResponseBuilder) WithPath(path string) *ResponseBuilder {
	rb.path = path
	return rb
}

// WithHeader adds a header to the response.
func (rb *ResponseBuilder) WithHeader(key, value string) *ResponseBuilder {
	rb.headers[key] = value
	return rb
}

// WithHeaders adds multiple headers to the response.
func (rb *ResponseBuilder) WithHeaders(headers map[string]string) *ResponseBuilder {
	for key, value := range headers {
		rb.headers[key] = value
	}
	return rb
}

// WithCORS adds CORS headers to the response.
func (rb *ResponseBuilder) WithCORS() *ResponseBuilder {
	rb.headers["Access-Control-Allow-Origin"] = "*"
	rb.headers["Access-Control-Allow-Headers"] = "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token"
	rb.headers["Access-Control-Allow-Methods"] = "OPTIONS,POST,GET,PUT,DELETE"
	return rb
}

// WithCacheControl adds cache control headers.
func (rb *ResponseBuilder) WithCacheControl(maxAge int) *ResponseBuilder {
	rb.headers["Cache-Control"] = fmt.Sprintf("max-age=%d", maxAge)
	return rb
}

// OK creates a 200 OK response with the given data.
func (rb *ResponseBuilder) OK(data interface{}) Response {
	return rb.buildResponse(200, data)
}

// Created creates a 201 Created response with the given data.
func (rb *ResponseBuilder) Created(data interface{}) Response {
	return rb.buildResponse(201, data)
}

// NoContent creates a 204 No Content response.
func (rb *ResponseBuilder) NoContent() Response {
	return rb.buildResponse(204, nil)
}

// BadRequest creates a 400 Bad Request error response.
func (rb *ResponseBuilder) BadRequest(message string, err error) Response {
	return rb.buildErrorResponse(400, message, err)
}

// Unauthorized creates a 401 Unauthorized error response.
func (rb *ResponseBuilder) Unauthorized(message string) Response {
	return rb.buildErrorResponse(401, message, nil)
}

// Forbidden creates a 403 Forbidden error response.
func (rb *ResponseBuilder) Forbidden(message string) Response {
	return rb.buildErrorResponse(403, message, nil)
}

// NotFound creates a 404 Not Found error response.
func (rb *ResponseBuilder) NotFound(message string) Response {
	return rb.buildErrorResponse(404, message, nil)
}

// MethodNotAllowed creates a 405 Method Not Allowed error response.
func (rb *ResponseBuilder) MethodNotAllowed(message string) Response {
	return rb.buildErrorResponse(405, message, nil)
}

// Conflict creates a 409 Conflict error response.
func (rb *ResponseBuilder) Conflict(message string, err error) Response {
	return rb.buildErrorResponse(409, message, err)
}

// UnprocessableEntity creates a 422 Unprocessable Entity error response.
func (rb *ResponseBuilder) UnprocessableEntity(message string, err error) Response {
	return rb.buildErrorResponse(422, message, err)
}

// TooManyRequests creates a 429 Too Many Requests error response.
func (rb *ResponseBuilder) TooManyRequests(message string) Response {
	return rb.buildErrorResponse(429, message, nil)
}

// InternalServerError creates a 500 Internal Server Error response.
func (rb *ResponseBuilder) InternalServerError(message string, err error) Response {
	return rb.buildErrorResponse(500, message, err)
}

// ServiceUnavailable creates a 503 Service Unavailable error response.
func (rb *ResponseBuilder) ServiceUnavailable(message string) Response {
	return rb.buildErrorResponse(503, message, nil)
}

// Custom creates a response with a custom status code and data.
func (rb *ResponseBuilder) Custom(statusCode int, data interface{}) Response {
	return rb.buildResponse(statusCode, data)
}

// buildResponse creates a response with the given status code and data.
func (rb *ResponseBuilder) buildResponse(statusCode int, data interface{}) Response {
	headers := rb.getDefaultHeaders()

	var body string
	if data != nil {
		if statusCode >= 200 && statusCode < 300 {
			// Success response
			successResponse := SuccessResponse{
				Data:      data,
				RequestID: rb.requestID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			bodyBytes, _ := json.Marshal(successResponse)
			body = string(bodyBytes)
		} else {
			// Direct data marshaling for custom responses
			bodyBytes, _ := json.Marshal(data)
			body = string(bodyBytes)
		}
	}

	return Response{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}

// buildErrorResponse creates an error response with the given status code and message.
func (rb *ResponseBuilder) buildErrorResponse(statusCode int, message string, err error) Response {
	headers := rb.getDefaultHeaders()

	errorResponse := ErrorResponse{
		Message:   message,
		RequestID: rb.requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      rb.path,
	}

	if err != nil {
		errorResponse.Error = err.Error()
	}

	bodyBytes, _ := json.Marshal(errorResponse)

	return Response{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(bodyBytes),
	}
}

// getDefaultHeaders returns the default headers for all responses.
func (rb *ResponseBuilder) getDefaultHeaders() map[string]string {
	headers := make(map[string]string)

	// Set default headers
	headers["Content-Type"] = "application/json"
	if rb.requestID != "" {
		headers["X-Request-ID"] = rb.requestID
	}

	// Add custom headers
	for key, value := range rb.headers {
		headers[key] = value
	}

	return headers
}

// GetDefaultHeaders returns standard headers for Lambda responses.
func GetDefaultHeaders(requestID string) map[string]string {
	return map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		"Access-Control-Allow-Methods": "OPTIONS,POST,GET,PUT,DELETE",
		"X-Request-ID":                 requestID,
		"Cache-Control":                "max-age=300",
	}
}

// CreateSuccessResponse creates a standardized success response.
func CreateSuccessResponse(statusCode int, data interface{}, requestID string) Response {
	return NewResponseBuilder().
		WithRequestID(requestID).
		WithCORS().
		WithCacheControl(300).
		Custom(statusCode, data)
}

// CreateErrorResponse creates a standardized error response.
func CreateErrorResponse(statusCode int, message string, err error, requestID, path string) Response {
	return NewResponseBuilder().
		WithRequestID(requestID).
		WithPath(path).
		WithCORS().
		buildErrorResponse(statusCode, message, err)
}

// ValidateStatusCode ensures the status code is valid.
func ValidateStatusCode(statusCode int) bool {
	return statusCode >= 100 && statusCode < 600
}

// GetStatusText returns the standard text for HTTP status codes.
func GetStatusText(statusCode int) string {
	statusTexts := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		409: "Conflict",
		422: "Unprocessable Entity",
		429: "Too Many Requests",
		500: "Internal Server Error",
		503: "Service Unavailable",
	}

	if text, exists := statusTexts[statusCode]; exists {
		return text
	}

	return "Unknown Status"
}

// AddSecurityHeaders adds common security headers to the response.
func AddSecurityHeaders(headers map[string]string) {
	headers["X-Content-Type-Options"] = "nosniff"
	headers["X-Frame-Options"] = "DENY"
	headers["X-XSS-Protection"] = "1; mode=block"
	headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
}

// SetCacheControl sets appropriate cache control headers based on the response type.
func SetCacheControl(headers map[string]string, isPublic bool, maxAge int) {
	if isPublic {
		headers["Cache-Control"] = fmt.Sprintf("public, max-age=%d", maxAge)
	} else {
		headers["Cache-Control"] = fmt.Sprintf("private, max-age=%d", maxAge)
	}
}

// ParseContentLength safely parses content length from headers.
func ParseContentLength(headers map[string]string) int64 {
	if contentLength, exists := headers["Content-Length"]; exists {
		if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			return length
		}
	}
	return 0
}
