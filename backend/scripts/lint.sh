#!/usr/bin/env bash
#
# lint.sh - Run linters and formatters
#
# Usage:
#   ./scripts/lint.sh          - Run all linters
#   ./scripts/lint.sh -fix     - Auto-fix issues
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

FIX=false
if [[ "${1:-}" == "-fix" ]]; then
    FIX=true
fi

echo "🔍 Running linters..."

# Format check.
#
# Capture the file list instead of piping into `grep -q`: under `set -o pipefail`
# grep's early exit sends SIGPIPE to gofmt, so the pipeline reports failure even
# when every file is formatted. Assigning to a variable keeps the test on the
# output itself, which is what we actually care about.
echo "  📝 Checking formatting..."
unformatted="$(gofmt -l .)"
if [[ -z "$unformatted" ]]; then
    echo "    ✅ Code is formatted"
elif [[ "$FIX" == true ]]; then
    echo "    Fixing formatting..."
    gofmt -w .
    echo "    ✅ Formatting fixed"
else
    echo "    ❌ Code is not formatted. Run: gofmt -w ."
    echo "$unformatted" | sed 's/^/       /'
    exit 1
fi

# Vet check
echo "  🔎 Running go vet..."
if go vet ./...; then
    echo "    ✅ go vet passed"
else
    echo "    ❌ go vet found issues"
    exit 1
fi

# golangci-lint (if available)
if command -v golangci-lint &>/dev/null; then
    echo "  🔧 Running golangci-lint..."
    if [[ "$FIX" == true ]]; then
        golangci-lint run --fix ./...
    else
        golangci-lint run ./...
    fi
    echo "    ✅ golangci-lint passed"
else
    echo "    ⚠️  golangci-lint not installed (optional)"
fi

echo "✅ Linting complete"
