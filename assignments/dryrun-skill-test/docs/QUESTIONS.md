# Intake Questions — dryrun-skill-test

Assignment: `DedupeSort(items []string) []string` (dedupe + alphabetical
sort), wrapped in `POST /dedupe` (`{"items":[...]}` → `{"result":[...]}`).
Throwaway assignment used only to dry-run the `/new-go-assignment` skill.

## Complexity & Scale
- **Expected input size / volume / request rate:** Unspecified; trivial for
  this dry-run, not a real constraint.
- **Memory constraints:** None.
- **Latency / throughput requirements:** None.
- **Optimize for time, space, or readability:** Readability — this is an
  interview-style assignment, not a performance exercise.

## Correctness & Edge Cases
- **How should invalid input be handled?** OPEN — see Q1 below.
- **Numeric edge cases:** N/A, input is `[]string`.
- **Is the input guaranteed non-empty, or must empty/nil be handled?** Must
  handle empty/nil gracefully — an empty `items` array returns `{"result":[]}`.
  Distinct from malformed JSON, which is Q1.
- **Ordering guarantees on the input:** None assumed; output must be
  alphabetically sorted regardless of input order.

## Concurrency
- **Single-goroutine or concurrent calls?** Standard `net/http` per-request
  handling; no shared state, so no additional concurrency handling needed.
- **Shared state / mutex / channel:** None.
- **Timeout or cancellation via `context.Context`?** Not needed — no I/O,
  pure in-memory computation. Handler still accepts `context.Context` per
  repo idiom even though unused here.

## Persistence
- **Should any state survive between calls or runs?** No.
- **Storage requirement:** None — stateless, in-memory only.

## Output
- **Exact output format required?** Yes: `{"result": [...]}` JSON, array
  alphabetically sorted, deduplicated.
- **Should errors be surfaced to the caller?** Yes, per the repo's HTTP
  Error Response Contract: `{"error": "...", "code": "..."}` with an
  appropriate status.

## Open Questions — Resolved

**Q1. Missing/malformed `items` field:** Return `400` with
`{"error": "items must be an array of strings", "code": "invalid_input"}`.
Applies to a missing `items` key, wrong type, or invalid JSON body.
Distinct from a present-but-empty `items: []`, which is valid and returns
`{"result": []}`.

**Q2. Case sensitivity:** Case-sensitive. `"Apple"` and `"apple"` are
distinct entries; sort uses standard Go lexicographic (byte) ordering.
