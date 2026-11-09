# ADR-0013: Local Phase 3 annuity loan servicing

Status: Accepted
Date: 2026-11-10
Decision basis: Owner selected annuity, nominal-band pricing and minimal local servicing rules, then requested finalization for implementation.

## Context

Applications currently exist without financial payout. Phase 3 requires reproducible pricing/schedules, one payout and cross-KCP repayments while preserving savings correctness and old application APIs.

## Decision

Follow the Final Local Implementation Decisions in [the tech plan](../tech-plan-phase-2-and-3.md).

- Six lending tables plus immutable loan_interest_policies; retain legacy application identity/contracts during separation from disbursed loans.
- Fixed monthly annuity using nominal annual bps divided by 12, half-up IDR rounding and final-principal reconciliation.
- Whole-approved-principal pricing bands selected at approval; persist policy/rate snapshots. Tariffs and approval limits are explicit owner configuration with no invented seed/default rates.
- version_number/version_type/is_current_version identify schedules. One current version per active loan, preserved old financial history; restructuring commands deferred.
- Seven audit fields on every table with truthful nullable legacy/system attribution. Operational soft-delete predicates have explicit financial-history/replay/job exceptions.
- One full cash payout per application; verified recipient, signed agreement and condition attestations plus a note are required. Payout and schedule/journal/audit/replay creation commit atomically.
- Payments allocate oldest installment first, interest then principal, support partial/advance scheduled payments, reject overpayment and generate no penalty/rebate. Closure requires all contractual principal and interest paid.
- Central cooperative ledger: payout Dr loan_principal_receivable / Cr cash; repayment Dr cash / Cr principal receivable and loan_interest_income. Recognize interest on receipt for this local scope. Preserve responsible KCP separately from servicing cash KCP.
- Preserve current role/authority checks; explicit revisions and database uniqueness prevent stale confirmations and double payout. Uncertain responses replay the original command.
- Retain one Loans sidebar menu with Applications, Disbursement and Loan Accounts views, EN/ID and light/dark; implement backend first.

## Alternatives considered

Manual per-approval rates were superseded by amount-band policies. Flat/fixed-principal methods were discussed; annuity was chosen. A workflow/rule engine, separate KCP books, floating-rate changes and restructuring execution add scope not required for this local iteration.

## Consequences

Forward migrations, compatibility tests, a backup/maintenance deployment for the application table split and populated restore reconciliation are required. Loan entries require kind-specific accounting rather than the current savings-only two-line assumptions.

This ADR supplements ADR-0003 and records the concrete local loan posting model alongside the proposed general safeguards in ADR-0005; it does not rewrite prior savings decisions or claim full bank accounting coverage.

## Deferred scope

Restructuring carryover, exceptional interest waivers/settlement, penalties, write-offs, posted-entry reversal UI, separate branch clearing, bank reporting and member self-service.
