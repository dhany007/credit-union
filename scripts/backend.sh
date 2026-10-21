#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [ -f .env ]; then set -a; source .env; set +a; fi
cd backend
command="${1:-api}"
if [ "$#" -gt 0 ]; then shift; fi
case "$command" in
  api) target=./cmd/api ;;
  migrate) target=./cmd/migrate ;;
  seed) target=./dev/cmd/seed ;;
  *) echo "Usage: bash scripts/backend.sh [api|migrate|seed]" >&2; exit 2 ;;
esac
if [ "$#" -ne 0 ]; then echo "Unexpected arguments" >&2; exit 2; fi
exec /usr/local/go/bin/go run "$target"
