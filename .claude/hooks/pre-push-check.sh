#!/bin/bash
# Pre-push quality gate. Keep in sync with Tooling & Quality Gates in STANDARDS.md.
# Exit 0 allows the Bash call; exit 2 blocks it and shows stderr to Claude.

block() {
  echo "Push blocked: $*" >&2
  exit 2
}

input=$(cat)

# Match `push` only as git's subcommand, anywhere in the payload: after
# global options (`git -C dir push`, `git --no-pager push`), after a chain
# (`&&`, `;`), or on a new line (a JSON-escaped `\n`). A commit message or
# description that merely mentions "push" must not match: at a Red step the
# tests fail on purpose, so a match would block the commit.
push_re='(^|[^[:alnum:]_-]|[\][nt])git([[:space:]]+(-[Cc][[:space:]]+[^[:space:]]+|--?[[:alnum:]-]+(=[^[:space:]]+)?))*[[:space:]]+push\b'
grep -Eq "$push_re" <<<"$input" || exit 0

[ -n "$CLAUDE_PROJECT_DIR" ] || block "CLAUDE_PROJECT_DIR is not set."
cd "$CLAUDE_PROJECT_DIR" || block "cannot cd to $CLAUDE_PROJECT_DIR."

# PATH first, then where `go install` puts it.
lint=$(command -v golangci-lint)
if [ -z "$lint" ]; then
  gobin="$(go env GOPATH)/bin"
  for candidate in "$gobin/golangci-lint" "$gobin/golangci-lint.exe"; do
    if [ -x "$candidate" ]; then
      lint=$candidate
      break
    fi
  done
fi
[ -n "$lint" ] ||
  block "golangci-lint not found on PATH or in $(go env GOPATH)/bin. Install v2."
case "$("$lint" version --short 2>/dev/null)" in
  2.*) ;;
  *) block "golangci-lint v2 required, found: $("$lint" version --short 2>&1)." ;;
esac

unformatted=$(gofmt -l .) || block "gofmt failed."
[ -z "$unformatted" ] || block "not gofmt-aligned (fix with: gofmt -w .):
$unformatted"

# Each assignment is its own module; there is no root go.mod.
shopt -s nullglob
for modfile in assignments/*/go.mod; do
  moddir=$(dirname "$modfile")

  out=$(cd "$moddir" && go test ./... 2>&1) ||
    block "go test failed in $moddir:
$out"

  out=$(cd "$moddir" && "$lint" run ./... 2>&1) ||
    block "golangci-lint failed in $moddir:
$out"

  out=$(cd "$moddir" && "$lint" run --build-tags=integration ./... 2>&1) ||
    block "golangci-lint failed in $moddir (--build-tags=integration):
$out"
done

exit 0
