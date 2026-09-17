package repository

import (
	"context"
	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// Default and cap for Pagination.Normalize: a page defaults to 50 rows
// when the caller doesn't specify a limit, and is clamped to at most 200
// rows even if the caller asks for more, so a single request can't be
// used to pull an unbounded result set.
const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

// Pagination narrows a List* call to a page of results via Limit and
// Offset. It is deliberately its own type rather than folded into
// FindingFilter: paging is an orthogonal concern from content filtering,
// and every List* method needs it, not just ListFindingsByScan.
type Pagination struct {
	Limit  int
	Offset int
}

// Normalize returns p with sensible bounds applied: a non-positive Limit
// (including the zero value, so "no pagination given" just works) defaults
// to defaultPageLimit, and a Limit above maxPageLimit is clamped down to
// it rather than rejected — a common, low-friction API convention: an
// over-eager client gets a smaller page instead of an error. Offset is
// passed through unchanged; a negative offset is a client mistake the
// handler layer rejects outright (see handlers.parsePagination) rather
// than something to silently correct here.
func (p Pagination) Normalize() Pagination {
	switch {
	case p.Limit <= 0:
		p.Limit = defaultPageLimit
	case p.Limit > maxPageLimit:
		p.Limit = maxPageLimit
	}
	return p
}

// ProjectRepository defines database operations for projects.
type ProjectRepository interface {
	ListProjects(ctx context.Context, p Pagination) ([]domain.Project, error)
	GetProjectByID(ctx context.Context, id int64) (domain.Project, error)
	CreateProject(ctx context.Context, name string) (domain.Project, error)
	DeleteProject(ctx context.Context, id int64) error
}

// ScanRepository defines database operations for scans.
type ScanRepository interface {
	ListScansByProject(ctx context.Context, projectID int64, p Pagination) ([]domain.Scan, error)
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
	ListFindingsByScan(ctx context.Context, scanID int64, filter FindingFilter, p Pagination) ([]domain.Finding, error)
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
