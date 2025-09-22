package lambda

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		field         string
		value         string
		expectedCode  int
		expectedType  string
	}{
		{
			name:         "basic validation error",
			message:      "field is required",
			field:        "username",
			value:        "",
			expectedCode: 400,
			expectedType: "ValidationError",
		},
		{
			name:         "validation error with special characters",
			message:      "field must be alphanumeric",
			field:        "user_id",
			value:        "user@123",
			expectedCode: 400,
			expectedType: "ValidationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.message, tt.field, tt.value)

			assert.Equal(t, tt.expectedCode, err.StatusCode)
			assert.Equal(t, tt.expectedType, err.Type)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.value, err.Value)
			assert.NotEmpty(t, err.Timestamp)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("test message", "testField", "testValue")
	expected := "validation error for field 'testField': test message"
	assert.Equal(t, expected, err.Error())
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
			err:      NewNotFoundError("resource", "123"),
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
	tests := []struct {
		name         string
		resource     string
		identifier   string
		expectedCode int
		expectedType string
	}{
		{
			name:         "user not found",
			resource:     "user",
			identifier:   "123",
			expectedCode: 404,
			expectedType: "NotFoundError",
		},
		{
			name:         "product not found",
			resource:     "product",
			identifier:   "abc-def",
			expectedCode: 404,
			expectedType: "NotFoundError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNotFoundError(tt.resource, tt.identifier)

			assert.Equal(t, tt.expectedCode, err.StatusCode)
			assert.Equal(t, tt.expectedType, err.Type)
			assert.Contains(t, err.Message, tt.resource)
			assert.Contains(t, err.Message, tt.identifier)
			assert.Equal(t, tt.resource, err.Resource)
			assert.Equal(t, tt.identifier, err.Identifier)
			assert.NotEmpty(t, err.Timestamp)
		})
	}
}

func TestNotFoundError_Error(t *testing.T) {
	err := NewNotFoundError("user", "123")
	expected := "user with identifier '123' not found"
	assert.Equal(t, expected, err.Error())
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not found error",
			err:      NewNotFoundError("user", "123"),
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
	operation := "database query"
	err := NewTimeoutError(operation)

	assert.Equal(t, 408, err.StatusCode)
	assert.Equal(t, "TimeoutError", err.Type)
	assert.Contains(t, err.Message, operation)
	assert.Contains(t, err.Message, "timeout")
	assert.Equal(t, operation, err.Operation)
	assert.NotEmpty(t, err.Timestamp)
}

func TestTimeoutError_Error(t *testing.T) {
	err := NewTimeoutError("test operation")
	expected := "operation 'test operation' timed out"
	assert.Equal(t, expected, err.Error())
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      NewTimeoutError("operation"),
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
	err := NewInternalError(message)

	assert.Equal(t, 500, err.StatusCode)
	assert.Equal(t, "InternalError", err.Type)
	assert.Equal(t, message, err.Message)
	assert.NotEmpty(t, err.Timestamp)
}

func TestNewInternalErrorWithOperation(t *testing.T) {
	operation := "user creation"
	message := "failed to save user"
	cause := assert.AnError

	err := NewInternalErrorWithOperation(operation, message, cause)

	assert.Equal(t, 500, err.StatusCode)
	assert.Equal(t, "InternalError", err.Type)
	assert.Contains(t, err.Message, operation)
	assert.Contains(t, err.Message, message)
	assert.Equal(t, operation, err.Operation)
	assert.Equal(t, cause, err.Cause)
	assert.NotEmpty(t, err.Timestamp)
}

func TestInternalError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *InternalError
		expected string
	}{
		{
			name: "simple internal error",
			err: &InternalError{
				LambdaError: LambdaError{
					Message: "simple error",
				},
			},
			expected: "internal error: simple error",
		},
		{
			name: "internal error with operation",
			err: &InternalError{
				LambdaError: LambdaError{
					Message: "operation failed",
				},
				Operation: "user creation",
			},
			expected: "internal error in operation 'user creation': operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInternalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "internal error",
			err:      NewInternalError("test"),
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

func TestLambdaError_GetStatusCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "validation error",
			err:          NewValidationError("test", "field", "value"),
			expectedCode: 400,
		},
		{
			name:         "not found error",
			err:          NewNotFoundError("user", "123"),
			expectedCode: 404,
		},
		{
			name:         "timeout error",
			err:          NewTimeoutError("operation"),
			expectedCode: 408,
		},
		{
			name:         "internal error",
			err:          NewInternalError("test"),
			expectedCode: 500,
		},
		{
			name:         "generic error",
			err:          assert.AnError,
			expectedCode: 500,
		},
		{
			name:         "nil error",
			err:          nil,
			expectedCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetStatusCode(tt.err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestErrorSerialization(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "validation error",
			err:  NewValidationError("test message", "field", "value"),
		},
		{
			name: "not found error",
			err:  NewNotFoundError("user", "123"),
		},
		{
			name: "timeout error",
			err:  NewTimeoutError("operation"),
		},
		{
			name: "internal error",
			err:  NewInternalError("test message"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that error implements error interface
			assert.Implements(t, (*error)(nil), tt.err)

			// Test that Error() method returns non-empty string
			errorString := tt.err.Error()
			assert.NotEmpty(t, errorString)

			// Test that error message is meaningful
			assert.Contains(t, errorString, "error")
		})
	}
}

func TestErrorTimestamps(t *testing.T) {
	tests := []struct {
		name    string
		errFunc func() error
	}{
		{
			name:    "validation error",
			errFunc: func() error { return NewValidationError("test", "field", "value") },
		},
		{
			name:    "not found error",
			errFunc: func() error { return NewNotFoundError("user", "123") },
		},
		{
			name:    "timeout error",
			errFunc: func() error { return NewTimeoutError("operation") },
		},
		{
			name:    "internal error",
			errFunc: func() error { return NewInternalError("test") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errFunc()

			// All error types should have timestamps
			switch e := err.(type) {
			case *ValidationError:
				assert.NotEmpty(t, e.Timestamp)
			case *NotFoundError:
				assert.NotEmpty(t, e.Timestamp)
			case *TimeoutError:
				assert.NotEmpty(t, e.Timestamp)
			case *InternalError:
				assert.NotEmpty(t, e.Timestamp)
			default:
				t.Errorf("Unexpected error type: %T", err)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	originalErr := assert.AnError
	internalErr := NewInternalErrorWithOperation("test operation", "test message", originalErr)

	assert.Equal(t, originalErr, internalErr.Cause)
	assert.Contains(t, internalErr.Error(), "test operation")
	assert.Contains(t, internalErr.Error(), "test message")
}

func TestConcurrentErrorCreation(t *testing.T) {
	// Test that error creation is safe for concurrent use
	const numGoroutines = 100
	errors := make([]error, numGoroutines)
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			errors[index] = NewValidationError("concurrent test", "field", "value")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all errors were created successfully
	for i, err := range errors {
		assert.NotNil(t, err, "Error %d should not be nil", i)
		assert.IsType(t, &ValidationError{}, err, "Error %d should be ValidationError", i)
	}
}
