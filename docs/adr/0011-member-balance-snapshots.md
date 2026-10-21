# ADR-0011: Store member balance snapshots on cash movements

Status: Accepted
Date: 2026-10-21

## Context
Staff need to see the member savings balance before and after a cash transaction. Existing journal_lines store balanced accounting entries, while transactions identify each savings movement. Neither previously stored a balance snapshot.

## Decision
Keep transactions as the member movement ledger and add balance_before/balance_after integer-rupiah snapshots. Capture actual values from the balance UPDATE RETURNING under the existing member lock; write the transaction, journals, audit and retry result in the same database transaction. The browser never supplies balances.

Retain explicit null for older rows without snapshots. Do not reconstruct historical balances from timestamp/reference sorting: those values cannot prove posting serialization. Nullable paired/nonnegative/arithmetic constraints protect recorded snapshots. New application postings always record both values; nullable storage permits historical rows and migration-before-deploy compatibility.

Expose snapshots on all cash-history read projections and display them with EN/ID labels. Zero is a valid balance. Unknown historic values use a dash and explanation. Values describe member savings, not KCP cash.

## Alternatives considered
A separate ledger table duplicates the existing cash movement without adding required behavior. Display-time cumulative sums would be incorrect under date filtering/pagination and could invent historical order. A full account sequence or general-ledger subsystem is outside the small Phase 1 scope.

## Consequences
Migration3 adds columns/constraints without changing money or old records. Pause writers, migrate, deploy new writers; retain additive columns on application rollback. No write payload, response, fingerprint or replay changes. History remains sorted by timestamp/reference for compatibility; adjacent displayed rows are not guaranteed to be the posting sequence under concurrent requests. Posted snapshots have no application edit endpoint; this does not restrict database administrators.
