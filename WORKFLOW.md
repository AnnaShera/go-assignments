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
   │ Commit at every Red, Green, Refactor.   │
   │ Every push gated by the pre-push hook:  │
   │ • gofmt, go test, golangci-lint         │
   │                                         │
   │ STOPS AFTER (practice mode only):       │
   │ • Test skeleton written (Red)           │
   │ • First test group passes (Green)       │
   │ • Each package complete                 │
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
        ┌─────────────────────────────────────┐
        │ SUBMISSION READINESS CHECKS         │
        │                                     │
        │ Run in order:                       │
        │ 1. /code-review       (correctness) │
        │ 2. /go-tests-scanner  (test signal) │
        │ 3. integration tests + go test -race│
        │    (by hand; the gate skips both)   │
        │ 4. /go-interviewer-review  (score)  │
        └─────────────────────────────────────┘
```

Fix correctness first, then test quality, then score, so the interviewer
review grades the final code rather than code that's about to change.
