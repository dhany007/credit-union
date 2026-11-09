# Local Phase 3 backend operations

Phase 3 adds the backend only. Existing member/savings screens remain available; the new loan frontend is a separate implementation.

## Prerequisites and pricing

Use the existing Docker Compose environment. Keep the configured approval limit; the software never raises it or seeds a loan rate. A zero approval limit blocks priced approvals and payouts.

A Supervisor creates a prospective loan policy through POST /api/loan-interest/policies. Both the base annual nominal rate and every band must be at least 300 bps (3% p.a.) and at most the technical ceiling of 10000 bps. Explicit bands: [] is valid for a base-only policy. Omitted/null tariff fields and below-minimum rates are rejected. Policy effective dates must be tomorrow or later in Asia/Jakarta; no current policy means priced operations are unavailable until configuration takes effect.

The system uses one whole-principal rate, fixed annuity, annual nominal rate / 12, half-up IDR rounding and final-principal reconciliation. No loan tariff is taken from savings.

## Backend routes

| Route | Access |
| --- | --- |
| GET /api/loans; POST /api/loans; POST /api/loans/:id/decision | Existing application API; existing role/authority rules |
| GET/POST /api/loan-interest/policies | Supervisor |
| POST /api/loans/preview | Teller/Supervisor |
| GET /api/loan-applications; GET /api/loan-applications/:id | Teller/Supervisor |
| POST /api/loan-applications/:id/disbursement-preview | Teller |
| POST /api/loan-applications/:id/disburse | Teller |
| GET /api/loan-accounts; GET /api/loan-accounts/:id | Teller/Supervisor |
| GET /api/loan-accounts/:id/installments | Teller/Supervisor |
| GET /api/loan-accounts/:id/schedule-versions | Teller/Supervisor |
| GET /api/loan-accounts/:id/repayments | Teller/Supervisor |
| POST /api/loan-accounts/:id/repayment-preview | Teller |
| POST /api/loan-accounts/:id/repayments | Teller |

New collections return {items,next_cursor}, 20 records per page. Keep filters unchanged when using a cursor. Legacy GET /api/loans retains its latest-100 array contract. Applications expose approval_ready as basic approval state; payout preview provides authoritative authority/cash eligibility for the acting KCP.

Policy commands, decisions, disbursement and repayment require the existing Idempotency-Key and same-origin/session controls. Previews save no financial/application/replay rows and require no idempotency key.

## Workflow

1. Teller submits through POST /api/loans using existing member_id, amount, tenor_months, purpose, risk and responsible_branch_id.
2. Preview with amount, tenor_months and expected_disbursement_date (today through one year ahead). Read policy_id and annual_rate_bps.
3. Supervisor decides through the existing decision route. For priced approval provide approved_amount, approved_tenor_months, expected_policy_id, expected_rate_bps and expected_approval_revision alongside decision/reason. Approved principal/tenor cannot exceed the requested values. Default revision before priced approval is 0.
4. Legacy approvals remain readable/unpriced. The same authorized decision route may complete pricing once; a fully priced approval is immutable and not reapproved in place.
5. Teller requests payout preview using expected_approval_revision. Review actual payout_date, cash and schedule.
6. Confirm with expected_approval_revision, expected_policy_id, expected_rate_bps, expected_principal, expected_tenor_months and expected_payout_date. recipient_verified, agreement_signed and conditions_verified must be true; verification_note must contain 5..500 trimmed characters; note is optional. The server rechecks all terms and cash under locks.
7. Read the returned loan and schedule. Only initial issuance is executable; old versions remain available and restructuring is not enabled.
8. Preview repayment with amount. Confirm with amount, expected_payment_revision, expected_schedule_version_id and optional note.
9. Retry uncertain commands using the same key and body. A new key cannot create a second payout. Reprinting is a read, not a new posting.

Repayments allocate oldest installment first, unpaid interest then principal. Partial/advance scheduled payments are supported without penalty or automatic interest waiver. Reject overpayment. paid_off requires all scheduled principal and interest paid.

## Maintenance migration and recovery

Migration 5 adds audit columns and backfills known actors/timestamps without inventing identities. Unknown legacy/system attribution stays nullable. Migration 6 renames the former application-shaped loans table to loan_applications and creates disbursed loans and servicing tables. Original application IDs, decision facts and command replay payloads remain intact.

This table split requires stopping the old API before migration:
1. Verify a populated backup in an isolated database and run the new migrations there.
2. Build the API image before downtime: docker compose --env-file .env.docker build api.
3. Stop API writers with docker compose --env-file .env.docker stop api.
4. Take a final pg_dump -Fc backup of credit_union to the ignored .local/backups directory.
5. Run docker compose --env-file .env.docker run --rm --no-deps migrate.
6. Start the compatible API with docker compose --env-file .env.docker up -d --no-deps --wait api.
7. Check http://localhost:3000/api/health and the main page, then inspect the interest scheduler.
8. Run docs/testing/sql/phase-1-reconciliation.sql and savings-interest-reconciliation.sql against the intended database. The historical filename is retained; it now supports loans. Verify the original seed batch with backend/dev/data/demo_2026_reconcile.sql inside a transaction.

If the transactional migration fails, it rolls back; restart the unchanged old API container. After successful migration, use the compatible API and forward-fix issues. Do not run a pre-Phase-3 binary against the renamed schema or restore an old backup over newer financial postings.

Operational master queries filter is_active/deleted_at. Financial/replay/history/job queries retain documented exceptions. Ledger checks include servicing-KCP cash, loan receivables, loan interest income and unchanged member savings. A repayment may have three journal lines. Keep all audit and financial rows together in backups.

The legacy /meta disbursement_enabled field remains false for the existing frontend, which has no payout screen. New backend endpoints are available under the documented guards and configuration; this field is not a blanket authorization.
