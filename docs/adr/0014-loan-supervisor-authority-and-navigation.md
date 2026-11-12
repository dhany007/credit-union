# ADR-0014: Supervisor loan operations and sidebar navigation

Status: Accepted by owner, 2026-11-11. Supersedes ADR-0013's Teller payout authority.

## Decision

Tellers may create loan applications, generate estimates and read loan records. Supervisors approve/reject applications, preview/post disbursements and manage interest policies. Deposit/withdrawal permissions are unchanged.

Use sidebar Applications and Loan accounts submenus, plus Supervisor-only Interest rates. Application creation is a workspace page with Back; success returns to the updated application list and displays a toast. Application details host payout verification only for Supervisors on fully priced, approved, undisbursed applications. No separate payout menu or new database status is needed.

Display stages derive from existing application status and linked loan. Awaiting-disbursement filtering reuses approved plus payout_pending=true. All shows disbursed application history too.

HTTP and command-service checks enforce operational authority. Existing approval completeness, revision, pricing, cash availability, verification, transaction locks, idempotency and balanced journals remain required. Historical Teller loan postings remain valid in reconciliation; all new loan disbursements require Supervisor authorization.

## Consequences

No schema migration or new dependencies. The owner explicitly accepts the same Supervisor deciding and disbursing in this initial local workflow. This is not a separate-operator approval model.

Installment payment UI and repayment authority changes are deferred to a separate PR.
