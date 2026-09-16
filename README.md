# vuln-findings-api

A REST API modeling how SAST/scanning tools structure results, built as
interview-prep practice — not a real product. The goal is to practice
clean REST design, correct HTTP semantics (PATCH vs PUT), and SQL
relationships in Go.

## Domain model

```
Project  1---* Scan  1---* Finding
```

- **Project** — a codebase being scanned.
- **Scan** — one run of a tool against a project.
- **Finding** — a single issue a scan turned up: severity, status,
  file path, and line number, mirroring real SAST output.

See [`migrations/0001_init.sql`](migrations/0001_init.sql) for the exact
schema (foreign keys, cascade deletes, and constraints on `severity` /
`status`).

## Planned API

| Method | Path                | Notes                              |
|--------|---------------------|-------------------------------------|
| POST   | `/projects`         |                                     |
| GET    | `/projects`         |                                     |
| POST   | `/projects/{id}/scans` |                                  |
| GET    | `/projects/{id}/scans` |                                  |
| POST   | `/scans/{id}/findings` |                                  |
| GET    | `/scans/{id}/findings` | supports `?severity=` and `?status=` filters |
| PATCH  | `/findings/{id}`    | partial update (e.g. status change) |
| DELETE | `/findings/{id}`    |                                     |

## Stack

- **Go** (stdlib `net/http`, 1.22+ method+path routing — no router
  dependency)
- **Postgres**, run via Docker Compose
- Manual testing with Postman (no automated API client yet)

See [`STANDARDS.md`](STANDARDS.md) for the full set of technical
defaults and tradeoffs this repo follows.

## Status

Schema, `docker-compose.yml`, the Go module skeleton, and the
repository layer are done. Project HTTP handlers (list/get/create/delete)
are implemented and wired up in `cmd/api/main.go`; scan and finding
handlers are not yet implemented.

## Running locally

1. Copy the env template and set a real password:
   ```
   cp .env.example .env
   ```
2. Start Postgres:
   ```
   docker compose up -d
   ```
3. Apply the schema:
   ```
   docker compose exec -T postgres psql -U vfa -d vuln_findings < migrations/0001_init.sql
   ```
4. Run the API:
   ```
   go run ./cmd/api
   ```
   Listens on `:8080`.
