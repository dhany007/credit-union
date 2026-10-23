# Tech Plan - Phase 2 savings interest

Status: Accepted by the project owner on 2026-10-23; implemented locally.
Evidence: [TDD and recovery validation](testing/savings-interest.tdd.md).
Date: 2026-10-23
Baseline: main af6b3aa (merged PR #3).

## Memory Context

No project memory-system directory or project-owned AGENTS.md was found. Follow the existing Gin modular monolith, integer-IDR amounts, pgx caller-owned transactions, immutable migrations, and durable request replay. Read docs/savings-interest.md, ADR-0007/0008, accounting.md, roadmap.md, open-questions.md, the cash/member/auth/platform packages, migration 3, integration tests, and the Phase 1 reconciliation SQL.

ADR-0007 already accepts annual whole-balance interest, daily calculation, monthly credit, global base rate, and Supervisor bands. [ADR-0012](adr/0012-savings-interest-calculation-and-posting.md) accepts the remaining rules and starting rates; it does not supersede ADR-0007 or constitute real-money operational sign-off.

## Requirement

Pay savings interest into the same account once per month without changing KCP cash. Persist reproducible daily calculations and policy history. Recover missed runs without duplicate credits, changing completed calculations, or inventing historical eligibility.

Accepted rules: per-account Jakarta closing balance; Actual/365 Fixed, including leap years; integer basis-point rates; whole-IDR rounding once per account/month, half up, no fractional carry; first calendar day after month-end, normally after 00:05 Jakarta. Late credits earn interest from their actual posting day, with no backdating. See ADR-0012 for boundaries, examples, exceptions, accounting, and approval gates.

Non-goals: loan pricing/disbursement, tax engine, account closure/freeze workflow, historical adjustments, general ledger framework, product overrides, approval engine, separate worker service, queue, or metrics platform. Initial activation is a future full business date; no pre-activation interest or synthetic historical backfill.

## Backend Scope

One new internal/interest domain with domain.go, service.go, postgres_store.go, http.go, and job.go as needed. Separate the pure calculator from persistence. Keep main.go as wiring/lifecycle only; reuse auth, fault/httpapi, config, slog, pgx, audit, and command replay patterns.

Frontend code is outside this backend plan. Integration prerequisite before enabling credits: explicitly render interest_credit as a positive non-cash movement in history/receipts, support System attribution and EN/ID, preserve light/dark tokens, and provide a small Supervisor policy/status view. Existing deposit-versus-everything-else rendering is unsafe for the new kind. Do not add another cash menu.

## Codebase Context

- internal/platform/database/schema.sql currently restricts transactions.kind to deposit/withdrawal, requires actor_id, and restricts journal accounts to cash/member_savings. Never edit this applied baseline.
- internal/platform/database/migrations/0003_member_balance_snapshots.sql validates only cash directions. A forward migration must also admit positive interest credits.
- internal/cash/service.go and postgres_store.go own cash member/branch locks and atomic before/after snapshots. internal/member/service.go registers and posts opening cash within the same command transaction.
- Cash History/List/Search INNER JOIN users. Adapt them to LEFT JOIN for system credits while preserving the actor string response.
- internal/platform/database/command.go owns staff retry/fingerprint storage. Reuse it for policy creation; automatic monthly uniqueness is account/period, independent of staff request keys.
- platform/app/app.go wires modules; cmd/api/main.go owns shutdown. The scheduler must stop and join before closing the database pool.
- docs/testing/sql/phase-1-reconciliation.sql assumes cash equals savings and non-deposit means withdrawal. Seed verifiers and integration assertions need an explicit per-kind review.

## API Design

Keep existing status/error shape {"error":"safe message"}, same-origin checks, strict decoding, session and role guards.

| Route | Permission | Contract |
| --- | --- | --- |
| GET /api/interest/policies | Supervisor | Ordered policy versions, including pending/superseded versions; items array. Small cooperative configuration history, no generalized query builder. |
| POST /api/interest/policies | Supervisor | Idempotency-Key; effective_date YYYY-MM-DD, base_rate_bps integer, bands array of {min_balance, max_balance nullable, rate_bps}, reason, expected_latest_id nullable for first policy. Returns persisted version. |
| GET /api/interest/status | Supervisor | enabled, start_date, last_completed_date, outstanding_months, last_success_at, failure_class; no identities or raw internal errors. Bound the pending month list to the activation-to-current period. |

Use 200 for successful creation/replay, matching current command conventions. 400 malformed date/rate/bands, 401 no session, 403 role/origin denied, 409 stale expected version or changed replay payload, 422 disallowed historical effective date, 503 temporary availability/lock timeout; unexpected errors use sanitized 500. Validate every integer bound and request-body limit. DB errors preserve their Go cause internally. Never expose SQL or policy request bodies in logs.

Policy versions are immutable. Under a policy-write advisory lock, compare expected_latest_id and read the DB clock again; effective_date must still be tomorrow or later. A new version can supersede a pending version for the same date; retain both and select the highest ID on that date. For any business date, select the greatest effective_date <= date, then greatest ID. Do not permit deletions or retroactive edits. A single Supervisor may create a version, with reason and durable audit; no maker-checker engine.

Compatibility classification: new policy/status endpoints are additive. Adding interest_credit to existing transaction read endpoints has client impact: current frontend assumes a closed two-kind set. Release API/read projections and frontend support together before enablement. Keep response fields, actor string (System for automatic credits), filters, dates, 50-row cursor ordering, and old staff replay payloads stable. POST /api/cash must continue rejecting interest_credit. Do not provide an HTTP manual arbitrary-date credit endpoint.

## Data and exact calculation

Add four small tables through the next forward migration (currently 0004), registered by database/migrate.go:

1. interest_policies: ID, effective_date, base_rate_bps, complete bands JSONB, created_by, reason, created_at. Whole-version validation is in the service; database constraints also bound rates and required metadata. Index effective_date/ID for dated resolution. The first policy establishes immutable activation start; reject any later version with an earlier effective date, even while activation is still pending.
2. interest_daily: member_id + business_date primary key, policy_id, closing_balance bigint, rate_bps integer, numerator numeric(40,0), calculated_at. Store B * rate_bps exactly; denominator is always 3,650,000. Nonnegative checks and foreign keys; include zero-rate/zero-balance days so absence never means zero. Index by period/date only if the chosen queries require it.
3. interest_days: business_date primary key, eligible_account_count, policy_id and completed_at. Persist the completion marker in the same transaction as all account/day rows, including days with no accounts. This is the durable catch-up watermark; counts and eligible rows must agree.
4. interest_monthly: member_id + period_start primary key (first of month), eligible_days, numerator_total numeric(40,0), credit_amount bigint, transaction_id nullable unique, posted_at. A completed zero-credit result exists with no financial transaction; positive amounts require a linked transaction. Totals must agree with daily rows.

Use math/big integers for multiplication/summing/rounding, not float64 and not daily decimal truncation. Credit = floor((sum(numerators) + 1,825,000) / 3,650,000). Reject overflow before converting to bigint and before balance addition. Also enforce the existing config.MaxAmount technical balance ceiling (currently IDR 1,000,000,000,000), as the current posting paths do; exceeding it fails that settlement without a partial credit. Sub-IDR residual is disclosed and discarded after monthly rounding. Example: IDR 15,000,000 at 450 bps for 30 days gives numerator 202,500,000,000 and credit IDR 55,479.

Expand existing transaction/journal checks for interest_credit and savings_interest_expense. Allow actor_id NULL only for interest_credit; cash keeps its non-null teller identity. Automatic audit events may have NULL actor only for the named system-interest action; policy events retain the Supervisor actor. Store system origin, period, and posting reference. Use the member home KCP as an accounting dimension, not a servicing teller KCP. Keep existing history timestamps and unknown snapshots untouched; new interest snapshots are required and positive.

## Daily cutoff and concurrency

A daily job cannot safely read current balance alone or trust the existing now() transaction-start default. A pre-midnight transaction may commit after midnight.

Introduce one cooperative-wide transaction-scoped advisory barrier using a reserved, documented lock key. Every savings writer acquires it SHARED before any member/branch lock or registration INSERT. Staff command replay locking may precede it; no closer takes staff request locks. Interest credit uses the same barrier then member lock; cash retains member-before-branch order. Policy creation takes the shared barrier before its policy lock to serialize with daily calculation.

After the savings writer has acquired its relevant locks, read DB clock_timestamp once and explicitly use it as the financial row created_at. No client timestamp or backdating. Derive Jakarta business dates from this value. Registration eligibility starts at the opening-deposit posting timestamp, not members.created_at, whose current default is transaction-start time.

For each completed business date d, the closer acquires the barrier EXCLUSIVE. It waits for in-flight writers; only then read the DB clock and query using a fresh READ COMMITTED statement snapshot. Reject an unfinished/future date. The closing balance is current member balance minus signed posted movements at or after next midnight Jakarta. Deposit and interest_credit are positive, withdrawal negative. Select only accounts opened before that cutoff. Pre-activation movements form the opening balance but earn no historical interest. Do not reconstruct from nullable historical before/after snapshots or a timestamp-sorted running balance.

Within one bounded transaction per business date, resolve the dated policy, verify eligibility and existing data, and insert all eligible account/day records. Commit the interest_days completion marker atomically with all account rows, including an empty day. Resume from the last contiguous completed date; never use MAX(account daily date) as proof that an entire day completed. Existing records and marker counts must match on replay; never overwrite a discrepancy. Process dates in order from activation; a missing policy/failed date blocks dependent credits, with no silent zero rate. The brief exclusive gate is a deliberate small-MVP throughput tradeoff; measure it on the current 15 x 500 dataset. Abort and report if a bounded lock/statement timeout is exceeded, rather than adding distributed infrastructure.

## Monthly posting and recovery

One cancellable in-process loop runs on startup and every minute when explicitly enabled. No public job route or separate service. Compute due dates from the DB clock: allow yesterday after 00:05 Jakarta; before that process only older dates. A database job advisory lock on a dedicated acquired connection prevents duplicate concurrent catch-up loops; release it before returning that connection, including errors/cancellation. Database uniqueness is still authoritative. Use a bounded work chunk per tick and resume from completed rows; re-query due work rather than trusting in-memory progress.

After all eligible daily records of a closed month exist, credit each account in its own transaction. Process months oldest first. Eligibility is max(activation day, opening-deposit business date) through month-end. Completeness means every expected date exists with a valid policy and arithmetic; never pay a partial period. Mid-month activation earns only the eligible days. A per-account failure can be reported while other complete accounts progress.

Acquire the shared barrier then member FOR UPDATE, check the unique monthly row, sum daily numerators, round once, and atomically write the positive interest transaction, actual before/after balance, Dr savings_interest_expense / Cr member_savings, monthly completion row, and system audit. Zero result writes completion/audit only. If another process already completed it, return the existing result without posting again. No cash journal or branch cash lock is needed. Do not call cash.Post, which hardcodes a cash counter-entry.

Actual credit time is recorded even for a missed month. It enters that posting day's closing balance, never past completed daily rows. The job recovers missed calculation and credit work; it does not retroactively compensate for lost compounding during downtime. Disclose this limitation. Failed transactions fully roll back, including monthly completion; retry safely after failure or an uncertain commit. Respect context cancellation, short DB lock/statement deadlines, bounded per-tick work and a shutdown wait. No unbounded retries; log a classified failure and retry on a later tick.

## Reconciliation and operational signals

- Each positive cash/interest posting has exactly two correctly directed equal journal lines.
- Member balance = deposits - withdrawals + interest credits = member_savings credits minus debits.
- Monthly credit equals rounded exact daily total; unique account/period; no missing eligible dates.
- Cash stays deposits - withdrawals; interest never changes branch cash.
- For this restricted model, total savings = cash + cumulative posted interest expense. Do not retain Phase 1's cash = savings assertion once interest posts; re-evaluate when other accounting events are introduced.
- System actor rules and home-KCP interest dimensions are checked separately from teller servicing-KCP attribution.

Reuse slog. Log job date/period, duration, account counts, completed/skipped/failed counts, lock timeouts and lag, with a safe failure class. No passwords, NIK, account numbers, raw payloads or financial row dumps. Status plus structured logs are sufficient initially; no monitoring dependencies. Verify backup restores policy history, daily/monthly records, financial rows, audit and uniqueness as well as balances.

## Implementation Plan

1. Owner accepted ADR-0012 rules, worked examples and starting rates on 2026-10-23. Implement these contracts; no production tax/accounting compliance claim.
2. Use TDD for the calculator and policy validation first. Prove RED/GREEN for exact boundaries, rounding, dates and invalid inputs.
3. Implement forward schema expansion, policy APIs/audit/replay, and explicit read compatibility. Preserve migrations 1-3 and existing cash behavior.
4. Add the shared writer barrier and explicit posting timestamp, including registration; test midnight and lock order before enabling daily calculation.
5. Implement ordered daily calculation, monthly non-cash posting and their uniqueness/rollback guarantees. Add the small lifecycle-managed loop and disabled-by-default enable flag.
6. Integrate the required EN/ID/light-dark frontend support separately; update cash/interest reconciliation, demo verifier and operational docs. Reject insertion of backdated demo rows after activation; existing seed batches may be checked read-only. Seed a fresh local database before activation.
7. Rehearse an isolated full and partial month, crash/retry, two workers, and populated restore. Enable locally only after all prerequisite writers/readers/checks are deployed. No live-money activation in this plan.

## Test Plan

- Pure table tests: zero; exact 1m/10m/100m boundaries; global fallback; daily band crossings; changing dated policies; 365/366-day years with fixed divisor; opening/activation dates; monthly half-up ties; sub-IDR zero completion; overflow.
- Policy API/auth tests: Teller denied, origin/session guards, malformed bands, overlapping intervals, explicit gap fallback, inclusive lower/exclusive upper, stale expected_latest_id, pending supersession, tomorrow validation across midnight, unchanged historical policy selection, durable replay/conflict and audit rollback.
- Database integration: opening/deposit/withdrawal concurrent with close, writer waiting before midnight then posting after midnight, pre-midnight posting committing late, fresh snapshot after lock wait, empty dates, missed days, partial first month, missing accrual blocks credit, two workers and two monthly attempts, duplicate scheduler tick, restart after commit-before-ack, journal/audit failure rolls everything back.
- Cash regressions: unchanged overdraft checks, branch cash, opening deposit, snapshots, request fingerprint/replay, history/search pagination, System projection; POST /cash cannot create interest. Legacy rows and null snapshots stay unchanged.
- Reconciliation: exact daily-to-month totals, per-kind journals, liability and balance equality, cash unchanged on interest, zero result, restore all tables/sequences and run jobs again without duplicates.
- Isolated database only: cd backend && /usr/local/go/bin/go test -race ./... with an explicitly disposable TEST_DATABASE_URL; integration fixtures must never target an unreviewed populated database. Run go vet ./..., existing scripts/lint.sh, and compile the API. Set required environment without printing secrets. See the evidence report for executed checks.
- Before release, corresponding frontend lint/typecheck/tests/build and browser checks must prove explicit interest display, EN/ID and both themes. This backend plan does not replace that integration verification.

## Risks & Mitigations

Schema/rollout: PostgreSQL 16 with the existing embedded transactional migrator and 5-second migration lock timeout. Add tables and expand checks via forward migration; validate old rows on an isolated populated restore. These ALTERs can lock tables: use a local maintenance window and a verified backup. No monetary backfill or old migration edits. Keep the job disabled until every writer participates in the barrier and every reader supports interest. New enum values make a rollback to an old frontend unsafe after posting; disable the loop and forward-fix compatible readers instead of deleting credits. Never remove credited rows or restore a stale backup over newer money movements.

Midnight: the exclusive close/shared writer barrier and post-lock DB timestamp are mandatory correctness work, not optional scheduler tuning. All administrative/seed money writes must comply or be unavailable after activation. Assume the database clock is operated correctly; detect an attempted write into an already completed date and fail rather than revise that date.

Policy/tax/accounting: gross-only calculation is for local validation. No assertion that zero deductions is appropriate for live cooperative operations. Production enablement needs a separately confirmed deduction treatment and approved expense account mapping. Closure/freeze/backdate/correction endpoints are absent and remain out of scope; if introduced, revisit eligibility and period rules first.

## Open Questions

The owner accepted the rules and starting rates on 2026-10-23. Activation is selected through the first policy. Real-money deduction/accounting and operational readiness remain separate gates.

## Implementation Handoff

Use tdd-workflow (RED/GREEN/refactor), coding-standard, go-patterns and go-lint-workflow. Apply go-build-resolver only if validation fails. Follow error-handling, api-contract-review, observability-check, database-migrations and postgres-patterns for the boundaries above. Keep Go business logic outside main.go, propagate context, avoid speculative interfaces/dependencies, and keep implementation scoped to the approved backend plan. Implementation and local validation were subsequently authorized on 2026-10-23. Commit, push and PR creation still require a separate user request.
