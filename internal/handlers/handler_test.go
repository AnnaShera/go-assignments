package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
)

// fakeRepo is a package-local test double for Repository. It can't reuse
// repository.FakeProjectRepository because that type is defined in a
// _test.go file: Go excludes _test.go files from the compiled package
// archive, so nothing outside package repository can import it. This fake
// mirrors its behavior exactly (see internal/repository/project_test.go).
type fakeRepo struct {
	projects map[int64]domain.Project
	nextID   int64
	listErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{projects: make(map[int64]domain.Project), nextID: 1}
}

func (f *fakeRepo) ListProjects(ctx context.Context) ([]domain.Project, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	projects := make([]domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		projects = append(projects, p)
	}
	return projects, nil
}

func (f *fakeRepo) GetProjectByID(ctx context.Context, id int64) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeRepo) CreateProject(ctx context.Context, name string) (domain.Project, error) {
	p := domain.Project{ID: f.nextID, Name: name, CreatedAt: time.Now()}
	if err := p.Validate(); err != nil {
		return domain.Project{}, err
	}
	f.projects[p.ID] = p
	f.nextID++
	return p, nil
}

func (f *fakeRepo) DeleteProject(ctx context.Context, id int64) error {
	if _, ok := f.projects[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.projects, id)
	return nil
}

// testEnvelope mirrors the wire shape of envelope/errorBody for decoding
// responses in assertions, keeping Data as raw JSON since its shape varies
// per endpoint (a single project vs. a list).
type testEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Message string `json:"error"`
		Code    string `json:"code"`
	} `json:"error"`
}

func newTestHandler() (*Handler, *fakeRepo) {
	repo := newFakeRepo()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(repo, log), repo
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) testEnvelope {
	t.Helper()
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response body %q: %v", w.Body.String(), err)
	}
	return env
}

func TestListProjects_Returns200WithProjects(t *testing.T) {
	h, repo := newTestHandler()
	if _, err := repo.CreateProject(context.Background(), "Project A"); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := repo.CreateProject(context.Background(), "Project B"); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	w := httptest.NewRecorder()

	h.ListProjects(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	env := decodeEnvelope(t, w)
	if env.Error != nil {
		t.Fatalf("expected no error, got %+v", env.Error)
	}
	var projects []domain.Project
	if err := json.Unmarshal(env.Data, &projects); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestListProjects_EmptyReturns200WithEmptyArray(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	w := httptest.NewRecorder()

	h.ListProjects(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	env := decodeEnvelope(t, w)
	if string(env.Data) != "[]" {
		t.Errorf("expected empty array \"[]\", got %q", string(env.Data))
	}
}

func TestGetProject_Returns200WithProject(t *testing.T) {
	h, repo := newTestHandler()
	created, err := repo.CreateProject(context.Background(), "Test Project")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+strconv.FormatInt(created.ID, 10), nil)
	req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
	w := httptest.NewRecorder()

	h.GetProject(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var project domain.Project
	if err := json.Unmarshal(env.Data, &project); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if project.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, project.ID)
	}
	if project.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", project.Name)
	}
}

func TestGetProject_NotFoundReturns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/projects/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.GetProject(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil {
		t.Fatal("expected error body, got none")
	}
	if env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %q", CodeNotFound, env.Error.Code)
	}
}

func TestGetProject_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/projects/not-a-number", nil)
	req.SetPathValue("id", "not-a-number")
	w := httptest.NewRecorder()

	h.GetProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

func TestCreateProject_ValidInput_Returns201WithProject(t *testing.T) {
	h, _ := newTestHandler()

	body := bytes.NewBufferString(`{"name":"My Project"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects", body)
	w := httptest.NewRecorder()

	h.CreateProject(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var project domain.Project
	if err := json.Unmarshal(env.Data, &project); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if project.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if project.Name != "My Project" {
		t.Errorf("expected name 'My Project', got %q", project.Name)
	}
}

func TestCreateProject_MissingName_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/projects", body)
	w := httptest.NewRecorder()

	h.CreateProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil {
		t.Fatal("expected error body, got none")
	}
	if env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %q", CodeInvalidInput, env.Error.Code)
	}
}

func TestCreateProject_InvalidJSON_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	body := bytes.NewBufferString(`{not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/projects", body)
	w := httptest.NewRecorder()

	h.CreateProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

// TestCreateProject_DuplicateName_Returns400 is skipped: neither the
// domain.Project.Validate rules nor the projects table (see
// migrations/0001_init.sql) enforce a unique name constraint, so the repo
// does not reject duplicates. Per the assignment's own fallback
// ("otherwise skip"), this is documented as a skip rather than omitted
// silently.
func TestCreateProject_DuplicateName_Returns400(t *testing.T) {
	t.Skip("no unique constraint on project name in schema or domain validation; repo does not reject duplicates")
}

func TestDeleteProject_Returns204NoContent(t *testing.T) {
	h, repo := newTestHandler()
	created, err := repo.CreateProject(context.Background(), "Temp Project")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+strconv.FormatInt(created.ID, 10), nil)
	req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
	w := httptest.NewRecorder()

	h.DeleteProject(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
	if _, err := repo.GetProjectByID(context.Background(), created.ID); err != domain.ErrNotFound {
		t.Errorf("expected project to be deleted, got err=%v", err)
	}
}

func TestDeleteProject_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/projects/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.DeleteProject(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}
