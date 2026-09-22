# Go Tests Scanner Skill

This skill evaluates the quality and completeness of a Go test suite, identifying low-value tests, coverage gaps, redundancy opportunities, and mocking practices. It outputs a signal-to-noise ratio assessment and actionable recommendations.

## Overview

The tests scanner analyzes your test suite from a pragmatist's angle: not "how much coverage percentage" but "are your tests *earning their keep*?" It detects patterns that waste time during development and maintenance without catching real bugs.

## What It Evaluates

### 1. **Low-Value Tests** (Noise)
Flags tests that add no discriminating power:
- Testing trivial getters/setters with no logic
- Tautological assertions (`if err != nil` → `require.NoError(t, err)`)
- Tests for code that cannot fail (e.g., `json.Marshal` on a concrete struct)
- Empty test tables or table entries that don't vary meaningfully
- Assertions that always pass (comparing a value to itself, no side effects tested)

### 2. **Coverage Gaps** (Signal Missing)
Identifies critical paths not tested:
- Happy path only; error branches untested
- Function exported and used but never called in tests
- Edge cases mentioned in comments but not in test cases
  - Nil pointers not checked
  - Empty slices not handled
  - Zero values not considered
  - Boundary conditions (min, max, off-by-one)
- External dependencies (DB queries, HTTP calls) not integration-tested

### 3. **Redundancy Opportunities** (Consolidation)
Detects tests that should be merged:
- Nearly identical test functions differing only in one input → consolidate to table-driven
- Test tables with repeated setup/teardown → extract fixture or helper
- Multiple sequential `t.Run` subtests with independent concerns → consider regrouping by theme

### 4. **Mocking Practices** (Smell Check)
Flags over-mocking and drift risks:
- Mocking a simple pure function or standard library function
- Mock that differs materially from the real implementation (e.g., error behavior)
- Mocking to avoid real DB when integration test is the point
- Missing mock-vs-real regression tests (comparing mock behavior to actual)

### 5. **Test Organization** (Structure)
Evaluates readability and maintainability:
- Test names describe behavior (`TestOrderTotal_ReturnsErrorWhenNegativeAmount`)
- Subtests grouped logically and consistently (`t.Run("error cases", ...)`)
- Setup/teardown (fixtures, helpers) separate from assertions
- No copy-paste test code; common patterns extracted

### 6. **Assertions & Error Messages** (Debuggability)
Checks test failure messages:
- Assertion errors include actual vs. expected values
- Custom error messages explain *why* the assertion matters
- `t.Errorf` used over `t.Fatalf` where flow can continue
- Table-driven test output identifies failing case clearly

## Output Format

The scanner produces a report with:

1. **Signal-to-Noise Ratio** — Percentage of tests pulling their weight (rough: high-value / total)
2. **Low-Value Tests Found** — List with line numbers and why (triviality, tautology, etc.)
3. **Coverage Gaps** — List of untested critical paths and recommendations
4. **Redundancy Opportunities** — Tests that should be consolidated with before/after examples
5. **Mocking Issues** — Specific calls to review or replace with fakes
6. **Recommended Actions** — Prioritized list: remove, consolidate, add (in that order)

## How to Use

Invoke this skill on your completed assignment:

```
/go-tests-scanner
```

Or scan a specific package:

```
/go-tests-scanner ./pkg/orders
```

The skill will:
1. Parse all `*_test.go` files in the current directory/package
2. Analyze test function names, table structure, assertions, mocking patterns
3. Cross-reference with coverage data (if available via `go test -cover`)
4. Produce a prioritized list of low-value tests and coverage gaps
5. Suggest consolidations and additions

## Scoring

- **Signal-to-Noise > 75%** — Healthy test suite, small improvements possible
- **Signal-to-Noise 50–75%** — Solid core, some noise to clean up
- **Signal-to-Noise < 50%** — Too many low-value tests, significant consolidation needed

## Common Patterns to Avoid

### ❌ Low-Value: Testing Trivial Getters
```go
func TestUser_GetEmail(t *testing.T) {
    u := &User{Email: "test@example.com"}
    if u.GetEmail() != "test@example.com" {
        t.Errorf("got %s, want test@example.com", u.GetEmail())
    }
}
```
**Delete this.** It tests the language itself, not your logic.

### ❌ Low-Value: Tautological Assertions
```go
func TestParseOrder(t *testing.T) {
    o, err := ParseOrder(validJSON)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if o == nil {
        t.Errorf("got nil")
    }
}
```
**Why?** If `err != nil`, you'd catch it. If `ParseOrder` is well-written, `o` won't be nil when `err` is nil. Test the actual parse logic.

### ✅ Good: Table-Driven with Edge Cases
```go
func TestOrderTotal(t *testing.T) {
    tests := []struct {
        name      string
        items     []Item
        wantTotal int
        wantErr   bool
    }{
        {"empty order", nil, 0, false},
        {"single item", []Item{{Price: 10}}, 10, false},
        {"negative price returns error", []Item{{Price: -5}}, 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            total, err := OrderTotal(tt.items)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
            if total != tt.wantTotal {
                t.Errorf("got %d, want %d", total, tt.wantTotal)
            }
        })
    }
}
```
**Why?** Clear cases, edge cases covered, subtest names identify failing case instantly.

## Integration Test Guidance

For database-backed code:
- Real DB integration tests use `docker-compose`, run in `*_integration_test.go` with `// +build integration` tags (or `_test.go` in a separate suite)
- Mock only at the business logic layer (e.g., a `Repository` interface with fakes for testing handlers)
- Never mock `database/sql` or the DB driver; test queries against real schema

## After the Review

The scanner does *not* auto-fix. Use the report to:
1. **Delete** low-value tests immediately (they slow you down)
2. **Consolidate** near-duplicate tests into one table-driven test
3. **Add** tests for identified coverage gaps (especially error branches and edge cases)
4. **Refactor mocks** to fakes for pure-logic tests, reserve mocks for boundaries only

Re-run the scanner after changes to confirm improvement.
