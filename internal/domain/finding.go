package domain

import (
	"errors"
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
	if f.Title == "" {
		return errors.New("title is required")
	}
	if f.Severity == "" {
		return ErrInvalidSeverity
	}
	if !isValidSeverity(f.Severity) {
		return ErrInvalidSeverity
	}
	if f.Status == "" {
		return ErrInvalidStatus
	}
	if !isValidStatus(f.Status) {
		return ErrInvalidStatus
	}
	if f.FilePath == "" {
		return errors.New("file path is required")
	}
	if f.LineNumber <= 0 {
		return errors.New("line number must be greater than 0")
	}
	return nil
}

func isValidSeverity(s string) bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}

func isValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusConfirmed, StatusFalsePositive, StatusResolved:
		return true
	}
	return false
}
