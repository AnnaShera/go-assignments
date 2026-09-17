// Package handlers implements the HTTP boundary for the vuln-findings API:
// request parsing, response envelopes, and mapping domain errors to HTTP
// status codes.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
	"github.com/AnnaShera/vuln-findings-api/internal/repository"
)

// Error codes returned in the JSON error envelope. Clients should branch on
// these, not on the human-readable message.
const (
	CodeNotFound        = "not_found"
	CodeInvalidInput    = "invalid_input"
	CodeInternal        = "internal_error"
	CodeRequestTooLarge = "request_too_large"
)

// maxBodyBytes caps the size of a decoded request body. 1 MiB comfortably
// covers this API's small JSON payloads (project/scan/finding fields, no
// file uploads); an unauthenticated client sending an unbounded body would
// otherwise be able to drive memory/CPU exhaustion on a single request.
const maxBodyBytes = 1 << 20

// Handler-local sentinel errors for request-parsing failures that never
// reach the repository (bad path param, unparsable body). Kept private:
// callers only ever see the mapped HTTP response.
var (
	errInvalidID   = errors.New("invalid id")
	errInvalidBody = errors.New("invalid request body")
)

// validationErrors lists every domain sentinel that represents a client
// input problem rather than a not-found or internal condition. It is a
// flat slice rather than a marker interface or wrapped type because the
// domain package already defines these as plain sentinels (see
// STANDARDS.md "Error Handling" default); adding a type per error would be
// pure boilerplate for a fixed, small set.
var validationErrors = []error{
	domain.ErrProjectNameRequired,
	domain.ErrProjectIDRequired,
	domain.ErrScanIDRequired,
	domain.ErrInvalidSeverity,
	domain.ErrInvalidStatus,
	domain.ErrToolRequired,
	domain.ErrTitleRequired,
	domain.ErrFilePathRequired,
	domain.ErrInvalidLineNumber,
}

// isValidationError reports whether err is (or wraps) one of the known
// domain input-validation sentinels.
func isValidationError(err error) bool {
	for _, v := range validationErrors {
		if errors.Is(err, v) {
			return true
		}
	}
	return false
}

// mapError maps a domain/handler error to an HTTP status code, a stable
// machine-readable code, and a message safe to send to a client. Errors
// that aren't recognized as not-found or validation are treated as
// internal: the client gets a generic message, the real error is logged
// by the caller.
func mapError(err error) (status int, code string, message string) {
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, CodeNotFound, "resource not found"
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge, CodeRequestTooLarge, "request body too large"
	case errors.Is(err, errInvalidID), errors.Is(err, errInvalidBody), isValidationError(err):
		return http.StatusBadRequest, CodeInvalidInput, err.Error()
	default:
		return http.StatusInternalServerError, CodeInternal, "internal server error"
	}
}

// envelope is the consistent JSON response shape for every handler:
// exactly one of Data or Error is populated.
type envelope struct {
	Data  any        `json:"data"`
	Error *errorBody `json:"error"`
}

// errorBody is the error object nested in envelope.Error.
type errorBody struct {
	Message string `json:"error"`
	Code    string `json:"code"`
}

// Repository is the persistence dependency the handlers package needs:
// projects, scans, and findings, now that all three have handlers.
// Depending on this interface rather than the full repository.Repository
// keeps the package boundary honest about what it actually uses, and lets
// tests satisfy it with a small fake instead of implementing a Close
// method it doesn't exercise.
type Repository interface {
	repository.ProjectRepository
	repository.ScanRepository
	repository.FindingRepository
}

// Handler holds the dependencies HTTP handlers need: a repository and a
// structured logger.
type Handler struct {
	repo Repository
	log  *slog.Logger
}

// NewHandler returns a Handler backed by repo, logging through log.
func NewHandler(repo Repository, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

// writeJSON writes status and body as the response, setting the JSON
// content type. Encode failures are logged rather than ignored (the
// header and status are already sent, so there's nothing left to do for
// the caller) but never panic or leak into the response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.Error("encode response failed", "error", err)
	}
}

// writeData writes a success envelope wrapping data at the given status.
func (h *Handler) writeData(w http.ResponseWriter, status int, data any) {
	h.writeJSON(w, status, envelope{Data: data})
}

// writeError maps err to a status/code/message via mapError, logs it at a
// level matching severity (server errors at Error, client errors at
// Warn), and writes the error envelope. This is the single call site
// STANDARDS.md's "HTTP Error Response Contract" asks for, so every
// endpoint maps errors identically.
func (h *Handler) writeError(w http.ResponseWriter, err error) {
	status, code, message := mapError(err)
	if status == http.StatusInternalServerError {
		h.log.Error("request failed", "error", err)
	} else {
		h.log.Warn("request failed", "error", err, "code", code)
	}
	h.writeJSON(w, status, envelope{Error: &errorBody{Message: message, Code: code}})
}

// parseID parses a path parameter as a project/scan/finding ID, returning
// errInvalidID (mapped to 400 invalid_input) rather than the raw
// strconv error, which would leak parser internals to the client.
func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, errInvalidID
	}
	return id, nil
}

// decodeJSON decodes r's body into dst, capped at maxBodyBytes via
// http.MaxBytesReader so an unauthenticated client can't drive memory/CPU
// exhaustion with an oversized body. A body over the limit surfaces as
// *http.MaxBytesError (mapped to 413 request_too_large by mapError) and is
// returned as-is so errors.As can detect it there; any other decode
// failure returns errInvalidBody (mapped to 400 invalid_input) rather than
// the raw json error, which would leak decoder internals to the client.
// Every handler that accepts a JSON body (CreateProject, CreateScan,
// CreateFinding, UpdateFindingStatus) calls this instead of decoding
// inline, per STANDARDS.md's single-mapping-point default for the error
// contract.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return err
		}
		return errInvalidBody
	}
	return nil
}

// statusRecorder wraps http.ResponseWriter to capture the status code
// written, so withLogging can report it after the handler returns control.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records status before delegating, so a handler that never
// calls WriteHeader explicitly (relying on the implicit 200) still gets
// logged correctly via the http.ResponseWriter zero-value default.
func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// WithLogging wraps next to log method, path, status, and latency at info
// level once the request completes, per STANDARDS.md's logging default.
// Exported so route registration in cmd/api/main.go can apply it per route.
func (h *Handler) WithLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		h.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"latency", time.Since(start),
		)
	}
}

// createProjectRequest is the JSON body accepted by CreateProject.
type createProjectRequest struct {
	Name string `json:"name"`
}

// ListProjects handles GET /projects.
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.repo.ListProjects(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, projects)
}

// GetProject handles GET /projects/{id}.
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	project, err := h.repo.GetProjectByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, project)
}

// CreateProject handles POST /projects.
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	project, err := h.repo.CreateProject(r.Context(), req.Name)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusCreated, project)
}

// DeleteProject handles DELETE /projects/{id}.
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.repo.DeleteProject(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createScanRequest is the JSON body accepted by CreateScan.
type createScanRequest struct {
	Tool string `json:"tool"`
}

// ListScans handles GET /projects/{projectID}/scans.
func (h *Handler) ListScans(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r.PathValue("projectID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	scans, err := h.repo.ListScansByProject(r.Context(), projectID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, scans)
}

// GetScan handles GET /scans/{id}.
func (h *Handler) GetScan(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	scan, err := h.repo.GetScanByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, scan)
}

// CreateScan handles POST /projects/{projectID}/scans.
func (h *Handler) CreateScan(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r.PathValue("projectID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req createScanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	scan, err := h.repo.CreateScan(r.Context(), projectID, req.Tool)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusCreated, scan)
}

// DeleteScan handles DELETE /scans/{id}.
func (h *Handler) DeleteScan(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.repo.DeleteScan(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createFindingRequest is the JSON body accepted by CreateFinding.
type createFindingRequest struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Status     string `json:"status"`
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
}

// updateFindingStatusRequest is the JSON body accepted by
// UpdateFindingStatus.
type updateFindingStatusRequest struct {
	Status string `json:"status"`
}

// parseFindingFilter reads the optional ?severity= and ?status= query
// params off r and validates any non-empty value against the domain's
// known enums. An invalid value is rejected as domain.ErrInvalidSeverity /
// domain.ErrInvalidStatus (mapped to 400 by mapError via validationErrors)
// rather than silently passed through to become a filter that matches
// nothing.
func parseFindingFilter(r *http.Request) (repository.FindingFilter, error) {
	q := r.URL.Query()

	severity := q.Get("severity")
	if severity != "" && !domain.IsValidSeverity(severity) {
		return repository.FindingFilter{}, domain.ErrInvalidSeverity
	}

	status := q.Get("status")
	if status != "" && !domain.IsValidStatus(status) {
		return repository.FindingFilter{}, domain.ErrInvalidStatus
	}

	return repository.FindingFilter{Severity: severity, Status: status}, nil
}

// ListFindings handles GET /scans/{scanID}/findings, optionally narrowed by
// the ?severity= and/or ?status= query params.
func (h *Handler) ListFindings(w http.ResponseWriter, r *http.Request) {
	scanID, err := parseID(r.PathValue("scanID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	filter, err := parseFindingFilter(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	findings, err := h.repo.ListFindingsByScan(r.Context(), scanID, filter)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, findings)
}

// GetFinding handles GET /findings/{id}.
func (h *Handler) GetFinding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	finding, err := h.repo.GetFindingByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, finding)
}

// CreateFinding handles POST /scans/{scanID}/findings.
func (h *Handler) CreateFinding(w http.ResponseWriter, r *http.Request) {
	scanID, err := parseID(r.PathValue("scanID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req createFindingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	finding := domain.Finding{
		ScanID:     scanID,
		Title:      req.Title,
		Severity:   req.Severity,
		Status:     req.Status,
		FilePath:   req.FilePath,
		LineNumber: req.LineNumber,
	}
	created, err := h.repo.CreateFinding(r.Context(), finding)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusCreated, created)
}

// UpdateFindingStatus handles PATCH /findings/{id}. On success it returns
// the finding as it stands after the update, saving the client a
// follow-up GET.
func (h *Handler) UpdateFindingStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req updateFindingStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.repo.UpdateFindingStatus(r.Context(), id, req.Status); err != nil {
		h.writeError(w, err)
		return
	}

	finding, err := h.repo.GetFindingByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeData(w, http.StatusOK, finding)
}

// DeleteFinding handles DELETE /findings/{id}.
func (h *Handler) DeleteFinding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.repo.DeleteFinding(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
