#!/bin/bash
input=$(cat)
command=$(echo "$input" | grep -o '"command" *: *"[^"]*"' | head -1 | sed -E 's/.*"command" *: *"//; s/"$//')

case "$command" in
  *"git push"*) ;;
  *) exit 0 ;;
esac

if [ -z "$CLAUDE_PROJECT_DIR" ]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR" || exit 0

unformatted=$(gofmt -l .)
gofmt_status=$?

if [ $gofmt_status -ne 0 ]; then
  echo "Push blocked, gofmt failed:" >&2
  exit 2
fi

if [ -n "$unformatted" ]; then
  echo "Push blocked, not gofmt-aligned:" >&2
  echo "$unformatted" >&2
  echo "Fix with: gofmt -w ." >&2
  exit 2
fi

test_output=$(go test ./... 2>&1)
test_status=$?

if [ $test_status -ne 0 ]; then
  echo "Push blocked, go test failed:" >&2
  echo "$test_output" >&2
  exit 2
fi

lint_bin="$(go env GOPATH)/bin/golangci-lint.exe"

if [ ! -x "$lint_bin" ]; then
  echo "Push blocked, golangci-lint not found at $lint_bin:" >&2
  exit 2
fi

lint_output=$("$lint_bin" run ./... 2>&1)
lint_status=$?

if [ $lint_status -ne 0 ]; then
  echo "Push blocked, golangci-lint failed:" >&2
  echo "$lint_output" >&2
  exit 2
fi

exit 0
