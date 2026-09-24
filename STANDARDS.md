# Engineering Standards & Decision Reference (Go)

This file holds the technical decisions that come up in every assignment,
with their trade-offs and one **default** each. `CLAUDE.md` covers *how we
work*; this file covers *what we build with*.

**How to use it:**

1. During intake, pull the applicable questions from the checklist below into
   `docs/QUESTIONS.md`.
2. During standards review, take the default for every topic that applies.
3. Deviate only when the brief or an intake answer gives a concrete reason,
   and record it in `docs/DECISIONS.md` in this form:
   `**Topic:** chose X instead of the default Y, because Z.`
   Deviations are fine. Unexplained ones are what a reviewer marks down.

---

## Defaults at a Glance

| Topic | Default |
|---|---|
| Layout | One module per assignment, `cmd/` for wiring, `internal/` for everything else |
| Errors | Domain sentinels + `ValidationError`, wrapped with `%w`, mapped to HTTP in one place |
| Modeling & validation | Plain structs, `Validate() error`, money as integer minor units, time in UTC |
| Configuration | Env vars parsed once in `main` into a typed `Config`, fail fast |
| Concurrency | Synchronous; `context.Context` first on anything that does I/O |
| Logging | `log/slog` JSON to stdout; log once, where the error is handled |
| Routing | `net/http` with Go 1.22+ method and path patterns |
| Server | Explicit `http.Server` timeouts, body size cap, graceful shutdown, `/healthz` |
| API | `snake_case` JSON, `201`/`204` where they apply, paginated lists, one error envelope |
| Auth | Not built unless the brief asks; where it would go is documented |
| Security | Parameterized SQL only, no secrets in code or logs, no internals in error responses |
| Database | `database/sql` + `pgx/v5/stdlib`, hand-written SQL, `golang-migrate` |
| Testing | Table-driven, `httptest` through the router, hand-written fakes, real deps behind `integration` tag |
| Dependencies | Stdlib first; each module justified in one line in DECISIONS |

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
├── .env.example           every config variable with a dummy value, if there is config
└── go.mod                 module github.com/AnnaShera/go-assignments/assignments/<name>
```

- `main.go` wires things together and holds no business logic, so everything
  worth testing lives in `internal/`.
- Name packages after what they provide (`handlers`, `repository`, `domain`),
  never `utils`, `common`, or `helpers`.
- Dependencies point inward: `handlers` → `domain` ← `repository`. The
  `domain` package imports neither of them, and nothing imports `handlers`
  except `main`.
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

**Default for this repo:**

- The `domain` package owns the error vocabulary: sentinels for fixed
  conditions (`ErrNotFound`, `ErrConflict`) and one `ValidationError`
  type carrying the field and the reason.
- The repository translates driver errors into that vocabulary, so nothing
  above it ever sees a `pgconn.PgError` or `sql.ErrNoRows`:
  `sql.ErrNoRows` → `ErrNotFound`, unique violation (`23505`) → `ErrConflict`,
  foreign key violation (`23503`) → `ErrNotFound` or `ValidationError`,
  depending on which side of the relation the caller controls.
- Every layer wraps with `%w` and adds what it was doing:
  `fmt.Errorf("get scan %d: %w", id, err)`.
- The HTTP layer maps errors to responses in exactly one place (see
  [HTTP Error Response Contract](#http-error-response-contract)).
- An error is either handled (logged, turned into a response) or returned,
  never both. Logging and returning produces the same failure three times in
  the logs.

---

## Data Modeling & Validation

### Options

#### 1. Plain structs with JSON tags
**Best for:** everything by default. `type Order struct { ID int64 \`json:"id"\`; ... }`

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

**Default for this repo:** plain structs with a manual `Validate() error` that
returns a `ValidationError`. Pull in `validator` only when the field count
makes hand-written checks noisy, and justify it under
[Dependency Policy](#dependency-policy).

- Validate at the boundary, once, before anything is persisted. Code past
  the boundary trusts its input.
- Normalize before validating (trim whitespace, lower-case what's
  case-insensitive), so `"  "` is rejected as empty rather than stored.
- For a single value, check inline. Don't build a validation abstraction for
  one check:

  ```go
  if amount < 0 {
      return fmt.Errorf("amount must be non-negative, got %d", amount)
  }
  ```

### Modeling traps reviewers look for

- **Money:** `int64` minor units (cents), never `float64`. `0.1 + 0.2 != 0.3`
  in floating point, and a reviewer will check.
- **Time:** `time.Time` in UTC everywhere, `TIMESTAMPTZ` in Postgres, RFC 3339
  in JSON. Code that reads the current time takes it as a dependency
  (`now func() time.Time`) so tests are deterministic.
- **IDs:** generated by the database (`BIGSERIAL` or `gen_random_uuid()`), not
  by the client, unless the brief says otherwise.
- **Enums:** a named string type with a fixed set of constants and a
  validity check (`type Severity string`), not a bare `string` accepted from
  the request.

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

## Concurrency

### Options

#### 1. Synchronous (default)
**Best for:** most assignments, including most REST handlers. One request, one goroutine (the stdlib server already does this for you), no manual concurrency needed.

#### 2. Goroutines + `sync.WaitGroup` / `errgroup`
**Best for:** fan-out work within a single request (e.g. call three downstream things concurrently, wait for all). Use `errgroup` when any failure should cancel the rest.

**Pros:** straightforward, well understood, easy to reason about
**Cons:** needs care with shared state (`sync.Mutex` or channels), easy to leak goroutines if you forget cancellation

#### 3. Channels for coordination
**Best for:** producer/consumer patterns, worker pools, pipeline stages

**Pros:** idiomatic Go, "share memory by communicating"
**Cons:** overkill for a single request/response; adds real complexity, don't reach for it unless the problem is actually concurrent

#### 4. `context.Context` for cancellation/timeout
**Best for:** any operation that could hang: DB call, downstream HTTP call, long computation. Should be threaded through from the HTTP handler down.

**Default for this repo:**

- Synchronous. Add goroutines only when the assignment has genuinely
  independent work.
- `context.Context` is the first parameter of anything that does I/O, and
  it's actually passed on: `QueryContext`, `ExecContext`,
  `http.NewRequestWithContext`. Never store a context in a struct.
- Every goroutine has an owner and a way to stop. If you can't say what
  ends it, it leaks.
- Anything concurrent gets a `go test -race` run before submission (see
  [Tooling & Quality Gates](#tooling--quality-gates)).

---

## Logging

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

**Default for this repo:** `log/slog` with the JSON handler to stdout, built
in `main` and passed down (no global logger calls inside `internal/`).

- One request-logging middleware logs every request: method, path, status,
  duration.
- Errors are logged once, where they're handled, usually in `writeError`
  for 5xx responses. See the "handled or returned, never both" rule in
  [Error Handling](#error-handling).
- Never log secrets, tokens, passwords, or full request bodies.

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

**Default for this repo:** stdlib `net/http` with 1.22+ routing. Middleware is
a plain `func(http.Handler) http.Handler`, applied in `main`; you don't need
`chi` for two or three of them. Reach for `chi` only if the middleware chain
gets long and you can justify it in thirty seconds if asked.

---

## HTTP Server Hardening

A bare `http.ListenAndServe(addr, mux)` has no timeouts, so one slow client
can hold a connection forever. `gosec` flags it, and reviewers do too.

**Default for this repo:**

- Build an explicit `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`,
  `WriteTimeout`, and `IdleTimeout` set.
- Cap request bodies with `http.MaxBytesReader` before decoding JSON, and
  answer `413` when the cap is hit.
- Decode with `json.NewDecoder(r.Body)` and call `DisallowUnknownFields()`
  when the contract is strict, so a typoed field fails loudly instead of
  being silently ignored.
- A recovery middleware turns a panic into a `500` with the standard error
  envelope and logs the stack, instead of dropping the connection.
- Shut down gracefully: `signal.NotifyContext` for SIGINT/SIGTERM, then
  `server.Shutdown(ctx)` with a timeout so in-flight requests finish.
- Expose `GET /healthz` returning `200` once the server is ready, and
  check the DB connection if there is one.

---

## API Design

**Default for this repo:**

- **Status codes:** `201 Created` with the created resource (and a
  `Location` header) for POSTs that create, `200` for reads and updates,
  `204 No Content` for deletes. Error statuses are listed under
  [HTTP Error Response Contract](#http-error-response-contract).
- **Pagination:** any list endpoint that can grow is paginated from day one
  (`?limit=&offset=` is fine for an assignment; cursor-based if the brief
  mentions large data). Apply a default `limit` and enforce a maximum.
- **Empty lists:** return `[]`, never `null`. Initialize slices you encode.
- **JSON naming:** `snake_case` field names, consistent across every
  endpoint. Timestamps as RFC 3339 in UTC.
- **Idempotency:** PUT and DELETE are idempotent by definition. If a POST
  must be safe to retry, say how in `docs/DECISIONS.md` (idempotency key or
  a natural unique constraint).
- **Versioning:** no `/v1` prefix unless the brief asks for it. Mention in
  DECISIONS where it would go.

---

## HTTP Error Response Contract

All handler errors return the same JSON envelope:

```json
{"error": "human-readable message", "code": "not_found"}
```

| Error | Status | `code` |
|---|---|---|
| `ValidationError`, malformed JSON | `400` | `invalid_input` |
| `ErrNotFound` | `404` | `not_found` |
| `ErrConflict` | `409` | `conflict` |
| Body over the size cap | `413` | `payload_too_large` |
| Anything else | `500` | `internal` |

For `500`, the message is always generic (`"internal error"`). The real error
is logged, never sent to the client.

**Default for this repo:** one `writeError(w, err)` helper in the handlers
package does this mapping with `errors.Is`/`errors.As`, so every endpoint stays
consistent without repeating switch statements.

---

## Authentication & Authorization

**Default for this repo:** no auth implemented unless the brief asks for it, but
document the decision explicitly in `docs/DECISIONS.md`: where JWT middleware would
sit (before the handler, via `net/http` middleware chaining), and which endpoints
would need role checks (typically destructive deletes and admin-only writes). An
interviewer will ask "how would you secure this" — having the answer ready is the
point, not building it.

---

## Security Baseline

Non-negotiable in every assignment, whatever else is in scope:

- **SQL:** parameterized queries only (`$1`, `$2`). Never build SQL with
  `fmt.Sprintf` or string concatenation, including for `ORDER BY` columns:
  map user input to a fixed allow-list instead.
- **Secrets:** only from environment variables. Never in code, committed
  files, logs, or error messages.
- **Errors:** clients get the error envelope, never stack traces, SQL, or
  driver messages.
- **Input:** every request body is size-capped and validated before use (see
  [HTTP Server Hardening](#http-server-hardening)).
- **Dependencies:** run `govulncheck ./...` before submitting.

---

## Database Access

### Options

#### 1. `database/sql` + `pgx` driver, hand-written SQL
**Best for:** most interview assignments. Register `pgx/v5/stdlib` as the driver and use the standard `*sql.DB` API.

**Pros:** no ORM magic, every query is visible and explainable, the API every Go reviewer knows
**Cons:** manual `Scan()` into each struct field, manual transaction handling (`BeginTx`/`Commit`/`Rollback`)

#### 2. Native `pgxpool`
**Best for:** Postgres-only assignments that need Postgres-specific types (arrays, `COPY`, `LISTEN/NOTIFY`) or pgx's performance

**Pros:** better type support, faster, richer error details
**Cons:** ties every repository signature to pgx, and it's less familiar to reviewers than `database/sql`

#### 3. `sqlx`
**Best for:** reducing struct-scanning boilerplate once row-to-struct mapping repeats across several resources

**Pros:** `StructScan`/`Get`/`Select` convenience on top of the same raw SQL, still fully explainable, thin enough to justify in seconds if asked
**Cons:** external dependency

#### 4. GORM
**Best for:** rarely, for an interview assignment

**Pros:** less code to write
**Cons:** hides the SQL and the relations/cascade behavior behind magic, which directly undercuts a reviewer's ability to see that you understand foreign keys, constraints, and cascade deletes

**Default for this repo:** `database/sql` + `pgx/v5/stdlib`. Move to `sqlx`
only if scanning across resources gets genuinely repetitive, don't reach for
it preemptively.

- Every query takes the request's context (`QueryContext`, `ExecContext`,
  `QueryRowContext`).
- Check `rows.Err()` after every `rows.Next()` loop. Skipping it silently
  truncates results when iteration fails partway.
- Deferred cleanup errors are not ignored (`errcheck` enforces this). Use a
  named `err` return and report the cleanup error unless an earlier error
  already exists:

  ```go
  func (r *repo) ListScans(ctx context.Context) (_ []Scan, err error) {
      rows, err := r.db.QueryContext(ctx, `SELECT ...`)
      if err != nil {
          return nil, fmt.Errorf("query scans: %w", err)
      }
      defer func() {
          if cerr := rows.Close(); cerr != nil && err == nil {
              err = fmt.Errorf("close rows: %w", cerr)
          }
      }()
      // ... scan loop, then rows.Err()
  }
  ```

- Multi-statement writes run in a transaction. Start with `BeginTx(ctx, nil)`
  and immediately defer a rollback in the same shape as above, ignoring only
  `sql.ErrTxDone` (what `Rollback` returns after a successful `Commit`). On
  any early return, the rollback undoes the partial work.
- Set pool limits explicitly (`SetMaxOpenConns`, `SetMaxIdleConns`,
  `SetConnMaxLifetime`) instead of relying on unlimited defaults.
- Constraints (`NOT NULL`, `UNIQUE`, `FOREIGN KEY ... ON DELETE`, `CHECK`)
  live in the schema, not only in Go. The database is the last line of
  defense for data integrity.

### Migrations

**Default:** `golang-migrate`, versioned SQL files (`0001_init.sql`, `0002_...sql`). Simple CLI, works cleanly with `docker compose` startup, no need to justify it in an interview since it's the de facto standard.

---

## Testing Patterns

### Options

#### 1. Table-driven tests (default)
**Best for:** almost everything. A slice of `{name, input, want}` structs run through `t.Run` subtests.

**Pros:** DRY, scales cleanly, each case shows individually in test output, this is the Go idiom interviewers expect to see
**Cons:** none

#### 2. `httptest` for HTTP handlers
**Best for:** testing REST endpoints without a real network call. `httptest.NewRequest` + `httptest.NewRecorder`, served through the same router `main` builds.

**Pros:** fast, no server needed, stdlib, and going through the router covers routing, path params, and method handling too
**Cons:** none, this is the standard way to test Go HTTP handlers

#### 3. `testify` (`assert`/`require`)
**Best for:** reducing boilerplate on assertions, especially comparing structs

**Pros:** more readable failure messages than raw `if got != want`
**Cons:** external dependency; plain stdlib comparisons are fine and dependency-free if the assignment favors minimal deps

#### 4. Interfaces + hand-written fakes for mocking
**Best for:** isolating a handler from a dependency (DB, downstream client) by depending on a small interface

**Pros:** idiomatic Go (accept interfaces, return structs), no mocking framework needed for small assignments
**Cons:** for large interfaces, consider `gomock`/`mockgen` instead of hand-writing every fake

**Default for this repo:** table-driven tests everywhere, `httptest` through
the router for handlers, stdlib assertions unless struct comparisons get
unwieldy (then `testify`), hand-written fakes over a mocking framework unless
the interface is large.

- **Interfaces as test seams:** the consumer defines the interface it needs
  (the handler package declares the store interface it calls), kept to the
  methods it actually uses. This is the one place an interface is justified
  with a single real implementation. The fake is the second.
- **Helpers:** call `t.Helper()` in every test helper so failures point at
  the caller, and register cleanup with `t.Cleanup` rather than `defer`.
- **Determinism:** no `time.Sleep`, no real clock, no dependence on map
  iteration order or test execution order.
- **Integration test isolation:** each test creates the data it needs and
  cleans up after itself (`t.Cleanup`, or a transaction that's rolled back).
  Never depend on seed data or on another test having run first.
- **What to test:** behavior through exported APIs, not private helpers.
  Error paths get as much coverage as the happy path. There's no coverage
  percentage target; a missing error-path test is what matters.

---

## Dependency Policy

**Default for this repo:** stdlib first. Every third-party module in `go.mod`
gets one line in `docs/DECISIONS.md` saying what it replaces and why the
stdlib wasn't enough. "It's popular" isn't a reason; "hand-rolling this would
be 200 lines of error-prone code" is.

Pre-approved when the need is real: `pgx` (Postgres driver), `golang-migrate`
(migrations), `golang.org/x/sync/errgroup` (concurrent fan-out with
cancellation), `testify` (assertions, only if comparisons get unwieldy).

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

With no `.golangci.yml`, golangci-lint v2 runs its standard linter set:
`errcheck`, `govet`, `ineffassign`, `staticcheck`, and `unused`. `govet` runs
roughly the same analyzers as `go vet`, so `go vet ./...` is still worth
running by hand before submitting.

**Go version:** the `go` directive in each `go.mod` matches the toolchain in
the README prerequisites (Go 1.26). Don't rely on features newer than it.

Known gaps, deliberately left for now:

- **No `-race`.** The race detector needs cgo, which isn't set up on the
  Windows dev machine. Run `go test -race ./...` on Linux or in CI before
  submitting anything concurrent.
- **Integration tests are linted but not run** by the gate, because they
  need Docker up. Run them by hand before submitting.
- **No `gosec` or `revive` yet.** They need a `.golangci.yml`, which will
  come with a cleanup pass on existing code.
- **The gate only covers pushes made through Claude Code.** A push from a
  plain terminal isn't checked.
