#!/usr/bin/env bash
# test_all.sh — Run all Go tests in the okr module.
#
# Usage:
#   ./scripts/test_all.sh        # all packages
#   ./scripts/test_all.sh -v     # verbose
#
# Environment:
#   DATABASE_URL  If set, uses existing PostgreSQL (DevContainer).
#                 If unset, testcontainers-go starts a Docker container.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== All Go Tests ==="
echo "Go root: ${GO_ROOT}"

if [[ -n "${DATABASE_URL:-}" ]]; then
    echo "DB mode: existing PostgreSQL (DATABASE_URL)"
else
    echo "DB mode: testcontainers-go"
fi
echo ""

cd "${GO_ROOT}"

go test ./... -timeout 180s -count=1 "$@"

echo ""
echo "=== Done ==="
