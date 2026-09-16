package domain

import (
	"strings"
	"time"
)

// Finding represents a security vulnerability finding.
type Finding struct {
	ID         int64
	ScanID     int64
	Title      string
	Severity   string // low, medium, high, critical
	Status     string // open, confirmed, false_positive, resolved
	FilePath   string
	LineNumber int
	CreatedAt  time.Time
}

// Validate checks that a finding has required fields and valid values.
func (f Finding) Validate() error {
	if f.ScanID == 0 {
		return ErrScanIDRequired
	}
	if strings.TrimSpace(f.Title) == "" {
		return ErrTitleRequired
	}
	if f.Severity == "" {
		return ErrInvalidSeverity
	}
	if !IsValidSeverity(f.Severity) {
		return ErrInvalidSeverity
	}
	if f.Status == "" {
		return ErrInvalidStatus
	}
	if !IsValidStatus(f.Status) {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(f.FilePath) == "" {
		return ErrFilePathRequired
	}
	if f.LineNumber <= 0 {
		return ErrInvalidLineNumber
	}
	return nil
}

// IsValidSeverity checks if a severity string is valid.
func IsValidSeverity(s string) bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusConfirmed, StatusFalsePositive, StatusResolved:
		return true
	}
	return false
}
