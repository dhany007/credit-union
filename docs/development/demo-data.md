# Local demo data

The development-only fixture is `backend/dev/data/demo_2026.sql`, with independent assertions in `demo_2026_reconcile.sql`. It inserts directly into the local PostgreSQL database; it does not exercise HTTP commands or create fake idempotency receipts.

## Fresh clone setup

Git carries the SQL fixture and its instructions, not another developer's database volume. On a fresh checkout:

1. Copy `.env.docker.example` to `.env.docker` and set local passwords as described in the [Docker runbook](../deployment/docker-local.md).
2. Run `make seed` from WSL/Linux with GNU Make, Bash, and Docker Compose available. It builds/starts the stack, applies migrations, seeds staff, pauses writers, saves a backup, imports and reconciles the fixture, then resumes services.
3. Open http://localhost:3000. Run the fixture before registering other members: it refuses unrelated existing financial data. `make up` starts services without adding seed data.

`make seed` is the recommended shortcut; the lower-level PowerShell commands below remain available. Normal startup does not seed sample financial data. An already populated local database keeps its data across restarts using the Docker volume. The fixture SQL, Makefile, and wrapper scripts are versioned together so a fresh clone can reproduce the same dataset.

## Dataset

- 15 clearly marked DEMO members, each with exactly 500 transactions, including the IDR 50,000 opening deposit.
- 7,500 total movements: 3,971 deposits and 3,529 withdrawals for the recorded cutoff.
- Every member has activity in every month from January 2026 through October 21, 2026, across both KCPs.
- Synthetic posting hours are Monday through Saturday, 09:00–16:00 Asia/Jakarta. Amounts are multiples of IDR 5,000. No interest or loan activity is simulated.
- NIKs deliberately use a non-real `00` region prefix; names, phone numbers, addresses, and notes are labeled synthetic. Do not use these as real KTP examples.
- Account numbers are allocated by the existing sequence. The verified empty local database allocated `1000000001` through `1000000015`.

Random amounts and servicing branches use a fixed seed. Withdrawals cannot exceed the member balance or the servicing KCP's available cash. Each movement has linked before/after balances, two correctly directed journal lines, and an attributed audit event. Registration creates an additional audit event per member, plus one batch manifest.

## Reproducible execution

Use only the local demo database, after migrations and the existing staff seed. The fixture refuses nonempty financial data. Keep a database backup and pause API/web writers before applying it. It never deletes or replaces existing data.

From the project root in PowerShell:

```powershell
# Fixed historical cutoff reproduces the verified dataset, including on later dates.
$cutoff = '2026-10-21T09:05:52.915146Z'
docker compose --env-file .env.docker cp backend/dev/data/demo_2026.sql db:/tmp/demo_2026.sql
docker compose --env-file .env.docker cp backend/dev/data/demo_2026_reconcile.sql db:/tmp/demo_2026_reconcile.sql
# Preview temporary tables only; no account numbers or persistent IDs are consumed.
docker compose --env-file .env.docker exec -T db psql -U credit_union -d credit_union -v ON_ERROR_STOP=1 -v local_demo=true -v apply=false -v "cutoff=$cutoff" -f /tmp/demo_2026.sql
# After reviewing the preview and taking a backup:
docker compose --env-file .env.docker stop web api
docker compose --env-file .env.docker exec -T db psql -U credit_union -d credit_union -v ON_ERROR_STOP=1 -v local_demo=true -v apply=true -v "cutoff=$cutoff" -f /tmp/demo_2026.sql
# Inspect the exit status; apply commits only after all assertions pass.
docker compose --env-file .env.docker exec -T db psql -U credit_union -d credit_union --single-transaction -v ON_ERROR_STOP=1 -f /tmp/demo_2026_reconcile.sql
docker compose --env-file .env.docker start api web
```

The fixture is specific to 2026 and a cutoff date represented in the generated business-day schedule. The example uses the recorded historical cutoff to reproduce the verified dataset on later dates. To generate through a different supported business date in 2026, capture the database clock once instead; use that same cutoff for preview and apply. Batch `demo-2026-15x500-v1` reruns verify the committed manifest and existing data, then exit without inserting rows or consuming sequences. A conflicting or modified batch fails verification; it does not repair or reseed it.

## Verified local result — October 21, 2026

The full import and repeat-run behavior passed first in a separate database restored from the empty local backup. The fixture was then committed to the main local application database and independently reconciled again.

| Measure | Verified value |
| --- | ---: |
| Members | 15 |
| Transactions per member | 500 |
| Total transactions | 7,500 |
| Journal lines | 15,000 |
| Total deposits | IDR 843,685,000 |
| Total withdrawals | IDR 731,710,000 |
| Total member closing balances | IDR 111,975,000 |
| KCP Pusat closing cash | IDR 50,700,000 |
| KCP Utara closing cash | IDR 61,275,000 |
| Journal debits / credits, each | IDR 1,575,395,000 |

Checks cover each member's counts/months, opening deposit, chronological balance chain, registration/time bounds, nonnegative historical member and KCP balances, actor/branch mapping, each transaction's journal direction/amount, audit attribution, and source-to-target checksum. The committed plan checksum is `663c06142a8d26b3d284ad3ff061cada`; the captured cutoff is `2026-10-21T09:05:52.915146Z`. Backup and detailed reports remain ignored under `.local/`.

Open the Transactions menu, choose a custom period from January 1 through October 21, 2026, and optionally filter by account number. The default Today filter intentionally shows only today's subset.

API smoke verification also passed: 15 members; account-filtered custom-period pagination returned exactly 500 unique references across 10 pages, with before/after snapshots present. Today returned 15 transactions. Required-flag rejection returned a nonzero status, and a repeat run against the main local database reconciled successfully without inserting rows.
