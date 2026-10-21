# TDD Evidence - Member balance ledger snapshots

## Source
[Tech plan](../tech-plan-member-balance-ledger.md), [ADR-0011](../adr/0011-member-balance-snapshots.md).

## Testable guarantees
| Guarantee | Test | RED | GREEN |
|---|---|---|---|
| Opening 0 to 50000, cross-KCP deposit/withdrawal snapshots, all three read projections, replay/rejection/audit rollback | TestMemberBalanceLedgerPostingAndReads | Existing JSON snapshots were nil | PASS |
| Concurrent postings form an actual balance chain under the member lock | TestMemberBalanceLedgerConcurrentChain | balance_before column missing | PASS |
| Legacy rows remain unknown, migration reruns preserve money/journals, invalid partial/negative/arithmetic pairs rejected | TestMemberBalanceLedgerMigration | snapshot columns missing | PASS |

The account-number migration preservation test originally serialized the entire transaction record. Adding columns correctly changed that JSON shape; its comparison now removes only the new snapshot metadata while checking every pre-existing transaction field, members, journals and retry records. This retains the financial preservation guarantee rather than treating additive metadata as a monetary rewrite.

## Validation
Focused new race-enabled integration tests: PASS (5.127s). Go lint: zero issues after line formatting fixes. Frontend ESLint, TypeScript and all 22 existing utility tests: PASS. Full uncached race-enabled suite: PASS (integration package 95.026s), including existing migration/funds/replay/concurrency regressions. Go vet: PASS. Read-only architect review found no confirmed correctness issues. Docker Go and frontend production builds: PASS. Browser checks in disposable credit-union-ledger-ui verified all four rows, exact account filtering, 0 to 50000 opening balance, 50000 to 75000 deposit, 75000 to 65000 withdrawal, null/dash legacy row, EN/ID, dark/light, and matching member-specific cash history.

## Limits
No reconstruction/backfill of historic balances, full ledger/reporting engine, new mutation payload, fingerprint change or per-account sequence. Display timestamp ordering is retained for compatibility and is not claimed to be serialization order. Historic unknown values are explicit null; new application posts always capture both values in the same transaction. No new dependency. Database administrator access remains outside application append-only guarantees.

## Local deployment evidence
A PostgreSQL custom-format backup was saved to the ignored `.local/backups/before-member-ledger-20261021.dump` with mode600 before rollout; archive listing was verified, not a full restore drill. Main Docker database had zero members/transactions/journal lines and zero savings before migration. Writers were stopped, migration3 applied, then updated API/web started against the retained volume.

The isolated browser fixture retained savings105000, four transactions, eight journal lines, and equal debit/credit totals125000 after read-only checks. Its browser tab, Compose stack and test-only volume were removed. Main application data was not populated during the ledger browser test; a subsequent explicitly requested demo import is documented in [local demo data](../development/demo-data.md). Screenshot saved as member-balance-ledger.png in the task visualization directory.
