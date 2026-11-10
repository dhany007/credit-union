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

## Automatic schedule display (2026-11-10)

- Eligible application details load the server-generated estimate automatically. Changes to amount/term debounce for 350 ms, abort obsolete work and hide any result not bound to the current request. Inputs remain editable during recalculation.
- Approval is disabled without a current successful schedule; an old confirmation cannot submit unpriced approval after inputs change. Failed calculations use the existing error modal and expose Reload schedule.
- Account summary now displays the persisted current contractual schedule directly. Historical-version navigation remains available; summary always returns to the current version. A disbursed application's details also display its linked persisted schedule.
- Estimates and committed schedules remain distinct. Read-only approved cases awaiting payout retain their approved terms; the Teller payout view loads the authoritative approved-pricing schedule automatically. No client-side loan calculation or repricing was added.
- All tables retain the existing 20-row pagination; EN/ID labels and theme tokens are preserved.
- Frontend typecheck, ESLint, all 50 tests and production Docker rebuild/restart passed. New tests prove the debounce, cancellation, out-of-order-response and obsolete-error guarantees.
- Read-only Chrome checks on local PJ-1 confirmed 12 rows appear without a preview click, invalid input removes the old table and disables approval, and changing to IDR 90,000,000 / six months automatically displays six installments. Original fields were restored without submitting a decision.
- Isolated Chrome checks on disbursed LN-1 confirmed the stored 12-row schedule is visible while Summary remains selected. No new financial posting or approval was performed.

## Compact application/calculation summary (2026-11-11)

Application details combine the editable terms and server-generated calculations in one responsive two-column summary. The schedule below omits repeated amount, term, tariff and totals; the standalone intake preview retains its self-contained summary. Original requested terms appear only when they differ from the reviewed or approved terms.

Typecheck, ESLint, all 50 tests and production Docker rebuild/restart passed. Read-only Chrome checks on PJ-1 verified two side-by-side definition lists with internally aligned colons, one regular-installment label, no repeated tariff-source paragraph, and automatic six-row schedule refresh after editing amount/term. Original inputs were restored without submitting a decision. No browser error-level logs were captured.

## Consistent table paging and retained estimates (2026-11-11)

This follow-up supersedes the earlier 20-row defaults and removal of old estimates during recalculation.

- Shared controls default to 10 rows, with 10 / 50 / 100 / All choices. Table viewports reserve ten row heights and use internal scrolling with sticky headers; All retains the active filters.
- Server collections accumulate bounded cursor pages. GET /members accepts optional positive before_id while preserving its latest-100 array response for existing callers.
- Previous successful estimates remain dimmed during recomputation or invalid input. Only the current successful estimate enables approval. Receipts still print all allocations.
- Member pagination integration test demonstrated RED then GREEN. Full Go race suite, go vet and golangci-lint passed (zero lint issues).
- Frontend typecheck, ESLint and all 56 tests passed; API and web production images rebuilt and restarted healthy on localhost:3000.
- Isolated Chrome UAT used 151 synthetic members: 50 rows and All (151 rows) retained the same 778 px viewport; scrolling to the oldest row retained the sticky header and selecting that member loaded its profile beyond the bootstrap's first 100.
- Loan tables initially displayed ten rows. Clearing the reviewed amount retained ten dimmed schedule rows with approval disabled. Changing the estimate to IDR 900,000 / six months retained the table while loading, then displayed six updated installments and enabled approval only after the response completed.
- No approval or financial posting was submitted during these checks. No browser error-level logs were captured. EN/ID strings and existing light/dark tokens remain supported.

## Full-page loan details (2026-11-11)

Loan details now replace the list in the existing workspace instead of opening a detail dialog. The list remains mounted and hidden, preserving filter inputs, cursor cache and display-page selection when Back is used. This uses existing workspace navigation; no new URL route or API is introduced.

Application approval, rejection and payout confirmation use one shared dialog above the detail page. It includes the member/account, amount, term/rate where applicable and decision reason. Returning from confirmation preserves inputs. The confirmation closes before command processing/recovery begins. Successful writes refresh the selected details rather than navigating away; receipts remain available in their existing modal.

Frontend typecheck, ESLint, all 56 tests and production Docker web rebuild/restart passed. Read-only Chrome checks on PJ-1 verified details as a page with no initial dialog, exactly one approval-confirmation dialog, cancellation preserving the reason, and Back retaining the 50-row display selection. No decision or financial write was submitted; no error-level browser logs were captured.

## Success toast follow-up (2026-11-11)

Non-financial loan successes (including approval/rejection) now use shadcn Sonner notifications at the top right, with EN/ID text, theme tokens, close control and a five-second duration. Financial receipts retain their read/print modal. Confirmations and uncertain-command recovery remain explicit dialogs. Frontend typecheck, ESLint and 56 tests passed; no approval was posted to trigger a visual test.

## Loan submenu and Supervisor operational authority (2026-11-11)

- Sidebar contains Applications, Loan accounts and Supervisor-only Interest rates. Removed the workspace tabs and separate payout menu.
- Teller application creation uses a page with Back; success refreshes the application list with an EN/ID toast. Approved without a linked loan displays Awaiting disbursement; linked loans display Disbursed.
- Supervisor alone may preview/post payout and repayment. Teller reads remain available. Payout/revision/verification/cash and duplicate prevention checks are retained.
- Integration tests proved RED when Teller preview was incorrectly accepted, then GREEN after role enforcement. Tests reject payout for submitted/rejected applications and reject Teller payout/repayment; concurrency, rollback, cross-KCP attribution, journal and savings reconciliation remain covered.
- Full Go race suite passed (integration 93.617 seconds), Go lint zero issues, frontend typecheck/ESLint and 58 tests passed. API and web images rebuilt/restarted healthy.
- Chrome verified main Teller submenu, full-page intake/Back, Awaiting-disbursement status with no Teller payout action. On isolated port-3002 data, Teller submitted synthetic PJ-27 and returned to the refreshed list with the expected toast and no success dialog. Supervisor saw the additional tariff submenu and no payout action on submitted or already-disbursed applications. No main financial write occurred.
