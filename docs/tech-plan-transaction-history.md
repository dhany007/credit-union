# Tech Plan - Account-filtered transaction history

## Requirement and context

Add a dedicated Transactions menu so staff can inspect cash transactions, optionally filtered by a full savings account number. Keep Phase 1 small: latest 100 cash transactions, including opening deposits, across KCPs. No loan events, general ledger explorer, analytics, export, or extra filters.

Existing English scope, account-number plan/ADR, cash domain, staff authorization, and PostgreSQL schema are the repository context. The user explicitly requests implementation. Existing account-number changes are uncommitted and must be preserved. No memory-system folder is present in the known project context.

## Backend scope

Cash domain owns a new read-only listing projection and HTTP route. Keep existing member-specific history, posting, journals, account-number allocation, idempotency, and loan routes unchanged. No schema or dependency changes.

## API design

- `GET /api/transactions?account_number=1000000001`.
- Existing authenticated Teller/Supervisor access, cooperative-wide, matching member/history access. No anonymous or public account lookup.
- Omit/empty filter for all KCPs; trim surrounding whitespace; a nonempty filter must match `^[1-9][0-9]{9}$`.
- Return a JSON array, maximum 100, ordered `created_at DESC, id DESC` for deterministic ties.
- Include existing transaction reference, kind, positive integer amount, note, timestamp, servicing KCP, actor; add member ID, member name, `CU-` membership number, and account-number string. Do not expose NIK, phone, address, sessions, or internal errors.
- Valid unknown account and no history return `200 []`. Invalid filter returns 400; anonymous request returns 401. Existing safe internal error mapping remains unchanged.
- Additive compatible endpoint; no mutation/fingerprint changes.

## Implementation plan

1. Write integration tests using isolated PostgreSQL fixtures, prove RED against the absent endpoint.
2. Add a cash-local read model, service validation, parameterized PostgreSQL query, and guarded route. Embed the existing transaction projection only in the new read model.
3. Use separate fixed SQL query paths for unfiltered vs exact-account listing. Join members/branches/users, compute membership number with Go formatting, enforce limit and stable ordering.
4. Run focused and existing regressions, Go formatting/lint/vet/build.
5. Document user-facing scope, actual validation, and deploy updated local Docker images.

Frontend integration: add a separate small component and Transactions navigation item; localized EN/ID labels, existing light/dark tokens, account filter/apply/reset, loading/error/retry/empty states, semantic table, member/account identity, positive/negative display direction, and explicit latest-100 limit. Abort obsolete requests and never display previous results as matching a newly applied filter. Reuse the existing API helper through a small shared module without changing request behavior. Cash posting screens remain separate.

## Architect review and tradeoffs

Read-only architect confirmed cash ownership, exact-filter semantics, masked staff projection, deterministic order, and unchanged old routes. Existing unique account-number index and member/date transaction index support filtering. The unfiltered MVP feed may scan/sort the small table; LIMIT bounds the response, not scanning. Defer an additional global date index until query plans/data justify it, before a large deployment. No claim of production-scale reporting performance.

## Test plan

- Staff authorization, both roles, cooperative-wide visibility, opening deposits, correct member/account/servicing KCP/actor attribution, deposit and withdrawal amounts.
- Exact account filtering; no substring match; malformed filters rejected; valid unknown account and empty database return arrays.
- At least 101 transactions prove latest-100 cap and descending reference tie-breaker for equal timestamps.
- Read requests preserve member balances, transaction/journal totals and idempotency outcomes; safe errors do not expose SQL details.
- Existing Go race-enabled regression suite, `scripts/lint.sh`, Go build/vet, frontend typecheck and Docker production build.
- Browser checks in a separate disposable test stack with synthetic data: show all, filter one account, no matches, reset, and EN/ID. Never insert synthetic financial records in the main local database.

## Risks, rollout, and observability

No data migration is required. Current account-number migration remains prerequisite for the new binary. Keep the main Docker volume. Existing no-store HTTP responses, request context timeout, safe route-template logs, and error mapping remain. Do not log account query values, notes, member names, or bodies. No new metrics framework. Page shows at most 100 latest results; older-history pagination is deliberately deferred and the UI states the bound.

## Open questions and implementation handoff

None blocking. Use tdd-workflow, coding-standard/go-patterns/go-lint-workflow, API contract review, error-handling, observability, and PostgreSQL guidance. Preserve all ongoing account-number work. Do not commit or push without an explicit request.
