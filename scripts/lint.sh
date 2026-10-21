#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project_dir="$PWD"
export PATH="/usr/local/go/bin:$PATH"
cd backend
if [ "$#" -eq 0 ]; then set -- run ./...; fi
exec "$project_dir/.local/bin/golangci-lint" "$@"
