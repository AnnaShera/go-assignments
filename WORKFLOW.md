# Assignment Workflow

This document describes the complete flow of a Go assignment from start to finish, including which files are read and created at each stage.

## The Complete Assignment Flow

```
┌─────────────────────────────────────────────────────────┐
│ SESSION STARTS                                          │
│ CLAUDE.md is automatically loaded (TDD rules)           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ /new-go-assignment SKILL LAUNCHED                       │
│ (Orchestrates 5-phase workflow)                         │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
   ┌─────────────┐          ┌──────────────────┐
   │ PHASE 1:    │          │ Reads:           │
   │ INTAKE      │◄─────────┤ STANDARDS.md     │
   │             │          │ (decision guide) │
   │ Creates:    │          └──────────────────┘
   │ docs/       │
   │ QUESTIONS.md│
   └────────────┬┘
                │ (STOP & WAIT FOR APPROVAL)
                │
                ▼
   ┌─────────────────────────┐
   │ PHASE 2:                │
   │ STANDARDS REVIEW        │
   │                         │
   │ Creates:                │
   │ docs/DECISIONS.md       │
   │ (log "we chose X        │
   │  because of Y")         │
   └────────────┬────────────┘
                │ (STOP & WAIT FOR APPROVAL)
                │
                ▼
   ┌─────────────────────────┐
   │ PHASE 3:                │
   │ DESIGN                  │
   │                         │
   │ Creates:                │
   │ docs/DESIGN.md          │
   │ (package layout,        │
   │  signatures, checklist) │
   └────────────┬────────────┘
                │ (STOP & WAIT FOR APPROVAL)
                │
                ▼
   ┌─────────────────────────────────────────┐
   │ PHASE 4: TDD IMPLEMENTATION             │
   │                                         │
   │ Red → Green → Refactor cycles           │
   │                                         │
   │ Each commit checked against:            │
   │ • CLAUDE.md (gofmt, go test, lint)     │
   │ • STANDARDS.md (for patterns)          │
   │                                         │
   │ STOPS AFTER:                            │
   │ • Test skeleton written (Red)           │
   │ • First test passes (Green)             │
   │ • Each package complete                 │
   │ • All tests + linting pass              │
   │                                         │
   │ Creates: Implementation code            │
   └────────────┬────────────────────────────┘
                │
                ▼
   ┌─────────────────────────┐
   │ PHASE 5: DEBRIEF        │
   │                         │
   │ Creates:                │
   │ docs/DEBRIEF.md         │
   │ (tradeoffs, lessons)    │
   └────────────┬────────────┘
                │
                ▼
        ┌──────────────────────────────┐
        │ SUBMISSION READINESS CHECKS  │
        │                              │
        │ Run in order:                │
        │ 1. /security-review-check   │
        │ 2. /code-review             │
        │ 3. /go-interviewer-review   │
        │ 4. /grill-me                │
        └──────────────────────────────┘
```

## File Connections

### CLAUDE.md (Project Rules — Loaded First)
Enforced via:
- Pre-push hooks: `gofmt`, `go test`, `golangci-lint`
- TDD discipline: Red → Green → Refactor
- Testing strategy: unit vs integration tests
- Error handling: every error handled or wrapped
- Code quality: package boundaries, DRY, naming

### STANDARDS.md (Decision Reference)
Consulted during:
- **docs/QUESTIONS.md** — What questions to ask during intake
- **docs/DECISIONS.md** — What choices were made and why
- **docs/DESIGN.md** — Validate architecture against standards
- **Implementation code** — Follow patterns and conventions

### docs/ Files (Assignment Lifecycle)

```
docs/
├── QUESTIONS.md   ← What we need to know (Intake phase)
├── DECISIONS.md   ← Choices we made & reasoning (Standards phase)
├── DESIGN.md      ← Architecture, signatures, checklist (Design phase)
├── [code/tests]   ← Red → Green → Refactor (Implementation phase)
└── DEBRIEF.md     ← What we learned & reflections (Debrief phase)
```

## The Flow in Words

### 1. Session Starts
**CLAUDE.md** is automatically loaded. This contains the TDD discipline, code quality rules, and testing strategy that apply to all your code.

### 2. Run the Skill
```bash
/new-go-assignment
```
This launches the guided 5-phase workflow. The skill:
- Reads **STANDARDS.md** to understand your decision framework
- Guides you through each phase with stop points for review
- Creates docs files to capture your thinking
- Tracks your progress

### 3. Phase 1: Intake
You read the assignment and answer questions from the **Universal Questions Checklist** (in STANDARDS.md). This creates:
- **docs/QUESTIONS.md** — Edge cases, complexity, concurrency, persistence questions

### 4. Phase 2: Standards Review
You pick technical decisions from **STANDARDS.md** (error handling, data modeling, testing strategy, etc.). This creates:
- **docs/DECISIONS.md** — Each choice with reasoning ("We chose X because...")

### 5. Phase 3: Design
You design the solution (package structure, exported signatures, checklist). This creates:
- **docs/DESIGN.md** — Architecture, data structures, algorithm outline

### 6. Phase 4: TDD Implementation
Red → Green → Refactor cycles:
- **Red**: Write failing tests (guided by docs/DESIGN.md)
- **Green**: Write minimal code to pass (follow STANDARDS.md patterns)
- **Refactor**: Clean up (follow CLAUDE.md quality gates)

Each commit is checked against:
- **CLAUDE.md** rules (gofmt, go test, golangci-lint)
- **STANDARDS.md** patterns

### 7. Phase 5: Debrief
You reflect on what you built. This creates:
- **docs/DEBRIEF.md** — Tradeoffs, surprises, what you'd do differently

### 8. Submission Readiness
Before shipping, run review skills in order:
1. **`/security-review-checklist`** — Security audit (OWASP, injection, auth)
2. **`/code-review`** — Bugs and simplifications
3. **`/go-interviewer-review`** — Comprehensive 8-dimension assessment
4. **`/grill-me`** — Practice interview questions

## Key Insights

- **CLAUDE.md** = Rules enforced by hooks (fail fast on bad code)
- **STANDARDS.md** = Options you choose from (explicit decisions)
- **docs/** = Your thinking captured (shows process, not just result)
- **Skills** = Guided checkpoints and reviews

The flow ensures:
- Every decision is intentional (logged in docs/DECISIONS.md)
- Code quality is consistent (CLAUDE.md rules + hooks)
- TDD discipline is maintained (Red → Green → Refactor)
- Learning is captured (docs/DEBRIEF.md)
- Reviews are comprehensive (four-skill sequence before submission)
