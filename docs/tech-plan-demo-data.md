# Tech Plan - Reconciled local demonstration data

## Requirement and context
Populate the existing local Docker application with 15 clearly synthetic members, each with exactly 500 cash transactions (including its IDR 50,000 registration deposit), January 2026 through the current local date, with randomized deposits/withdrawals that reconcile. Preserve existing uncommitted member-ledger work. No production seed/runtime dependency/schema change.

## Scope and design
Add a psql development-only SQL fixture plus an independent reconciliation query/report. Use fixed randomness, deterministic batch/transaction identifiers, synthetic 16-digit NIKs using a non-real 00 region prefix, obvious DEMO names/addresses/notes, existing Teller accounts/KCP mapping and generated savings account numbers. Dates span every elapsed month and include today, never the future; synthetic operating-hour convention is Monday–Saturday 09:00–16:00 Asia/Jakarta.

Generate the full 7,500-row plan in temporary tables first, ordered by actual synthetic posting timestamp. Simulate member and KCP cash balances, restrict withdrawals to both available balances, and preserve a positive IDR denomination. First movement per member is a 50,000 deposit from zero at its home KCP. Other servicing KCPs are randomized; actor belongs to servicing KCP. Capture correct before/after values in every row.

## Execution and verification
1. Inspect the target schema/data and read-only architect review; create a local database backup.
2. Dry-run only temporary plan/reconciliation, without persistent rows or nextval calls, then rollback. This avoids consuming account numbers during preview.
3. Apply all member, transaction, journal and audit records in one database transaction, after pausing local writers. Never delete/reset data or renumber existing identities.
4. Use a fixed batch marker and collision checks: completed batch reruns must not duplicate anything; partial/conflicting data must fail clearly. Marker commits with all rows. Never seed production or reuse real personal identities/passwords.
5. Before commit independently check 15 × 500 counts, both cash directions, every elapsed month/member, opening 50,000, no transaction before registration or after cutoff, exact balance chain, nonnegative member/KCP running balances, per-member signed cash sum equals closing balance, two correctly directed journal lines per transaction, and per-transaction/global debit=credit. Verify existing data counts/digest unchanged.
6. Reconcile committed data with a separate SQL report, grouped by member/month/KCP. Confirm the API/UI lists and account/date filters expose the data. Resume local services and retain evidence.

## Risks and limits
This is labeled local/sample confidence. Calendar dates and volumes are artificial; no interest, loans, reversals or external financial transfers are simulated. Direct SQL fixture insertion is development-only; audit rows identify synthetic provenance and no fake HTTP idempotency receipts are created. Do not mark success from counts alone. No fabricated historical snapshot is inferred: snapshots are created with the generated chronology. Do not commit/push without a new user request.

## Verification outcome

Validated preview, full apply in a temporary restored database, repeat-run safety, main local import, and independent post-commit reconciliation. All checks passed: 15 members, 7,500 transactions, 15,000 balanced journal lines, and IDR 111,975,000 total savings. Missing required flags fail with a nonzero psql status. Local API smoke checks confirmed 15 members and exactly 500 unique account-filtered transactions across 10 pages with non-null balance snapshots. Today correctly returns 15 rows. The temporary validation database was removed; the main local application services are healthy. See [dataset and execution evidence](development/demo-data.md).
