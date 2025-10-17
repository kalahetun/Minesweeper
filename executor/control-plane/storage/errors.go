package storage

import "errors"

// ErrNotFound is returned when a requested resource is not found.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned when trying to create a resource that already exists.
var ErrAlreadyExists = errors.New("resource already exists")

// ErrInvalidInput is returned when input data is invalid.
var ErrInvalidInput = errors.New("invalid input")
