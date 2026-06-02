#!/usr/bin/env bash
# test_repository.sh
#
# Usage:
#   ./scripts/test_repository.sh           # run all repository tests
#   ./scripts/test_repository.sh -v        # verbose
#   ./scripts/test_repository.sh -run User # filter by test name
#   ./scripts/test_repository.sh -run Team
#
# Environment:
#   DATABASE_URL  If set, connects to existing PostgreSQL (DevContainer).
#                 If unset, testcontainers-go starts a Docker container (CI).
#
# Example (DevContainer):
#   DATABASE_URL=postgresql://postgres:pass@postgres:5432/learning \
#       ./scripts/test_repository.sh -v

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== Repository Integration Tests ==="
echo "Go root: ${GO_ROOT}"

if [[ -n "${DATABASE_URL:-}" ]]; then
    echo "Mode:    existing PostgreSQL (DATABASE_URL set)"
else
    echo "Mode:    testcontainers-go (Docker required)"
fi
echo ""

cd "${GO_ROOT}"

# Pass any extra args (e.g. -v, -run Xxx) straight to go test
go test ./infrastructure/repository/... \
    -timeout 120s \
    -count=1 \
    "$@"

echo ""
echo "=== Done ==="
