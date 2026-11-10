# Phase 3 frontend verification

Date: 2026-11-10. Scope: local implementation, synthetic money only.

## Delivery

The existing Loans sidebar entry now contains Applications, Disbursement and Loan Accounts. Supervisor tariff configuration is within the same workspace. No new dependency or backend schema change is introduced.

- Exact savings account lookup for application intake; optional server annuity preview.
- Final-term approval and rejection, current authority display, policy-bound previews and legacy pricing completion.
- Teller-only payout eligibility, recipient/agreement/conditions checks, verification note, explicit confirmation and committed receipt.
- Actual outstanding principal, current/prior schedule pages, version history, partial repayments, server allocations and payment history.
- Account/reference/status filtering and server cursor pagination, at most 20 rows per page.
- Committed payout/payment receipt retrieval and printing without another money command.
- EN/ID labels/errors and theme tokens for light/dark.
- The original command is persisted in session storage before sending. An uncertain response, server failure, expired session or throttling retains its key/body for replay. Cash actions stay disabled until recovery. A definitive stale-terms conflict requires a fresh review. Storage failure prevents sending.

## Automated verification

Frontend typecheck, ESLint with zero warnings and all 41 Node tests pass. Tests cover server expectation binding, Jakarta date boundaries, original-key recovery after an ambiguous response, session/throttle/server errors, definitive rejections, fail-closed storage and application references with a null linked loan number. Existing API/i18n tests also pass.

The production multi-stage frontend Docker build passes. The main web service was rebuilt/restarted; database, API and web report healthy.

## Chrome UAT

A dedicated local Compose stack at port 3002 used a fresh database, synthetic staff credentials, a synthetic member and 24 applications for pagination. The main dataset was not used for these money movements.

| Scenario | Result |
| --- | --- |
| Teller login, Loans workspace, 20-row first page and next page | PASS |
| Exact account lookup and submit application PJ-25 | PASS |
| Supervisor final-term preview and approval | PASS |
| Whole-principal band selects 4.5% p.a.; final schedule closes at zero | PASS |
| Payout verification and one IDR 1,000,000 cash payout | PASS |
| Loan account, current version and schedule navigation | PASS |
| IDR 50,000 partial payment allocation | PASS: interest 3,750; principal 46,250 |
| Actual remaining principal after payment | PASS: IDR 953,750 |
| Payment history and receipt retrieval | PASS |
| Supervisor future policy save and native minimum 3% validation | PASS |
| Existing contract keeps 4.5% after future policy creation | PASS |
| Indonesian/dark and English/light views | PASS |
| Browser error-level console messages | None captured |
| Loan-aware ledger reconciliation after UAT | PASS |

The effective UAT fixture policy date was advanced directly in the disposable test database to avoid waiting overnight. Production policy creation remains prospective. Approval limits and loan tariffs in the main dataset were not raised or seeded.

## Limits

Restructuring execution, penalties, write-offs and reversal actions remain outside this delivery. Physical cash handover and live-money operational acceptance are outside synthetic UAT. Automated component/E2E tooling was not added; browser scenarios were exercised with the connected Chrome UI. A dedicated narrow-viewport and assistive-technology audit remains separate.

## Modal interaction follow-up (2026-11-10)

- Added shadcn-style Dialog primitives backed by the existing Radix dependency; no dependency/lockfile changes.
- Application intake opens in a wide dialog with scrollable body and persistent Cancel / Preview / Submit footer.
- Loan details and their review/confirmation steps open in a dialog. Successful commands, errors and uncertain-command recovery no longer insert notification panels above the workspace.
- Tariff-save success has no Print action. Committed financial receipts retain printing and read-only retrieval.
- Shared notifications cover member/cash actions, savings tariff saves, transaction read failures and login errors. Cash confirmation also uses a dialog.
- During writes, closing/repeated submission is disabled. Uncertain results retain the original command key/body; dismissing the recovery modal leaves an explicit recovery action and new loan writes stay blocked.
- Read/write validation: typecheck, ESLint and all 46 Node tests pass; production web image rebuild/restart and health check pass.
- Connected Chrome UAT on the isolated port-3002 dataset verified initial focus, modal focus trapping, Escape dismissal, nested lookup errors preserving form values, formatted amount input, annuity preview, synthetic application PJ-26 submission and success dialog.
- UAT verified ID/dark and EN/light; policy-save notification has zero Print buttons. Modal body remained within 90% of viewport height while the footer remained visible.
- Synthetic IDR 1,000 cash deposit was confirmed through the new dialog and displayed the committed reference in a success dialog. No browser error-level console messages were captured.
- Main financial rows were not used for these UAT writes. Printing was preserved in code; a physical printer and screen reader were not exercised.

## Application summary alignment (2026-11-10)

Application details now use a semantic label/value definition list with a dedicated aligned colon column. Approval authority is read fresh from /meta when application details open rather than reusing the workspace bootstrap value.

Frontend typecheck, ESLint, all 46 Node tests and the production web rebuild/restart passed. Read-only Chrome verification of local PJ-1 confirmed every colon shares the same horizontal position and the displayed Supervisor limit is IDR 100,000,000. No approval or financial posting was performed during this layout check.
