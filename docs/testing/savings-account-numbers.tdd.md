# TDD Evidence - Savings account numbers

Date: 2026-10-21
Source plan: [Savings account numbers](../tech-plan-savings-account-numbers.md)

## Testable guarantees

| Guarantee | Test target | RED evidence | GREEN evidence |
| --- | --- | --- | --- |
| Registration allocates a separate account, replay is stable, lookup works across KCPs, malformed/unknown/unauthenticated lookup is rejected | `TestSavingsAccountRegistrationAndLookup` | AccountNumber was empty; expected 1000000001 separately from CU-000001 | Passed with race detector, including cross-KCP deposit and unchanged account |
| Concurrent registrations have distinct 10-digit accounts | `TestSavingsAccountConcurrentAllocation` | Empty account number rejected | Six concurrent registrations passed |
| Existing rows receive stable unique accounts without altering previous financial fields/history; migration reruns serialize | `TestSavingsAccountMigrationBackfill` | account_number column absent | Populated baseline financial JSON snapshots matched before/after; concurrent reruns and constraints passed |
| Exhausted numbering cannot partially register or post funds | `TestSavingsAccountSequenceExhaustionIsAtomic` | savings_account_number_seq absent | Last valid number allocated; next registration failed with no extra member, transaction, balance, or request outcome |
| Older accounts are directly resolvable outside the latest-100 list | `TestSavingsAccountLookupBeyondListLimit` | Added after core GREEN as an additional regression guard | Passed with 101 later members; exact lookup still found the original account |
| Old stored responses and financial protections remain compatible | Existing MVP, persisted replay, cash locking/rollback, auth and loan regression suite | Existing baseline | Full suite passed with race detector |

## Validation commands

- `bash scripts/test.sh -run TestSavingsAccount`: initial four focused tests failed for the missing account-number behavior before production edits; final five account tests passed.
- `bash scripts/test.sh`: full backend race-enabled suite passed after the implementation, including legacy replay and financial concurrency/rollback cases.
- `bash scripts/lint.sh fmt` and `bash scripts/lint.sh`: passed, final result 0 issues. One new 121-character test fixture line was wrapped before final validation.
- `go build ./...` and `go vet ./...`: passed.
- Frontend `npm run typecheck`: passed; Docker Next.js production build including TypeScript passed.
- Docker Go/API and frontend images built; updated main local stack became healthy.
- Existing initial schema file was retained unchanged; account change is migration 2.

## Local rollout and reconciliation

Main local Docker database was stopped for writes, backed up as a custom-format archive under ignored `.local/backups/`, and its archive catalog verified with `pg_restore --list`. Migration 2 and normal migration rerun succeeded. Main database pre/post member count, total savings, cash transaction count, and journal-line count were all zero; versions 1 and 2 were recorded. This is empty-local-data validation, not evidence of a production backfill. The populated isolated migration test supplies non-empty field/financial-history reconciliation evidence.

## Browser verification in a disposable stack

Used a separate Compose project `credit-union-account-ui` on port 3001, with its own disposable database volume and synthetic identities only. No synthetic member or cash data was inserted into the main local database.

1. Teller KCP Pusat registered a synthetic member. Feedback, list, and detail showed member CU-000001 and account 1000000001.
2. Added 101 later zero-balance fixture members, ensuring the original member was outside the latest-100 list.
3. Signed in as Teller KCP Utara and resolved account 1000000001 directly. The original member name and IDR 50,000 balance appeared.
4. Changed the number to an unknown account: previous target/history cleared, transaction button disabled, and the localized not-found message appeared.
5. Resolved the original account again, confirmed a synthetic IDR 10,000 deposit, and observed balance IDR 60,000. Confirmation showed both member name and account number.
6. Database reconciliation showed the same account/home KCP 1, opening transaction at KCP 1, and new deposit at KCP 2. EN/ID feedback and IDR formatting remained correct.
7. Closed the test tab and removed only the disposable Compose project and its synthetic volume. Main stack health and unauthenticated account-lookup rejection remained correct.

## Gaps and operational limits

No production load, staged large-table backfill, or full backup-restore drill was performed. Account migration is a small local maintenance migration with a bounded lock wait; do not infer zero-downtime readiness. Sequence gaps are expected. Existing persisted registration JSON is preserved, and current member projections provide account numbers for old replay results. No member renumbering, multiple savings accounts, or new financial product behavior was added.
