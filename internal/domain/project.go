package domain

import (
	"strings"
	"time"
)

// Project represents a security scanning project.
type Project struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Validate checks that a project has required fields.
func (p Project) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrProjectNameRequired
	}
	return nil
}
