# Assignment Workflow

How an assignment moves from brief to merge, and which files each step
reads and writes. `CLAUDE.md` and `STANDARDS.md` hold the rules; the
`/new-go-assignment` skill runs these steps. Every `docs/` path means
`assignments/<name>/docs/`.

```
┌───────────────────────────────────────────────┐
│ SESSION STARTS                                │
│ CLAUDE.md loads automatically (how we work)   │
└───────────────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 0: SETUP                                │
│ Branch assignment/<name> off main             │
│ Creates: assignments/<name>/docs/, go.mod     │
│ Adds: a row to the root README.md table       │
└───────────────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 1: INTAKE                               │
│ Reads: STANDARDS.md Traits + Questions        │
│ Creates: docs/QUESTIONS.md (brief, answers)   │
│          docs/DECISIONS.md (traits first)     │
└───────────────────────────────────────────────┘
                        │
                        │  (STOP: wait for go-ahead)
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 2: STANDARDS REVIEW                     │
│ Applies: every 'always' section, plus each    │
│          whose 'Applies when:' matches        │
│ Updates: docs/DECISIONS.md (deviations,       │
│          layer shape, dependencies)           │
└───────────────────────────────────────────────┘
                        │
                        │  (STOP: wait for go-ahead)
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 3: DESIGN                               │
│ Creates: docs/DESIGN.md (layout, layer        │
│          shape, signatures, data flow,        │
│          build-order checklist)               │
└───────────────────────────────────────────────┘
                        │
                        │  (STOP: wait for go-ahead)
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 4: TDD IMPLEMENTATION                   │
│ Red -> Green -> Refactor, one unit at a time  │
│ Commit every phase: (Red) (Green) (Refactor)  │
│ Push the branch only from green; the hook     │
│ gates gofmt, go test, golangci-lint           │
│                                               │
│ STOPS (practice mode only):                   │
│ - first test skeleton                         │
│ - first passing test group                    │
│ - each package complete                       │
└───────────────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│ PHASE 5: DEBRIEF & SUBMISSION CHECK           │
│ Creates: docs/DEBRIEF.md, assignment README   │
│ Runs: every applicable command in CLAUDE.md   │
│       (integration, -race, govulncheck, ...)  │
│ Then: follow the README from a clean clone    │
└───────────────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│ REVIEW, in this order:                        │
│ 1. /code-review            correctness        │
│ 2. /go-tests-scanner       test signal        │
│ 3. /go-interviewer-review  score, verdict     │
└───────────────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│ MERGE assignment/<name> into main             │
│ The user's call, never automatic              │
└───────────────────────────────────────────────┘
```

Review order matters: fix correctness first, then test quality, then score,
so the interviewer review grades the final code rather than code that's
about to change.

**Live mode** skips the practice stops in Phase 4 and defers `DEBRIEF.md`
until after submission. The assignment README and the submission check
still happen, because a reviewer who can't run the code stops there.
