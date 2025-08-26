package service

import "errors"

// Business logic errors
var (
	ErrInvalidPolicy = errors.New("invalid policy")
	ErrPolicyExists  = errors.New("policy already exists")
)

// Validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
