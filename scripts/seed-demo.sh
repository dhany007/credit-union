#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker_cli="${DOCKER:-docker}"
compose() { "$docker_cli" compose --env-file .env.docker "$@"; }

# make seed starts/migrates the stack before running this script.
compose run --rm --no-deps seed
restore_services() {
  local status=$?
  trap - EXIT
  if ! compose up -d --no-build --wait api web; then
    echo 'Could not restart services; inspect docker compose logs.' >&2
    status=1
  fi
  exit "$status"
}
trap restore_services EXIT
compose stop web api

stamp="$(date -u +%Y%m%dT%H%M%SZ)-$$"
backup=".local/backups/before-demo-${stamp}.dump"
mkdir -p .local/backups
compose exec -T db pg_dump -U credit_union -d credit_union -Fc -f "/tmp/before-demo-${stamp}.dump"
compose cp "db:/tmp/before-demo-${stamp}.dump" "$backup"
chmod 600 "$backup"
echo "Local database backup: $backup"

compose cp backend/dev/data/demo_2026.sql db:/tmp/demo_2026.sql
compose cp backend/dev/data/demo_2026_reconcile.sql db:/tmp/demo_2026_reconcile.sql
# Fixed historical snapshot; reproduce the same fixture even when run later.
compose exec -T db psql -U credit_union -d credit_union \
  -v ON_ERROR_STOP=1 -v local_demo=true -v apply=true \
  -v cutoff=2026-10-21T09:05:52.915146Z -f /tmp/demo_2026.sql
compose exec -T db psql -U credit_union -d credit_union \
  --single-transaction -v ON_ERROR_STOP=1 -f /tmp/demo_2026_reconcile.sql
echo 'Demo data verified: 15 members, 500 transactions each (including opening deposits).'
