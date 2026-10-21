#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [ -f .env ]; then set -a; source .env; set +a; fi
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-$DATABASE_URL}"
cd backend
export GIN_MODE=test
exec /usr/local/go/bin/go test -race -v ./... "$@"
