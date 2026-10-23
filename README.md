# Credit Union

A small multi-KCP cooperative application: Go API, PostgreSQL, and Next.js with shadcn/ui and the selected tweakcn Zen Inspired Theme.

Start with [Local Docker setup](#local-docker-setup) and [Makefile commands](#makefile-commands). The root [Makefile](Makefile) provides `make up`, `make seed`, and `make release VERSION=v0.1.0`.

## Implemented MVP

- Teller/Supervisor login with hashed passwords and database-backed HTTP-only sessions.
- Member registration using unique 16-digit NIK, member number, home KCP, and an automatically allocated 10-digit savings account number.
- IDR 50,000 opening cash deposit posted atomically with registration.
- Cross-KCP cash deposits and withdrawals by exact account-number lookup, with balance/cash checks and transaction history.
- Dedicated Transactions menu with exact account-number filtering and cooperative-wide Teller/Supervisor visibility with Today, Last 7 days, Last 30 days, inclusive custom dates, and 50-row pagination (including opening deposits).
- Loan applications, assigned responsible KCP, Supervisor decisions, and decision reasons.
- Balanced transaction journals, atomic before/after member balance snapshots, actor/KCP audit records, request retry protection, and concurrent withdrawal protection.

No disbursement, repayment, automatic savings interest, or member portal is enabled. Loan approval does not move cash or create outstanding principal.

## Local Docker setup

See the [local Docker runbook](docs/deployment/docker-local.md) for configuration, backups, and troubleshooting. Docker uses its own PostgreSQL volume and a separate ignored `.env.docker`; the existing WSL database and `.env` remain separate.

On a fresh checkout only, copy `.env.docker.example` to `.env.docker` and set unique passwords. Keep the existing file if already configured. Then run from the project root:

```bash
make up
# Optional: seed staff plus 15 demo members / 7,500 cash transactions.
make seed
```

Open **http://localhost:3000**. Sign in with a development username below and the `DEV_SEED_PASSWORD` from `.env.docker`. Only the frontend port is published, bound to localhost. Seed is explicit and does not run on normal startup. For staff accounts without demo members, run `docker compose --env-file .env.docker run --rm seed` after `make up`.

### Optional demo members and transactions

`make seed` creates the KCPs and staff accounts, then loads the reconciled local dummy fixture. The lower-level Compose `seed` service creates only KCPs and staff. Database contents live in a local Docker volume and are not included in a Git clone.

To populate a fresh local database with 15 synthetic members and 7,500 reconciled cash transactions, follow [Local demo data](docs/development/demo-data.md#fresh-clone-setup). `make seed` handles startup, staff seeding, backup, dummy import, reconciliation, and service restart. Ordinary startup never inserts demo financial data. Existing local installations that already loaded the fixture do not need to run it again.

## Makefile commands

The root [Makefile](Makefile) is the command entry point. Run it from the repository root. It requires GNU Make, Bash, Docker with Compose, and a configured `.env.docker`. On Windows, use WSL with Docker Desktop integration enabled. No host Go or Node installation is needed for the Docker workflow. Run `make help` (or just `make`) to list the available commands.

```bash
make up                         # Build, migrate, start, and wait for healthy services
make seed                       # Also seed staff and the fixed 2026 dummy dataset
make release VERSION=v0.1.0      # New GitHub tag and Release from remote main
```

`make seed` keeps a private backup under `.local/backups/`, pauses financial writers during import, and resumes services even if the import fails. A matching unchanged fixture is verified without duplicating data. Existing financial data that is not this batch is refused.

`make release` requires GitHub CLI (`gh`), `gh auth login`, repository write access, and a new stable `vMAJOR.MINOR.PATCH` version. It resolves the configured GitHub repository and remote `main` SHA, rejects an existing tag, and publishes a GitHub Release with generated notes at that SHA. Local working files and the current branch are not uploaded. Merge the intended changes into `main` before releasing. This command publishes a release; it does not deploy containers or build release assets.

## Local WSL setup

Requirements: PostgreSQL, Node 24+, npm, and Go 1.26+ (the Go module can download its matching toolchain). Commands run from this project root.

```bash
cp .env.example .env
# Set a unique DEV_SEED_PASSWORD of at least 12 characters in .env.
# Create a dedicated database owned by your PostgreSQL role:
sudo -u postgres createdb --owner="$USER" credit_union
# Adjust DATABASE_URL if your database user/connection differs.
cd backend && go mod download && cd ..
npm --prefix frontend ci
bash scripts/backend.sh migrate
bash scripts/backend.sh seed
```

The development environment created during implementation already has `.env`, the database, and seed accounts. Do not overwrite that `.env` when resuming local work. Never commit `.env` or use development accounts for production.

Start in two terminals:

```bash
bash scripts/backend.sh
```

```bash
npm --prefix frontend run dev
```

Open **http://localhost:3000**. Next.js proxies `/api` to `127.0.0.1:8080`. Use `localhost`, matching `APP_ORIGIN`; changing the browser origin requires updating the backend setting. All UI financial operations use whole rupiah, with a technical maximum of IDR 1 trillion per operation/account (not a lending policy).

### Development users

| Username | Role | KCP |
| --- | --- | --- |
| teller.a | Teller | KCP Pusat |
| teller.b | Teller | KCP Utara |
| supervisor | Supervisor | KCP Pusat; cooperative-wide decision scope for this MVP |

Passwords are the local `DEV_SEED_PASSWORD` supplied when seed first runs. Seed does not overwrite existing users/passwords. There is no public registration for staff and no password reset/admin UI in this phase.

### Approval policy

`APPROVAL_LIMIT_IDR=0` disables approval until the owner chooses a limit. Rejection and application intake remain available. A positive explicitly configured value enables one small rule: `regular` product, `low` risk, amount up to the limit, Supervisor role, across both KCPs. Medium/high risk or higher amounts remain blocked. Teller records a preliminary risk category; Supervisor must verify it and record decision reasons. This is not automated credit scoring. Restart the API after changing the setting.

No production limit has been invented or enabled. The tests use a clearly isolated test-only limit to verify successful and blocked approval.

### Financial assumptions and limits

- Initial IDR 50,000 is treated as an ordinary savings deposit, not membership capital or a fee. KTP must be checked manually; 16 digits are format validation, not Dukcapil verification.
- Home KCP is the registering Teller's KCP. The responsible KCP is explicitly selected at loan intake; no reassignment interface yet.
- No minimum retained savings balance beyond zero and no fees have been added. A withdrawal also requires sufficient recorded cash at the servicing KCP.
- Branch cash starts at zero and is funded by recorded deposits. Cash transfers, opening-cash setup, and reversal interfaces are not implemented. The app blocks insufficient branch cash rather than inventing funds.
- The central ledger uses KCP dimensions; branch cash can be reconciled, but separate fully balanced branch books/inter-KCP clearing are not implemented.
- Member search returns at most 100 matches; search by name, member number, full account number, or exact NIK to locate others. The cash form resolves a full account number directly, independent of the latest-100 member list. Select the member before opening cash/loan forms when needed. Member-specific history and loan lists are limited to 100. The dedicated Transactions menu has date filtering and pagination.

These assumptions need business review before real-money use. The standard staff seed creates no member or financial sample data; the optional local demo fixture is documented separately above.

## Verification

```bash
bash scripts/test.sh
npm --prefix frontend run build
```

Integration tests create and drop only a randomly named `cu_test_*` schema in the configured database; they never truncate application tables. They cover duplicate membership, cross-KCP service, idempotency, concurrent withdrawals, failed journal rollback, authorization/origin checks, approval rules, and session invalidation. Use a dedicated test database in shared environments.

## Structure

```text
backend/cmd/    Thin API and migration entry points
backend/internal/{auth,member,cash,loan}/
                Domain-local Gin handlers, services, PostgreSQL stores
backend/internal/platform/
                Wiring, config, database commands, HTTP utilities
backend/integration/
                HTTP/PostgreSQL regression tests
frontend/       Next.js, shadcn/ui components, exact theme token snapshot
scripts/        Local API/migration/seed and test commands
docs/           Business decisions, diagrams, ADRs, and roadmap
```

The API uses Gin. Direct Go commands are `go run ./cmd/api`, `go run ./cmd/migrate`, and `go run ./dev/cmd/seed` from `backend/`; the existing shell wrappers above remain supported.

Schema setup uses the explicit migration command with an ordered version ledger and transaction-scoped migration lock. The immutable initial schema is version 1; version 2 assigns savings account numbers to existing and future members; version 3 records before/after member balance snapshots for new cash movements. Run migrations before starting an updated API; see the Docker runbook for the local maintenance procedure. Frontend UI supports Indonesian/English and light/dark themes, with browser-persisted preferences on login and staff screens. Project documentation is English.

Before a real pilot: approve product and accounting rules, configure approval authority, provide correction/reversal and cash-handling procedures, test backups/restores, configure HTTPS/secure cookies, and replace development identities. This build is for local review, not a production rollout.

## Backend refactor

See [backend tech plan](docs/tech-plan-phase-1-backend.md) and [TDD evidence](docs/testing/gin-refactor.tdd.md). Endpoints and financial posting behavior are preserved. Dependency failure during authentication now returns 503 instead of incorrectly claiming invalid credentials or an expired session. The API uses a 15-second request context budget within its existing HTTP server timeouts. No frontend source or financial schema migration is needed.

### Go lint tooling

The project uses `.local/bin/golangci-lint` (ignored by Git). Install the pinned validation tool once if absent:

```bash
GOBIN="$PWD/.local/bin" /usr/local/go/bin/go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
bash scripts/lint.sh config verify
bash scripts/lint.sh
```

Configuration is at `backend/.golangci.yml`. The skill-derived correctness/security/readability linters remain enabled for production and test code, with 120-character lines and shared formatting commands. Narrow annotations explain synthetic credential fixtures and environment-controlled local HTTP cookies.

GitHub Actions runs the same pinned linter and formatting checks for relevant backend changes on branch pushes and pull requests through [.github/workflows/lint.yml](.github/workflows/lint.yml). The workflow can also be started manually and does not require a database or application secrets.

## Automated quality checks

GitHub Actions runs Go lint/formatting and backend vet/race-enabled PostgreSQL tests for backend changes, and frontend ESLint/typecheck/unit tests/production build for frontend changes. Shared Makefile/Compose configuration changes run both stacks; documentation-only changes skip these checks. Frontend local commands are npm run lint, npm run typecheck, npm test, and npm run build from frontend/ using Node24. See [quality checks and coverage](docs/testing/quality-checks.md) for database isolation, limits, and required-check activation.

### Savings interest (Phase 2)

Supervisor can configure the global annual rate and balance bands under Savings interest. Policies take effect tomorrow or later. Set SAVINGS_INTEREST_ENABLED=true in your local .env.docker after reviewing the configuration, then make up. Daily closing and monthly non-cash credits run inside the API; no extra service is needed. See [interest operations](docs/deployment/savings-interest.md) and [accepted rules](docs/savings-interest.md). EN/ID and light/dark remain supported.
