package domain

import "errors"

// Domain validation errors.
var (
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectIDRequired   = errors.New("project ID is required")
	ErrScanIDRequired      = errors.New("scan ID is required")
	ErrNotFound            = errors.New("not found")
	ErrInvalidSeverity     = errors.New("invalid severity")
	ErrInvalidStatus       = errors.New("invalid status")
	ErrToolRequired        = errors.New("tool is required")
	ErrTitleRequired       = errors.New("title is required")
	ErrFilePathRequired    = errors.New("file path is required")
	ErrInvalidLineNumber   = errors.New("line number must be greater than 0")
)

// Severity levels for findings.
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Status values for findings.
const (
	StatusOpen          = "open"
	StatusConfirmed     = "confirmed"
	StatusFalsePositive = "false_positive"
	StatusResolved      = "resolved"
)
