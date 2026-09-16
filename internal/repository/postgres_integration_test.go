//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
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
