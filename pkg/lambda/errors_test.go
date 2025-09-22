package lambda

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		field    string
		value    interface{}
	}{
		{
			name:    "basic validation error",
			message: "field is required",
			field:   "username",
			value:   "",
		},
		{
			name:    "validation error with number value",
			message: "value must be positive",
			field:   "age",
			value:   -5,
		},
		{
			name:    "validation error with complex value",
			message: "invalid format",
			field:   "user_data",
			value:   map[string]string{"invalid": "data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.message, tt.field, tt.value)

			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.value, err.Value)
			assert.Nil(t, err.Err)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "validation error with field",
			err: &ValidationError{
				Message: "test message",
				Field:   "testField",
				Value:   "testValue",
			},
			expected: "validation error for field 'testField': test message",
		},
		{
			name: "validation error without field",
			err: &ValidationError{
				Message: "test message",
				Field:   "",
				Value:   "testValue",
			},
			expected: "validation error: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewValidationErrorWithCause(t *testing.T) {
	cause := assert.AnError
	err := NewValidationErrorWithCause("test message", "field", "value", cause)

	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "field", err.Field)
	assert.Equal(t, "value", err.Value)
	assert.Equal(t, cause, err.Err)
}

func TestValidationError_Unwrap(t *testing.T) {
	cause := assert.AnError
	err := NewValidationErrorWithCause("test", "field", "value", cause)

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)

	// Test with no cause
	errNoCause := NewValidationError("test", "field", "value")
	assert.Nil(t, errNoCause.Unwrap())
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      NewValidationError("test", "field", "value"),
			expected: true,
		},
		{
			name:     "not found error",
			err:      NewNotFoundError("resource not found"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	message := "resource not found"
	err := NewNotFoundError(message)

	assert.Equal(t, message, err.Message)
	assert.Empty(t, err.Resource)
	assert.Empty(t, err.ID)
}

func TestNewResourceNotFoundError(t *testing.T) {
	resource := "user"
	id := "123"
	message := "not found"

	err := NewResourceNotFoundError(resource, id, message)

	assert.Equal(t, message, err.Message)
	assert.Equal(t, resource, err.Resource)
	assert.Equal(t, id, err.ID)
}

func TestNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotFoundError
		expected string
	}{
		{
			name: "full resource not found error",
			err: &NotFoundError{
				Message:  "not found",
				Resource: "user",
				ID:       "123",
			},
			expected: "user not found: not found with ID '123'",
		},
		{
			name: "resource error without ID",
			err: &NotFoundError{
				Message:  "not found",
				Resource: "user",
				ID:       "",
			},
			expected: "user not found: not found",
		},
		{
			name: "simple not found error",
			err: &NotFoundError{
				Message:  "not found",
				Resource: "",
				ID:       "",
			},
			expected: "not found: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not found error",
			err:      NewNotFoundError("not found"),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", "field", "value"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTimeoutError(t *testing.T) {
	message := "operation timed out"
	timeout := 30 * time.Second

	err := NewTimeoutError(message, timeout)

	assert.Equal(t, message, err.Message)
	assert.Equal(t, timeout, err.Timeout)
}

func TestTimeoutError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TimeoutError
		expected string
	}{
		{
			name: "timeout with duration",
			err: &TimeoutError{
				Message: "operation timed out",
				Timeout: 30 * time.Second,
			},
			expected: "timeout: operation timed out (after 30s)",
		},
		{
			name: "timeout without duration",
			err: &TimeoutError{
				Message: "operation timed out",
				Timeout: 0,
			},
			expected: "timeout: operation timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      NewTimeoutError("timeout", 30*time.Second),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", "field", "value"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTimeoutError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewInternalError(t *testing.T) {
	message := "database connection failed"
	cause := assert.AnError

	err := NewInternalError(message, cause)

	assert.Equal(t, message, err.Message)
	assert.Equal(t, cause, err.Err)
	assert.Empty(t, err.Operation)
}

func TestNewInternalErrorWithOperation(t *testing.T) {
	operation := "user creation"
	message := "failed to save user"
	cause := assert.AnError

	err := NewInternalErrorWithOperation(operation, message, cause)

	assert.Equal(t, message, err.Message)
	assert.Equal(t, operation, err.Operation)
	assert.Equal(t, cause, err.Err)
}

func TestInternalError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *InternalError
		expected string
	}{
		{
			name: "internal error with operation",
			err: &InternalError{
				Message:   "failed to save",
				Operation: "user creation",
			},
			expected: "internal error during user creation: failed to save",
		},
		{
			name: "simple internal error",
			err: &InternalError{
				Message:   "database error",
				Operation: "",
			},
			expected: "internal error: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInternalError_Unwrap(t *testing.T) {
	cause := assert.AnError
	err := NewInternalError("test", cause)

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)

	// Test with no cause
	errNoCause := NewInternalError("test", nil)
	assert.Nil(t, errNoCause.Unwrap())
}

func TestIsInternalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "internal error",
			err:      NewInternalError("test", nil),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", "field", "value"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewConflictError(t *testing.T) {
	message := "resource already exists"
	err := NewConflictError(message)

	assert.Equal(t, message, err.Message)
	assert.Empty(t, err
