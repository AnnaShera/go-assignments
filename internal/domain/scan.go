package domain

import (
	"errors"
	"time"
)

// Scan represents a security scan run on a project.
type Scan struct {
	ID        int64
	ProjectID int64
	Tool      string
	StartedAt time.Time
}

// Validate checks that a scan has required fields.
func (s Scan) Validate() error {
	if s.ProjectID == 0 {
		return errors.New("project ID is required")
	}
	if s.Tool == "" {
		return errors.New("tool is required")
	}
	return nil
}
