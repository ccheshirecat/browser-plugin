#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="$REPO_ROOT/build/bin"
TEST_DIR="$REPO_ROOT/tests/integration"

if [ ! -d "$TEST_DIR" ]; then
  echo "Integration tests not implemented yet" >&2
  exit 1
fi

cd "$TEST_DIR"
go test ./...
