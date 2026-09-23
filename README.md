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

| Assignment | What it is | Read first |
|---|---|---|
| [vuln-findings-api](assignments/vuln-findings-api) | REST API for vulnerability findings, backed by Postgres | `docs/DEBRIEF.md` |

## How an assignment runs

Start with `/new-go-assignment`. It moves the assignment through five phases:

1. **Intake.** Read the brief, extract requirements, log open questions in `docs/QUESTIONS.md`.
2. **Standards review.** Pick technical defaults from `STANDARDS.md` and log them in `docs/DECISIONS.md`.
3. **Design.** Package layout, exported signatures, and an implementation checklist in `docs/DESIGN.md`.
4. **TDD implementation.** Red → Green → Refactor with explicit checkpoints, committing after each step.
5. **Debrief.** Trade-offs, lessons learned, and next steps in `docs/DEBRIEF.md`.

Before submitting, run the review skills in the order given in [`WORKFLOW.md`](WORKFLOW.md).

## Review skills

- **`/code-review`:** bugs and simplification opportunities.
- **`/go-tests-scanner`:** test suite quality and signal-to-noise ratio.
- **`/security-review-checklist`:** audit against OWASP Top 10, OWASP API Security Top 10, and OWASP LLM Top 10.
- **`/go-interviewer-review`:** 8-dimension scoring with a senior-level verdict.
- **`/grill-me`:** interview-style stress test of a plan or design.
- **`/pair-explainer`:** step-by-step explanation of every code chunk while writing.

## Quality gate

`.claude/hooks/pre-push-check.sh` runs for **each module under `assignments/*/`**. A push is blocked unless:

- `gofmt` reports no unformatted files
- `go test ./...` passes
- `golangci-lint` is clean

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
