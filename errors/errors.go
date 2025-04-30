package errors

import (
	"fmt"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Configuration errors
	ErrConfigParse   ErrorType = "CONFIG_PARSE_ERROR"
	ErrConfigInvalid ErrorType = "CONFIG_INVALID_ERROR"

	// AWS errors
	ErrAWSClient   ErrorType = "AWS_CLIENT_ERROR"
	ErrAWSInstance ErrorType = "AWS_INSTANCE_ERROR"

	// Terraform errors
	ErrTerraformState  ErrorType = "TERRAFORM_STATE_ERROR"
	ErrTerraformConfig ErrorType = "TERRAFORM_CONFIG_ERROR"

	// Drift checker errors
	ErrDriftChecker ErrorType = "DRIFT_CHECKER_ERROR"
)

// CustomError represents a custom error with additional context
type CustomError struct {
	Type       ErrorType
	Message    string
	Context    map[string]interface{}
	WrappedErr error
}

// New creates a new custom error
func New(errorType ErrorType, message string, context map[string]interface{}, wrappedErr error) *CustomError {
	return &CustomError{
		Type:       errorType,
		Message:    message,
		Context:    context,
		WrappedErr: wrappedErr,
	}
}

// Error implements the error interface
func (e *CustomError) Error() string {
	if e.WrappedErr != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.WrappedErr)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *CustomError) Unwrap() error {
	return e.WrappedErr
}

// Is checks if the error is of a specific type
func Is(err error, errType ErrorType) bool {
	if err == nil {
		return false
	}

	if customErr, ok := err.(*CustomError); ok {
		return customErr.Type == errType
	}

	return false
}
