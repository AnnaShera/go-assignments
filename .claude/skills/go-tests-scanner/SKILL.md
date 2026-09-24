---
name: go-tests-scanner
description: Evaluates a Go assignment's test suite in this repo for signal-to-noise, finding low-value tests, coverage gaps against the CLAUDE.md build order, consolidation opportunities, fake/mock misuse, and non-determinism, with file:line citations and a prioritized remove/consolidate/add list. Use when the user asks to review, scan, or audit tests, or as step 2 of the submission review order in WORKFLOW.md. Read-only; it doesn't change code.
---

# Go Tests Scanner

Evaluates a test suite from a pragmatist's angle. Not "what's the
coverage percentage" but "is each test earning its keep, and is anything
that matters untested?"

**Read-only.** Report findings; don't fix them.

## Before scanning

1. Read the assignment's `docs/DECISIONS.md` (for the traits and the layer
   shape) and `docs/QUESTIONS.md` (for the edge cases that were agreed).
2. Read Testing Patterns in `STANDARDS.md` and the build order in
   `CLAUDE.md`. They define what "covered" means here.
3. From `assignments/<name>/`, run `go test -shuffle=on ./...` and
   `go test -cover ./...`. Coverage numbers point at untested files; they
   are not a target.

## What it evaluates

### 1. Low-value tests (noise)
- Tests of trivial getters or setters with no logic.
- Assertions that can't fail: comparing a value to itself, or asserting a
  constant.
- Tests of code that can't fail, such as `json.Marshal` on a plain struct.
- **Weak assertions:** checking only `err == nil` or `x != nil`, never the
  value that matters.
- Table rows that don't vary anything meaningful.

### 2. Coverage gaps (missing signal)
Check each build-order step whose trait applies:
- **Domain:** `Validate()` with a valid case, each invalid field, and
  several invalid fields at once, asserting that **every** field is
  reported.
- **Service** (business-rules shape only): happy path, not found,
  conflict, and validation passthrough, each against fakes.
- **Stream processing** (Large input): empty input, a line longer than
  64 KiB, and a malformed record reported with its line number.
- **HTTP:** status code, JSON body, and the error envelope for **each**
  error type, through the router.
- **CLI:** output, the returned error, usage errors, and a table test for
  `exitCode`.
- **Messages:** success, retryable failure, and a duplicate message.
- **Concurrency:** a test hitting the shared state from many goroutines
  at once.
- **Repository** (Database): happy path, not found, unique violation, FK
  violation, and pagination order.
- **Wiring:** one smoke test against the real database.

Also flag:
- error branches with no test
- exported functions never exercised
- edge cases agreed in `QUESTIONS.md` with no matching test

### 3. Consolidation opportunities
- Near-identical test functions that differ only in input should become
  one table-driven test.
- Repeated setup should become a helper that calls `t.Helper()` and
  registers cleanup with `t.Cleanup`.

### 4. Fakes and mocks
- **Faking what shouldn't be faked:** a pure function, the standard
  library, `database/sql`, or the driver. Queries are tested against the
  real schema.
- **An integration test that silently uses a fake**, when the real
  dependency is the point of the test.
- **Fakes that drift from the real thing:** a fake whose behavior (errors
  especially) isn't pinned by an integration test of the real
  implementation.
- **Interface placement:** interfaces should be declared by the consumer
  and kept to the methods it uses. Hand-written fakes are preferred over a
  mocking framework unless the interface is large.

### 5. Determinism and isolation
- `time.Sleep`, the real clock, or dependence on map or test order. Time
  should be injected, or tested with `testing/synctest` in concurrent
  code.
- Integration tests that rely on seed data or on another test running
  first. Each test creates its own data and cleans up after itself.
- Integration tests that aren't behind `//go:build integration`, or that
  are mixed into the fast unit suite.

### 6. Organization and debuggability
- Test names describe behavior (`TestOrderTotal_ReturnsErrorWhenNegativeAmount`),
  and subtest names identify the case.
- Failure messages show got vs. want. Use `t.Fatalf` only when the test
  can't meaningfully continue.
- Large expected outputs live in golden files under `testdata/`.

## Output

1. **Signal-to-noise:** high-value tests / total, as a rough percentage.
   Above 75% is healthy, 50–75% needs cleanup, and below 50% needs
   significant consolidation.
2. **Low-value tests:** `file:line` and why.
3. **Coverage gaps:** grouped by build-order step, each with the test to
   add.
4. **Consolidations:** which tests merge, with a short before and after.
5. **Fake and determinism issues:** `file:line` and the fix.
6. **Recommended actions,** in order: remove, consolidate, add.

## Reference: weak vs. strong

A weak test passes whenever `ParseOrder` returns anything at all:

```go
func TestParseOrder(t *testing.T) {
    o, err := ParseOrder(validJSON)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if o == nil {
        t.Fatal("got nil order")
    }
}
```

A strong one pins the behavior, including *which* error:

```go
func TestOrderTotal(t *testing.T) {
    tests := []struct {
        name    string
        items   []Item
        want    int64
        wantErr error
    }{
        {"empty order totals zero", nil, 0, nil},
        {"single item", []Item{{PriceCents: 1000}}, 1000, nil},
        {"negative price is rejected", []Item{{PriceCents: -5}}, 0, ErrNegativePrice},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := OrderTotal(tt.items)
            if !errors.Is(err, tt.wantErr) {
                t.Fatalf("error = %v, want %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("total = %d, want %d", got, tt.want)
            }
        })
    }
}
```

`errors.Is(nil, nil)` is true, so the same line covers both the success
rows and the error rows.
