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
