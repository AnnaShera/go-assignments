package domain

import (
	"strings"
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
		return ErrProjectIDRequired
	}
	if strings.TrimSpace(s.Tool) == "" {
		return ErrToolRequired
	}
	return nil
}
