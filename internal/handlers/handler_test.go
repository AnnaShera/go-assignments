package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
	"github.com/AnnaShera/vuln-findings-api/internal/repository"
)

// paginateByID sorts items by ID ascending, mirroring the real
// repository's ORDER BY id, and returns the window p.Normalize()
// describes. Shared by fakeRepo's three List methods below, which each
// need identical windowing logic over a different element type.
func paginateByID[T any](items []T, id func(T) int64, p repository.Pagination) []T {
	sort.Slice(items, func(i, j int) bool { return id(items[i]) < id(items[j]) })
	p = p.Normalize()
	if p.Offset >= len(items) {
		return []T{}
	}
	end := p.Offset + p.Limit
	if end > len(items) {
		end = len(items)
	}
	return items[p.Offset:end]
}

// fakeRepo is a package-local test double for Repository. It can't reuse
// repository.FakeProjectRepository because that type is defined in a
// _test.go file: Go excludes _test.go files from the compiled package
// archive, so nothing outside package repository can import it. This fake
// mirrors its behavior exactly (see internal/repository/project_test.go).
type fakeRepo struct {
	projects map[int64]domain.Project
	nextID   int64
	listErr  error

	scans      map[int64]domain.Scan
	nextScanID int64

	findings      map[int64]domain.Finding
	nextFindingID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:      make(map[int64]domain.Project),
		nextID:        1,
		scans:         make(map[int64]domain.Scan),
		nextScanID:    1,
		findings:      make(map[int64]domain.Finding),
		nextFindingID: 1,
	}
}

func (f *fakeRepo) ListProjects(ctx context.Context, pg repository.Pagination) ([]domain.Project, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	projects := make([]domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		projects = append(projects, p)
	}
	return paginateByID(projects, func(p domain.Project) int64 { return p.ID }, pg), nil
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

func (f *fakeRepo) ListScansByProject(ctx context.Context, projectID int64, pg repository.Pagination) ([]domain.Scan, error) {
	scans := []domain.Scan{}
	for _, s := range f.scans {
		if s.ProjectID == projectID {
			scans = append(scans, s)
		}
	}
	return paginateByID(scans, func(s domain.Scan) int64 { return s.ID }, pg), nil
}

func (f *fakeRepo) GetScanByID(ctx context.Context, id int64) (domain.Scan, error) {
	s, ok := f.scans[id]
	if !ok {
		return domain.Scan{}, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeRepo) CreateScan(ctx context.Context, projectID int64, tool string) (domain.Scan, error) {
	s := domain.Scan{ProjectID: projectID, Tool: tool}
	if err := s.Validate(); err != nil {
		return domain.Scan{}, err
	}
	// Mirrors the real FK check (postgres.go's CreateScan maps a 23503
	// violation to ErrNotFound): the fake has no real foreign key, so it
	// checks the projects map directly instead.
	if _, ok := f.projects[projectID]; !ok {
		return domain.Scan{}, domain.ErrNotFound
	}
	s.ID = f.nextScanID
	s.StartedAt = time.Now()
	f.scans[s.ID] = s
	f.nextScanID++
	return s, nil
}

func (f *fakeRepo) DeleteScan(ctx context.Context, id int64) error {
	if _, ok := f.scans[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.scans, id)
	return nil
}

// ListFindingsByScan returns findings for scanID, optionally narrowed by
// filter.Severity and/or filter.Status, and paged per pg.
func (f *fakeRepo) ListFindingsByScan(ctx context.Context, scanID int64, filter repository.FindingFilter, pg repository.Pagination) ([]domain.Finding, error) {
	findings := []domain.Finding{}
	for _, fd := range f.findings {
		if fd.ScanID != scanID {
			continue
		}
		if filter.Severity != "" && fd.Severity != filter.Severity {
			continue
		}
		if filter.Status != "" && fd.Status != filter.Status {
			continue
		}
		findings = append(findings, fd)
	}
	return paginateByID(findings, func(fd domain.Finding) int64 { return fd.ID }, pg), nil
}

func (f *fakeRepo) GetFindingByID(ctx context.Context, id int64) (domain.Finding, error) {
	fd, ok := f.findings[id]
	if !ok {
		return domain.Finding{}, domain.ErrNotFound
	}
	return fd, nil
}

func (f *fakeRepo) CreateFinding(ctx context.Context, finding domain.Finding) (domain.Finding, error) {
	if err := finding.Validate(); err != nil {
		return domain.Finding{}, err
	}
	// Mirrors the real FK check on scan_id, same reasoning as CreateScan
	// above.
	if _, ok := f.scans[finding.ScanID]; !ok {
		return domain.Finding{}, domain.ErrNotFound
	}
	finding.ID = f.nextFindingID
	finding.CreatedAt = time.Now()
	f.findings[finding.ID] = finding
	f.nextFindingID++
	return finding, nil
}

func (f *fakeRepo) UpdateFindingStatus(ctx context.Context, id int64, status string) error {
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

func (f *fakeRepo) DeleteFinding(ctx context.Context, id int64) error {
	if _, ok := f.findings[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.findings, id)
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

// TestListProjects_Pagination covers the ?limit=/?offset= query params on
// GET /projects: unset defaults, explicit values page correctly, an
// offset past the end returns an empty array rather than an error, and a
// non-numeric or negative value is rejected as 400 invalid_input rather
// than silently defaulted or clamped.
func TestListProjects_Pagination(t *testing.T) {
	h, repo := newTestHandler()
	for i := 0; i < 5; i++ {
		if _, err := repo.CreateProject(context.Background(), fmt.Sprintf("Project %d", i)); err != nil {
			t.Fatalf("seed project %d: %v", i, err)
		}
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{name: "no params returns all seeded (under default limit)", query: "", wantStatus: http.StatusOK, wantCount: 5},
		{name: "explicit limit narrows the page", query: "?limit=2", wantStatus: http.StatusOK, wantCount: 2},
		{name: "explicit limit and offset selects a sub-page", query: "?limit=2&offset=3", wantStatus: http.StatusOK, wantCount: 2},
		{name: "offset past the end returns empty array", query: "?limit=10&offset=100", wantStatus: http.StatusOK, wantCount: 0},
		{name: "non-numeric limit returns 400", query: "?limit=abc", wantStatus: http.StatusBadRequest},
		{name: "negative limit returns 400", query: "?limit=-1", wantStatus: http.StatusBadRequest},
		{name: "non-numeric offset returns 400", query: "?offset=abc", wantStatus: http.StatusBadRequest},
		{name: "negative offset returns 400", query: "?offset=-1", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/projects"+tt.query, nil)
			w := httptest.NewRecorder()

			h.ListProjects(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			if tt.wantStatus != http.StatusOK {
				if env.Error == nil || env.Error.Code != CodeInvalidInput {
					t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
				}
				return
			}
			var projects []domain.Project
			if err := json.Unmarshal(env.Data, &projects); err != nil {
				t.Fatalf("decode data: %v", err)
			}
			if len(projects) != tt.wantCount {
				t.Errorf("expected %d projects, got %d", tt.wantCount, len(projects))
			}
		})
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

// TestDecodeJSON_BodySizeLimit drives decodeJSON directly (it's
// package-private, decodeJSON lives in handler.go) across the three cases
// STANDARDS.md's error-handling default cares about here: a normal small
// body still decodes, a too-large body is distinguished from an ordinary
// malformed body so it can map to 413 instead of 400, and a malformed-but-
// small body keeps the existing 400 behavior.
func TestDecodeJSON_BodySizeLimit(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantStatus int
		wantCode   string
	}{
		{
			name:    "body under limit decodes fine",
			body:    `{"name":"ok"}`,
			wantErr: false,
		},
		{
			name:       "malformed JSON under limit returns 400",
			body:       `{not valid json`,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeInvalidInput,
		},
		{
			name:       "body over limit returns 413",
			body:       `{"name":"` + strings.Repeat("a", maxBodyBytes+1) + `"}`,
			wantErr:    true,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   CodeRequestTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			var dst payload
			err := decodeJSON(w, req, &dst)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			status, code, _ := mapError(err)
			if status != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, status)
			}
			if code != tt.wantCode {
				t.Errorf("expected code %q, got %q", tt.wantCode, code)
			}
		})
	}
}

// TestCreateProject_BodyTooLarge_Returns413 confirms the size limit is
// actually wired into the handler call site, not just decodeJSON in
// isolation.
func TestCreateProject_BodyTooLarge_Returns413(t *testing.T) {
	h, _ := newTestHandler()

	body := bytes.NewBufferString(`{"name":"` + strings.Repeat("a", maxBodyBytes+1) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects", body)
	w := httptest.NewRecorder()

	h.CreateProject(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeRequestTooLarge {
		t.Errorf("expected code %q, got %+v", CodeRequestTooLarge, env.Error)
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

// --- Scan handlers ---

func TestListScans_Returns200WithScans(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := repo.CreateScan(context.Background(), project.ID, "nmap"); err != nil {
		t.Fatalf("seed scan: %v", err)
	}
	if _, err := repo.CreateScan(context.Background(), project.ID, "zap"); err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	projectIDStr := strconv.FormatInt(project.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/projects/"+projectIDStr+"/scans", nil)
	req.SetPathValue("projectID", projectIDStr)
	w := httptest.NewRecorder()

	h.ListScans(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var scans []domain.Scan
	if err := json.Unmarshal(env.Data, &scans); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(scans) != 2 {
		t.Errorf("expected 2 scans, got %d", len(scans))
	}
}

func TestListScans_EmptyReturns200WithEmptyArray(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	projectIDStr := strconv.FormatInt(project.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/projects/"+projectIDStr+"/scans", nil)
	req.SetPathValue("projectID", projectIDStr)
	w := httptest.NewRecorder()

	h.ListScans(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if string(env.Data) != "[]" {
		t.Errorf("expected empty array \"[]\", got %q", string(env.Data))
	}
}

// TestListScans_Pagination covers the ?limit=/?offset= query params on
// GET /projects/{projectID}/scans.
func TestListScans_Pagination(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := repo.CreateScan(context.Background(), project.ID, fmt.Sprintf("tool-%d", i)); err != nil {
			t.Fatalf("seed scan %d: %v", i, err)
		}
	}
	projectIDStr := strconv.FormatInt(project.ID, 10)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{name: "explicit limit narrows the page", query: "?limit=2", wantStatus: http.StatusOK, wantCount: 2},
		{name: "negative offset returns 400", query: "?offset=-1", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/projects/"+projectIDStr+"/scans"+tt.query, nil)
			req.SetPathValue("projectID", projectIDStr)
			w := httptest.NewRecorder()

			h.ListScans(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			if tt.wantStatus != http.StatusOK {
				if env.Error == nil || env.Error.Code != CodeInvalidInput {
					t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
				}
				return
			}
			var scans []domain.Scan
			if err := json.Unmarshal(env.Data, &scans); err != nil {
				t.Fatalf("decode data: %v", err)
			}
			if len(scans) != tt.wantCount {
				t.Errorf("expected %d scans, got %d", tt.wantCount, len(scans))
			}
		})
	}
}

func TestGetScan_Returns200WithScan(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	created, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/scans/"+idStr, nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.GetScan(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var scan domain.Scan
	if err := json.Unmarshal(env.Data, &scan); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if scan.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, scan.ID)
	}
	if scan.Tool != "nmap" {
		t.Errorf("expected tool 'nmap', got %q", scan.Tool)
	}
}

func TestGetScan_NotFoundReturns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/scans/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.GetScan(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

func TestCreateScan_ValidInput_Returns201WithScan(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	projectIDStr := strconv.FormatInt(project.ID, 10)
	body := bytes.NewBufferString(`{"tool":"nmap"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectIDStr+"/scans", body)
	req.SetPathValue("projectID", projectIDStr)
	w := httptest.NewRecorder()

	h.CreateScan(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var scan domain.Scan
	if err := json.Unmarshal(env.Data, &scan); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if scan.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if scan.ProjectID != project.ID {
		t.Errorf("expected project ID %d, got %d", project.ID, scan.ProjectID)
	}
	if scan.Tool != "nmap" {
		t.Errorf("expected tool 'nmap', got %q", scan.Tool)
	}
}

func TestCreateScan_NonexistentProject_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	// Project 999 was never created, simulating the FK violation the real
	// repository maps to ErrNotFound (see postgres.go CreateScan).
	body := bytes.NewBufferString(`{"tool":"nmap"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects/999/scans", body)
	req.SetPathValue("projectID", "999")
	w := httptest.NewRecorder()

	h.CreateScan(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

func TestCreateScan_MissingTool_Returns400(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	projectIDStr := strconv.FormatInt(project.ID, 10)
	body := bytes.NewBufferString(`{"tool":""}`)
	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectIDStr+"/scans", body)
	req.SetPathValue("projectID", projectIDStr)
	w := httptest.NewRecorder()

	h.CreateScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

func TestDeleteScan_Returns204NoContent(t *testing.T) {
	h, repo := newTestHandler()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	created, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodDelete, "/scans/"+idStr, nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.DeleteScan(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
	if _, err := repo.GetScanByID(context.Background(), created.ID); err != domain.ErrNotFound {
		t.Errorf("expected scan to be deleted, got err=%v", err)
	}
}

func TestDeleteScan_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/scans/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.DeleteScan(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

// --- Finding handlers ---

// seedScan seeds a project and a scan under it, returning the scan. Shared
// by the finding tests below, which all need a valid scan ID to hang
// findings off.
func seedScan(t *testing.T, repo *fakeRepo) domain.Scan {
	t.Helper()
	project, err := repo.CreateProject(context.Background(), "Project A")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	scan, err := repo.CreateScan(context.Background(), project.ID, "nmap")
	if err != nil {
		t.Fatalf("seed scan: %v", err)
	}
	return scan
}

func TestListFindings_Returns200WithFindings(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	finding := domain.Finding{
		ScanID: scan.ID, Title: "SQLi", Severity: domain.SeverityHigh,
		Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 10,
	}
	if _, err := repo.CreateFinding(context.Background(), finding); err != nil {
		t.Fatalf("seed finding: %v", err)
	}
	if _, err := repo.CreateFinding(context.Background(), finding); err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	scanIDStr := strconv.FormatInt(scan.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/scans/"+scanIDStr+"/findings", nil)
	req.SetPathValue("scanID", scanIDStr)
	w := httptest.NewRecorder()

	h.ListFindings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var findings []domain.Finding
	if err := json.Unmarshal(env.Data, &findings); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}

func TestListFindings_EmptyReturns200WithEmptyArray(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)

	scanIDStr := strconv.FormatInt(scan.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/scans/"+scanIDStr+"/findings", nil)
	req.SetPathValue("scanID", scanIDStr)
	w := httptest.NewRecorder()

	h.ListFindings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if string(env.Data) != "[]" {
		t.Errorf("expected empty array \"[]\", got %q", string(env.Data))
	}
}

// TestListFindings_QueryFilters covers ?severity= and ?status= filtering
// on GET /scans/{scanID}/findings, including both together and neither
// (existing unfiltered behavior, preserved as one case of this table).
func TestListFindings_QueryFilters(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantCount    int
		wantSeverity string // "" = don't check
		wantStatus   string // "" = don't check
	}{
		{name: "no query params returns unfiltered", query: "", wantCount: 3},
		{name: "severity filter returns only matching", query: "severity=high", wantCount: 2, wantSeverity: domain.SeverityHigh},
		{name: "status filter returns only matching", query: "status=open", wantCount: 2, wantStatus: domain.StatusOpen},
		{name: "severity and status filter together", query: "severity=high&status=open", wantCount: 1, wantSeverity: domain.SeverityHigh, wantStatus: domain.StatusOpen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newTestHandler()
			scan := seedScan(t, repo)
			seed := []domain.Finding{
				{ScanID: scan.ID, Title: "A", Severity: domain.SeverityHigh, Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 1},
				{ScanID: scan.ID, Title: "B", Severity: domain.SeverityHigh, Status: domain.StatusResolved, FilePath: "b.go", LineNumber: 2},
				{ScanID: scan.ID, Title: "C", Severity: domain.SeverityLow, Status: domain.StatusOpen, FilePath: "c.go", LineNumber: 3},
			}
			for _, fd := range seed {
				if _, err := repo.CreateFinding(context.Background(), fd); err != nil {
					t.Fatalf("seed finding: %v", err)
				}
			}

			scanIDStr := strconv.FormatInt(scan.ID, 10)
			target := "/scans/" + scanIDStr + "/findings"
			if tt.query != "" {
				target += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			req.SetPathValue("scanID", scanIDStr)
			w := httptest.NewRecorder()

			h.ListFindings(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			var findings []domain.Finding
			if err := json.Unmarshal(env.Data, &findings); err != nil {
				t.Fatalf("decode data: %v", err)
			}
			if len(findings) != tt.wantCount {
				t.Fatalf("expected %d findings, got %d", tt.wantCount, len(findings))
			}
			for _, fd := range findings {
				if tt.wantSeverity != "" && fd.Severity != tt.wantSeverity {
					t.Errorf("expected severity %q, got %q", tt.wantSeverity, fd.Severity)
				}
				if tt.wantStatus != "" && fd.Status != tt.wantStatus {
					t.Errorf("expected status %q, got %q", tt.wantStatus, fd.Status)
				}
			}
		})
	}
}

// TestListFindings_InvalidFilterValue_Returns400 confirms an invalid
// ?severity= or ?status= value is rejected as a 400 rather than silently
// treated as "no matches."
func TestListFindings_InvalidFilterValue_Returns400(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "invalid severity", query: "severity=extreme"},
		{name: "invalid status", query: "status=unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newTestHandler()
			scan := seedScan(t, repo)

			scanIDStr := strconv.FormatInt(scan.ID, 10)
			req := httptest.NewRequest(http.MethodGet, "/scans/"+scanIDStr+"/findings?"+tt.query, nil)
			req.SetPathValue("scanID", scanIDStr)
			w := httptest.NewRecorder()

			h.ListFindings(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			if env.Error == nil || env.Error.Code != CodeInvalidInput {
				t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
			}
		})
	}
}

// TestListFindings_Pagination covers the ?limit=/?offset= query params on
// GET /scans/{scanID}/findings, orthogonal to the ?severity=/?status=
// filters covered by TestListFindings_QueryFilters.
func TestListFindings_Pagination(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	for i := 0; i < 3; i++ {
		fd := domain.Finding{
			ScanID: scan.ID, Title: fmt.Sprintf("Finding %d", i), Severity: domain.SeverityHigh,
			Status: domain.StatusOpen, FilePath: "a.go", LineNumber: i + 1,
		}
		if _, err := repo.CreateFinding(context.Background(), fd); err != nil {
			t.Fatalf("seed finding %d: %v", i, err)
		}
	}
	scanIDStr := strconv.FormatInt(scan.ID, 10)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{name: "explicit limit narrows the page", query: "?limit=2", wantStatus: http.StatusOK, wantCount: 2},
		{name: "non-numeric limit returns 400", query: "?limit=abc", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/scans/"+scanIDStr+"/findings"+tt.query, nil)
			req.SetPathValue("scanID", scanIDStr)
			w := httptest.NewRecorder()

			h.ListFindings(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			if tt.wantStatus != http.StatusOK {
				if env.Error == nil || env.Error.Code != CodeInvalidInput {
					t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
				}
				return
			}
			var findings []domain.Finding
			if err := json.Unmarshal(env.Data, &findings); err != nil {
				t.Fatalf("decode data: %v", err)
			}
			if len(findings) != tt.wantCount {
				t.Errorf("expected %d findings, got %d", tt.wantCount, len(findings))
			}
		})
	}
}

func TestGetFinding_Returns200WithFinding(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	created, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID: scan.ID, Title: "SQLi", Severity: domain.SeverityHigh,
		Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 10,
	})
	if err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/findings/"+idStr, nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.GetFinding(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var finding domain.Finding
	if err := json.Unmarshal(env.Data, &finding); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if finding.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, finding.ID)
	}
	if finding.Title != "SQLi" {
		t.Errorf("expected title 'SQLi', got %q", finding.Title)
	}
}

func TestGetFinding_NotFoundReturns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/findings/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.GetFinding(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

func TestCreateFinding_ValidInput_Returns201WithFinding(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)

	scanIDStr := strconv.FormatInt(scan.ID, 10)
	body := bytes.NewBufferString(`{"title":"SQL Injection","severity":"high","status":"open","file_path":"a.go","line_number":42}`)
	req := httptest.NewRequest(http.MethodPost, "/scans/"+scanIDStr+"/findings", body)
	req.SetPathValue("scanID", scanIDStr)
	w := httptest.NewRecorder()

	h.CreateFinding(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var finding domain.Finding
	if err := json.Unmarshal(env.Data, &finding); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if finding.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if finding.ScanID != scan.ID {
		t.Errorf("expected scan ID %d, got %d", scan.ID, finding.ScanID)
	}
	if finding.Title != "SQL Injection" {
		t.Errorf("expected title 'SQL Injection', got %q", finding.Title)
	}
	if finding.LineNumber != 42 {
		t.Errorf("expected line number 42, got %d", finding.LineNumber)
	}
}

func TestCreateFinding_NonexistentScan_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	// Scan 999 was never created, simulating the FK violation the real
	// repository maps to ErrNotFound (see postgres.go CreateFinding).
	body := bytes.NewBufferString(`{"title":"SQLi","severity":"high","status":"open","file_path":"a.go","line_number":1}`)
	req := httptest.NewRequest(http.MethodPost, "/scans/999/findings", body)
	req.SetPathValue("scanID", "999")
	w := httptest.NewRecorder()

	h.CreateFinding(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

func TestCreateFinding_InvalidSeverity_Returns400(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)

	scanIDStr := strconv.FormatInt(scan.ID, 10)
	body := bytes.NewBufferString(`{"title":"SQLi","severity":"extreme","status":"open","file_path":"a.go","line_number":1}`)
	req := httptest.NewRequest(http.MethodPost, "/scans/"+scanIDStr+"/findings", body)
	req.SetPathValue("scanID", scanIDStr)
	w := httptest.NewRecorder()

	h.CreateFinding(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

func TestCreateFinding_InvalidStatus_Returns400(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)

	scanIDStr := strconv.FormatInt(scan.ID, 10)
	body := bytes.NewBufferString(`{"title":"SQLi","severity":"high","status":"unknown","file_path":"a.go","line_number":1}`)
	req := httptest.NewRequest(http.MethodPost, "/scans/"+scanIDStr+"/findings", body)
	req.SetPathValue("scanID", scanIDStr)
	w := httptest.NewRecorder()

	h.CreateFinding(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

func TestUpdateFindingStatus_ValidInput_Returns200WithUpdatedFinding(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	created, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID: scan.ID, Title: "SQLi", Severity: domain.SeverityHigh,
		Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 10,
	})
	if err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	body := bytes.NewBufferString(`{"status":"resolved"}`)
	req := httptest.NewRequest(http.MethodPatch, "/findings/"+idStr, body)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.UpdateFindingStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	var finding domain.Finding
	if err := json.Unmarshal(env.Data, &finding); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if finding.Status != domain.StatusResolved {
		t.Errorf("expected status %q, got %q", domain.StatusResolved, finding.Status)
	}
}

func TestUpdateFindingStatus_InvalidStatus_Returns400(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	created, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID: scan.ID, Title: "SQLi", Severity: domain.SeverityHigh,
		Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 10,
	})
	if err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	body := bytes.NewBufferString(`{"status":"bogus"}`)
	req := httptest.NewRequest(http.MethodPatch, "/findings/"+idStr, body)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.UpdateFindingStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeInvalidInput {
		t.Errorf("expected code %q, got %+v", CodeInvalidInput, env.Error)
	}
}

func TestUpdateFindingStatus_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	body := bytes.NewBufferString(`{"status":"resolved"}`)
	req := httptest.NewRequest(http.MethodPatch, "/findings/999", body)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.UpdateFindingStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}

func TestDeleteFinding_Returns204NoContent(t *testing.T) {
	h, repo := newTestHandler()
	scan := seedScan(t, repo)
	created, err := repo.CreateFinding(context.Background(), domain.Finding{
		ScanID: scan.ID, Title: "SQLi", Severity: domain.SeverityHigh,
		Status: domain.StatusOpen, FilePath: "a.go", LineNumber: 10,
	})
	if err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodDelete, "/findings/"+idStr, nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.DeleteFinding(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
	if _, err := repo.GetFindingByID(context.Background(), created.ID); err != domain.ErrNotFound {
		t.Errorf("expected finding to be deleted, got err=%v", err)
	}
}

func TestDeleteFinding_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/findings/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	h.DeleteFinding(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil || env.Error.Code != CodeNotFound {
		t.Errorf("expected code %q, got %+v", CodeNotFound, env.Error)
	}
}
