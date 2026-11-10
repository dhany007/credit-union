# Phase 3 backend TDD evidence

Source: docs/tech-plan-phase-2-and-3.md and ADR-0013.

## Guarantees and evidence

| Guarantee | RED | GREEN |
| --- | --- | --- |
| Exact annuity, final principal and month-end dates | CalculateSchedule undefined | Calculator unit tests passed |
| All-table audit and pricing API | 14 original tables lacked audit fields | Audit/policy integration tests passed |
| Priced approval, single payout and cross-KCP repayment | Extended decision returned HTTP 400 before implementation | End-to-end loan tests passed |
| Minimum tariff 3% including missing/null inputs | Below-minimum/omitted fields were incorrectly accepted | Unit and API regressions passed; DB floor tested |
| Populated-interest migration | SQLSTATE 55006 from deferred FK events before ALTER | Explicit constraint validation; populated regression and restored migration passed |
| Replay, concurrent payout/payment, journal rollback | Behavior is covered by focused financial safety tests | Safety tests passed |
| Recovery SQL supports loan journals | Old script assumed two-line savings-only postings | Revised checks passed against 3-line repayment test and populated restore |
| Existing money/identity data preserved | Original migration assertion needed additive audit projection | Original-column digests and original sequence states preserved across 13 populated tables |

## Validation recorded

- Full backend go test -race ./... and go vet ./... passed after the first review fixes.
- Focused tests passed after the populated-restore fix.
- Repository Go lint passed with 0 issues before the final regression additions.
- A populated main-dataset backup was restored into an isolated PostgreSQL container and migrated through versions 1..6. Original column values and sequence states matched.
- Loan-aware ledger, existing savings-interest reconciliation and original seed-batch checks passed on that restored database.
- Reviewers rechecked audit, financial time binding, tariff validation, compatibility and migration corrections; no additional concrete blocker was reported.
- Final full backend race tests and go vet passed, including the populated migration and pagination regressions. Final golangci-lint: 0 issues.
- Rebuilt the local API image and restarted the API successfully after migrating the main dummy database to version 6. Health endpoint returned ok; API, web and database containers are healthy.
- Took a fresh predeployment backup; original column digests and sequence states matched before/after migration. Main ledger, savings-interest and demo-batch reconciliation checks all passed. No loan policy, approval limit or synthetic loan movement was added to the main dataset.

## Scope

No frontend implementation. No loan policy or approval limit is silently seeded/raised. Restructuring actions, penalties, special interest waivers and reversal UI remain deferred. Test policies and money movements are synthetic and confined to isolated databases/schemas.
