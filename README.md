# Go Assignments

Production-grade Go take-home assignments, built test-first with Claude as a pair programmer.

## Why this repo exists

I use this repo to practice Go backend interview assignments the way I would ship real code: tests first, decisions documented, quality enforced by tooling.
The goal is a **repeatable process that holds up under interview time pressure**, not just finished assignments.

## What it demonstrates

- **Strict TDD.** Every behavior starts as a failing test (Red → Green → Refactor).
- **Documented trade-offs.** Each assignment records its choices in `docs/DECISIONS.md`, checked against the defaults in `STANDARDS.md`.
- **Enforced quality.** Claude Code cannot push until every module passes `gofmt`, `go test`, and `golangci-lint` (including `integration`-tagged code).
- **AI as pair programmer, not autopilot.** Claude skills guide the process and review the result. I make the design decisions and can explain every line.

## Assignments

| Assignment | What it is | Status |
|---|---|---|
| [vuln-findings-api](assignments/vuln-findings-api) | REST API for vulnerability findings, backed by Postgres | In progress. Built before the workflow existed, so there is no `docs/` yet |

## Getting started

**Prerequisites**

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/) v2 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`), found on `PATH` or in `$(go env GOPATH)/bin`
- Docker (only needed for integration tests)
- [Claude Code](https://claude.com/claude-code), for the skills and the push gate

**Run an assignment's checks**

```bash
cd assignments/<name>
go test -shuffle=on ./...                      # unit tests, in random order
golangci-lint run ./...                        # lint
docker compose up -d                           # only if the assignment has integration tests
go test -tags=integration ./...                # integration tests against the real dependency
```

The full list, including race, `govulncheck`, and benchmarks, is the
Commands table in [`CLAUDE.md`](CLAUDE.md).

## How an assignment runs

Start with `/new-go-assignment`. It scaffolds the assignment on its own
`assignment/<name>` branch, then moves it through five phases, stopping for
review between each:

1. **Intake.** Classify the assignment by traits (HTTP, Database, CLI,
   Concurrency, and so on), then log the brief and every open question in
   `docs/QUESTIONS.md` and get them answered.
2. **Standards review.** Apply every `STANDARDS.md` section that matches
   those traits, and log deviations in `docs/DECISIONS.md`.
3. **Design.** Package layout, layer shape, exported signatures, data flow,
   and a build-order checklist in `docs/DESIGN.md`.
4. **TDD implementation.** Red → Green → Refactor, with a commit at every
   phase so the git log shows the process.
5. **Debrief and submission check.** Trade-offs in `docs/DEBRIEF.md`, the
   assignment's own README, every check passing, and a run from a clean
   clone.

Then run the review skills in the order given in [`WORKFLOW.md`](WORKFLOW.md).

## Skills

These ship with the repo in `.claude/skills/`:

- **`/new-go-assignment`:** scaffolds an assignment and runs the five phases above.
- **`/go-tests-scanner`:** test suite signal-to-noise, and coverage gaps against the build order.
- **`/go-interviewer-review`:** 8-dimension scoring against `STANDARDS.md` and the assignment's own decisions, with a hiring verdict.

Also used: **`/code-review`** (bugs and simplification opportunities), from Anthropic's official `code-review` Claude Code plugin.

## Quality gate

`.claude/hooks/pre-push-check.sh` runs whenever Claude Code runs `git push`. A push is blocked unless:

- `gofmt` reports no unformatted files anywhere in the repo
- `go test ./...` passes in each module under `assignments/*/`
- `golangci-lint` is clean in each module, both with and without the `integration` build tag

It also blocks if `golangci-lint` v2 isn't found on `PATH` or in `$(go env GOPATH)/bin`.

## Repository layout

```
go-assignments/
├── CLAUDE.md          How we work: working rules, TDD, build order, workflow
├── STANDARDS.md       What we build with: traits, defaults, intake checklist
├── WORKFLOW.md        End-to-end flow diagram
├── .claude/
│   ├── skills/        Assignment and review skills
│   └── hooks/         Pre-push quality gate
└── assignments/
    └── <name>/        One Go module per assignment
        ├── cmd/       Entry point, if there's a binary
        ├── internal/  Application code
        ├── docs/      QUESTIONS, DECISIONS, DESIGN, DEBRIEF
        ├── README.md  How to run, test, and use it
        └── go.mod
```

The full per-assignment layout, including migrations, `testdata/`, and the
Dockerfile, is Project Layout in [`STANDARDS.md`](STANDARDS.md).

See [`CLAUDE.md`](CLAUDE.md) for the full TDD rules and conventions.
