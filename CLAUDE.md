# Home Assignment Instructions (Go, TDD)

Read `STANDARDS.md` before designing any solution and use its "Default for
this repo" choices unless the assignment gives a specific reason to deviate.

Pushes are automatically gated by a `.claude/settings.json` hook —
`.claude/hooks/pre-push-check.sh` blocks on `gofmt`, `go test`, and
`golangci-lint` failures, checked separately in each module under
`assignments/*/`.

## Repo Layout & Conventions

- Every assignment lives in its own folder, `assignments/<name>/`, and is
  its own Go module. There is no root `go.mod`, so run Go commands from
  inside the assignment folder.
- Module path: `github.com/AnnaShera/go-assignments/assignments/<name>`.
- Inside an assignment: `cmd/<binary>/main.go` for entry points,
  `internal/<package>/` for everything else, `migrations/` if there is a
  database, `docs/` for the workflow files.
- Workflow docs go in `assignments/<name>/docs/`, never in the repo root.
- When a new assignment is created, add a row for it to the Assignments
  table in `README.md`.

## Commands

Run these from `assignments/<name>/`:

| Purpose | Command |
|---|---|
| Unit tests | `go test ./...` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run ./...` |
| Lint integration code | `golangci-lint run --build-tags=integration ./...` |
| Integration tests | `docker compose up -d`, then `go test -tags=integration ./...` |
| Format (repo-wide, from root) | `gofmt -l .` to check, `gofmt -w .` to fix |

## TDD Discipline

- Workflow is **Red → Green → Refactor**. Write a failing test before any
  implementation. No production code without a test driving it first.
- Test names describe behavior: `TestOrderTotal_ReturnsErrorWhenNegativeAmount`,
  or for table-driven tests, a descriptive subtest name per case:
  `t.Run("negative amount returns error", func(t *testing.T) {...})`.
- Commit after every Green or Refactor phase. Small commits, not one commit
  at the end. This is what shows an interviewer your actual process, not
  just your final answer.
- Refactor only on green. Never refactor and add behavior in the same step.
- `go test ./...` must pass before every commit, not just before Refactor is "done."

## Code Quality Standards

- Code must pass `gofmt`, `go vet`, and `golangci-lint` clean before a
  Refactor phase is considered done.
- **Every returned error is handled or explicitly wrapped and returned.**
  No `_ = err`, no ignored error from a function call, ever. This is
  non-negotiable in Go and is one of the fastest ways to fail a review.
- DRY, but avoid premature abstraction: don't extract an interface or a
  shared helper until you have three concrete call sites that need it.
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
- Every exported identifier (function, type, const) gets a doc comment
  starting with its name, per Go convention: `// Validate checks that...`.
  This is standard Go, not optional polish.
- Self-explanatory naming reduces the need for comments. A function called
  `calc` needs a comment; a function called `totalWithTax` doesn't.

## Testing Strategy

- **Unit tests** cover business logic: validation, state transitions,
  anything that doesn't touch an external dependency. They use small
  interfaces and hand-written fakes, run in milliseconds, and are part of
  the normal Red → Green → Refactor loop.
- **Integration tests** cover the edges where the real dependency's behavior
  matters: queries, constraints, transactions, cascade deletes, wire
  formats. They run against the real thing (e.g. Postgres via
  `docker compose`), not a mock. A mock only confirms what you already
  believe the dependency does.
- Integration tests are a separate, slower group behind the `integration`
  build tag (`//go:build integration`), never mixed into the fast unit
  suite. The dependency must be running before they're runnable.
- HTTP handlers are tested with `httptest`, through the real router, so
  routing, status codes, and the JSON error envelope are covered too.

## Structured Workflow

Complex assignments use the `/new-go-assignment` skill, which orchestrates
the phases below. Every `docs/` path means `assignments/<name>/docs/`.

1. **Intake** — read the assignment, fill `docs/QUESTIONS.md` from the
   Universal Questions Checklist in `STANDARDS.md`, get answers before
   designing.
2. **Standards review** — pick the relevant options from `STANDARDS.md`,
   log the choice and reasoning in `docs/DECISIONS.md`.
3. **Design** — produce `docs/DESIGN.md`: package layout, exported
   function signatures, a checklist of what needs implementing.
4. **TDD implementation** — Red → Green → Refactor per unit, with stopping
   points for review after: test file skeleton is written, first passing
   test group, each package's implementation is complete.
5. **Debrief** — `docs/DEBRIEF.md`: what tradeoffs were made and why,
   what you'd do differently with more time.

Each stopping point requires explicit go-ahead before advancing to the
next phase. Do not skip ahead on your own.

## Live-Interview Mode

The checkpoint structure above is for **practice reps**, where pausing for
review is the point. If this is running during an actual timed, AI-assisted
interview: skip the stop-and-wait approvals in phase 4, keep `docs/DECISIONS.md`
to one line per decision instead of a full writeup, and save the debrief for
after submission. Say "live mode" at the start of a session to switch.
