---
name: go-interviewer-review
description: Reviews a Go assignment in this repo the way a senior backend interviewer would, scoring 8 dimensions 0-10 with file:line citations, graded against the repo's STANDARDS.md defaults and the assignment's own docs/DECISIONS.md rather than generic style opinions. Use after TDD implementation, when the user asks for an interviewer review, a score, or "would this pass". Read-only; it doesn't change code.
---

# Go Interviewer Review

Reviews an assignment the way a senior backend interviewer would in a
take-home debrief. It grades against the defaults in `STANDARDS.md`, the
rules in `CLAUDE.md`, and the choices recorded in the assignment's
`docs/DECISIONS.md`, not against generic style opinions.

**Read-only.** Report findings; don't fix them.

## Before scoring

1. Read the assignment's `docs/QUESTIONS.md` (the brief and its answers),
   `docs/DECISIONS.md` (traits, deviations, layer shape), and
   `docs/DESIGN.md`, plus `STANDARDS.md` and `CLAUDE.md`.
   - Grade only the `STANDARDS.md` sections that apply: the `always`
     sections, plus those whose `Applies when:` matches a trait answered
     yes.
   - A deviation with a concrete reason in `DECISIONS.md` is **not** a
     deduction. An unexplained deviation is.
2. From `assignments/<name>/`, run the checks before reading code, and let
   failures show where to look first: `gofmt -l .`, `go vet ./...`,
   `golangci-lint run ./...` (also with `--build-tags=integration`),
   `go test -shuffle=on ./...`, the integration tests if Docker is up,
   `govulncheck ./...`, and `go mod tidy -diff`. Run `-race` only where
   cgo is available (see Tooling & Quality Gates), and say so in the
   report when it wasn't run. Don't claim a check passed that you didn't
   run.
3. Every finding cites `file:line`. No vague "error handling could be
   better".

## Scoring dimensions (0–10 each)

### 1. Requirements & Delivery
- Everything the brief asks for is implemented and behaves as specified.
  Check against `## Brief` and the answered questions in `QUESTIONS.md`.
- The assignment `README.md` has the sections listed under Delivery, and
  the code runs from a clean clone in 3 commands or fewer.
- Nothing half-finished: no leftover stubs, `TODO`s, commented-out code,
  or skipped tests. Anything cut is listed under "Not done".

### 2. Architecture & Design
- The package layout matches Project Layout, and the layer shape matches
  the one recorded in `DECISIONS.md`. Business rules live in `service`
  (business-rules shape), never in handlers or SQL.
- `domain` imports no other internal package. Only `cmd` wires concrete
  implementations. Interfaces are declared by the package that consumes
  them.
- Adapters don't leak: handlers don't run SQL, the repository knows
  nothing about HTTP, and driver errors are translated in the repository.

### 3. Idiomatic Go
- Errors are wrapped with `%w` and checked with `errors.Is`/`errors.As`.
  None are dropped, deferred cleanup included (the named-return pattern
  in Database Access). `_ = err` is an automatic flag.
- `context.Context` is the first parameter on I/O and is actually passed
  on. No context is stored in a struct.
- No `interface{}`/`any` or reflection used to dodge a proper type.
  Naming follows Go convention. Every package and every exported
  identifier has a doc comment.

### 4. Correctness & Edge Cases
- The edge cases in `QUESTIONS.md` are handled: empty/nil input, zero and
  negative numbers, overflow, malformed input.
- The traps named in `STANDARDS.md` are avoided:
  - `rows.Err()` checked after every loop
  - no typed-nil `*ValidationError` returned as an `error`
  - empty lists encoded as `[]`
  - pagination with a unique tiebreaker
  - an oversize body answered with `413`
  - money in integer minor units, time in UTC
- Concurrency: shared state guarded next to the data, no unbounded
  goroutines, every goroutine with a way to stop.

### 5. Security
- Everything in the Security Baseline: parameterized SQL only (including
  `ORDER BY` via an allow-list), secrets only from the environment,
  nothing internal in error output, every input size-bounded and
  validated.
- `govulncheck` is clean.

### 6. Performance & Resource Management
- Pool limits are set, `db.PingContext` runs at startup, and there are no
  N+1 queries.
- The `http.Server` has timeouts, the request body is capped, and
  shutdown is graceful.
- Large input is streamed rather than loaded into memory. With the
  Performance trait: complexity is stated in the README, `b.Loop()`
  benchmarks exist, and any optimization is backed by numbers.

### 7. Clean Code & Readability
- No abstraction before three call sites, except a consumer-defined test
  seam. No duplicated logic, and no function doing two jobs.
- Names explain themselves. Comments say why, not what.
- `gofmt` and `golangci-lint` are clean. Per `CLAUDE.md` they should be
  already, so flag it if not.

### 8. Testability & Test Quality
- Tests follow the build order in `CLAUDE.md` for every trait that
  applies: `Validate()` reporting every invalid field, adapter tests
  covering the error envelope for each error type, repository
  integration tests covering unique and FK violations, and a wiring smoke
  test.
- Tests are table-driven, go through the router, use hand-written fakes,
  and hit the real database in integration tests. They're deterministic:
  no `time.Sleep`, no real clock, and `-shuffle=on` passes.
- The git log shows TDD: for each unit, a `(Red)` commit before its
  `(Green)` commit, and a `(Refactor)` after it where there was something
  to refactor.
- For a deep test-suite pass, recommend `/go-tests-scanner` rather than
  repeating it here.

## Output

For each dimension: the score, then its findings as
`file:line — issue — why it matters`. Keep it concise, and don't dwell on
what's already correct. End with:

- **Total:** out of 80.
- **Top 3 fixes:** ranked by what an interviewer would flag first.
- **Verdict:** Strong Hire, Hire, Feedback Required, or No Hire. A verdict
  is not an average. One critical flaw (ignored errors throughout, SQL
  built from strings, code that doesn't run from a clean clone) caps it
  at No Hire, whatever the other scores.
- **Interview talking points:** 2–3 questions an interviewer would ask to
  probe the design, each with the file it's about.
