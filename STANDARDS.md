# Engineering Standards & Decision Reference (Go)

This file captures recurring technical decisions with their tradeoffs.
Before designing any solution, read this file and pick the option that
fits the current assignment. Log the confirmed choice in the
per-assignment `docs/DECISIONS.md`.

---

## Universal Questions Checklist

Always include applicable questions from this list in `docs/QUESTIONS.md` during intake.

### Complexity & Scale
- What is the expected input size / volume / request rate?
- Are there memory constraints?
- Are there latency or throughput requirements?
- Should the solution optimize for time, space, or readability, or balance all three?

### Correctness & Edge Cases
- How should invalid input be handled: return an error, a zero value, or panic?
- Are there numeric edge cases: zero, negative numbers, float precision, overflow, integer division?
- Is the input guaranteed non-empty, or must empty/nil be handled?
- Are there ordering guarantees on the input, or must the solution handle arbitrary order?

### Concurrency
- Will this run single-goroutine, or could it be called concurrently?
- Is there shared state? Does it need a mutex, a channel, or can it be avoided by design?
- Does any operation need a timeout or cancellation via `context.Context`?

### Persistence
- Should any state survive between calls or runs?
- Is there a storage requirement: in-memory map, file, real database?

### Output
- Is there an exact output format required (JSON field names, status codes, sorting)?
- Should errors be surfaced to the caller (HTTP status + body) or logged only?

### API Surface (HTTP assignments)
- Is authentication or authorization expected, or explicitly out of scope?
- Do list endpoints need pagination, filtering, or sorting? What happens with a very large result?
- Must any write be idempotent (e.g. a retried POST must not create duplicates)?
- Is there an API versioning expectation (`/v1/...`)?

### Scope & Delivery
- What is the time budget, and what matters most if it runs out?
- What exactly does the reviewer receive: repo link, Dockerfile, run instructions, a demo?
- How will the reviewer run it (plain `go run`, `docker compose up`, a specific Go version)?
- Which external dependencies are allowed, if any?

---

## Project Layout

**Default for this repo:**

```
assignments/<name>/
├── cmd/<binary>/main.go   wiring only: config, dependencies, server start/stop
├── internal/<package>/    all application code; not importable from outside the module
├── migrations/            versioned SQL, if there is a database
├── docs/                  QUESTIONS, DECISIONS, DESIGN, DEBRIEF
└── go.mod                 module github.com/AnnaShera/go-assignments/assignments/<name>
```

- `main.go` wires things together and holds no business logic, so everything
  worth testing lives in `internal/`.
- Name packages after what they provide (`handlers`, `repository`, `domain`),
  never `utils`, `common`, or `helpers`.
- Don't create a package for a single small type. Split when a package
  starts doing two jobs, not before.
- Skip `pkg/`. For a single-module assignment it adds a level of nesting and
  says nothing.

---

## Error Handling

### Options

#### 1. Sentinel errors (`errors.New`, package-level `var`)
**Best for:** a small fixed set of known error conditions callers need to check with `errors.Is`

**Pros:** simple, stdlib-only, cheap to compare
**Cons:** no room for context (a message, a field name); doesn't scale past a handful of cases

#### 2. Custom error types (struct implementing `error`)
**Best for:** domain errors that carry data (which field failed, which ID was not found), checked with `errors.As`

**Pros:** carries structured context; callers can branch on type; good for HTTP handlers mapping errors to status codes
**Cons:** more boilerplate than a sentinel

#### 3. Wrapped errors (`fmt.Errorf("...: %w", err)`)
**Best for:** adding context as an error propagates up the call stack, while preserving the original for `errors.Is`/`errors.As`

**Pros:** idiomatic Go; preserves the chain; cheap
**Cons:** none really, this is close to mandatory practice once an error crosses more than one function boundary

#### 4. Panic
**Best for:** programmer errors only (nil pointer you control, invariant violation that should never happen). Never for expected failure paths like bad user input or a missing DB row.

**Default for this repo:** sentinel errors for known conditions, wrapped with `%w` as they propagate, custom error types at the boundary where a caller (HTTP handler, CLI) needs to distinguish "not found" from "invalid input" from "internal error."

---

## Data Modeling

### Options

#### 1. Plain structs with JSON tags
**Best for:** everything by default. `type Order struct { ID string \`json:"id"\`; ... }`

**Pros:** zero dependency, idiomatic, works with `encoding/json` out of the box
**Cons:** none for most assignments

#### 2. Struct + manual `Validate() error` method
**Best for:** structs that need validation beyond what JSON unmarshaling gives you (required fields, ranges, cross-field rules)

**Pros:** explicit, no dependency, easy to explain in an interview
**Cons:** boilerplate scales linearly with field count

#### 3. `go-playground/validator` (struct tags)
**Best for:** many fields with standard rules (required, min, max, email, oneof), API-boundary structs

**Pros:** declarative, less boilerplate at scale, industry-common
**Cons:** external dependency; tag-based validation is less discoverable than reading a `Validate()` method

**Default for this repo:** plain struct, manual `Validate() error` unless the assignment explicitly rewards showing dependency judgment, then justify pulling in `validator`.

---

## Input Validation

Same three options as Data Modeling above. For single-value validation (not a whole struct), just check inline at the boundary:

```go
if amount < 0 {
    return fmt.Errorf("amount must be non-negative, got %d", amount)
}
```

Don't build a validation abstraction for a single check.

---

## HTTP Routing / Web Framework

### Options

#### 1. `net/http` stdlib only (with 1.22+ method+path patterns)
**Best for:** most interview assignments. `mux.HandleFunc("GET /orders/{id}", handler)` covers path params and method routing natively since Go 1.22.

**Pros:** zero dependency, shows you know the stdlib, nothing to explain/justify to an interviewer
**Cons:** no middleware chaining helpers, no built-in request binding/validation glue

#### 2. `chi`
**Best for:** when you want middleware composition (logging, recovery, auth) without a full framework's opinions

**Pros:** thin, idiomatic, plays well with stdlib `http.Handler`
**Cons:** external dependency, needs justification if the assignment says "no dependencies"

#### 3. `gin` / `echo`
**Best for:** rarely, for an interview assignment. They bring their own conventions (binding, context object) that can read as "doesn't know plain Go" to a reviewer grading fundamentals.

**Pros:** fast to write
**Cons:** obscures whether you understand `net/http` underneath; risky choice for an interview unless the JD explicitly lists one of these

**Default for this repo:** stdlib `net/http` with 1.22+ routing. Reach for `chi` only if you need middleware and can justify it in thirty seconds if asked.

---

## HTTP Server Hardening

A bare `http.ListenAndServe(addr, mux)` has no timeouts, so one slow client
can hold a connection forever. `gosec` flags it, and reviewers do too.

**Default for this repo:**

- Build an explicit `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`,
  `WriteTimeout`, and `IdleTimeout` set.
- Cap request bodies with `http.MaxBytesReader` before decoding JSON.
- Decode with `json.NewDecoder(r.Body)` and call `DisallowUnknownFields()`
  when the contract is strict, so a typoed field fails loudly instead of
  being silently ignored.
- Shut down gracefully: `signal.NotifyContext` for SIGINT/SIGTERM, then
  `server.Shutdown(ctx)` with a timeout so in-flight requests finish.
- Expose `GET /healthz` returning `200` once the server is ready, and
  check the DB connection if there is one.

---

## API Design

**Default for this repo:**

- **Status codes:** `201 Created` with the created resource (and a
  `Location` header) for POSTs that create, `200` for reads and updates,
  `204 No Content` for deletes, `400` for validation, `404` for not found,
  `409` for conflicts, `500` for anything unexpected.
- **Pagination:** any list endpoint that can grow is paginated from day one
  (`?limit=&offset=` is fine for an assignment; cursor-based if the brief
  mentions large data). Enforce a max `limit`.
- **JSON naming:** `snake_case` field names, consistent across every
  endpoint. Timestamps as RFC 3339 in UTC.
- **Idempotency:** PUT and DELETE are idempotent by definition. If a POST
  must be safe to retry, say how in `docs/DECISIONS.md` (idempotency key or
  a natural unique constraint).
- **Versioning:** no `/v1` prefix unless the brief asks for it. Mention in
  DECISIONS where it would go.

---

## Authentication & Authorization

**Default for this repo:** no auth implemented unless the brief asks for it, but
document the decision explicitly in `docs/DECISIONS.md`: where JWT middleware would
sit (before the handler, via `net/http` middleware chaining), and which endpoints
would need role checks (typically destructive deletes and admin-only writes). An
interviewer will ask "how would you secure this" — having the answer ready is the
point, not building it.

---

## HTTP Error Response Contract

All handler errors return a consistent JSON envelope:
```go
{"error": "human-readable message", "code": "not_found"}
```
Map custom error types (from Error Handling above) to status codes at the handler
boundary: not-found → 404, validation → 400, internal → 500 (message generic,
details logged via slog, never leaked to the client).

**Default for this repo:** one `writeError(w, err)` helper in the handlers package
doing this mapping, so every endpoint stays consistent without repeating switch
statements.

---

## Configuration

**Default for this repo:**

- Read config from environment variables, once, in `main`, into a typed
  `Config` struct. Pass values down explicitly. No package reads `os.Getenv`
  on its own.
- Fail fast: a missing required variable stops startup with a clear error
  instead of surfacing later as a confusing connection failure.
- Defaults only for values that are safe to default (port, timeouts), never
  for credentials.
- Commit a `.env.example` per assignment listing every variable with dummy
  values. The real `.env` stays gitignored.
- No config library (`viper`, `envconfig`) unless there are enough variables
  that parsing by hand gets noisy.

---

## Database Access

### Options

#### 1. `database/sql` + `pgx` driver, hand-written SQL
**Best for:** most interview assignments. `pgx` as the driver under `database/sql`, or its native `pgxpool` for connection pooling and better type support (native `time.Time`, arrays, etc.).

**Pros:** no ORM magic, every query is visible and explainable, closest to what a fundamentals-focused reviewer wants to see
**Cons:** manual `Scan()` into each struct field, manual transaction handling (`Begin`/`Commit`/`Rollback`)

#### 2. `sqlx`
**Best for:** reducing struct-scanning boilerplate once row-to-struct mapping repeats across several resources

**Pros:** `StructScan`/`Get`/`Select` convenience on top of the same raw SQL, still fully explainable, thin enough to justify in seconds if asked
**Cons:** external dependency

#### 3. GORM
**Best for:** rarely, for an interview assignment

**Pros:** less code to write
**Cons:** hides the SQL and the relations/cascade behavior behind magic, which directly undercuts a reviewer's ability to see that you understand foreign keys, constraints, and cascade deletes

**Default for this repo:** `database/sql` + `pgx`. Move to `sqlx` only if scanning across resources gets genuinely repetitive, don't reach for it preemptively.

### Migrations

**Default:** `golang-migrate`, versioned SQL files (`0001_init.sql`, `0002_...sql`). Simple CLI, works cleanly with `docker compose` startup, no need to justify it in an interview since it's the de facto standard.

---

## Concurrency

### Options

#### 1. Synchronous (default)
**Best for:** most assignments, including most REST handlers. One request, one goroutine (the stdlib server already does this for you), no manual concurrency needed.

#### 2. Goroutines + `sync.WaitGroup`
**Best for:** fan-out work within a single request (e.g. call three downstream things concurrently, wait for all)

**Pros:** straightforward, well understood, easy to reason about
**Cons:** needs care with shared state (`sync.Mutex` or channels), easy to leak goroutines if you forget cancellation

#### 3. Channels for coordination
**Best for:** producer/consumer patterns, worker pools, pipeline stages

**Pros:** idiomatic Go, "share memory by communicating"
**Cons:** overkill for a single request/response; adds real complexity, don't reach for it unless the problem is actually concurrent

#### 4. `context.Context` for cancellation/timeout
**Best for:** any operation that could hang: DB call, downstream HTTP call, long computation. Should be threaded through from the HTTP handler down.

**Default for this repo:** synchronous by default. Add goroutines only when the assignment has genuine concurrent work (multiple independent I/O calls). Always pass `context.Context` as the first param on anything that does I/O, even if you don't use it yet, it's expected idiom and costs nothing.

---

## Logging Strategy

### Options

#### 1. No logging (default for pure algorithm assignments)

#### 2. `log/slog` (stdlib, Go 1.21+)
**Best for:** any assignment with meaningful runtime state, request handling, or error paths worth tracing

**Pros:** stdlib, structured (key-value, JSON-capable), no dependency, this is the modern idiomatic default
**Cons:** none, this replaced the old `log` package as the default choice

#### 3. `zerolog` / `zap`
**Best for:** when you specifically want to demonstrate familiarity with production-grade logging libraries

**Pros:** very fast, widely used in real backend codebases
**Cons:** external dependency; `slog` covers 95% of the same need now without one

**Default for this repo:** `log/slog`, structured, at minimum on every HTTP request (method, path, status, latency) and every error path.

---

## Testing Patterns

### Options

#### 1. Table-driven tests (default)
**Best for:** almost everything. A slice of `{name, input, want}` structs run through `t.Run` subtests.

**Pros:** DRY, scales cleanly, each case shows individually in test output, this is the Go idiom interviewers expect to see
**Cons:** none

#### 2. `httptest` for HTTP handlers
**Best for:** testing REST endpoints without a real network call. `httptest.NewRequest` + `httptest.NewRecorder`, call the handler directly.

**Pros:** fast, no server needed, stdlib
**Cons:** none, this is the standard way to test Go HTTP handlers

#### 3. `testify` (`assert`/`require`)
**Best for:** reducing boilerplate on assertions, especially comparing structs

**Pros:** more readable failure messages than raw `if got != want`
**Cons:** external dependency; plain stdlib comparisons are fine and dependency-free if the assignment favors minimal deps

#### 4. Interfaces + hand-written fakes for mocking
**Best for:** isolating a handler from a dependency (DB, downstream client) by depending on a small interface

**Pros:** idiomatic Go (accept interfaces, return structs), no mocking framework needed for small assignments
**Cons:** for large interfaces, consider `gomock`/`mockgen` instead of hand-writing every fake

**Default for this repo:** table-driven tests everywhere, `httptest` for handlers, stdlib assertions unless the struct comparisons get unwieldy (then `testify/assert`), hand-written fakes over a mocking framework unless the interface is large.

---

## Dependency Policy

**Default for this repo:** stdlib first. Every third-party module in `go.mod`
gets one line in `docs/DECISIONS.md` saying what it replaces and why the
stdlib wasn't enough. "It's popular" isn't a reason; "hand-rolling this would
be 200 lines of error-prone code" is.

Pre-approved when the need is real: `pgx` (Postgres driver), `golang-migrate`
(migrations), `testify` (assertions, only if comparisons get unwieldy).

---

## Tooling & Quality Gates

The gate is enforced by `.claude/hooks/pre-push-check.sh`, which runs whenever
Claude Code runs `git push`. Keep this section and the hook in sync.

| Check | Scope | Blocks push |
|---|---|---|
| `gofmt -l .` | whole repo | yes |
| `go test ./...` | each module under `assignments/*/` | yes |
| `golangci-lint run ./...` | each module | yes |
| `golangci-lint run --build-tags=integration ./...` | each module | yes |

Known gaps, deliberately left for now:

- **No `-race`.** The race detector needs cgo, which this Windows machine
  doesn't have set up. Run `go test -race ./...` on Linux or in CI before
  submitting anything concurrent.
- **Integration tests are linted but not run** by the gate, because they
  need Docker up. Run them by hand before submitting.
- **golangci-lint uses its default linter set** until a `.golangci.yml` is
  added.
- **The gate only covers pushes made through Claude Code.** A push from a
  plain terminal isn't checked.

---

**Default approach:** Log decisions in per-assignment `docs/DECISIONS.md` after confirming context-appropriate choices from this reference. When in doubt, prefer idiomatic Go and explicit code over clever abstractions.
