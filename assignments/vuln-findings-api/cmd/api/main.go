// Command api starts the vuln-findings HTTP server: it connects to
// Postgres, wires the repository into the handlers, and serves the
// project, scan, and finding endpoints with graceful shutdown on
// SIGINT/SIGTERM.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/AnnaShera/go-assignments/assignments/vuln-findings-api/internal/handlers"
	"github.com/AnnaShera/go-assignments/assignments/vuln-findings-api/internal/repository"
)

func main() {
	log := newLogger(os.Getenv("LOG_LEVEL"))

	db, err := sql.Open("pgx", connString())
	if err != nil {
		log.Error("open database connection", "error", err)
		os.Exit(1)
	}

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(pingCtx)
	cancelPing()
	if err != nil {
		log.Error("ping database", "error", err)
		os.Exit(1)
	}

	repo := repository.NewPostgresRepository(db)
	defer func() {
		// pgRepository.Close closes the underlying *sql.DB; closing db
		// separately here would double-close it.
		if err := repo.Close(); err != nil {
			log.Error("close repository", "error", err)
		}
	}()

	h := handlers.NewHandler(repo, log)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects", h.WithLogging(h.ListProjects))
	mux.HandleFunc("GET /projects/{id}", h.WithLogging(h.GetProject))
	mux.HandleFunc("POST /projects", h.WithLogging(h.CreateProject))
	mux.HandleFunc("DELETE /projects/{id}", h.WithLogging(h.DeleteProject))

	mux.HandleFunc("GET /projects/{projectID}/scans", h.WithLogging(h.ListScans))
	mux.HandleFunc("POST /projects/{projectID}/scans", h.WithLogging(h.CreateScan))
	mux.HandleFunc("GET /scans/{id}", h.WithLogging(h.GetScan))
	mux.HandleFunc("DELETE /scans/{id}", h.WithLogging(h.DeleteScan))

	mux.HandleFunc("GET /scans/{scanID}/findings", h.WithLogging(h.ListFindings))
	mux.HandleFunc("POST /scans/{scanID}/findings", h.WithLogging(h.CreateFinding))
	mux.HandleFunc("GET /findings/{id}", h.WithLogging(h.GetFinding))
	mux.HandleFunc("PATCH /findings/{id}", h.WithLogging(h.UpdateFindingStatus))
	mux.HandleFunc("DELETE /findings/{id}", h.WithLogging(h.DeleteFinding))

	addr := ":" + getenvDefault("PORT", "8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	run(server, log)
}

// run starts server in the background and blocks until either it fails
// or a SIGINT/SIGTERM arrives, in which case it shuts down gracefully
// with a bounded timeout.
func run(server *http.Server, log *slog.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Info("starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		log.Info("server stopped")
	}
}

// connString builds the Postgres connection string. DATABASE_URL, if set,
// is used verbatim; otherwise it's assembled from the same POSTGRES_*
// variables docker-compose.yml and .env.example already define, plus
// POSTGRES_HOST (default 127.0.0.1, matching the compose port mapping).
func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	host := getenvDefault("POSTGRES_HOST", "127.0.0.1")
	port := getenvDefault("POSTGRES_PORT", "5432")
	user := getenvDefault("POSTGRES_USER", "vfa")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := getenvDefault("POSTGRES_DB", "vuln_findings")
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		user, password, dbname, host, port)
}

// getenvDefault returns the environment variable named key, or def if it
// is unset or empty.
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// newLogger builds a structured JSON slog.Logger at the level named by
// levelStr (debug/info/warn/error, case-insensitive); an unrecognized or
// empty value defaults to info.
func newLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
