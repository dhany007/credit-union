# TDD Evidence - Account-filtered transaction history

Date: 2026-10-21
Source plan: [Transaction history](../tech-plan-transaction-history.md)

## Testable guarantees

| Guarantee | Target | RED evidence | GREEN evidence |
| --- | --- | --- | --- |
| Staff can inspect cash history across KCPs, with correct member/account attribution and exact filtering | `TestTransactionListingAndAccountFilter` | GET /api/transactions returned 404 rather than 200 | Passed; both staff roles, opening deposits, cross-KCP deposit, withdrawal, validation, unknown account and authentication verified |
| Latest 100 and deterministic timestamp ties; large member numbers are not truncated | `TestTransactionListingLatest100AndStableTies` | New endpoint returned 404 | Passed for 101 tied timestamps, including member ID 1000001 |
| Empty history is an array; infrastructure details stay private | `TestTransactionListingEmptyAndSafeFailure` | New endpoint returned 404 | Passed for empty/unknown accounts and controlled missing-table failure |
| Reads preserve financial and retry state | Transaction listing snapshot assertion | New endpoint absent | Savings, transaction count, journal count/debit/credit, and request outcomes matched before/after |
| Existing financial/account behavior is unchanged | Full existing integration suite | Existing baseline | Race-enabled account, MVP, legacy replay, concurrency, rollback, authentication and loan tests all passed |

## Commands and results

- `bash scripts/test.sh -run TestTransactionListing`: three intended endpoint-absent failures before implementation, then all passed.
- `bash scripts/test.sh`: full Go race-enabled suite passed after the implementation.
- `bash scripts/lint.sh fmt`; `bash scripts/lint.sh`: passed, 0 lint issues.
- `go vet ./...`: passed.
- Frontend `npm run typecheck`: passed.
- Docker build: Go API/migration/seed binaries and Next.js production build/TypeScript passed.
- Main Docker `up -d --no-build --wait`: passed; no schema migration added by this menu.
- Main health endpoint passed; unauthenticated transaction listing returned 401.
- Local PostgreSQL `EXPLAIN` for exact-account filtering used `members_account_number_key` and `transactions_member_idx`. No new index was introduced.

## Browser verification

Used a separate disposable Compose project `credit-union-transactions-ui` on port 3001, with two synthetic members, four cash postings and balanced journals. No test financial records were added to the main local database.

- Teller opened the Transactions menu and saw both accounts across KCPs: opening deposits, a cross-KCP deposit, and a withdrawal.
- Filtering account 1000000001 showed only its two postings; the second member was absent.
- A valid unknown account displayed zero rows and no stale first-member rows.
- Reset cleared the account input and restored all four postings.
- EN/ID labels, localized date/currency, and light/dark appearances worked.
- Stopped only the disposable API: reapplying the filter showed a localized error and retry control, hiding previous rows. Restarting that API and selecting Retry restored all four rows.
- Supervisor signed in and could view the same menu, filter the second account, and see its opening deposit and withdrawal.
- Final reconciliation matched fixture totals: savings IDR 120000, four transactions, eight journal lines, equal debit/credit totals IDR 130000, and zero mutation-request outcomes. History reads did not change these values.
- Closed the test tab and removed only the disposable project and volume.

## Contract, scope, and gaps

New endpoint is additive and explicitly restricted to Teller/Supervisor. Existing member-specific history, cash command JSON/fingerprints, account allocations, and journal posting remain unchanged. Query values and member data are not added to logs; SQL filtering remains parameterized, with existing no-store responses and bounded request context.

The page intentionally shows only the latest 100 matching cash postings. Pagination, exports and date/type/KCP filters are not part of this change. Account filtering has existing indexes; unfiltered small-MVP history can scan/sort the table. The inspected empty-local-data plan is not a production-scale benchmark. No new component/E2E dependency was installed; real-browser checks supplement Go tests.
