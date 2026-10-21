# Tech Plan - Savings account numbers

## Requirement and memory context

One cooperative, many KCPs. Each Phase 1 member has one savings account created during registration. Use an automatically assigned, globally unique 10-digit numeric string starting at `1000000001`, independent of NIK and home KCP. Existing `CU-xxxxxx` membership numbers remain separate. The user explicitly requested implementation; the plan is followed by TDD and implementation in the same task. No project memory-system directory is known; existing scope, schema, domain code, and regression fixtures supply context.

## Backend scope and codebase context

Member domain owns account creation/projection and lookup; cash continues to post by member ID using the existing caller-owned transaction. PostgreSQL 16; pgx; embedded immutable initial schema; explicit migration executable. No new dependency, account table, multi-account framework, loan change, or accounting redesign.

Read-only architect consultation recommended an independent bounded PostgreSQL sequence over deriving account numbers from internal member IDs. Adopt that recommendation to preserve the distinction between membership IDs and durable account identifiers.

## API design

- New registration results and member-list projections add `account_number` as a string. Registration input remains unchanged and cannot accept a chosen account number.
- Add `GET /api/members/by-account/:account_number`: existing staff authentication, cooperative-wide exact lookup, current member projection with masked NIK; 400 malformed number, 404 absent account, 401 unauthenticated. Existing name/NIK/member-number search also recognizes exact account numbers.
- Existing `/cash` payload and idempotency fingerprint remain unchanged. The frontend resolves an account number to the existing member ID.
- Additive compatible contract. Old persisted registration responses may lack `account_number`; return them unchanged, and let the UI retrieve the current projection. Never rewrite persisted fingerprints or results.

## Implementation plan

1. Add integration tests first and prove RED for registration/account projection and exact lookup.
2. Keep the initial schema byte-identical. Add an immutable numbered SQL migration and a small version ledger in `database.Migrate`, with transaction-scoped advisory serialization and a bounded lock timeout.
3. Add `members.account_number` text, independent sequence starting 1000000001 / max9999999999 / NO CYCLE, default allocation, backfill, 10-digit check, NOT NULL, and UNIQUE. Sequence gaps on rollback are valid. No renumbering on migration rerun.
4. Return the database-assigned number from member insertion; expose it in new registration results, lists, and exact authenticated lookup. Preserve shared opening-deposit transaction, audit, and replay behavior.
5. Prove GREEN, run existing financial regressions/race tests and Go format/vet/lint. Document the decision and deployment steps.

Frontend integration: display account numbers in registration feedback, member list/details, selectors, and cash confirmation; add direct exact-account lookup in the cash form. Changing or failing lookup clears the old target. Avoid submitting unresolved or stale targets; prevent overlapping lookup/submission. Maintain EN/ID and light/dark support.

## Test plan

- Fresh registration returns a 10-digit string; replay returns the same number/result; concurrent registrations allocate unique accounts.
- Exact lookup from another KCP resolves the same member/balance; invalid, partial, unknown, and unauthenticated requests fail correctly; full generic search includes the account.
- A cross-KCP deposit resolved through the account posts once and preserves the account number; failed registration/account-sequence exhaustion cannot leave partial funds, members, or request records.
- Backfill an existing populated baseline schema; verify members, home KCPs, balances, journal/transaction/history, and account mappings survive. Rerun and concurrent migration calls are stable.
- Preserve legacy replay fixture behavior.
- `bash scripts/test.sh`; `bash scripts/lint.sh fmt`; `bash scripts/lint.sh`; Go build/vet; frontend production build; Docker health and browser lookup/form checks.

## Risks and rollout

Mixed schema/data migration, transactional local maintenance rollout, not zero downtime. Stop API/web before applying the migration to the explicitly local Docker stack; keep the database volume. Bounded lock wait fails cleanly. Backfill locks and updates the small MVP members table; large deployments require a separate batched plan. Take a local database backup before applying. Sequence exhaustion fails safely; never recycle numbers. Application rollback retains the additive schema and allocated numbers; no destructive down migration. Old binaries tolerate the defaulted column. Runtime logs retain route templates/status/error class, never account numbers/query strings or NIK. No new metrics infrastructure.

## Open questions

None blocking for the one-account Phase 1 scope. Number formatting is an initial implementation convention recommended to the owner, not an external banking standard.

## Implementation handoff

Use tdd-workflow, coding-standard, go-patterns, go-lint-workflow, error-handling, API contract review, observability, database-migrations, and postgres-patterns. Preserve actor authorization, exact rupiah values, context cancellation, single financial commit, safe error messages, and structured redacted logs. Do not commit or push unless explicitly requested.
