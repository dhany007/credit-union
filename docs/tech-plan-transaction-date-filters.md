# Tech Plan - Transaction date filters and pagination

## Memory Context
No project memory-system folder or AGENTS.md was found. Follow the existing cash domain, transaction-history plan, staff authorization, English docs, and EN/ID interface. Preserve all existing uncommitted account and transaction features.

## Requirement
Today (default), Last 7 days (today plus six prior dates), Last 30 days (today plus 29 prior dates), and Custom (inclusive start/end dates). Use Asia/Jakarta consistently. Combine dates with the exact optional account filter. Paginate matching results without a latest-100 ceiling. No exports, analytics, or financial mutations.

## Backend Scope and API Design
Add GET /api/transactions/search for Teller/Supervisor. Preserve the flat-array legacy GET /transactions and member history. Query: period=today|last_7_days|last_30_days|custom, start_date/end_date required only for custom, optional account_number and cursor. Default period=today. Return {items,next_cursor,start_date,end_date,time_zone}, 50 rows per page, ordered created_at DESC,id DESC. Dates use YYYY-MM-DD, start <= end, years 0001..9999; reject incomplete/invalid/irrelevant fields and incompatible/malformed cursors with safe 400 messages. Unknown accounts return empty items. Auth and internal error handling stay unchanged.

Resolve presets on the server. Query >= local start midnight and < midnight after end. The next page uses returned explicit dates to freeze the range across midnight. A bounded URL-safe base64 cursor holds the last timestamp/reference and account/date scope; it is a navigation value, not an authorization token. Validate scope and timestamp. Concurrent later postings appear on refresh, not as duplicates during forward traversal. No snapshot/export guarantee.

## Implementation Plan
1. Write deterministic date boundary unit tests and isolated PostgreSQL API tests; prove RED.
2. Add cash-local request/range/cursor parsing, bounded keyset query, and response envelope. Keep all values parameterized and contexts propagated. Reuse the staff projection and safe route-template logging; no account/date/cursor values in logs.
3. Add preset/date controls, inclusive date validation, combined filtering, reset-to-Today, and previous/next navigation in the existing UI. New filter starts page one, obsolete requests abort, stale results stay hidden. Show applied dates/timezone; retain EN/ID and light/dark.
4. Validate focused/full race tests, lint/vet, frontend typecheck/build, browser checks and local Docker rollout. Update documentation and evidence.

## Test Plan
Fixed clock timezone rollover, leap days, 7/30-day inclusivity, custom single date, invalid/reversed dates and cursors. API lower/upper boundary precision, combined account filter, >100 rows, equal timestamps without duplicates, cross-filter cursor rejection, empty/auth/safe failure. Existing financial regressions must remain green.

## Risks and Mitigations
Separate route is additive compatible. Existing indexes support member filtering; global date scans remain a documented local-MVP limitation and warrant a date/id index before large datasets, not speculative migration now. No schema/dependency changes. Keep Docker volume. API runtime already includes tzdata. Roll back binaries together; data unchanged. Read-only architect reviewed and agreed with the separate endpoint, inclusive local dates, filter-bound cursor, frozen page range, and runtime timezone availability.

## Open Questions
None blocking. User explicitly requested implementation.

## Implementation Handoff
Apply tdd-workflow, coding-standard, go-patterns, go-lint-workflow, api-contract-review, error-handling, observability-check and PostgreSQL guidance. Write tests before production code and record actual RED/GREEN. Do not commit/push without a new request.
