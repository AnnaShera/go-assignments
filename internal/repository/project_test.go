package repository

import (
	"context"
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

func (f *FakeProjectRepository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	var projects []domain.Project
	for _, p := range f.projects {
		projects = append(projects, p)
	}
	return projects, nil
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
		Name:      name,
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

	projects, err := repo.ListProjects(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects_ReturnsAllProjects(t *testing.T) {
	repo := NewFakeProjectRepository()
	repo.CreateProject(context.Background(), "Project A")
	repo.CreateProject(context.Background(), "Project B")

	projects, err := repo.ListProjects(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
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
