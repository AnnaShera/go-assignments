package domain

import "errors"

// Domain validation errors.
var (
	ErrProjectNameRequired = errors.New("project name is required")
	ErrNotFound            = errors.New("not found")
	ErrInvalidSeverity     = errors.New("invalid severity")
	ErrInvalidStatus       = errors.New("invalid status")
)
