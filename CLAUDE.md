# Home Assignment Instructions (Go, TDD)

**Version 1.4** (2026-09-24). Earlier versions are in git history.

This file is *how we work*. `STANDARDS.md` is *what we build with*: read it
before designing anything, and use its defaults unless the assignment gives a
concrete reason to deviate (logged in `docs/DECISIONS.md`). TDD process lives
here only; technical rules live in `STANDARDS.md` only. If the two files
disagree, stop and flag it.

Pushes are automatically gated by a `.claude/settings.json` hook.
`.claude/hooks/pre-push-check.sh` blocks on `gofmt`, `go test`, and
`golangci-lint` failures, checked separately in each module under
`assignments/*/`.

## Working Rules

These hold in every phase, in practice and live mode alike.

- **Tests are the spec.** Never weaken, skip, or delete a failing test to
  get to green. If a test itself is wrong, say so, and fix it as its own
  step with its own commit.
- **Never bypass the gate.** No `--no-verify`, and no loosening the hook or
  lint config to make a failure go away. Fix the code.
- **No new dependency without asking first.** Every one needs a line in
  `docs/DECISIONS.md` (see Dependency Policy in `STANDARDS.md`).
- **Stay inside the brief.** Good ideas outside it go into
  `docs/DEBRIEF.md` under "with more time", not into the code.
- **Nothing half-finished at submission.** No leftover stubs, `TODO`s,
  commented-out code, or feature flags in the submitted code. If something
  won't be finished, cut it and list it under "Not done" in the assignment
  README. Red-phase stubs are fine while a cycle is in progress.
- **Don't guess requirements.** An ambiguity is a question for the user. A
  decision that comes up mid-implementation gets one line in
  `docs/DECISIONS.md` before the code that depends on it.
- **Every line must be explainable in the interview.** Work in small steps,
  explain anything non-obvious while writing it, and prefer the boring
  solution over the clever one. If the user can't explain a line, it
  gets explained or rewritten before moving on.

## Repo Layout & Conventions

- Every assignment lives in its own folder, `assignments/<name>/`, and is
  its own Go module. There is no root `go.mod`, so run Go commands from
  inside the assignment folder.
- Module path: `github.com/AnnaShera/go-assignments/assignments/<name>`.
- Layout inside an assignment follows Project Layout in `STANDARDS.md`,
  including the assignment's own `README.md` (see Delivery).
- Workflow docs go in `assignments/<name>/docs/`, never in the repo root.
- When a new assignment is created, add a row for it to the Assignments
  table in the repo-root `README.md`.

## Commands

Run these from `assignments/<name>/`:

| Purpose | Command |
|---|---|
| Unit tests | `go test ./...` |
| Unit tests, random order | `go test -shuffle=on ./...` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run ./...` |
| Lint integration code | `golangci-lint run --build-tags=integration ./...` |
| Integration tests | `docker compose up -d`, then `go test -tags=integration ./...` |
| Race detector (needs cgo: not on this Windows machine; run on Linux or in CI) | `go test -race ./...` |
| Benchmarks (Performance trait) | `go test -bench=. -benchmem ./...` |
| Known vulnerabilities | `govulncheck ./...` |
| Module tidiness (prints nothing when clean) | `go mod tidy -diff` |
| Format (repo-wide, from root) | `gofmt -l .` to check, `gofmt -w .` to fix |

Before submission, every row that applies passes, including integration,
race, and `govulncheck`.

## TDD Discipline

- Workflow is **Red → Green → Refactor**. No production code without a
  failing test driving it first.
- **Red fails for the right reason:** on an assertion, not a compile error.
  Add the new function's signature as a stub that returns zero values
  first, so the test compiles and fails on its check.
- **Green is the smallest change that passes.** Hardcoding is fine if the
  next test will force generalization.
- Test names describe behavior: `TestOrderTotal_ReturnsErrorWhenNegativeAmount`,
  or for table-driven tests, a descriptive subtest name per case:
  `t.Run("negative amount returns error", func(t *testing.T) {...})`.
- **Commit at every phase, Red included.** The Red commit is the evidence
  that the test came first, and the git log is what shows an interviewer
  your process. A Red commit fails only on the new tests; Green and
  Refactor commits pass `go test ./...` in full.
- **Commit messages:** an imperative summary with the phase as a suffix.
  Commits outside a TDD cycle (docs, chores) have no suffix.
  - `Add failing tests for severity filtering (Red)`
  - `Implement severity filtering on ListFindings (Green)`
  - `Extract pagination normalization (Refactor)`
- **Push only from green.** The gate runs the tests at push time, so a Red
  HEAD is blocked anyway.
- Refactor only on green. Never refactor and add behavior in the same step.

**Build order (inside out).** Take every step whose trait applies, in this
order:

1. **Domain** (always): `Validate()` table tests (valid case, each invalid
   field, multiple invalid fields at once, asserting every field is
   reported).
2. **Service** (only with the business-rules layer shape; see Layers in
   `STANDARDS.md`): business rules against a hand-written fake of each
   dependency (happy path, not found, conflict, validation passthrough).
3. **Stream processing** (Large input): the core function against an
   in-memory `strings.Reader`, with output captured in a `bytes.Buffer`:
   empty input, a line longer than 64 KiB, and a malformed record reported
   with its line number.
4. **Adapter**, one per entry point:
   - **HTTP:** through the router with `httptest`, against a fake of the
     interface the handler calls (the service, or the store for thin CRUD).
     Status code, JSON body, and the error envelope for each error type.
   - **CLI:** `run` with `bytes.Buffer` for stdout and stderr: output,
     returned error, and usage errors; plus a table test for `exitCode`.
   - **Messages:** the handler function called directly: success, retryable
     failure, duplicate message.
5. **Concurrency:** a test that hits the shared state from many goroutines
   at once, so the race detector has something to catch.
6. **Repository** (Database): integration tests (`integration` build tag)
   against real Postgres: happy path, not found, unique violation, FK
   violation, pagination order.
7. **Wiring** (HTTP, `integration` tag): one smoke test that builds the
   real handler with the real database, serves it on `httptest.NewServer`,
   and hits `/healthz` plus one real endpoint.

**For an algorithm assignment:** start with the simplest edge case
(empty/nil input), then the happy path, then the remaining edge cases from
`docs/QUESTIONS.md`, one test per cycle. With the Performance trait, add
the benchmark once the tests are green.

## Code Quality Standards

- Code must pass `gofmt`, `go vet`, and `golangci-lint` clean before a
  Refactor phase is considered done.
- **Every returned error is handled or explicitly wrapped and returned.**
  No `_ = err`, no ignored error from a function call, ever. This is
  non-negotiable in Go and one of the fastest ways to fail a review.
  Deferred cleanup counts: for a `Close()` that returns an error (files,
  `resp.Body`, `*sql.Rows`), use the named-return pattern under Database
  Access in `STANDARDS.md`.
- DRY, but avoid premature abstraction: a shared helper or abstraction
  needs three concrete call sites first. The one exception is a small,
  consumer-defined interface used as a test seam (see Testing Patterns in
  `STANDARDS.md`).
- Package boundaries should reflect responsibility, not just "where the
  code happened to go." A package name that needs "and" in its description
  is a sign it's doing two jobs.
- Prioritize maintainability and readability over micro-optimization
  unless the assignment states a performance requirement.
- No naked `interface{}`/`any` or reflection to dodge typing a struct
  properly, unless the problem genuinely requires it.

## Documentation Approach

- Comments explain *why*, not *what*. If the code needs a comment to say
  what it does, the code should probably be clearer instead.
- Comments only when: an invariant isn't obvious from the code, or there's
  a workaround for a specific bug/edge case that would otherwise look like
  a mistake.
- Every package gets a `// Package <name> ...` comment in one of its files,
  and every exported identifier (function, type, const) gets a doc comment
  starting with its name: `// Validate checks that...`. This is standard
  Go, not optional polish.
- Self-explanatory naming reduces the need for comments. A function called
  `calc` needs a comment; a function called `totalWithTax` doesn't.

## Testing Strategy

- **Unit tests** cover business logic: validation, state transitions,
  anything that doesn't touch an external dependency. They use fakes, run
  in milliseconds, and are part of every Red → Green → Refactor loop.
- **Integration tests** cover the edges where the real dependency's behavior
  matters: queries, constraints, transactions, cascade deletes, broker
  delivery. They run against the real thing (e.g. Postgres via
  `docker compose`), never a mock, because a mock only confirms what you
  already believe the dependency does.
- Integration tests live behind the `integration` build tag
  (`//go:build integration`) and never mix into the fast unit suite.
- Patterns (table-driven, `httptest` through the router, fakes, isolation,
  determinism) are under Testing Patterns in `STANDARDS.md`.

## Structured Workflow

Every assignment runs through the `/new-go-assignment` skill. For a small
algorithmic exercise, phases 2 and 3 can be a few lines each, but they still
happen. Every `docs/` path means `assignments/<name>/docs/`.

1. **Intake:** read the assignment, answer the Traits questions in
   `STANDARDS.md` (record them as the first lines of `docs/DECISIONS.md`),
   fill `docs/QUESTIONS.md` from the Universal Questions Checklist, and get
   answers before designing.
2. **Standards review:** go through every `STANDARDS.md` section whose
   `Applies when:` matches, take its default, and log any deviation in
   `docs/DECISIONS.md`.
3. **Design:** produce `docs/DESIGN.md`: package layout, exported
   function signatures, the layer shape (thin CRUD or business rules), data
   flow (one line per request, command, or message path, e.g.
   `POST /scans → handlers → service → repository → Postgres`), and a
   checklist of what needs implementing, in the build order above.
4. **TDD implementation:** Red → Green → Refactor per unit, with stopping
   points for review after: test file skeleton is written, first passing
   test group, each package's implementation is complete. At each stop,
   report in this form and nothing longer: a 1-line summary of what was
   done, the files changed (as links), and the next step.
5. **Debrief and submission check:** write `docs/DEBRIEF.md` (tradeoffs
   made and why, what you'd do differently with more time), then run every
   applicable command above and follow the assignment README from a clean
   clone (see Delivery in `STANDARDS.md`).

Each stopping point requires explicit go-ahead before advancing to the
next phase. Do not skip ahead on your own.

## Live-Interview Mode

The checkpoint structure above is for **practice reps**, where pausing for
review is the point. If this is running during an actual timed, AI-assisted
interview: skip the stop-and-wait approvals in phase 4, keep `docs/DECISIONS.md`
to one line per decision instead of a full writeup, and save the debrief for
after submission. The Working Rules and TDD commits still apply, and so does
the README (a reviewer who can't run it stops there). If the interviewer is
unavailable, record your assumption for each open question in
`docs/QUESTIONS.md` and continue. Say "live mode" at the start of a session
to switch.
