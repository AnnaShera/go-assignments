package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// pgRepository implements Repository using Postgres.
type pgRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a new Postgres repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

// ListProjects returns all projects.
func (r *pgRepository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, created_at FROM projects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []domain.Project{}
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}

// GetProjectByID returns a project by ID.
func (r *pgRepository) GetProjectByID(ctx context.Context, id int64) (domain.Project, error) {
	var p domain.Project
	err := r.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM projects WHERE id = $1", id).
		Scan(&p.ID, &p.Name, &p.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("get project: %w", err)
	}

	return p, nil
}

// CreateProject inserts a new project and returns it with ID set.
func (r *pgRepository) CreateProject(ctx context.Context, name string) (domain.Project, error) {
	p := domain.Project{Name: name}
	if err := p.Validate(); err != nil {
		return domain.Project{}, err
	}

	err := r.db.QueryRowContext(ctx,
		"INSERT INTO projects (name) VALUES ($1) RETURNING id, created_at",
		name).Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}

	return p, nil
}

// DeleteProject removes a project by ID.
func (r *pgRepository) DeleteProject(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ListScansByProject returns all scans for a project.
func (r *pgRepository) ListScansByProject(ctx context.Context, projectID int64) ([]domain.Scan, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, project_id, tool, started_at FROM scans WHERE project_id = $1 ORDER BY id",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()

	scans := []domain.Scan{}
	for rows.Next() {
		var s domain.Scan
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Tool, &s.StartedAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		scans = append(scans, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scans: %w", err)
	}

	return scans, nil
}

// GetScanByID returns a scan by ID.
func (r *pgRepository) GetScanByID(ctx context.Context, id int64) (domain.Scan, error) {
	var s domain.Scan
	err := r.db.QueryRowContext(ctx,
		"SELECT id, project_id, tool, started_at FROM scans WHERE id = $1",
		id).Scan(&s.ID, &s.ProjectID, &s.Tool, &s.StartedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Scan{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Scan{}, fmt.Errorf("get scan: %w", err)
	}

	return s, nil
}

// CreateScan inserts a new scan.
func (r *pgRepository) CreateScan(ctx context.Context, projectID int64, tool string) (domain.Scan, error) {
	s := domain.Scan{ProjectID: projectID, Tool: tool}
	if err := s.Validate(); err != nil {
		return domain.Scan{}, err
	}

	err := r.db.QueryRowContext(ctx,
		"INSERT INTO scans (project_id, tool) VALUES ($1, $2) RETURNING id, started_at",
		projectID, tool).Scan(&s.ID, &s.StartedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Scan{}, fmt.Errorf("project %d: %w", projectID, domain.ErrNotFound)
		}
		return domain.Scan{}, fmt.Errorf("create scan: %w", err)
	}

	return s, nil
}

// DeleteScan removes a scan by ID. Findings under it are removed by the
// database via ON DELETE CASCADE.
func (r *pgRepository) DeleteScan(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM scans WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete scan: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ListFindingsByScan returns all findings for a scan.
func (r *pgRepository) ListFindingsByScan(ctx context.Context, scanID int64) ([]domain.Finding, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, scan_id, title, severity, status, file_path, line_number, created_at FROM findings WHERE scan_id = $1 ORDER BY id",
		scanID)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	findings := []domain.Finding{}
	for rows.Next() {
		var f domain.Finding
		if err := rows.Scan(&f.ID, &f.ScanID, &f.Title, &f.Severity, &f.Status, &f.FilePath, &f.LineNumber, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		findings = append(findings, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}

	return findings, nil
}

// GetFindingByID returns a finding by ID.
func (r *pgRepository) GetFindingByID(ctx context.Context, id int64) (domain.Finding, error) {
	var f domain.Finding
	err := r.db.QueryRowContext(ctx,
		"SELECT id, scan_id, title, severity, status, file_path, line_number, created_at FROM findings WHERE id = $1",
		id).Scan(&f.ID, &f.ScanID, &f.Title, &f.Severity, &f.Status, &f.FilePath, &f.LineNumber, &f.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Finding{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Finding{}, fmt.Errorf("get finding: %w", err)
	}

	return f, nil
}

// CreateFinding inserts a new finding.
func (r *pgRepository) CreateFinding(ctx context.Context, finding domain.Finding) (domain.Finding, error) {
	if err := finding.Validate(); err != nil {
		return domain.Finding{}, err
	}

	err := r.db.QueryRowContext(ctx,
		"INSERT INTO findings (scan_id, title, severity, status, file_path, line_number) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at",
		finding.ScanID, finding.Title, finding.Severity, finding.Status, finding.FilePath, finding.LineNumber).
		Scan(&finding.ID, &finding.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Finding{}, fmt.Errorf("scan %d: %w", finding.ScanID, domain.ErrNotFound)
		}
		return domain.Finding{}, fmt.Errorf("create finding: %w", err)
	}

	return finding, nil
}

// UpdateFindingStatus updates a finding's status.
func (r *pgRepository) UpdateFindingStatus(ctx context.Context, id int64, status string) error {
	if !domain.IsValidStatus(status) {
		return domain.ErrInvalidStatus
	}

	result, err := r.db.ExecContext(ctx, "UPDATE findings SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("update finding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// DeleteFinding removes a finding by ID.
func (r *pgRepository) DeleteFinding(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM findings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete finding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Close closes the database connection.
func (r *pgRepository) Close() error {
	return r.db.Close()
}
