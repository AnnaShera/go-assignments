# Engineering Standards & Decision Reference (Go)

**Version 1.1** (2026-09-24). v1.0 is in git history.

This file holds the technical decisions that come up in every assignment,
with their trade-offs and one **default** each. `CLAUDE.md` covers *how we
work*; this file covers *what we build with*.

**How to use it:**

1. During intake, answer the [Traits](#traits) questions and record the
   answers as the first lines of `docs/DECISIONS.md`.
2. Pull the applicable questions from the
   [Universal Questions Checklist](#universal-questions-checklist) into
   `docs/QUESTIONS.md`.
3. During standards review, read **every section whose `Applies when:` line
   matches a trait you answered yes**, plus every `always` section. Take the
   default in each.
4. Deviate only when the brief or an intake answer gives a concrete reason,
   and record it in `docs/DECISIONS.md` in this form:
   `**Topic:** chose X instead of the default Y, because Z.`
   Deviations are fine. Unexplained ones are what a reviewer marks down.
5. If the assignment has a need no section covers, apply the `always`
   sections, pick the closest section, and log the gap in `DECISIONS.md`.

---

## Traits

Traits compose. An algorithm + REST task answers yes to HTTP and
Performance, and gets both sets of sections. No merging needed.

| Trait | Yes when the assignment... | Adds these sections |
|---|---|---|
| **HTTP** | exposes an HTTP API | Logging, HTTP Routing, HTTP Server Hardening, API Design, HTTP Error Response Contract, Authentication |
| **Database** | persists to a real database | Database Access |
| **Concurrency** | runs work in parallel, **or** keeps state shared across requests or goroutines | Concurrency |
| **CLI** | is run as a command-line program | CLI |
| **Large input** | reads files or streams that may not fit in memory | File & Stream Processing |
| **Performance** | states a latency, throughput, memory, or complexity requirement | Performance & Benchmarking |
| **Messages** | consumes from a queue or broker | Logging, Message Consumers |

**Trap:** an HTTP server with an in-memory store is **Concurrency: yes**.
Every request runs on its own goroutine, so the store is shared state.

Always applies, whatever the traits: Project Layout, Context, Error
Handling, Data Modeling & Validation, Configuration, Security Baseline,
Testing Patterns, Dependency Policy, Delivery, Tooling & Quality Gates.

---

## Defaults at a Glance

| Topic | Applies when | Default |
|---|---|---|
| Layout | always | One module per assignment, `cmd/` for wiring, `internal/` for everything else; `main` only calls `run`; a `service` package only when there are business rules |
| Context | always | `context.Context` first on anything that does I/O; never stored in a struct |
| Errors | always | Domain sentinels + multi-field `ValidationError`, wrapped with `%w`, mapped to output in one place |
| Modeling & validation | always | Plain structs, `Validate() error`, money as integer minor units, time in UTC |
| Configuration | always | Env vars (or flags for a CLI) parsed once into a typed `Config`, fail fast |
| Security | always | Parameterized SQL only, no secrets in code or logs, no internals in error output |
| Testing | always | Table-driven, hand-written fakes, real deps behind `integration` tag, `-shuffle=on` clean |
| Dependencies | always | Stdlib first; each module justified in one line in DECISIONS |
| Delivery | always | Per-assignment README; runs from a clean clone in 3 commands or fewer |
| Logging | HTTP, Messages | `log/slog` JSON to stdout; log once, where the error is handled |
| Routing | HTTP | `net/http` with Go 1.22+ method and path patterns |
| Server | HTTP | Explicit `http.Server` timeouts, body size cap, graceful shutdown, `/healthz` |
| API | HTTP | `snake_case` JSON, `201`/`204` where they apply, paginated lists with stable order, one error envelope |
| Auth | HTTP | Not built unless the brief asks; where it would go is documented |
| Database | Database | `database/sql` + `pgx/v5/stdlib`, hand-written SQL, `golang-migrate` |
| Concurrency | Concurrency | Synchronous unless work is independent; bounded goroutines with an owner; mutex next to the data |
| CLI | CLI | `flag` inside `run`, stdout for results, stderr for diagnostics, exit codes 0/1/2 |
| Streams | Large input | Accept `io.Reader`, stream record by record, report the line number of bad input |
| Performance | Performance | Measure with `b.Loop()` benchmarks before optimizing; state complexity in README |
| Consumers | Messages | Idempotent handler, ack after success, bounded retries then dead-letter |

---

## Universal Questions Checklist

Always include applicable questions from this list in `docs/QUESTIONS.md`
during intake.

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
- For a malformed record in a batch: skip and report it, or fail the whole run?

### Concurrency
- Will this run single-goroutine, or could it be called concurrently?
- Is there shared state? Does it need a mutex, a channel, or can it be avoided by design?
- Does any operation need a timeout or cancellation via `context.Context`?

### Persistence
- Should any state survive between calls or runs?
- Is there a storage requirement: in-memory map, file, real database?

### Output
- Is there an exact output format required (JSON field names, status codes, sorting)?
- Should errors be surfaced to the caller (HTTP status + body, exit code) or logged only?

### API Surface (HTTP)
- Is authentication or authorization expected, or explicitly out of scope?
- Do list endpoints need pagination, filtering, or sorting? What happens with a very large result?
- Must any write be idempotent (e.g. a retried POST must not create duplicates)?
- Is there an API versioning expectation (`/v1/...`)?

### Messages
- What delivery guarantee does the broker give (at-least-once is the usual answer)?
- Does processing order matter, globally or per key?
- What should happen to a message that keeps failing?

### Scope & Delivery
- What is the time budget, and what matters most if it runs out?
- What exactly does the reviewer receive: repo link, Dockerfile, run instructions, a demo?
- How will the reviewer run it (plain `go run`, `docker compose up`, a specific Go version)?
- Which external dependencies are allowed, if any?

---

## Project Layout

**Applies when:** always.

**Default for this repo:**

```
assignments/<name>/
├── cmd/<binary>/main.go   if there is a binary: calls run(), nothing else
├── internal/<package>/    all application code; not importable from outside the module
├── migrations/            Database: versioned SQL
├── testdata/              fixture and golden files; ignored by the go tool
├── docs/                  QUESTIONS, DECISIONS, DESIGN, DEBRIEF
├── README.md              how to run it (see Delivery)
├── Dockerfile             HTTP or Database
├── compose.yaml           Database, or any external service
├── .env.example           every config variable with a dummy value, if there is config
└── go.mod                 module github.com/AnnaShera/go-assignments/assignments/<name>
```

- **Library or algorithm assignment with no binary:** put the package in
  the module root, named after what it provides. No `cmd/`, no
  `internal/`. The reviewer reads one package.
- `main.go` holds no logic. It builds the process inputs and calls `run`,
  which returns an error. That makes the whole wiring path testable:

  ```go
  func main() {
      ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
      err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr)
      stop()
      if err != nil {
          fmt.Fprintln(os.Stderr, err)
          os.Exit(1)
      }
  }
  ```

  `stop()` is called before `os.Exit` because `os.Exit` skips deferred
  calls. A CLI maps errors to exit codes 0/1/2 instead of always 1; see
  [CLI](#cli).
- Name packages after what they provide (`handlers`, `service`,
  `repository`, `domain`), never `utils`, `common`, or `helpers`.
- **Layers.** Pick the shape by whether there's logic beyond validation:
  - **Thin CRUD** (validate, store, return): `handlers` → `repository`.
    The handler package declares the small store interface it calls.
  - **Business rules** (state transitions, cross-entity checks,
    calculations): add a `service` package, `handlers` → `service` →
    `repository`. The handler package declares the service interface it
    calls; the service package declares the repository interface it calls.
    Business rules live in `service`, never in handlers or SQL.

  Either way, every package may import `domain`, and `domain` imports no
  other internal package. Only `cmd` imports the concrete implementations
  and wires them together. Record which shape you picked in
  `docs/DECISIONS.md`.
- Don't create a package for a single small type. Split when a package
  starts doing two jobs, not before.
- Skip `pkg/`. For a single-module assignment it adds a level of nesting and
  says nothing.

---

## Context

**Applies when:** always (anything that does I/O or can block).

- `context.Context` is the first parameter of anything that does I/O or can
  block, and it's actually passed on: `QueryContext`, `ExecContext`,
  `http.NewRequestWithContext`, a `select` on `ctx.Done()`.
- Never store a context in a struct.
- In tests, use `t.Context()` (Go 1.24+). It's canceled when the test ends.

---

## Error Handling

**Applies when:** always.

### Options

#### 1. Sentinel errors (`errors.New`, package-level `var`)
**Best for:** a small fixed set of known error conditions callers need to check with `errors.Is`

**Pros:** simple, stdlib-only, cheap to compare
**Cons:** no room for context (a message, a field name); doesn't scale past a handful of cases

#### 2. Custom error types (struct implementing `error`)
**Best for:** domain errors that carry data (which fields failed, which ID was not found), checked with `errors.As`

**Pros:** carries structured context; callers can branch on type; good for handlers mapping errors to status codes
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
  type that collects **every** invalid field, so the client fixes them in
  one round trip:

  ```go
  // FieldError describes one invalid field.
  type FieldError struct {
      Field  string `json:"field"`
      Reason string `json:"reason"`
  }

  // ValidationError reports all invalid fields of one input.
  type ValidationError struct {
      Fields []FieldError
  }
  ```

  `Validate()` appends to `Fields` and returns `nil` when the slice is
  empty, never an empty `*ValidationError` (a typed nil inside an `error`
  is not `== nil`). Callers check it with `errors.As`.
- The repository translates driver errors into that vocabulary, so nothing
  above it ever sees a `*pgconn.PgError` or `sql.ErrNoRows`:
  `sql.ErrNoRows` → `ErrNotFound`, unique violation (`23505`) → `ErrConflict`,
  foreign key violation (`23503`) → `ErrNotFound` or `ValidationError`,
  depending on which side of the relation the caller controls. With the
  `pgx/v5/stdlib` driver the error is still a `*pgconn.PgError`: match it
  with `errors.As` and compare its `Code` field.
- Every layer wraps with `%w` and adds what it was doing:
  `fmt.Errorf("get scan %d: %w", id, err)`.
- Errors are turned into output in exactly one place per adapter:
  `writeError` for HTTP (see
  [HTTP Error Response Contract](#http-error-response-contract)), the exit
  code in `main` for a CLI, the retry decision for a consumer.
- An error is either handled (logged, turned into a response) or returned,
  never both. Logging and returning produces the same failure three times in
  the logs.

---

## Data Modeling & Validation

**Applies when:** always.

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

**Applies when:** always (skip for a pure library).

**Default for this repo:**

- Read config once, in `run`, into a typed `Config` struct: environment
  variables for services, flags for a CLI. Pass values down explicitly. No
  package reads `os.Getenv` on its own; `run` receives `getenv` as a
  parameter so tests can supply their own.
- Fail fast: a missing required variable stops startup with a clear error
  instead of surfacing later as a confusing connection failure.
- Defaults only for values that are safe to default (port, timeouts), never
  for credentials.
- Commit a `.env.example` per assignment listing every variable with dummy
  values. The real `.env` stays gitignored.
- No config library (`viper`, `envconfig`) unless there are enough variables
  that parsing by hand gets noisy.

---

## Security Baseline

**Applies when:** always. Non-negotiable, whatever else is in scope.

- **SQL:** parameterized queries only (`$1`, `$2`). Never build SQL with
  `fmt.Sprintf` or string concatenation, including for `ORDER BY` columns:
  map user input to a fixed allow-list instead.
- **Secrets:** only from environment variables. Never in code, committed
  files, logs, or error messages. Local-dev credentials in `compose.yaml`
  and `.env.example` are fine if they're obviously dummy values.
- **Errors:** clients get the error envelope (or a plain message on stderr
  for a CLI), never stack traces, SQL, or driver messages.
- **Input:** every request body or input record is size-bounded and
  validated before use (see [HTTP Server Hardening](#http-server-hardening)
  and [File & Stream Processing](#file--stream-processing)).
- **Dependencies:** run `govulncheck ./...` before submitting.

---

## Testing Patterns

**Applies when:** always.

### Options

#### 1. Table-driven tests (default)
**Best for:** almost everything. A slice of `{name, input, want}` structs run through `t.Run` subtests.

**Pros:** DRY, scales cleanly, each case shows individually in test output, this is the Go idiom interviewers expect to see
**Cons:** none

#### 2. `httptest` for HTTP handlers
**Best for:** testing REST endpoints without a real network call. `httptest.NewRequest` + `httptest.NewRecorder`, served through the same router `run` builds.

**Pros:** fast, no server needed, stdlib, and going through the router covers routing, path params, and method handling too
**Cons:** none, this is the standard way to test Go HTTP handlers

#### 3. `testify` (`assert`/`require`)
**Best for:** reducing boilerplate on assertions, especially comparing structs

**Pros:** more readable failure messages than raw `if got != want`
**Cons:** external dependency; plain stdlib comparisons are fine and dependency-free if the assignment favors minimal deps

#### 4. Interfaces + hand-written fakes for mocking
**Best for:** isolating a unit from a dependency (DB, downstream client, broker) by depending on a small interface

**Pros:** idiomatic Go (accept interfaces, return structs), no mocking framework needed for small assignments
**Cons:** for large interfaces, consider `gomock`/`mockgen` instead of hand-writing every fake

**Default for this repo:** table-driven tests everywhere, `httptest` through
the router for handlers, stdlib assertions unless struct comparisons get
unwieldy (then `testify`), hand-written fakes over a mocking framework unless
the interface is large.

- **Interfaces as test seams:** the consumer defines the interface it needs,
  kept to the methods it actually uses: the handler package declares the
  store or service interface it calls, the service package declares the
  repository interface (see Layers under [Project Layout](#project-layout)).
  This is the one place an interface is justified with a single real
  implementation. The fake is the second.
- **Helpers:** call `t.Helper()` in every test helper so failures point at
  the caller, and register cleanup with `t.Cleanup` rather than `defer`.
- **Determinism:** no `time.Sleep`, no real clock, no dependence on map
  iteration order or test execution order. `go test -shuffle=on ./...`
  proves the last one; it must pass.
- **Fixtures:** input files and large expected outputs (golden files) live
  in `testdata/`.
- **Integration test isolation:** each test creates the data it needs and
  cleans up after itself (`t.Cleanup`, or a transaction that's rolled back).
  Never depend on seed data or on another test having run first.
- **What to test:** behavior through exported APIs, not private helpers.
  Error paths get as much coverage as the happy path. There's no coverage
  percentage target; a missing error-path test is what matters.

---

## Dependency Policy

**Applies when:** always.

**Default for this repo:** stdlib first. Every third-party module in `go.mod`
gets one line in `docs/DECISIONS.md` saying what it replaces and why the
stdlib wasn't enough. "It's popular" isn't a reason; "hand-rolling this would
be 200 lines of error-prone code" is.

Pre-approved when the need is real: `pgx` (Postgres driver), `golang-migrate`
(migrations), `golang.org/x/sync/errgroup` (concurrent fan-out with
cancellation), `golang.org/x/time/rate` (rate limiting, unless building the
limiter *is* the assignment), `testify` (assertions, only if comparisons get
unwieldy).

---

## Delivery

**Applies when:** always. "I couldn't run it" ends a review before the code
is read.

**Default for this repo:**

- **Per-assignment `README.md`** with these sections, in this order:
  1. **Run:** 3 commands or fewer from a clean clone.
  2. **Test:** unit and integration commands.
  3. **Usage:** one example per endpoint (`curl`) or per CLI mode, with
     real output.
  4. **Assumptions:** the open questions and the answer you assumed (link
     `docs/QUESTIONS.md`).
  5. **Design and trade-offs:** 3 to 5 bullets, linking
     `docs/DECISIONS.md`.
  6. **Not done / with more time:** honest and short.
- **HTTP or Database:** `docker compose up --build` brings up everything.
  Postgres has a `pg_isready` healthcheck, a `migrate` service (the
  `migrate/migrate` image) runs after it's healthy, and the app waits on
  `migrate` with `condition: service_completed_successfully`.
- **Dockerfile:** multi-stage. Build in `golang:<version>` with
  `CGO_ENABLED=0`, run from `gcr.io/distroless/static-debian12:nonroot`.
  Small image, no shell, non-root user.
- **Repo hygiene:** `go.sum` committed, `go mod tidy -diff` prints nothing,
  `.gitignore` covers binaries, `.env`, and coverage output.
- **Before submitting:** clone the repo into an empty folder and follow the
  README exactly, as the reviewer will.

---

## Tooling & Quality Gates

**Applies when:** always.

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
the README prerequisites (Go 1.26). Move both together, never one alone.
Nothing in this file needs more than Go 1.25.

Known gaps, deliberately left for now:

- **No `-race` locally.** The race detector needs cgo, which isn't set up on
  the Windows dev machine. Planned fix: a GitHub Actions workflow on Linux
  that runs `go test -race -shuffle=on ./...`, the integration tests against
  a Postgres service container, and `govulncheck`. Until it exists, run
  those on Linux by hand before submitting anything with the Concurrency
  trait.
- **Integration tests are linted but not run** by the gate, because they
  need Docker up. Run them by hand before submitting.
- **No `gosec` or `revive` yet.** They need a `.golangci.yml`, which will
  come with a cleanup pass on existing code.
- **The gate only covers pushes made through Claude Code.** A push from a
  plain terminal isn't checked.

---

## Logging

**Applies when:** HTTP, Messages, or any long-running process. A CLI writes
diagnostics to stderr instead (see [CLI](#cli)).

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
in `run` and passed down (no global logger calls inside `internal/`).

- One request-logging middleware logs every request: method, path, status,
  duration.
- Errors are logged once, where they're handled, usually in `writeError`
  for 5xx responses. See the "handled or returned, never both" rule in
  [Error Handling](#error-handling).
- Never log secrets, tokens, passwords, or full request bodies.

---

## HTTP Routing / Web Framework

**Applies when:** HTTP.

### Options

#### 1. `net/http` stdlib only (with 1.22+ method+path patterns)
**Best for:** most interview assignments. `mux.HandleFunc("GET /orders/{id}", handler)` with `r.PathValue("id")` covers path params and method routing natively since Go 1.22.

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
a plain `func(http.Handler) http.Handler`, applied in `run`; you don't need
`chi` for two or three of them. Reach for `chi` only if the middleware chain
gets long and you can justify it in thirty seconds if asked.

---

## HTTP Server Hardening

**Applies when:** HTTP.

A bare `http.ListenAndServe(addr, mux)` has no timeouts, so one slow client
can hold a connection forever. `gosec` flags it, and reviewers do too.

**Default for this repo:**

- Build an explicit `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`,
  `WriteTimeout`, and `IdleTimeout` set.
- Cap request bodies with `http.MaxBytesReader` before decoding JSON. When
  the cap is hit, the decode error wraps a `*http.MaxBytesError`: detect it
  with `errors.As` and answer `413`.
- Decode with `json.NewDecoder(r.Body)` and call `DisallowUnknownFields()`
  when the contract is strict, so a typoed field fails loudly instead of
  being silently ignored.
- A recovery middleware turns a panic into a `500` with the standard error
  envelope and logs the stack, instead of dropping the connection.
- Shut down gracefully: `run` receives the `signal.NotifyContext` context
  from `main`, and when it's canceled calls `server.Shutdown` with a
  timeout so in-flight requests finish.
- Expose `GET /healthz` returning `200` once the server is ready, and
  check the DB connection if there is one.

---

## API Design

**Applies when:** HTTP.

**Default for this repo:**

- **Status codes:** `201 Created` with the created resource (and a
  `Location` header) for POSTs that create, `200` for reads and updates,
  `204 No Content` for deletes. Error statuses are listed under
  [HTTP Error Response Contract](#http-error-response-contract).
- **Responses:** one `writeJSON` helper sets
  `Content-Type: application/json`, writes the status, then encodes. An
  encode error is logged (the status is already sent, so it can't change).
- **Pagination:** any list endpoint that can grow is paginated from day one
  (`?limit=&offset=` is fine for an assignment; cursor-based if the brief
  mentions large data). Default `limit` 20, maximum 100: a non-integer or
  negative value is a `400`, a value above the maximum is clamped. The
  query has a **stable order with a unique tiebreaker**
  (`ORDER BY created_at DESC, id DESC`). Without the tiebreaker, rows with
  equal sort keys can repeat or vanish between pages.
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

**Applies when:** HTTP.

All handler errors return the same JSON envelope. `details` appears only on
validation errors:

```json
{
  "error": "invalid input",
  "code": "invalid_input",
  "details": [{"field": "severity", "reason": "must be one of low, medium, high"}]
}
```

| Error | Status | `code` |
|---|---|---|
| `ValidationError`, malformed JSON, bad query parameter | `400` | `invalid_input` |
| `ErrNotFound` | `404` | `not_found` |
| `ErrConflict` | `409` | `conflict` |
| Body over the size cap (`*http.MaxBytesError`) | `413` | `payload_too_large` |
| Anything else | `500` | `internal` |

For `500`, the message is always generic (`"internal error"`). The real error
is logged, never sent to the client.

**Default for this repo:** one `writeError(w, err)` helper in the handlers
package does this mapping with `errors.Is`/`errors.As`, so every endpoint stays
consistent without repeating switch statements.

---

## Authentication & Authorization

**Applies when:** HTTP.

**Default for this repo:** no auth implemented unless the brief asks for it,
but document the decision explicitly in `docs/DECISIONS.md`: where JWT
middleware would sit (before the handler, via `net/http` middleware
chaining), and which endpoints would need role checks (typically destructive
deletes and admin-only writes). An interviewer will ask "how would you
secure this". Having the answer ready is the point, not building it.

---

## Database Access

**Applies when:** Database.

### Options

#### 1. `database/sql` + `pgx` driver, hand-written SQL
**Best for:** most interview assignments. Register `pgx/v5/stdlib` as the driver and use the standard `*sql.DB` API.

**Pros:** no ORM magic, every query is visible and explainable, the API every Go reviewer knows
**Cons:** manual `Scan()` into each struct field, manual transaction handling (`BeginTx`/`Commit`/`Rollback`)

#### 2. Native `pgxpool`
**Best for:** Postgres-only assignments that need Postgres-specific types (arrays, `COPY`, `LISTEN/NOTIFY`) or pgx's performance

**Pros:** better type support, faster, richer error details
**Cons:** ties every repository signature to pgx, and it's less familiar to reviewers than `database/sql`

If you choose it, its API differs from `database/sql`: `pgx.Rows.Close()`
returns nothing, so use `pgx.CollectRows` (which checks `rows.Err()` for
you), and `pgx.BeginFunc` so commit and rollback are handled for you.

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

- `sql.Open` doesn't connect. Call `db.PingContext` at startup so a wrong
  DSN fails fast instead of on the first request.
- Every query takes the request's context (`QueryContext`, `ExecContext`,
  `QueryRowContext`).
- Check `rows.Err()` after every `rows.Next()` loop. Skipping it silently
  truncates results when iteration fails partway.
- Deferred cleanup errors are not ignored (`errcheck` enforces this).
  `*sql.Rows.Close()` returns an error. Use a named `err` return and report
  the cleanup error unless an earlier error already exists:

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

**Default:** `golang-migrate`, versioned SQL files in up/down pairs:
`000001_init.up.sql` and `000001_init.down.sql`. The tool only recognizes
files named `{version}_{title}.up.sql` / `.down.sql`. Simple CLI, runs as a
compose service, no need to justify it in an interview since it's the de
facto standard.

---

## Concurrency

**Applies when:** Concurrency (parallel work, or state shared across
requests or goroutines).

### Options

#### 1. Synchronous (default)
**Best for:** most assignments, including most REST handlers. One request, one goroutine (the stdlib server already does this for you), no manual concurrency needed.

#### 2. Goroutines + `sync.WaitGroup` / `errgroup`
**Best for:** fan-out work within a single request or job (e.g. call three downstream things concurrently, wait for all). Use `errgroup.WithContext` when any failure should cancel the rest; `sync.WaitGroup.Go` (Go 1.25+) when there's no error to collect.

**Pros:** straightforward, well understood, easy to reason about
**Cons:** needs care with shared state (`sync.Mutex` or channels), easy to leak goroutines if you forget cancellation

#### 3. Channels for coordination
**Best for:** producer/consumer patterns, worker pools, pipeline stages

**Pros:** idiomatic Go, "share memory by communicating"
**Cons:** overkill for a single request/response; adds real complexity, don't reach for it unless the problem is actually concurrent

**Default for this repo:** synchronous. Add goroutines only when the
assignment has genuinely independent work.

- **Shared state:** the mutex lives in the same struct as the data it
  guards, unexported, and only that struct's methods lock it. Never return
  the internal map or slice; return a copy. Use pointer receivers so the
  struct (and its mutex) is never copied; `govet`'s `copylocks` check
  catches it if you do. `sync.RWMutex` only when reads clearly dominate.
- **Bounded work:** never one goroutine per input item on unbounded input.
  Use `errgroup` with `g.SetLimit(n)`, or a fixed worker pool.
- **Ownership:** every goroutine has an owner and a way to stop (context
  cancellation or a closed channel). If you can't say what ends it, it
  leaks. Only the sender closes a channel.
- **Time in concurrent code:** inject the clock, or test with
  `testing/synctest` (Go 1.25+): inside `synctest.Test(t, func(t *testing.T) {...})`
  time is virtual, and `synctest.Wait()` blocks until every goroutine in
  the test is idle. No `time.Sleep` in tests.
- **Proof:** a test that hits the shared state from many goroutines at
  once, run under `go test -race` (see
  [Tooling & Quality Gates](#tooling--quality-gates) for where that runs).

---

## CLI

**Applies when:** CLI.

**Default for this repo:** stdlib `flag`. Reach for `cobra` only if the
brief asks for nested subcommands.

- Parse flags inside `run` with `flag.NewFlagSet(name, flag.ContinueOnError)`
  and `fs.SetOutput(stderr)`, never the global `flag.Parse()`. That keeps
  argument parsing testable and stops the flag package from calling
  `os.Exit` on its own.
- **stdout** carries results only, so output can be piped. **stderr**
  carries diagnostics and errors.
- **Exit codes:** `0` success, `1` runtime failure, `2` usage error (the
  convention the `flag` package itself follows). `-h` / `-help` exits `0`:
  `fs.Parse` returns `flag.ErrHelp` for it. `run` reports usage problems on
  its own `stderr` (for a bad flag, `fs.Parse` already has) and returns
  `errUsage`; `main` maps the result and prints only runtime errors:

  ```go
  var errUsage = errors.New("usage error")

  func main() {
      ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
      err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr)
      stop()
      os.Exit(exitCode(err))
  }

  func exitCode(err error) int {
      switch {
      case err == nil, errors.Is(err, flag.ErrHelp):
          return 0
      case errors.Is(err, errUsage):
          return 2
      default:
          fmt.Fprintln(os.Stderr, err)
          return 1
      }
  }
  ```

  `exitCode` is a pure function, so a table test covers all three codes.
- **Every write checks its error.** Output goes through the `io.Writer`
  that `run` receives, and `errcheck` flags an unchecked
  `fmt.Fprintln(stdout, ...)`, even when the writer is a `*bufio.Writer`.
  Return the error: `if _, err := fmt.Fprintln(stdout, line); err != nil { return fmt.Errorf("write output: %w", err) }`.
  Only `fmt.Fprint*` to `os.Stderr`, as in `main` above, is exempt.
- Read input from an `io.Reader` (a file argument, or stdin when none is
  given), so tests pass a `strings.Reader`.
- Tests call `run` with `bytes.Buffer` for stdout and stderr and assert on
  output and the returned error. Large expected outputs are golden files in
  `testdata/`.

---

## File & Stream Processing

**Applies when:** Large input.

**Default for this repo:**

- Core logic accepts `io.Reader` / `io.Writer`, never a file path. Opening
  files happens in `run`. Tests pass in-memory readers.
- Stream record by record. No `os.ReadFile` or `io.ReadAll` on input that
  can be large; memory must stay flat as input grows.
- `bufio.Scanner` has a 64 KiB default token limit
  (`bufio.MaxScanTokenSize`). A longer line stops the scan with
  `bufio.ErrTooLong`. Raise the limit with `scanner.Buffer` when lines can
  be long, and always check `scanner.Err()` after the loop.
- CSV: `encoding/csv` `Reader.Read` in a loop until `io.EOF`. Set
  `FieldsPerRecord` to enforce the column count. `ReuseRecord` cuts
  allocations on big files, but copy any record you keep.
- Wrap output in `bufio.NewWriter` for throughput. Buffering doesn't exempt
  writes from `errcheck`: check every write's error, then check `Flush`,
  which reports anything the buffer hit on the way out.
- Bad records: follow the intake answer (skip and report, or fail). Either
  way, the message names the **line number**.
- Close files with the named-return pattern shown under
  [Database Access](#database-access).

---

## Performance & Benchmarking

**Applies when:** Performance.

**Default for this repo:**

- State the **time and space complexity** of the core algorithm in the
  README and in a doc comment on the function.
- Benchmarks use the Go 1.24+ loop form, which keeps setup out of the
  timing:

  ```go
  func BenchmarkTopK(b *testing.B) {
      input := makeInput(10_000) // setup, not timed
      for b.Loop() {
          TopK(input, 10)
      }
  }
  ```

  Run with `go test -bench=. -benchmem ./...`.
- Optimize only what a benchmark shows is slow. Compare before and after
  with `benchstat` (`golang.org/x/perf/cmd/benchstat`) and put the numbers
  in `DECISIONS.md`.
- Cheap wins that don't need a benchmark to justify: pre-size slices and
  maps with `make` when the size is known, `strings.Builder` for building
  strings in a loop, a map lookup instead of a nested scan.

---

## Message Consumers

**Applies when:** Messages.

**Default for this repo:**

- **Assume at-least-once delivery.** Every handler is idempotent: dedupe
  on the message ID, ideally with a unique constraint so the database
  enforces it.
- **Ack or commit only after processing succeeds.** Acking first loses
  messages on a crash.
- **Poison messages:** bounded retries with backoff, then move the message
  to a dead-letter queue (or log it and park it) so one bad message can't
  block the rest.
- **Structure:** the handler is a plain `func(ctx context.Context, msg Message) error`
  with no broker types in its signature. The broker client sits behind a
  small interface in `run`. Unit tests call the handler directly; broker
  tests go behind the `integration` tag against the broker in compose.
- **Ordering:** if order matters per key, process each key's messages
  sequentially and say how in `DECISIONS.md`.
- **Shutdown:** on context cancellation, stop fetching, finish in-flight
  messages, commit, then exit.
- **Client library:** the one the brief names. Otherwise pick the most
  widely used Go client for that broker and justify it under
  [Dependency Policy](#dependency-policy).
