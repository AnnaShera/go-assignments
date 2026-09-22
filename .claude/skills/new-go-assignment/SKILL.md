# New Go Assignment Workflow

This skill guides developers through test-driven development (TDD) assignments in Go, following a structured five-phase workflow with explicit review checkpoints.

## Overview

The workflow automates setup and guides users through five phases:

1. **Setup & Intake** — Creates folder structure, reads the assignment, documents requirements and edge cases in `docs/QUESTIONS.md`
2. **Standards Review** — Confirms technical decisions (error handling, package structure, testing strategy, etc.) in `docs/DECISIONS.md`
3. **Design** — Maps package layout, defines exported function signatures, and creates implementation checklist in `docs/DESIGN.md`
4. **TDD Execution** — Writes failing tests (Red), implements code (Green), refactors (Refactor), with regular review checkpoints
5. **Debrief** — Documents tradeoffs, lessons learned, and what you'd do differently in `docs/DEBRIEF.md`

## Key Features

- **Fast intake mode** (10–15 minutes) with compact documentation using the Universal Questions Checklist from `STANDARDS.md`
- **Explicit stop points** between phases for user review and approval before advancing
- **Red → Green → Refactor discipline** with commits after each phase transition
- **TDD pattern validation** during implementation with behavior coverage summaries
- **Integration testing support** with `docker-compose` for database-dependent tests
- **Standards checklist** ensuring error handling, code quality, and Go conventions are met

## Output Structure

Generates documentation under `docs/` with files:
- `docs/QUESTIONS.md` — Requirements and edge cases from the assignment
- `docs/DECISIONS.md` — Technical decisions made and rationale
- `docs/DESIGN.md` — Package layout, exported signatures, implementation checklist
- `docs/DEBRIEF.md` — Tradeoffs, lessons learned, reflections on the approach

Implementation goes into the project's existing package structure (`cmd/`, `pkg/`, or as specified by the assignment).

## Technical Constraints

- Go 1.22+
- Uses `go test ./...` for unit testing, integration tests via `docker-compose`
- Linting via `gofmt`, `go vet`, and `golangci-lint`
- Emphasizes error handling, clear package boundaries, and maintainable code
- Commits after every Green or Refactor step during TDD (small commits, not batch commits)
- No naked `interface{}` or reflection without justification
- DRY principle applies, but avoid premature abstraction (wait for 3 call sites)

## Workflow Phases

### Phase 1: Setup & Intake
- Read the assignment and extract requirements
- Fill `docs/QUESTIONS.md` using the Universal Questions Checklist from `STANDARDS.md`
- Clarify ambiguities before moving forward
- Create the initial project structure (cmd/, pkg/, or docs/ directories as needed)

### Phase 2: Standards Review
- Pick relevant choices from `STANDARDS.md` (validation approach, error handling, DB testing strategy, etc.)
- Log each decision in `docs/DECISIONS.md` with reasoning
- Align on technical direction before design work

### Phase 3: Design
- Produce `docs/DESIGN.md` with:
  - Package layout and responsibility boundaries
  - Exported function signatures (including error returns)
  - Interface definitions if needed
  - Data structures (types, structs)
  - High-level algorithm/flow notes
  - Checklist of what needs implementing

### Phase 4: TDD Implementation
- **Red:** Write failing tests first (describe behavior, then implement)
- **Green:** Write minimal code to pass tests
- **Refactor:** Clean up, optimize, extract common patterns (only when tests are passing)
- Commit after each Green or Refactor phase (small commits showing your process)
- Stop and wait for review approval after:
  - Test file skeleton is written (before any implementation)
  - First passing test group
  - Each package's implementation is complete
  - All tests passing and linters clean

### Phase 5: Debrief
- Document in `docs/DEBRIEF.md`:
  - What tradeoffs were made and why
  - What surprised you during implementation
  - What you'd do differently with more time
  - Patterns you discovered or applied
  - Edge cases you discovered after design

## Live Interview Mode

If running during a timed interview:
- Skip stop-and-wait approvals in Phase 4 (keep moving)
- Keep Phase 2 decisions brief (one line per decision, not full writeups)
- Save the debrief for after submission
- Say "live mode" at the start of a session to enable this

## Before You Start

Ensure you've read:
- The project's `STANDARDS.md` (for "Default for this repo" choices)
- The project's `CLAUDE.md` (for conventions and expectations)
- The assignment brief (requirements, constraints, examples)

The skill will prompt you for clarifications and guide you through each phase.
