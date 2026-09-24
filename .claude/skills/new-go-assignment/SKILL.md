---
name: new-go-assignment
description: Runs a Go take-home or interview-prep assignment end to end in this repo, scaffold, Intake, Standards review, Design, TDD implementation, then Debrief and submission check, writing each phase to assignments/<name>/docs/ and stopping for explicit go-ahead between phases. Use when the user says "new assignment", "new take-home", "start the assignment", or pastes an assignment brief, and when resuming an assignment that already has a docs/ folder.
---

# New Go Assignment

Runs the workflow in `CLAUDE.md` ("Structured Workflow") using the
defaults in `STANDARDS.md`. Those two files are the source of truth. This
skill only sequences them, so when they say something more specific,
follow them.

**Every phase transition needs the user's explicit go-ahead.** Never
advance because a phase "looks done." If you're about to write code before
`docs/DESIGN.md` is approved, stop.

All `docs/` paths below mean `assignments/<name>/docs/`.

## Phase 0: Setup

1. Confirm `CLAUDE.md` and `STANDARDS.md` exist at the repo root. They're
   shared by every assignment, never copied into one. If either is
   missing, stop and tell the user.
2. Ask whether this is a **practice rep** or **live mode**, unless the user
   already said "live mode" this session. Note the mode; it changes
   phases 4 and 5.
3. Get the brief. Don't go further without it.
4. Scaffold the assignment:
   - Agree on a short kebab-case `<name>` with the user.
   - Branch off an up-to-date `main`: `assignment/<name>`. Never work on
     `main`.
   - Create `assignments/<name>/docs/`.
   - From `assignments/<name>/`, run
     `go mod init github.com/AnnaShera/go-assignments/assignments/<name>`,
     and make sure the `go` directive matches the README prerequisites.
   - Add a row to the Assignments table in the repo-root `README.md`,
     with status "In progress".
   - Commit: `Scaffold <name> assignment`.

   Other folders (`cmd/`, `internal/`, `migrations/`, `testdata/`) are
   created in Phase 4, when a test first needs them.

## Phase 1: Intake

1. Answer every row of **Traits** in `STANDARDS.md` with yes or no and a
   one-line reason. Watch the trap it names: an HTTP server with an
   in-memory store is Concurrency: yes. Write the answers as the first
   lines of `docs/DECISIONS.md`.
2. Create `docs/QUESTIONS.md`:
   - Under `## Brief`, paste the brief verbatim, so later reviews can
     check requirements against it.
   - Go through the **Universal Questions Checklist**, keeping the groups
     that apply. For each question, either answer it from the brief
     (quote the line) or mark it **OPEN**.
3. Get every OPEN question answered. Use `AskUserQuestion` when there's a
   small set of reasonable options, plain conversation otherwise, and
   recommend an option when one is clearly better. Never guess to avoid
   asking. In live mode, if the interviewer can't be reached, record your
   assumption and mark it **ASSUMED**.
4. Update `docs/QUESTIONS.md` with the answers. Commit:
   `Intake: traits and questions for <name>`.
5. **Stop.** Summarize in 2–3 sentences: the traits, and anything the
   answers changed about scope. Wait for the go-ahead.

## Phase 2: Standards review

1. List the sections that apply: every `always` section, plus every
   section whose `Applies when:` matches a trait answered yes.
2. Take each section's default. For each deviation, add a line to
   `docs/DECISIONS.md` in the format `STANDARDS.md` prescribes:
   `**Topic:** chose X instead of the default Y, because Z.`
3. Also record everything else the sections ask to be recorded: the layer
   shape (Layers, under Project Layout), one line per third-party
   dependency (Dependency Policy), and where auth would go
   (Authentication & Authorization, HTTP only).
4. Keep it short. Defaults don't need restating, just a line naming the
   sections applied. In live mode, one line per decision.
5. Commit: `Standards review for <name>`.
6. **Stop.** List the sections applied and any deviations. Wait for the
   go-ahead.

## Phase 3: Design

1. Write `docs/DESIGN.md` with:
   - Package layout: a tree with a one-line purpose per package.
   - Layer shape: thin CRUD or business rules.
   - Exported types and function signatures per package, interfaces
     especially, each declared in the package that consumes it. Phase 4
     tests are written against these.
   - Data flow: one line per request, command, or message path.
   - An implementation checklist in the **build order** from `CLAUDE.md`,
     including only the steps whose trait applies.
2. Commit: `Design for <name>`.
3. **Stop.** Walk through the design in a few sentences. No test or
   implementation code before the go-ahead.

## Phase 4: TDD implementation

Take the checklist one unit at a time: a function, an endpoint, or a
package.

1. **Red:** add the signature as a stub that returns zero values, then
   write the failing tests. Run `go test ./...` and confirm that only the
   new tests fail, and that they fail on an assertion, not a compile
   error. Commit: `<summary> (Red)`.
2. **Green:** make the smallest change that passes. Run `go test ./...`
   in full. Commit: `<summary> (Green)`.
3. **Refactor:** only on green. Never add behavior here. `gofmt`,
   `go vet`, and `golangci-lint` must all be clean. Commit:
   `<summary> (Refactor)`. If there's nothing to refactor, skip it; never
   make an empty commit.

The Working Rules in `CLAUDE.md` apply throughout. In particular: never
weaken a failing test, no new dependency without asking, and a decision
made mid-implementation gets its line in `docs/DECISIONS.md` before the
code that depends on it. Push the `assignment/<name>` branch only from
green.

**Practice-mode stops.** Stop and wait at each of these:

- after the first test skeleton, before implementing
- after the first passing test group
- after each package is complete

At each stop, report exactly this and nothing longer: a 1-line summary of
what was done, the files changed (as links), and the next step.

**Live mode:** no stops. Move through Red → Green → Refactor continuously,
still committing every phase.

## Phase 5: Debrief and submission check

1. Write `docs/DEBRIEF.md`: the trade-offs made and why (link the
   `docs/DECISIONS.md` entries behind them), and what you'd do
   differently with more time. In live mode, defer this until the user
   asks for it after submission.
2. Write the assignment's own `README.md` with the sections, in the
   order, given under **Delivery** in `STANDARDS.md`. This isn't deferred
   in live mode: a reviewer who can't run the code stops there.
3. Run the submission check:
   - Every applicable row of the Commands table in `CLAUDE.md` must pass.
     That includes integration tests, `-shuffle=on`, `govulncheck`, and
     `go mod tidy -diff`. Run `-race` wherever the Tooling & Quality
     Gates section says it can run, and tell the user plainly if it
     wasn't run.
   - Clone the repo into an empty folder in the scratchpad, check out the
     branch, and follow the assignment README exactly, as the reviewer
     will. Fix whatever breaks.
4. Update the assignment's status in the repo-root README table. Commit.
5. Suggest the review order from `WORKFLOW.md`: `/code-review`, then
   `/go-tests-scanner`, then `/go-interviewer-review`.
6. Merging `assignment/<name>` into `main` is the user's call. Ask, don't
   merge.

## If resuming mid-assignment

If `assignments/<name>/docs/` already has some of these files, don't
restart. Read what exists, say which phase you think is next and why,
and confirm before continuing:

| Present | Next phase |
|---|---|
| nothing, or no Traits in `DECISIONS.md` | 1 |
| `QUESTIONS.md` with no OPEN items, but no standards review in `DECISIONS.md` | 2 |
| `DECISIONS.md` complete, no `DESIGN.md` | 3 |
| `DESIGN.md` with unchecked checklist items | 4 |
| every checklist item done, no `DEBRIEF.md` or assignment `README.md` | 5 |
