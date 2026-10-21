# TDD Evidence - Phase 1 Gin Backend Refactor

Date: 2026-10-21

## Source Plan

[Phase 1 backend tech plan](../tech-plan-phase-1-backend.md), including the owner-selected 6hc domain organization.

## Testable Guarantees and Evidence

| Guarantee | Test target | RED / baseline evidence | GREEN evidence |
| --- | --- | --- | --- |
| Existing HTTP and strict JSON contracts survive routing change | integration/TestTransportBaseline | Passed against the original ServeMux; characterization, not an invented failing test | Passed against Gin, including HEAD, 404, 405/Allow, unknown fields, trailing JSON, and body cap |
| Dependency failure is not reported as invalid credentials/session | integration/TestAuthDependencyFailure | Original login and me both returned 401; expected 503, causing two focused failures | Both pass with 503; normal credential/session tests still return 401 |
| Unique membership, cross-KCP cash, atomic journals, roles, approval limits | integration/TestMVP, eight original scenarios | Existing behavior suite retained while moving to public composition entry | Passed with PostgreSQL and race detector |
| Persisted pre-refactor requests replay without executing new effects | integration/TestPersistedLegacyReplay | Fixed old-format fingerprint fixtures independent of new projection code | Registration, cash, application, and decision all replay correctly |
| Same-KCP cash and concurrent same-key requests cannot overpay | integration/TestSharedBranchCashAndConcurrentRetry | New regression coverage around existing intended invariants | One payout when branch cash is insufficient for two; one deposit for four retries |
| Actual lock waits can be cancelled; ancillary persistence failures roll back | integration/TestCancelledLockAndAtomicFailures | New regression coverage; waits until PostgreSQL confirms a blocked lock, then cancels | Cancelled request stops; failed audit/result writes leave balances reconciled |
| Competing loan decisions cannot both win or move cash | integration/TestConcurrentLoanDecisions | New concurrent-state regression | One success, one conflict, no financial effects |
| Runtime settings retain safe defaults and reject malformed values | config/TestLoad | Characterization and boundary coverage for typed config | Seven configuration cases pass |
| Query identity, tokens, and panic values stay out of logs | httpapi/TestLoggingAndRecoveryDoNotExposeInputs | New privacy regression around chosen explicit Gin logging | Passed; route templates remain visible |

The auth error correction has genuine RED-to-GREEN evidence. Package extraction uses existing passing behavior and added characterization/regressions; no false claim that moving files itself required a failing behavior test.

## Validation Commands

- `bash scripts/test.sh`: PASS, PostgreSQL integration and unit/transport tests with `-race`.
- `go build ./...` from backend: PASS.
- `go vet ./...` from backend: PASS.
- `bash scripts/lint.sh config verify`: PASS.
- `bash scripts/lint.sh linters`: enabled set inspected; `lll` is enabled at 120 characters.
- `bash scripts/lint.sh run --output.json.path=../.local/lint.json ./...`: PASS, zero issues with golangci-lint v2.15.0.
- Local Gin API smoke: PASS for health, login, current user, KCP metadata, member/application reads, logout, and expired session denial. No member financial mutations in this smoke.
- Schema source copied byte-for-byte from the initial implementation; no application database migration was run for the refactor.

## Implementation Notes

Thin executables: `cmd/api`, `cmd/migrate`, and local-only `dev/cmd/seed`. Domain-local HTTP adapters, services, PostgreSQL stores, and named types live in auth/member/cash/loan. Explicit app wiring injects the cash posting capability into registration. One shared command transaction retains all posting, audit, and replay effects.

The only deliberate API failure correction is dependency failures in credential/session lookups becoming safe 503 responses rather than false 401 responses. Existing successful payloads, cookie model, origin checks, amount rules, and role boundaries remain. Request contexts have a 15-second budget. No unsafe forwarded-IP trust, new rate-limit policy, or automatic financial retries were added.

## Coverage / Gaps

No production security review, load test, restore drill, real-data validation, or full browser E2E was performed. Cancellation/atomicity/concurrency evidence comes from isolated PostgreSQL schemas, not real members. Production approval limits remain unconfigured; disbursement, repayments, and savings interest remain deferred. Identity corrections, cash funding/transfers, and reversal procedures remain operational work.

Some crypto-failure paths are handled but not fault-injected. No arbitrary coverage percentage is claimed. The current per-peer login throttle remains unchanged and still needs a separately scoped proxy-aware policy review.

## Source Safety

Existing uncommitted work was preserved. A recoverable copy of the original backend root files remains under ignored `.local/refactor-baseline/`. No secrets were copied into that baseline. Generator scripts were removed after use. No commits or pushes were created for this refactor.
