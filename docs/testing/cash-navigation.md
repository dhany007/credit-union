# Separate cash-service navigation

Date: 2026-10-22.

Cash Deposit / Setor Tunai and Cash Withdrawal / Tarik Tunai now have separate navigation entries, active states, headings, and member-detail shortcuts. Both use the same existing cash-form JSX, account lookup, confirmation, receipt/history, and `/api/cash` mutation. No backend, database, dependency, or theme-token change.

The selected menu determines the posted transaction kind. The in-form operation switch is removed. Entering either cash service clears the amount, note, previous notices and confirmation while retaining the selected account. Navigation and member shortcuts are disabled during pending operations.

## Validation

- Frontend ESLint: PASS.
- TypeScript: PASS.
- Existing Node utility tests: 22 PASS, no failures/skips.
- Docker frontend production build: PASS; updated frontend deployed locally without rebuilding the backend.
- Browser: both Indonesian menus/headings and English Cash Deposit/Cash Withdrawal labels visible; active menu uses `aria-current=page`.
- Browser: deposit and withdrawal each show the correct confirmation text without posting a movement.
- Asserted keyboard switching clears amount, note, and confirmation, and preserves the account.
- Both member-detail shortcuts open the matching cash service with the selected account.
- Existing dark/light controls render both services correctly; original Indonesian/dark preferences restored after checks.
- No browser error-level logs captured. Test login signed out and temporary tab closed.
- Screenshots saved as `cash-withdrawal-id-dark.png` and `cash-deposit-en-light.png` in the task visualization directory.

The browser check stops before final transaction posting; backend posting/role rules remain unchanged and were covered by the preceding Phase 1 UAT. No new financial transaction is required to validate this navigation change.

Final reconciliation: member/transaction/journal/loan/retry/audit table digests and all financial totals match the pre-browser baseline. Main remains 15 members, 7,500 movements, IDR 111,975,000 savings, and equal IDR 1,575,395,000 debit/credit totals. Docker Desktop was restarted after the interruption; DB/API/web returned healthy.
