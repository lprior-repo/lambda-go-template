// Package lambda provides common utilities and middleware for AWS Lambda functions.
package lambda

import (
	"fmt"
	"time"
)

// ValidationError represents a request validation error.
type ValidationError struct {
	Message string
	Field   string
	Value   interface{}
	Err     error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error.
func NewValidationError(message, field string, value interface{}) *ValidationError {
	return &ValidationError{
		Message: message,
		Field:   field,
		Value:   value,
	}
}

// NewValidationErrorWithCause creates a new validation error with an underlying cause.
func NewValidationErrorWithCause(message, field string, value interface{}, err error) *ValidationError {
	return &ValidationError{
		Message: message,
		Field:   field,
		Value:   value,
		Err:     err,
	}
}

// NotFoundError represents a resource not found error.
type NotFoundError struct {
	Message  string
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.Resource != "" && e.ID != "" {
		return fmt.Sprintf("%s not found: %s with ID '%s'", e.Resource, e.Message, e.ID)
	}
	if e.Resource != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.Message)
	}
	return fmt.Sprintf("not found: %s", e.Message)
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		Message: message,
	}
}

// NewResourceNotFoundError creates a new resource not found error.
func NewResourceNotFoundError(resource, id, message string) *NotFoundError {
	return &NotFoundError{
		Message:  message,
		Resource: resource,
		ID:       id,
	}
}

// ConflictError represents a resource conflict error.
type ConflictError struct {
	Message  string
	Resource string
	Err      error
}

func (e *ConflictError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("conflict with %s: %s", e.Resource, e.Message)
	}
	return fmt.Sprintf("conflict: %s", e.Message)
}

func (e *ConflictError) Unwrap() error {
	return e.Err
}

// NewConflictError creates a new conflict error.
func NewConflictError(message string) *ConflictError {
	return &ConflictError{
		Message: message,
	}
}

// NewResourceConflictError creates a new resource conflict error.
func NewResourceConflictError(resource, message string, err error) *ConflictError {
	return &ConflictError{
		Message:  message,
		Resource: resource,
		Err:      err,
	}
}

// UnauthorizedError represents an authentication error.
type UnauthorizedError struct {
	Message string
	Reason  string
}

func (e *UnauthorizedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("unauthorized: %s (%s)", e.Message, e.Reason)
	}
	return fmt.Sprintf("unauthorized: %s", e.Message)
}

// NewUnauthorizedError creates a new unauthorized error.
func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{
		Message: message,
	}
}

// NewUnauthorizedErrorWithReason creates a new unauthorized error with reason.
func NewUnauthorizedErrorWithReason(message, reason string) *UnauthorizedError {
	return &UnauthorizedError{
		Message: message,
		Reason:  reason,
	}
}

// ForbiddenError represents an authorization error.
type ForbiddenError struct {
	Message   string
	Resource  string
	Operation string
}

func (e *ForbiddenError) Error() string {
	if e.Resource != "" && e.Operation != "" {
		return fmt.Sprintf("forbidden: cannot %s %s - %s", e.Operation, e.Resource, e.Message)
	}
	if e.Resource != "" {
		return fmt.Sprintf("forbidden: access to %s denied - %s", e.Resource, e.Message)
	}
	return fmt.Sprintf("forbidden: %s", e.Message)
}

// NewForbiddenError creates a new forbidden error.
func NewForbiddenError(message string) *ForbiddenError {
	return &ForbiddenError{
		Message: message,
	}
}

// NewResourceForbiddenError creates a new resource forbidden error.
func NewResourceForbiddenError(resource, operation, message string) *ForbiddenError {
	return &ForbiddenError{
		Message:   message,
		Resource:  resource,
		Operation: operation,
	}
}

// TimeoutError represents a request timeout error.
type TimeoutError struct {
	Message string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	if e.Timeout > 0 {
		return fmt.Sprintf("timeout: %s (after %v)", e.Message, e.Timeout)
	}
	return fmt.Sprintf("timeout: %s", e.Message)
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(message string, timeout time.Duration) *TimeoutError {
	return &TimeoutError{
		Message: message,
		Timeout: timeout,
	}
}

// InternalError represents an internal system error.
type InternalError struct {
	Message   string
	Operation string
	Err       error
}

func (e *InternalError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("internal error during %s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("internal error: %s", e.Message)
}

func (e *InternalError) Unwrap() error {
	return e.Err
}

// NewInternalError creates a new internal error.
func NewInternalError(message string, err error) *InternalError {
	return &InternalError{
		Message: message,
		Err:     err,
	}
}

// NewInternalErrorWithOperation creates a new internal error with operation context.
func NewInternalErrorWithOperation(operation, message string, err error) *InternalError {
	return &InternalError{
		Message:   message,
		Operation: operation,
		Err:       err,
	}
}

// BusinessLogicError represents a business logic validation error.
type BusinessLogicError struct {
	Message string
	Code    string
	Details map[string]interface{}
}

func (e *BusinessLogicError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("business logic error [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("business logic error: %s", e.Message)
}

// NewBusinessLogicError creates a new business logic error.
func NewBusinessLogicError(message, code string) *BusinessLogicError {
	return &BusinessLogicError{
		Message: message,
		Code:    code,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the business logic error.
func (e *BusinessLogicError) WithDetail(key string, value interface{}) *BusinessLogicError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// ExternalServiceError represents an error from an external service.
type ExternalServiceError struct {
	Message     string
	Service     string
	StatusCode  int
	Retryable   bool
	Err         error
}

func (e *ExternalServiceError) Error() string {
	if e.Service != "" && e.StatusCode > 0 {
		return fmt.Sprintf("external service error [%s:%d]: %s", e.Service, e.StatusCode, e.Message)
	}
	if e.Service != "" {
		return fmt.Sprintf("external service error [%s]: %s", e.Service, e.Message)
	}
	return fmt.Sprintf("external service error: %s", e.Message)
}

func (e *ExternalServiceError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable.
func (e *ExternalServiceError) IsRetryable() bool {
	return e.Retryable
}

// NewExternalServiceError creates a new external service error.
func NewExternalServiceError(service, message string, statusCode int, retryable bool, err error) *ExternalServiceError {
	return &ExternalServiceError{
		Message:    message,
		Service:    service,
		StatusCode: statusCode,
		Retryable:  retryable,
		Err:        err,
	}
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsNotFoundError checks if an error is a not found error.
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IsConflictError checks if an error is a conflict error.
func IsConflictError(err error) bool {
	_, ok := err.(*ConflictError)
	return ok
}

// IsUnauthorizedError checks if an error is an unauthorized error.
func IsUnauthorizedError(err error) bool {
	_, ok := err.(*UnauthorizedError)
	return ok
}

// IsForbiddenError checks if an error is a forbidden error.
func IsForbiddenError(err error) bool {
	_, ok := err.(*ForbiddenError)
	return ok
}

// IsTimeoutError checks if an error is a timeout error.
func IsTimeoutError(err error) bool {
	_, ok := err.(*TimeoutError)
	return ok
}

// IsInternalError checks if an error is an internal error.
func IsInternalError(err error) bool {
	_, ok := err.(*InternalError)
	return ok
}

// IsBusinessLogicError checks if an error is a business logic error.
func IsBusinessLogicError(err error) bool {
	_, ok := err.(*BusinessLogicError)
	return ok
}

// IsExternalServiceError checks if an error is an external service error.
func IsExternalServiceError(err error) bool {
	_, ok := err.(*ExternalServiceError)
	return ok
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	if extErr, ok := err.(*ExternalServiceError); ok {
		return extErr.IsRetryable()
	}

	if IsTimeoutError(err) {
		return true
	}

	return false
}
