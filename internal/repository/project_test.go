package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// FakeProjectRepository is a test double for unit testing.
type FakeProjectRepository struct {
	projects map[int64]domain.Project
	nextID   int64
}

// NewFakeProjectRepository creates a new fake repository for testing.
func NewFakeProjectRepository() *FakeProjectRepository {
	return &FakeProjectRepository{
		projects: make(map[int64]domain.Project),
		nextID:   1,
	}
}

func (f *FakeProjectRepository) ListProjects(ctx context.Context, p Pagination) ([]domain.Project, error) {
	projects := []domain.Project{}
	for _, proj := range f.projects {
		projects = append(projects, proj)
	}
	return paginateByID(projects, func(proj domain.Project) int64 { return proj.ID }, p), nil
}

func (f *FakeProjectRepository) GetProjectByID(ctx context.Context, id int64) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
}

func (f *FakeProjectRepository) CreateProject(ctx context.Context, name string) (domain.Project, error) {
	p := domain.Project{
		ID:        f.nextID,
		Name:      strings.TrimSpace(name),
		CreatedAt: time.Now(),
	}
	if err := p.Validate(); err != nil {
		return domain.Project{}, err
	}
	f.projects[p.ID] = p
	f.nextID++
	return p, nil
}

func (f *FakeProjectRepository) DeleteProject(ctx context.Context, id int64) error {
	if _, ok := f.projects[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.projects, id)
	return nil
}

// Test cases

func TestListProjects_Empty(t *testing.T) {
	repo := NewFakeProjectRepository()

	projects, err := repo.ListProjects(context.Background(), Pagination{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects_ReturnsAllProjects(t *testing.T) {
	repo := NewFakeProjectRepository()
	if _, err := repo.CreateProject(context.Background(), "Project A"); err != nil {
		t.Fatalf("setup: create project A: %v", err)
	}
	if _, err := repo.CreateProject(context.Background(), "Project B"); err != nil {
		t.Fatalf("setup: create project B: %v", err)
	}

	projects, err := repo.ListProjects(context.Background(), Pagination{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

// TestListProjects_Pagination covers Pagination handling: no params
// applies the default page size, an explicit limit/offset selects a
// sub-page in id order, a limit above the cap is clamped, and an
// offset past the end of the result set returns an empty slice rather
// than an error.
func TestListProjects_Pagination(t *testing.T) {
	repo := NewFakeProjectRepository()
	var seeded []domain.Project
	for i := 0; i < 5; i++ {
		p, err := repo.CreateProject(context.Background(), fmt.Sprintf("Project %d", i))
		if err != nil {
			t.Fatalf("setup: create project %d: %v", i, err)
		}
		seeded = append(seeded, p)
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
			name:       "explicit limit returns that many",
			pagination: Pagination{Limit: 2},
			wantIDs:    []int64{seeded[0].ID, seeded[1].ID},
		},
		{
			name:       "explicit limit and offset selects a sub-page",
			pagination: Pagination{Limit: 2, Offset: 2},
			wantIDs:    []int64{seeded[2].ID, seeded[3].ID},
		},
		{
			name:       "limit above cap is clamped to maxPageLimit",
			pagination: Pagination{Limit: maxPageLimit + 1000},
			wantIDs:    []int64{seeded[0].ID, seeded[1].ID, seeded[2].ID, seeded[3].ID, seeded[4].ID},
		},
		{
			name:       "offset past the end returns empty slice not error",
			pagination: Pagination{Limit: 10, Offset: 100},
			wantIDs:    []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := repo.ListProjects(context.Background(), tt.pagination)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(projects) != len(tt.wantIDs) {
				t.Fatalf("expected %d projects, got %d", len(tt.wantIDs), len(projects))
			}
			for i, want := range tt.wantIDs {
				if projects[i].ID != want {
					t.Errorf("index %d: expected project ID %d, got %d", i, want, projects[i].ID)
				}
			}
		})
	}
}

func TestGetProjectByID_NotFound(t *testing.T) {
	repo := NewFakeProjectRepository()

	_, err := repo.GetProjectByID(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetProjectByID_ReturnsProject(t *testing.T) {
	repo := NewFakeProjectRepository()
	created, _ := repo.CreateProject(context.Background(), "Test Project")

	project, err := repo.GetProjectByID(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, project.ID)
	}
	if project.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got '%s'", project.Name)
	}
}

func TestCreateProject_ValidatesName(t *testing.T) {
	repo := NewFakeProjectRepository()

	_, err := repo.CreateProject(context.Background(), "")

	if err != domain.ErrProjectNameRequired {
		t.Errorf("expected ErrProjectNameRequired, got %v", err)
	}
}

func TestCreateProject_TrimsNameWhitespace(t *testing.T) {
	repo := NewFakeProjectRepository()

	project, err := repo.CreateProject(context.Background(), "  My Project  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Name != "My Project" {
		t.Errorf("expected trimmed name 'My Project', got '%s'", project.Name)
	}
}

func TestCreateProject_ReturnsProjectWithID(t *testing.T) {
	repo := NewFakeProjectRepository()

	project, err := repo.CreateProject(context.Background(), "My Project")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if project.Name != "My Project" {
		t.Errorf("expected name 'My Project', got '%s'", project.Name)
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	repo := NewFakeProjectRepository()

	err := repo.DeleteProject(context.Background(), 999)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteProject_RemovesProject(t *testing.T) {
	repo := NewFakeProjectRepository()
	created, _ := repo.CreateProject(context.Background(), "Temp Project")

	err := repo.DeleteProject(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.GetProjectByID(context.Background(), created.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
