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
- [golangci-lint](https://golangci-lint.run/) v2 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
- Docker (only needed for integration tests)
- [Claude Code](https://claude.com/claude-code), for the skills and the push gate

**Run an assignment's checks**

```bash
cd assignments/<name>
go test ./...                                  # unit tests
golangci-lint run ./...                        # lint
docker compose up -d                           # only if the assignment has integration tests
go test -tags=integration ./...                # integration tests against the real dependency
```

## How an assignment runs

Start with `/new-go-assignment`. It moves the assignment through five phases:

1. **Intake.** Read the brief, extract requirements, log open questions in `docs/QUESTIONS.md`.
2. **Standards review.** Pick technical defaults from `STANDARDS.md` and log them in `docs/DECISIONS.md`.
3. **Design.** Package layout, exported signatures, and an implementation checklist in `docs/DESIGN.md`.
4. **TDD implementation.** Red → Green → Refactor with explicit checkpoints, committing after each step.
5. **Debrief.** Trade-offs, lessons learned, and next steps in `docs/DEBRIEF.md`.

Before submitting, run the review skills in the order given in [`WORKFLOW.md`](WORKFLOW.md).

## Skills

These ship with the repo in `.claude/skills/`:

- **`/new-go-assignment`:** runs the five phases above.
- **`/go-tests-scanner`:** test suite quality and signal-to-noise ratio.
- **`/go-interviewer-review`:** 8-dimension scoring with a senior-level verdict.

Also used: **`/code-review`** (bugs and simplification opportunities), from Anthropic's official `code-review` Claude Code plugin.

## Quality gate

`.claude/hooks/pre-push-check.sh` runs whenever Claude Code runs `git push`. A push is blocked unless:

- `gofmt` reports no unformatted files anywhere in the repo
- `go test ./...` passes in each module under `assignments/*/`
- `golangci-lint` is clean in each module, both with and without the `integration` build tag

## Repository layout

```
go-assignments/
├── CLAUDE.md          TDD rules and the 5-phase workflow
├── STANDARDS.md       Go decision reference and intake checklist
├── WORKFLOW.md        End-to-end flow diagram
├── .claude/
│   ├── skills/        Assignment and review skills
│   └── hooks/         Pre-push quality gate
└── assignments/
    └── <name>/        One Go module per assignment
        ├── go.mod
        └── docs/      QUESTIONS, DECISIONS, DESIGN, DEBRIEF
```

See [`CLAUDE.md`](CLAUDE.md) for the full TDD rules and conventions.
