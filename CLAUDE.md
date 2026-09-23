# Home Assignment Instructions (Go, TDD)

Read `STANDARDS.md` before designing any solution and use its "Default for
this repo" choices unless the assignment gives a specific reason to deviate.

Pushes are automatically gated by a `.claude/settings.json` hook —
`.claude/hooks/pre-push-check.sh` blocks on `gofmt`, `go test`, and
`golangci-lint` failures.

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

## Testing Strategy for This Stack

- **Unit tests** for business logic (validation, status transitions, anything
  not touching the database) use fakes/interfaces. No real DB dependency,
  fast, part of the normal Red-Green-Refactor loop.
- **Integration tests** for the database layer (queries, foreign key and
  cascade delete behavior, transactions) run against a real Postgres
  instance via `docker-compose`, not a mock. A mocked DB won't catch a
  cascade delete that's wired wrong, only a real one will.
- Integration tests are a separate, slower test group (build tag or
  `_integration_test.go` suffix), not mixed into the fast unit suite.
  `docker-compose up` must be running before they're runnable.

## Structured Workflow

Complex assignments use the `/new-go-assignment` skill, which orchestrates:

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
