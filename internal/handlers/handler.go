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

	"github.com/AnnaShera/vuln-findings-api/internal/domain"
	"github.com/AnnaShera/vuln-findings-api/internal/repository"
)

// Error codes returned in the JSON error envelope. Clients should branch on
// these, not on the human-readable message.
const (
	CodeNotFound     = "not_found"
	CodeInvalidInput = "invalid_input"
	CodeInternal     = "internal_error"
)

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
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, CodeNotFound, "resource not found"
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

// Repository is the persistence dependency the handlers package needs.
// It is scoped to project operations today; scan/finding handlers can
// widen it later. Depending on this narrow interface rather than the full
// repository.Repository keeps the package boundary honest about what it
// actually uses, and lets tests satisfy it with a small fake instead of
// implementing scan/finding/Close methods they don't exercise.
type Repository interface {
	repository.ProjectRepository
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

// createProjectRequest is the JSON body accepted by CreateProject.
type createProjectRequest struct {
	Name string `json:"name"`
}

// ListProjects handles GET /projects.
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.repo.ListProjects(r.Context())
	if err != nil {
		status, code, message := mapError(err)
		h.log.Error("list projects failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(envelope{Data: projects}); err != nil {
		h.log.Error("encode response failed", "error", err)
	}
}

// GetProject handles GET /projects/{id}.
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		status, code, message := mapError(errInvalidID)
		h.log.Warn("get project failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	project, err := h.repo.GetProjectByID(r.Context(), id)
	if err != nil {
		status, code, message := mapError(err)
		if status == http.StatusInternalServerError {
			h.log.Error("get project failed", "error", err)
		} else {
			h.log.Warn("get project failed", "error", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(envelope{Data: project}); err != nil {
		h.log.Error("encode response failed", "error", err)
	}
}

// CreateProject handles POST /projects.
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, code, message := mapError(errInvalidBody)
		h.log.Warn("create project failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	project, err := h.repo.CreateProject(r.Context(), req.Name)
	if err != nil {
		status, code, message := mapError(err)
		if status == http.StatusInternalServerError {
			h.log.Error("create project failed", "error", err)
		} else {
			h.log.Warn("create project failed", "error", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(envelope{Data: project}); err != nil {
		h.log.Error("encode response failed", "error", err)
	}
}

// DeleteProject handles DELETE /projects/{id}.
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		status, code, message := mapError(errInvalidID)
		h.log.Warn("delete project failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	if err := h.repo.DeleteProject(r.Context(), id); err != nil {
		status, code, message := mapError(err)
		if status == http.StatusInternalServerError {
			h.log.Error("delete project failed", "error", err)
		} else {
			h.log.Warn("delete project failed", "error", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encErr := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Message: message, Code: code}}); encErr != nil {
			h.log.Error("encode error response failed", "error", encErr)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
