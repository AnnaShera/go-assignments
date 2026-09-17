//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Connect to the running Postgres instance from docker-compose
	db, err := sql.Open("pgx", "user=vfa password=devpassword dbname=vuln_findings host=127.0.0.1 port=5432 sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Clean up data before test
	if _, err := db.ExecContext(context.Background(), "DELETE FROM findings"); err != nil {
		t.Fatalf("failed to clear findings: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM scans"); err != nil {
		t.Fatalf("failed to clear scans: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM projects"); err != nil {
		t.Fatalf("failed to clear projects: %v", err)
	}

	return db
}

func TestNewPostgresRepository_ListProjects_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	projects, err := repo.ListProjects(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestNewPostgresRepository_CreateProject_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	project, err := repo.CreateProject(context.Background(), "Test Project")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if project.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got '%s'", project.Name)
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestNewPostgresRepository_GetProjectByID_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	created, _ := repo.CreateProject(context.Background(), "Get Test")

	project, err := repo.GetProjectByID(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, project.ID)
	}
	if project.Name != "Get Test" {
		t.Errorf("expected name 'Get Test', got '%s'", project.Name)
	}
}

func TestNewPostgresRepository_DeleteProject_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	created, _ := repo.CreateProject(context.Background(), "Delete Test")

	err := repo.DeleteProject(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's deleted
	_, err = repo.GetProjectByID(context.Background(), created.ID)
	if err.Error() != "not found" {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestNewPostgresRepository_CreateScan_ProjectNotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	_, err := repo.CreateScan(context.Background(), 999999, "nmap")

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNewPostgresRepository_CreateFinding_ScanNotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	finding := domain.Finding{
		ScanID:     999999,
		Title:      "SQL Injection",
		Severity:   domain.SeverityHigh,
		Status:     domain.StatusOpen,
		FilePath:   "internal/handler/user.go",
		LineNumber: 42,
	}

	_, err := repo.CreateFinding(context.Background(), finding)

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNewPostgresRepository_DeleteProject_CascadesToScansAndFindings_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	project, err := repo.CreateProject(context.Background(), "Cascade Test")
	if err != nil {
		t.Fatalf("setup: create project: %v", err)
	}
	scan, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("setup: create scan: %v", err)
	}
	finding, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID:     scan.ID,
		Title:      "SQL Injection",
		Severity:   domain.SeverityHigh,
		Status:     domain.StatusOpen,
		FilePath:   "internal/handler/user.go",
		LineNumber: 42,
	})
	if err != nil {
		t.Fatalf("setup: create finding: %v", err)
	}

	if err := repo.DeleteProject(context.Background(), project.ID); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	if _, err := repo.GetScanByID(context.Background(), scan.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected scan to be cascade-deleted, got %v", err)
	}
	if _, err := repo.GetFindingByID(context.Background(), finding.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected finding to be cascade-deleted, got %v", err)
	}
}

func TestNewPostgresRepository_DeleteScan_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	project, err := repo.CreateProject(context.Background(), "Delete Scan Test")
	if err != nil {
		t.Fatalf("setup: create project: %v", err)
	}
	scan, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("setup: create scan: %v", err)
	}

	if err := repo.DeleteScan(context.Background(), scan.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := repo.GetScanByID(context.Background(), scan.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestNewPostgresRepository_DeleteScan_NotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	err := repo.DeleteScan(context.Background(), 999999)

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNewPostgresRepository_DeleteFinding_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	project, err := repo.CreateProject(context.Background(), "Delete Finding Test")
	if err != nil {
		t.Fatalf("setup: create project: %v", err)
	}
	scan, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("setup: create scan: %v", err)
	}
	finding, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID:     scan.ID,
		Title:      "SQL Injection",
		Severity:   domain.SeverityHigh,
		Status:     domain.StatusOpen,
		FilePath:   "internal/handler/user.go",
		LineNumber: 42,
	})
	if err != nil {
		t.Fatalf("setup: create finding: %v", err)
	}

	if err := repo.DeleteFinding(context.Background(), finding.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := repo.GetFindingByID(context.Background(), finding.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// TestNewPostgresRepository_ListFindingsByScan_Filtering_Integration confirms
// the SQL-level severity/status filtering (parameterized WHERE clauses in
// pgRepository.ListFindingsByScan) works against a real Postgres instance,
// not just the in-memory fake.
func TestNewPostgresRepository_ListFindingsByScan_Filtering_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	}()

	repo := NewPostgresRepository(db)
	project, err := repo.CreateProject(context.Background(), "Filter Test")
	if err != nil {
		t.Fatalf("setup: create project: %v", err)
	}
	scan, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("setup: create scan: %v", err)
	}
	seed := []domain.Finding{
		{ScanID: scan.ID, Title: "A", Severity: domain.SeverityHigh, Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 1},
		{ScanID: scan.ID, Title: "B", Severity: domain.SeverityHigh, Status: domain.StatusResolved, FilePath: "b.go", LineNumber: 2},
		{ScanID: scan.ID, Title: "C", Severity: domain.SeverityLow, Status: domain.StatusOpen, FilePath: "c.go", LineNumber: 3},
	}
	for _, fd := range seed {
		if _, err := repo.CreateFinding(context.Background(), fd); err != nil {
			t.Fatalf("setup: create finding: %v", err)
		}
	}

	tests := []struct {
		name      string
		filter    FindingFilter
		wantCount int
	}{
		{name: "no filter returns all", filter: FindingFilter{}, wantCount: 3},
		{name: "severity only", filter: FindingFilter{Severity: domain.SeverityHigh}, wantCount: 2},
		{name: "status only", filter: FindingFilter{Status: domain.StatusOpen}, wantCount: 2},
		{name: "severity and status", filter: FindingFilter{Severity: domain.SeverityHigh, Status: domain.StatusOpen}, wantCount: 1},
		{name: "matches nothing", filter: FindingFilter{Severity: domain.SeverityCritical}, wantCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := repo.ListFindingsByScan(context.Background(), scan.ID, tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if findings == nil {
				t.Error("expected non-nil slice even when empty")
			}
			if len(findings) != tt.wantCount {
				t.Errorf("expected %d findings, got %d", tt.wantCount, len(findings))
			}
		})
	}
}

func TestNewPostgresRepository_DeleteFinding_NotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	err := repo.DeleteFinding(context.Background(), 999999)

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
