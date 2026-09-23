# Go Assignments

A workspace for test-driven Go assignments with guided workflows and comprehensive code review skills.

## What is this?

This repository scaffolds structured, interview-prep Go assignments with:
- **TDD discipline** — Red → Green → Refactor workflow with guided phases
- **Built-in skills** — Assignment scaffolding, code review, test quality evaluation, security review, interview prep
- **Standards reference** — Curated decision guide for Go patterns and tradeoffs
- **Pre-commit gates** — Automatic checks: gofmt, go test, golangci-lint

## Getting started

Start a new assignment:
```bash
/new-go-assignment
```

The skill guides you through five phases:
1. **Intake** — Read assignment, extract requirements, document edge cases
2. **Standards Review** — Pick technical decisions from STANDARDS.md
3. **Design** — Define package structure, exported signatures, checklist
4. **TDD Implementation** — Red → Green → Refactor with explicit checkpoints
5. **Debrief** — Document tradeoffs, lessons learned, reflections

## Available skills

- **`/security-review-checklist`** — Security audit against OWASP Top 10, OWASP LLM Top 10, OWASP API Security Top 10
- **`/code-review`** — Bug and simplification detection
- **`/go-interviewer-review`** — 8-dimension code assessment with senior-level verdict
- **`/go-tests-scanner`** — Test suite quality evaluation with signal-to-noise ratio
- **`/grill-me`** — Interview-style stress-test of a plan or design until reaching shared understanding
- **`/pair-explainer`** — Forces step-by-step explanation of every code chunk while writing

See `WORKFLOW.md` for the recommended order to run these in before submitting.

## Before pushing

Push is blocked by hooks until:
- Code passes `gofmt`
- All tests pass (`go test ./...`)
- `golangci-lint` is clean

## Repository structure

```
go-assignments/
├── CLAUDE.md — TDD discipline and conventions
├── STANDARDS.md — Go decision reference (error handling, HTTP, database, etc.)
├── README.md — This file
├── .claude/skills/ — Assignment skills
├── .gitignore — Go project exclusions
└── assignments/
    ├── vuln-findings-api/ — Example: REST API with Postgres
    ├── your-assignment/ — Next assignment
    └── ...
```

## Standards & Decision Framework

See [`STANDARDS.md`](STANDARDS.md) for:
- Universal questions checklist for project intake
- Error handling patterns (sentinel vs custom types vs wrapped errors)
- Data modeling and validation approaches
- HTTP routing, database access, testing patterns
- Code quality gates and conventions

Every assignment logs decisions in `docs/DECISIONS.md` with rationale from this reference.

## Recommended workflow

1. **Start** → Run `/new-go-assignment` in this workspace
2. **Implement** → Red → Green → Refactor, commit after each phase
3. **Review** → Run security, code, and interviewer review skills
4. **Iterate** → Fix feedback, push when hooks pass
5. **Debrief** → Document tradeoffs and lessons

---

See [`CLAUDE.md`](CLAUDE.md) for full TDD discipline and project conventions.
