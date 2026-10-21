# Tech Plan - Member balance ledger snapshots

## Context and requirement
The user wants each member transaction to show before/after balance. Existing transactions are savings movement records and journal_lines already provide balanced debit/credit entries. Reuse those records; do not introduce a second ledger engine. Follow existing cash-domain locking, caller-owned atomic transactions, retry fingerprints and historical replay compatibility. No project memory-system folder is present.

## Backend scope and API
Add nullable bigint balance_before/balance_after to transactions through forward migration3. Non-null values must appear together, be nonnegative, and match signed deposit/withdrawal amount. Existing rows remain null: transaction-start timestamps and random references cannot prove historical posting order. Never fabricate a backfill or change funds/journals to populate display data.

On each new cash posting, capture actual before/after values with UPDATE members SET balance=balance+delta RETURNING balance-delta,balance under the existing member lock. Store those values in the transaction INSERT, in the same transaction as journals, audit and replay result. This covers registration's opening deposit too. Rollback/retries/concurrent posting retain their existing guarantees. The public write payload and result/fingerprints do not change.

Expose additive balance_before/balance_after JSON fields as integer IDR or null on member history, legacy cooperative listing, and paginated date search. Existing sorting/filtering/pagination and staff authorization stay unchanged. Snapshot balances reflect posting serialization; timestamp presentation is not a per-account sequence guarantee.

## Implementation plan
1. Read-only architect consultation; integration tests first with RED for absent fields/columns.
2. Add migration3 and version wiring without editing applied migrations. Nullable expansion remains compatible with old binaries; deploy migration before new writer in a local maintenance window. New application writes always capture snapshots; legacy/old-binary rows can remain unknown.
3. Capture snapshots in the cash store and extend all read projections. Do not trust client-provided balances. Preserve context propagation and safe errors/logs.
4. Show EN/ID before/after columns in the transaction table and member cash history, with explicit unavailable text for null values. Display actual zero as currency, not unavailable.
5. Test opening/cross-KCP deposit/withdrawal chains, concurrent continuity, replay, atomic rollback, read API parity, and migration invariants/constraints. Run full race suite, Go lint/vet, frontend lint/types/tests/build and isolated browser checks.

## Migration and operational risks
Small local MVP table: ALTER TABLE locks within the existing five-second migration lock timeout. No monetary backfill, historical rewrite, index or new dependency. Preserve database volume and take a backup before local deployment. Roll back application binaries if needed; retain additive schema/snapshots, use forward fixes. Production deployment/migration is outside this request.

## Open questions and handoff
None blocking. Use tdd-workflow, coding-standard/go-patterns/go-lint-workflow, API contract/error/observability guidance and database-migrations/PostgreSQL practices. Append-only means no application endpoint edits posted snapshots; database administrator access is not made immutable by this change. Do not commit/push or merge without a new user request.
