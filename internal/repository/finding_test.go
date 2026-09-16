package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// FakeFindingRepository is a test double for unit testing.
type FakeFindingRepository struct {
	findings map[int64]domain.Finding
	nextID   int64
	// validScanIDs simulates the scans table's FK target, since the fake
	// has no real foreign key to violate.
	validScanIDs map[int64]bool
}

// NewFakeFindingRepository creates a new fake repository for testing.
func NewFakeFindingRepository() *FakeFindingRepository {
	return &FakeFindingRepository{
		findings:     make(map[int64]domain.Finding),
		nextID:       1,
		validScanIDs: make(map[int64]bool),
	}
}

// AddScan registers a scan ID as existing, so CreateFinding's FK check
// against it succeeds. Mirrors the real scans table for FK simulation.
func (f *FakeFindingRepository) AddScan(scanID int64) {
	f.validScanIDs[scanID] = true
}

func (f *FakeFindingRepository) ListFindingsByScan(ctx context.Context, scanID int64) ([]domain.Finding, error) {
	findings := []domain.Finding{}
	for _, fd := range f.findings {
		if fd.ScanID == scanID {
			findings = append(findings, fd)
		}
	}
	return findings, nil
}

func (f *FakeFindingRepository) GetFindingByID(ctx context.Context, id int64) (domain.Finding, error) {
	fd, ok := f.findings[id]
	if !ok {
		return domain.Finding{}, domain.ErrNotFound
	}
	return fd, nil
}

func (f *FakeFindingRepository) CreateFinding(ctx context.Context, finding domain.Finding) (domain.Finding, error) {
	finding.Title = strings.TrimSpace(finding.Title)
	finding.FilePath = strings.TrimSpace(finding.FilePath)

	if err := finding.Validate(); err != nil {
		return domain.Finding{}, err
	}
	if !f.validScanIDs[finding.ScanID] {
		return domain.Finding{}, domain.ErrNotFound
	}

	finding.ID = f.nextID
	finding.CreatedAt = time.Now()
	f.findings[finding.ID] = finding
	f.nextID++
	return finding, nil
}

func (f *FakeFindingRepository) UpdateFindingStatus(ctx context.Context, id int64, status string) error {
	if !domain.IsValidStatus(status) {
		return domain.ErrInvalidStatus
	}

	fd, ok := f.findings[id]
	if !ok {
		return domain.ErrNotFound
	}
	fd.Status = status
	f.findings[id] = fd
	return nil
}

func (f *FakeFindingRepository) DeleteFinding(ctx context.Context, id int64) error {
	if _, ok := f.findings[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.findings, id)
	return nil
}

// newValidFinding returns a finding that passes Validate(), for tests that
// only care about one invalid field at a time.
func newValidFinding(scanID int64) domain.Finding {
	return domain.Finding{
		ScanID:     scanID,
		Title:      "SQL Injection",
		Severity:   domain.SeverityHigh,
		Status:     domain.StatusOpen,
		FilePath:   "internal/handler/user.go",
		LineNumber: 42,
	}
}

// Test cases

func TestListFindingsByScan_Empty(t *testing.T) {
	repo := NewFakeFindingRepository()

	findings, err := repo.ListFindingsByScan(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestListFindingsByScan_ReturnsFindingsForScan(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	repo.AddScan(2)
	f1 := newValidFinding(1)
	f2 := newValidFinding(1)
	f3 := newValidFinding(2)
	if _, err := repo.CreateFinding(context.Background(), f1); err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}
	if _, err := repo.CreateFinding(context.Background(), f2); err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}
	if _, err := repo.CreateFinding(context.Background(), f3); err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	findings, err := repo.ListFindingsByScan(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 2 {
		t.Errorf("expected 2 findings for scan 1, got %d", len(findings))
	}
	for _, fd := range findings {
		if fd.ScanID != 1 {
			t.Errorf("expected only scan 1 findings, got finding for scan %d", fd.ScanID)
		}
	}
}

func TestGetFindingByID_NotFound(t *testing.T) {
	repo := NewFakeFindingRepository()

	_, err := repo.GetFindingByID(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetFindingByID_ReturnsFinding(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	created, err := repo.CreateFinding(context.Background(), newValidFinding(1))
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	finding, err := repo.GetFindingByID(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finding.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, finding.ID)
	}
	if finding.Title != "SQL Injection" {
		t.Errorf("expected title 'SQL Injection', got '%s'", finding.Title)
	}
}

func TestCreateFinding_ValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(f domain.Finding) domain.Finding
		wantErr error
	}{
		{
			name:    "missing scan ID",
			modify:  func(f domain.Finding) domain.Finding { f.ScanID = 0; return f },
			wantErr: domain.ErrScanIDRequired,
		},
		{
			name:    "missing title",
			modify:  func(f domain.Finding) domain.Finding { f.Title = ""; return f },
			wantErr: domain.ErrTitleRequired,
		},
		{
			name:    "whitespace-only title",
			modify:  func(f domain.Finding) domain.Finding { f.Title = "   "; return f },
			wantErr: domain.ErrTitleRequired,
		},
		{
			name:    "invalid severity",
			modify:  func(f domain.Finding) domain.Finding { f.Severity = "extreme"; return f },
			wantErr: domain.ErrInvalidSeverity,
		},
		{
			name:    "invalid status",
			modify:  func(f domain.Finding) domain.Finding { f.Status = "unknown"; return f },
			wantErr: domain.ErrInvalidStatus,
		},
		{
			name:    "missing file path",
			modify:  func(f domain.Finding) domain.Finding { f.FilePath = ""; return f },
			wantErr: domain.ErrFilePathRequired,
		},
		{
			name:    "zero line number",
			modify:  func(f domain.Finding) domain.Finding { f.LineNumber = 0; return f },
			wantErr: domain.ErrInvalidLineNumber,
		},
		{
			name:    "negative line number",
			modify:  func(f domain.Finding) domain.Finding { f.LineNumber = -1; return f },
			wantErr: domain.ErrInvalidLineNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeFindingRepository()
			repo.AddScan(1)
			finding := tt.modify(newValidFinding(1))

			_, err := repo.CreateFinding(context.Background(), finding)

			if err != tt.wantErr {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateFinding_TrimsWhitespace(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	finding := newValidFinding(1)
	finding.Title = "  SQL Injection  "
	finding.FilePath = "  internal/handler/user.go  "

	created, err := repo.CreateFinding(context.Background(), finding)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Title != "SQL Injection" {
		t.Errorf("expected trimmed title 'SQL Injection', got '%s'", created.Title)
	}
	if created.FilePath != "internal/handler/user.go" {
		t.Errorf("expected trimmed file path 'internal/handler/user.go', got '%s'", created.FilePath)
	}
}

func TestCreateFinding_FKViolation_ScanNotFound(t *testing.T) {
	repo := NewFakeFindingRepository()
	// Note: scan 1 is never registered via AddScan.

	_, err := repo.CreateFinding(context.Background(), newValidFinding(1))

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateFinding_ReturnsFindingWithID(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)

	finding, err := repo.CreateFinding(context.Background(), newValidFinding(1))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finding.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if finding.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestUpdateFindingStatus_InvalidStatus(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	created, err := repo.CreateFinding(context.Background(), newValidFinding(1))
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	err = repo.UpdateFindingStatus(context.Background(), created.ID, "bogus")

	if err != domain.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestUpdateFindingStatus_NotFound(t *testing.T) {
	repo := NewFakeFindingRepository()

	err := repo.UpdateFindingStatus(context.Background(), 999, domain.StatusResolved)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateFindingStatus_UpdatesStatus(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	created, err := repo.CreateFinding(context.Background(), newValidFinding(1))
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	err = repo.UpdateFindingStatus(context.Background(), created.ID, domain.StatusResolved)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := repo.GetFindingByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusResolved {
		t.Errorf("expected status %q, got %q", domain.StatusResolved, updated.Status)
	}
}

func TestDeleteFinding_NotFound(t *testing.T) {
	repo := NewFakeFindingRepository()

	err := repo.DeleteFinding(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteFinding_RemovesFinding(t *testing.T) {
	repo := NewFakeFindingRepository()
	repo.AddScan(1)
	created, err := repo.CreateFinding(context.Background(), newValidFinding(1))
	if err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	err = repo.DeleteFinding(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.GetFindingByID(context.Background(), created.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
