package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// ListProjects returns a page of projects, ordered by id, per p (see
// Pagination.Normalize for how defaults and the max page size are
// applied).
func (r *pgRepository) ListProjects(ctx context.Context, p Pagination) (_ []domain.Project, err error) {
	p = p.Normalize()
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, created_at FROM projects ORDER BY id LIMIT $1 OFFSET $2",
		p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close rows: %w", cerr)
		}
	}()

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

// CreateProject inserts a new project and returns it with ID set. name is
// trimmed of surrounding whitespace before validation and persistence, so
// "  Foo  " and "Foo" are stored identically rather than as distinct values.
func (r *pgRepository) CreateProject(ctx context.Context, name string) (domain.Project, error) {
	p := domain.Project{Name: strings.TrimSpace(name)}
	if err := p.Validate(); err != nil {
		return domain.Project{}, err
	}

	err := r.db.QueryRowContext(ctx,
		"INSERT INTO projects (name) VALUES ($1) RETURNING id, created_at",
		p.Name).Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}

	return p, nil
}

// DeleteProject removes a project by ID.
func (r *pgRepository) DeleteProject(ctx context.Context, id int64) error {
	return r.deleteByID(ctx, "DELETE FROM projects WHERE id = $1", "delete project", id)
}

// deleteByID runs a single-row delete statement and maps "no rows affected"
// to ErrNotFound. query must be a static, fully-formed statement with the ID
// as its only placeholder ($1); opLabel names the operation for error
// wrapping (e.g. "delete project").
func (r *pgRepository) deleteByID(ctx context.Context, query, opLabel string, id int64) error {
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", opLabel, err)
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

// ListScansByProject returns a page of scans for a project, ordered by id,
// per p (see Pagination.Normalize).
func (r *pgRepository) ListScansByProject(ctx context.Context, projectID int64, p Pagination) (_ []domain.Scan, err error) {
	p = p.Normalize()
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, project_id, tool, started_at FROM scans WHERE project_id = $1 ORDER BY id LIMIT $2 OFFSET $3",
		projectID, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close rows: %w", cerr)
		}
	}()

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

// CreateScan inserts a new scan. tool is trimmed of surrounding whitespace
// before validation and persistence (see CreateProject).
func (r *pgRepository) CreateScan(ctx context.Context, projectID int64, tool string) (domain.Scan, error) {
	s := domain.Scan{ProjectID: projectID, Tool: strings.TrimSpace(tool)}
	if err := s.Validate(); err != nil {
		return domain.Scan{}, err
	}

	err := r.db.QueryRowContext(ctx,
		"INSERT INTO scans (project_id, tool) VALUES ($1, $2) RETURNING id, started_at",
		projectID, s.Tool).Scan(&s.ID, &s.StartedAt)

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
	return r.deleteByID(ctx, "DELETE FROM scans WHERE id = $1", "delete scan", id)
}

// ListFindingsByScan returns a page of findings for a scan, ordered by id,
// optionally narrowed by filter.Severity and/or filter.Status and paged
// per p (see Pagination.Normalize). Each filter clause is appended only
// when its field is non-empty; every bound value, filters and
// limit/offset alike, travels as a parameter ($2, $3, ...), never
// string-concatenated into the query text.
func (r *pgRepository) ListFindingsByScan(ctx context.Context, scanID int64, filter FindingFilter, p Pagination) (_ []domain.Finding, err error) {
	p = p.Normalize()
	query := "SELECT id, scan_id, title, severity, status, file_path, line_number, created_at FROM findings WHERE scan_id = $1"
	args := []any{scanID}

	if filter.Severity != "" {
		args = append(args, filter.Severity)
		query += fmt.Sprintf(" AND severity = $%d", len(args))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	query += " ORDER BY id"

	args = append(args, p.Limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))
	args = append(args, p.Offset)
	query += fmt.Sprintf(" OFFSET $%d", len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close rows: %w", cerr)
		}
	}()

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

// CreateFinding inserts a new finding. Title and FilePath are trimmed of
// surrounding whitespace before validation and persistence (see
// CreateProject).
func (r *pgRepository) CreateFinding(ctx context.Context, finding domain.Finding) (domain.Finding, error) {
	finding.Title = strings.TrimSpace(finding.Title)
	finding.FilePath = strings.TrimSpace(finding.FilePath)
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
	return r.deleteByID(ctx, "DELETE FROM findings WHERE id = $1", "delete finding", id)
}

// Close closes the database connection.
func (r *pgRepository) Close() error {
	return r.db.Close()
}
