# TDD Evidence - Transaction date filters and pagination

## Source Plan
[Tech plan](../tech-plan-transaction-date-filters.md), [ADR-0010](../adr/0010-transaction-business-dates.md).

## Testable Guarantees
| Guarantee | Test target | RED | GREEN |
|---|---|---|---|
| Backend Jakarta dates, today/7/30 inclusive presets, leap date, custom validation | cash/TestSearchDateRanges | Missing SearchInput/resolveSearch compile failure | PASS |
| Inclusive Jakarta midnight bounds, account/date intersection, staff roles, invalid queries, unchanged legacy arrays | integration/TestTransactionDateSearch | New endpoint returned 404 | PASS |
| More than 100 tied rows, no duplicates/skips, scope-bound cursor | integration/TestTransactionSearchPagination | Endpoint returned 404 after correcting fixture setup | PASS |
| Exactly 50/51 lookahead, unknown account, safe database error | integration/TestTransactionSearchPageBoundaryAndFailure | Additional regression coverage after initial RED/GREEN | PASS |

Initial pagination fixture used unsupported multi-statement prepared SQL; the fixture was split before recording valid endpoint RED. This setup failure is not feature RED evidence.

## Automated Validation
- scripts/test.sh: full Go race-enabled suite PASS, integration package 78.706s.
- scripts/lint.sh: 0 issues.
- go vet ./...: PASS.
- npm run typecheck: PASS.
- Docker Go and Next.js production builds: PASS, including the final corrected date form.

## Browser Validation
Separate disposable Compose project `credit-union-dates-ui`, port 3001, synthetic members only. The fixture contains 101 same-day rows plus rows six, twenty-nine, and thirty days earlier, with balanced journal lines. Main local data is never used for synthetic posting.

Verified Today default, forward pages 50/50/1, Previous, combined account/7-day filter yielding one older row, and combined account/30-day filter yielding two older rows and excluding the thirty-days-ago row.

## Limits
No production load or snapshot guarantee. Global date queries can scan/sort the small MVP table; evaluate an index before large deployment. Existing posting, membership, journals, auth, migrations, loan approval and idempotency regression tests remain green. No dependency or schema changes for date filtering.

Final browser checks: inclusive same-day Custom with account filter, native reversed-date validation and recovery by editing the start date, unknown-account empty state with no stale rows, Reset restoring Today/all accounts/page one, English labels and dark-theme rendering all PASS. Custom dates read native FormData on submit after a browser check exposed controlled-input date event synchronization; rerun passed with the same date-fill sequence. No new test framework was added.

Synthetic totals remained savings 104, transactions 104, journal lines 208, debits 104 and credits 104 after read-only UI checks. The isolated test tab and Compose project/volume were removed. Main database counts were zero members, zero transactions, zero journal lines and zero savings before deployment. Main Docker is rebuilt without resetting its volume.

Visual proof: `transaction-date-filters.png` saved in the task visualization output directory. No commits or pushes were made for this feature.
