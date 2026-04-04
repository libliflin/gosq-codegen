#!/usr/bin/env bash
# snapshot.sh — Collect Go project state for the lathe agent.
# Output goes to stdout as markdown. The engine captures it.

set -euo pipefail

# macOS ships without `timeout`; use gtimeout (coreutils) or a no-op fallback
if command -v timeout &>/dev/null; then
    TIMEOUT=timeout
elif command -v gtimeout &>/dev/null; then
    TIMEOUT=gtimeout
else
    TIMEOUT=""
fi
_to() { if [[ -n "$TIMEOUT" ]]; then $TIMEOUT "$@"; else shift; "$@"; fi; }

echo "# Project Snapshot"
echo "Generated: $(date)"
echo ""

# Git state
echo "## Git Status"
git status --short 2>/dev/null || echo "(not a git repo)"
echo ""

echo "## Recent Commits (last 10)"
git log --oneline -10 2>/dev/null || echo "(no commits)"
echo ""

# Go module info
echo "## Go Module"
if [[ -f go.mod ]]; then
    head -3 go.mod
else
    echo "(no go.mod found — pre-modules project)"
fi
echo ""

# Build
echo "## Build"
echo '```'
_to 30 go build ./... 2>&1 && echo "OK — builds clean" || true
echo '```'
echo ""

# Tests
echo "## Tests"
echo '```'
_to 60 go test ./... -count=1 2>&1 || true
echo '```'
echo ""

# Vet
echo "## Vet"
echo '```'
_to 15 go vet ./... 2>&1 && echo "OK — vet clean" || true
echo '```'
echo ""

# Package structure
echo "## Packages"
go list ./... 2>/dev/null || echo "(no packages — may need go.mod)"
echo ""

# File overview
echo "## Go Files"
find . -name '*.go' -not -path './vendor/*' -not -path './.lathe/*' | sort | head -50
echo ""

# TODOs and FIXMEs
echo "## TODOs"
grep -rn 'TODO\|FIXME\|HACK\|XXX' --include='*.go' . 2>/dev/null | head -20 || echo "(none)"
echo ""

# Test coverage (quick, no output files)
echo "## Coverage"
echo '```'
_to 60 go test ./... -cover 2>&1 | grep -E 'coverage:|FAIL|ok' || echo "(no coverage data)"
echo '```'
