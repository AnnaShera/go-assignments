package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// FakeScanRepository is a test double for unit testing.
type FakeScanRepository struct {
	scans  map[int64]domain.Scan
	nextID int64
	// validProjectIDs simulates the projects table's FK target, since the
	// fake has no real foreign key to violate.
	validProjectIDs map[int64]bool
}

// NewFakeScanRepository creates a new fake repository for testing.
func NewFakeScanRepository() *FakeScanRepository {
	return &FakeScanRepository{
		scans:           make(map[int64]domain.Scan),
		nextID:          1,
		validProjectIDs: make(map[int64]bool),
	}
}

// AddProject registers a project ID as existing, so CreateScan's FK check
// against it succeeds. Mirrors the real projects table for FK simulation.
func (f *FakeScanRepository) AddProject(projectID int64) {
	f.validProjectIDs[projectID] = true
}

func (f *FakeScanRepository) ListScansByProject(ctx context.Context, projectID int64, p Pagination) ([]domain.Scan, error) {
	scans := []domain.Scan{}
	for _, s := range f.scans {
		if s.ProjectID == projectID {
			scans = append(scans, s)
		}
	}
	return paginateByID(scans, func(s domain.Scan) int64 { return s.ID }, p), nil
}

func (f *FakeScanRepository) GetScanByID(ctx context.Context, id int64) (domain.Scan, error) {
	s, ok := f.scans[id]
	if !ok {
		return domain.Scan{}, domain.ErrNotFound
	}
	return s, nil
}

func (f *FakeScanRepository) CreateScan(ctx context.Context, projectID int64, tool string) (domain.Scan, error) {
	s := domain.Scan{
		ID:        f.nextID,
		ProjectID: projectID,
		Tool:      strings.TrimSpace(tool),
		StartedAt: time.Now(),
	}
	if err := s.Validate(); err != nil {
		return domain.Scan{}, err
	}
	if !f.validProjectIDs[projectID] {
		return domain.Scan{}, domain.ErrNotFound
	}
	f.scans[s.ID] = s
	f.nextID++
	return s, nil
}

func (f *FakeScanRepository) DeleteScan(ctx context.Context, id int64) error {
	if _, ok := f.scans[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.scans, id)
	return nil
}

// Test cases

func TestListScansByProject_Empty(t *testing.T) {
	repo := NewFakeScanRepository()

	scans, err := repo.ListScansByProject(context.Background(), 1, Pagination{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("expected 0 scans, got %d", len(scans))
	}
}

func TestListScansByProject_ReturnsScansForProject(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)
	repo.AddProject(2)
	if _, err := repo.CreateScan(context.Background(), 1, "nmap"); err != nil {
		t.Fatalf("setup: create scan nmap: %v", err)
	}
	if _, err := repo.CreateScan(context.Background(), 1, "zap"); err != nil {
		t.Fatalf("setup: create scan zap: %v", err)
	}
	if _, err := repo.CreateScan(context.Background(), 2, "nikto"); err != nil {
		t.Fatalf("setup: create scan nikto: %v", err)
	}

	scans, err := repo.ListScansByProject(context.Background(), 1, Pagination{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scans) != 2 {
		t.Errorf("expected 2 scans for project 1, got %d", len(scans))
	}
	for _, s := range scans {
		if s.ProjectID != 1 {
			t.Errorf("expected only project 1 scans, got scan for project %d", s.ProjectID)
		}
	}
}

// TestListScansByProject_Pagination covers Pagination handling for a
// single project's scans: default page size, an explicit sub-page in id
// order, and an offset past the end returning an empty slice.
func TestListScansByProject_Pagination(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)
	var seeded []domain.Scan
	for i := 0; i < 5; i++ {
		s, err := repo.CreateScan(context.Background(), 1, fmt.Sprintf("tool-%d", i))
		if err != nil {
			t.Fatalf("setup: create scan %d: %v", i, err)
		}
		seeded = append(seeded, s)
	}

	tests := []struct {
		name       string
		pagination Pagination
		wantIDs    []int64
	}{
		{
			name:       "no params returns default page in id order",
			pagination: Pagination{},
			wantIDs:    []int64{seeded[0].ID, seeded[1].ID, seeded[2].ID, seeded[3].ID, seeded[4].ID},
		},
		{
			name:       "explicit limit and offset selects a sub-page",
			pagination: Pagination{Limit: 2, Offset: 3},
			wantIDs:    []int64{seeded[3].ID, seeded[4].ID},
		},
		{
			name:       "offset past the end returns empty slice not error",
			pagination: Pagination{Limit: 10, Offset: 100},
			wantIDs:    []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scans, err := repo.ListScansByProject(context.Background(), 1, tt.pagination)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(scans) != len(tt.wantIDs) {
				t.Fatalf("expected %d scans, got %d", len(tt.wantIDs), len(scans))
			}
			for i, want := range tt.wantIDs {
				if scans[i].ID != want {
					t.Errorf("index %d: expected scan ID %d, got %d", i, want, scans[i].ID)
				}
			}
		})
	}
}

func TestGetScanByID_NotFound(t *testing.T) {
	repo := NewFakeScanRepository()

	_, err := repo.GetScanByID(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetScanByID_ReturnsScan(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)
	created, err := repo.CreateScan(context.Background(), 1, "nmap")
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	scan, err := repo.GetScanByID(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scan.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, scan.ID)
	}
	if scan.Tool != "nmap" {
		t.Errorf("expected tool 'nmap', got '%s'", scan.Tool)
	}
	if scan.ProjectID != 1 {
		t.Errorf("expected project ID 1, got %d", scan.ProjectID)
	}
}

func TestCreateScan_ValidatesInput(t *testing.T) {
	tests := []struct {
		name      string
		projectID int64
		tool      string
		wantErr   error
	}{
		{"missing project ID", 0, "nmap", domain.ErrProjectIDRequired},
		{"missing tool", 1, "", domain.ErrToolRequired},
		{"whitespace-only tool", 1, "   ", domain.ErrToolRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeScanRepository()
			repo.AddProject(1)

			_, err := repo.CreateScan(context.Background(), tt.projectID, tt.tool)

			if err != tt.wantErr {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateScan_TrimsToolWhitespace(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)

	scan, err := repo.CreateScan(context.Background(), 1, "  nmap  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scan.Tool != "nmap" {
		t.Errorf("expected trimmed tool 'nmap', got '%s'", scan.Tool)
	}
}

func TestCreateScan_FKViolation_ProjectNotFound(t *testing.T) {
	repo := NewFakeScanRepository()
	// Note: project 1 is never registered via AddProject.

	_, err := repo.CreateScan(context.Background(), 1, "nmap")

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateScan_ReturnsScanWithID(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)

	scan, err := repo.CreateScan(context.Background(), 1, "nmap")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scan.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if scan.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
}

func TestDeleteScan_NotFound(t *testing.T) {
	repo := NewFakeScanRepository()

	err := repo.DeleteScan(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteScan_RemovesScan(t *testing.T) {
	repo := NewFakeScanRepository()
	repo.AddProject(1)
	created, err := repo.CreateScan(context.Background(), 1, "nmap")
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	err = repo.DeleteScan(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.GetScanByID(context.Background(), created.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
