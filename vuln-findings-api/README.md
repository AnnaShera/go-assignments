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

## API

| Method | Path                              | Notes                                        |
|--------|-----------------------------------|-----------------------------------------------|
| GET    | `/projects`                       | supports `?limit=`/`?offset=` paging         |
| POST   | `/projects`                       |                                               |
| GET    | `/projects/{id}`                  |                                               |
| DELETE | `/projects/{id}`                  | cascades to the project's scans and findings |
| GET    | `/projects/{projectID}/scans`     | supports `?limit=`/`?offset=` paging         |
| POST   | `/projects/{projectID}/scans`     |                                               |
| GET    | `/scans/{id}`                     |                                               |
| DELETE | `/scans/{id}`                     | cascades to the scan's findings              |
| GET    | `/scans/{scanID}/findings`        | supports `?severity=`/`?status=` filters and `?limit=`/`?offset=` paging |
| POST   | `/scans/{scanID}/findings`        |                                               |
| GET    | `/findings/{id}`                  |                                               |
| PATCH  | `/findings/{id}`                  | partial update (e.g. status change)          |
| DELETE | `/findings/{id}`                  |                                               |

### Pagination

`GET /projects`, `GET /projects/{projectID}/scans`, and
`GET /scans/{scanID}/findings` accept `?limit=` and `?offset=`. Omitting
either applies the default (`limit=50`); a `limit` above the max page
size (200) is silently clamped down rather than rejected. A non-numeric
or negative `limit`/`offset` value returns `400 invalid_input` — that's
a client mistake, unlike an over-cap limit, which is a reasonable clamp.

### Request size limit

Every request body is capped at 1 MiB (`http.MaxBytesReader`), enough
for this API's small JSON payloads. A body over the limit returns
`413` with error code `request_too_large`.

## Stack

- **Go** (stdlib `net/http`, 1.22+ method+path routing — no router
  dependency)
- **Postgres**, run via Docker Compose
- Manual testing with Postman (no automated API client yet)

See [`../STANDARDS.md`](../STANDARDS.md) for the full set of technical
defaults and tradeoffs this project follows.

## Status

Schema, `docker-compose.yml`, the Go module skeleton, and the
repository layer are done. Project, scan, and finding HTTP handlers
(list/get/create/delete, plus `?severity=`/`?status=` filtering on
`GET /scans/{id}/findings`, `?limit=`/`?offset=` paging on all three
list endpoints, and a 1 MiB request body cap) are implemented and
wired up in `cmd/api/main.go`. CRUD for all three resources —
projects, scans, and findings — is complete.

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
