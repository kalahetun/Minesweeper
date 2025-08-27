package service

import (
	"errors"
	"fmt"
)

// Business logic errors
var (
	ErrInvalidPolicy = errors.New("invalid policy")
	ErrPolicyExists  = errors.New("policy already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// Validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// DetailedError provides detailed error information with context
type DetailedError struct {
	Type    string
	Message string
	Details map[string]interface{}
}

func (e *DetailedError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewDetailedError creates a new detailed error
func NewDetailedError(errorType, message string, details map[string]interface{}) *DetailedError {
	return &DetailedError{
		Type:    errorType,
		Message: message,
		Details: details,
	}
}
