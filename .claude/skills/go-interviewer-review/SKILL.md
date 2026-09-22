# Go Interviewer Review Skill

This skill conducts a comprehensive code assessment of a completed Go assignment across eight quality dimensions, scoring each 0–10 with senior-level verdicts and actionable feedback.

## Overview

The review evaluates a Go codebase from an interviewer's perspective, assessing both technical correctness and software engineering judgment. It provides holistic scoring and targeted recommendations, not line-by-line nitpicks.

## Eight Quality Dimensions

### 1. **Error Handling** (0–10)
- Every error explicitly handled or wrapped and returned
- Errors carry context (`fmt.Errorf` with `%w`)
- Custom error types used appropriately for domain errors
- No `_ = err` or silent failures
- Error messages are actionable and descriptive

### 2. **Code Clarity & Readability** (0–10)
- Function and variable names are self-documenting
- Code structure matches responsibility (packages, functions, methods)
- Comments explain *why*, not *what*
- Unnecessary abstractions avoided (YAGNI principle applied)
- Cyclomatic complexity is reasonable per function

### 3. **Package Design & Architecture** (0–10)
- Packages reflect responsibility, not implementation layers
- Package boundaries are clear and intentional
- Interfaces used only at abstraction points (not over-designed)
- No `interface{}` without justification
- Exported API is minimal and well-documented

### 4. **Testing Coverage & Quality** (0–10)
- Happy path, error cases, and edge cases covered
- Table-driven tests used appropriately
- Unit tests isolated from external dependencies (fakes for interfaces)
- Integration tests against real database where applicable
- Test names describe behavior, not just "Test" + function name
- Mocking used judiciously (real DB tested, not mocked)

### 5. **Go Idioms & Conventions** (0–10)
- Idiomatic error handling (`if err != nil`, error wrapping)
- Follows Go naming conventions (CamelCase, acronyms)
- Uses `context.Context` for cancellation and deadlines
- Concurrency handled safely (no races, mutex/channel use appropriate)
- Defer used for cleanup (file closes, transaction rollbacks)
- No `panic` for expected failures

### 6. **Correctness & Robustness** (0–10)
- Logic handles edge cases (nil, empty, zero values, boundaries)
- Race-condition free (passing `-race` in tests)
- No resource leaks (connections, goroutines, file handles)
- Invariants are maintained (pre/post conditions of functions)
- SQL queries are safe from injection; parameters used correctly

### 7. **Performance Awareness** (0–10)
- Algorithms chosen appropriately for the problem
- No obvious inefficiencies (quadratic where linear is possible)
- String concatenation uses `strings.Builder` or `fmt.Sprintf` appropriately
- Database queries are efficient (indexes, prepared statements, N+1 detection)
- Premature micro-optimization avoided (readability not sacrificed for marginal gains)

### 8. **Meeting Requirements** (0–10)
- All specified features implemented
- Edge cases from requirements handled
- Behavior matches the assignment brief
- Integration tests passing
- No skipped tests or TODO comments left behind

## Output Format

The review produces a structured report with:

1. **Dimension Scores** — Each dimension scored 0–10 with brief rationale
2. **Strengths** — 3–5 specific positive observations (what the candidate did well)
3. **Growth Areas** — 3–5 specific improvements (ranked by priority: correctness > readability > optimization)
4. **Senior-Level Verdict** — Overall assessment: "Strong Hire" / "Hire" / "Feedback Required" / "No Hire"
5. **Interview Talking Points** — 2–3 questions an interviewer could ask to probe deeper understanding

## How to Use

Invoke this skill after you've completed an assignment and want a comprehensive code review:

```
/go-interviewer-review
```

Or review a specific file/package:

```
/go-interviewer-review ./pkg/orders
```

The skill will:
1. Read the assignment brief (if `docs/DESIGN.md` or `docs/INTAKE.md` exists)
2. Scan the implementation for patterns, errors, and structure
3. Run `go test -v -race ./...` to check test status and race conditions
4. Run `gofmt`, `go vet`, and `golangci-lint` to check code quality
5. Assess against the eight dimensions
6. Produce a scored report with actionable feedback

## Grading Scale

- **9–10:** Production-ready code, demonstrates senior-level judgment
- **7–8:** Solid, interview-quality code with minor areas for growth
- **5–6:** Acceptable, shows understanding but has meaningful gaps
- **3–4:** Below interview standard, significant issues to address
- **0–2:** Does not meet basic requirements

## Interview Context

This review is designed to mimic what a senior engineer would say in a debrief call after a take-home assignment. The verdict ("Hire", "Feedback Required", etc.) reflects whether the code would advance in a hiring process, not whether the candidate is good — it's about *readiness for this role*.

Scores are not averages; a single critical flaw (e.g., all errors ignored) can make the verdict "No Hire" even if most dimensions score 7–8.
