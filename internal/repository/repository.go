package repository

import (
	"context"
	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// ProjectRepository defines database operations for projects.
type ProjectRepository interface {
	ListProjects(ctx context.Context) ([]domain.Project, error)
	GetProjectByID(ctx context.Context, id int64) (domain.Project, error)
	CreateProject(ctx context.Context, name string) (domain.Project, error)
	DeleteProject(ctx context.Context, id int64) error
}

// ScanRepository defines database operations for scans.
type ScanRepository interface {
	ListScansByProject(ctx context.Context, projectID int64) ([]domain.Scan, error)
	GetScanByID(ctx context.Context, id int64) (domain.Scan, error)
	CreateScan(ctx context.Context, projectID int64, tool string) (domain.Scan, error)
	DeleteScan(ctx context.Context, id int64) error
}

// FindingFilter narrows a ListFindingsByScan call to findings matching the
// given severity and/or status. An empty field means no filter on that
// dimension; a zero-value FindingFilter matches everything, preserving the
// unfiltered behavior callers relied on before filtering existed.
type FindingFilter struct {
	Severity string
	Status   string
}

// FindingRepository defines database operations for findings.
type FindingRepository interface {
	ListFindingsByScan(ctx context.Context, scanID int64, filter FindingFilter) ([]domain.Finding, error)
	GetFindingByID(ctx context.Context, id int64) (domain.Finding, error)
	CreateFinding(ctx context.Context, finding domain.Finding) (domain.Finding, error)
	UpdateFindingStatus(ctx context.Context, id int64, status string) error
	DeleteFinding(ctx context.Context, id int64) error
}

// Repository groups all repository interfaces.
type Repository interface {
	ProjectRepository
	ScanRepository
	FindingRepository
	Close() error
}
