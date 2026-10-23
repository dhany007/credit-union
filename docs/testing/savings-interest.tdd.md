# TDD and recovery evidence - Savings interest

Date: 2026-10-23.
Source: [accepted tech plan](../tech-plan-savings-interest.md), [ADR-0012](../adr/0012-savings-interest-calculation-and-posting.md).
Result: local implementation and technical validation passed. The main scheduler was disabled during initial validation; see the authorized activation follow-up below.

## RED / GREEN evidence

| Guarantee | RED evidence | GREEN evidence |
| --- | --- | --- |
| Exact whole-balance rate selection and monthly half-up rounding | New domain tests failed to compile because Policy/Band/Numerator/RoundCredit did not exist | Three domain tests pass, including exact boundaries, fallback, invalid bands, overflow, leap-year fixed divisor and Jakarta closing delay |
| Supervisor policy authorization and durable replay | TestInterestPolicyContract expected 403 but received 404 before routes existed | Supervisor creation/list/status and semantic replay pass; Teller denied; stale expected version rejected |
| Missing completed-day calculations are rejected | TestInterestDetectsMissingCompletedDayRows returned success after all daily rows were removed while markers remained | Catch-up rejects marker/count inconsistencies; no settlement or false success |
| Atomic non-cash credit and retry | New integration tests exercise posting and injected journal failure | Balance, journals, monthly result and audit commit together; failed journal leaves no credit; retry posts once; zero result completes without money row |
| Failed accounts cannot starve healthy accounts | Review identified the first-100 retry starvation path | TestInterestFailedAccountsDoNotStarveHealthyAccount verifies one healthy account after 100 capped accounts receives credit across bounded ticks |

## Automated checks

- Backend: existing golangci-lint configuration and formatters, 0 issues.
- Backend: full go test -race ./... passed, including seven new interest integration test groups on isolated PostgreSQL 16 at localhost:55432/phase2_test. Tests use per-test schemas and clean them up.
- Partial activation/opening month, dated rate changes, actual-day late credit treatment, concurrent scheduler entry, immutable-period duplicate protection, zero result, fault rollback, writer-barrier cancellation, and closed-date refusal passed.
- Frontend: TypeScript typecheck and ESLint passed; 25 Node tests passed, including exact percentage parsing, Jakarta date minimum, and translated positive interest movements.
- Docker API and web production builds passed. No new dependency or scheduler service was added.
- Read-only code, Go, React/TypeScript and architecture reviews completed. Confirmed fairness, daily-marker integrity and diagnostic findings were fixed and the affected/full checks rerun.

The clock tests exercise date-boundary calculations and explicit lock blocking/cancellation; no wall-clock overnight wait was performed. Larger-load capacity and a real-money security/accounting audit are outside this validation.

## Populated interest and restore rehearsal

Main DB/API/web writers were briefly paused for a populated backup and resumed in a finally block. The main volume was preserved.

The archive was restored into the explicitly isolated phase2_uat database in the owned test container. Migration 0004 was applied there. A synthetic historical policy was inserted only into that isolated database to simulate a completed month on the existing 15-member / 7,500-cash-movement fixture. It was not inserted into main.

Three bounded catch-up invocations generated:

| Measure | Result |
| --- | ---: |
| Members | 15 |
| Daily account calculations | 780 |
| Completed business dates | 52 |
| Monthly settlements / positive interest credits | 15 / 15 |
| Total interest expense / savings increase | IDR 351,831 |
| Total savings after credit | IDR 112,326,831 |
| Cash, unchanged | IDR 111,975,000 |
| Journal debits | IDR 1,575,746,831 |
| Journal credits | IDR 1,575,746,831 |

Both reconciliation SQL scripts passed. The existing seeded cash batch also passed its unchanged-history checks after interest crediting.

A new populated archive, including interest policies, daily records, completion markers, monthly results, journals, transactions and audit, was restored into phase2_recovered. All 14 public tables and six sequences matched exact ordered-row fingerprints/state. Running catch-up again did not change the manifest or add interest. An actual API process with its scheduler enabled was also started against this isolated recovered database, logged a successful no-duplicate catch-up, and shut down with exit code 0. This was not activation on main.

Private archives/manifests are ignored under .local/; backup and manifest files have restricted permissions. No secrets, dumps or identity exports are committed.

## Browser checks in Chrome

The isolated web/API ran at localhost:3001. Checked:

- Teller cannot see the interest configuration menu; interest reads remain available in transaction history.
- Interest rows are positive and show the correct before/after balance, home KCP, period/reference and System attribution.
- English/light and Indonesian/dark controls and content render correctly.
- Supervisor can open policy/status/history, rejects overlapping bands with a clear error, then saves the corrected prospective version using the keyboard.
- Saving retains the earlier policy version. Rate values are 3%, 4%, 4.5%, 5% with a 3% global fallback.
- No error-level browser console entries were captured during the UAT checks.

Main was then migrated/deployed with the compatible API/web and scheduler disabled. Main member, transaction and journal table fingerprints match the pre-migration backup exactly: 15 members, 7,500 cash movements, IDR 111,975,000 savings, no interest policies or credits. The Supervisor screen is available for owner review. No backdated main financial posting was performed.

Screenshots are retained in the task visualization folder: phase2-interest-ledger-en-light.png, phase2-interest-settings-id-dark.png, phase2-main-interest.png.

## Activation boundary

Automatic approval review rejected the attempted main-local scheduler enablement because implementation approval was not considered explicit authorization for automatic financial postings to an existing populated dataset. That action did not execute. The compatible application/migration was deployed with the scheduler disabled instead.

Before activation, obtain explicit authorization to save the approved prospective policy on main and enable SAVINGS_INTEREST_ENABLED. The first policy fixes activation, with no pre-activation accrual. The first daily calculation follows the activation day; the first monthly credit follows its eligible month. Real-money deduction/accounting and operational readiness remain separate gates. No commit, push, PR or release was created.

Normalized source/restored schema dumps also match, including constraints and indexes (only pg_dump random restrict guards excluded). Temporary UAT/scheduler containers, their owned anonymous database volume and network were removed after retaining private archives. Main DB/API/web remain healthy.

## Authorized main dummy activation follow-up (2026-10-23)

The owner explicitly authorized saving the approved rates and enabling automatic interest on the main dummy dataset, effective tomorrow. Policy version 1 was saved through the authenticated Supervisor API, effective 2026-10-24 in Asia/Jakarta, with global 300 bps and whole-balance bands 300/400/450/500 bps at 1m/10m/100m boundaries. SAVINGS_INTEREST_ENABLED=true was set in the ignored local environment; compatible API/web returned healthy. The status endpoint confirmed enabled=true, start_date=2026-10-24, a successful scheduler pass and no failure class. Member/transaction/journal table fingerprints remain unchanged: 15 members, 7,500 transactions, IDR 111,975,000 savings, and no daily/monthly interest rows before activation. First daily close is due 2026-10-25 after 00:05 Jakarta; the first partial-month credit is due 2026-11-01 after the October dates complete. No pre-activation interest is credited.
