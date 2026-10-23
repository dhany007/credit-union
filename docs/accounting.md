# Accounting Foundations

Status: Proposed technical safeguards. Product accounting and accounting policies require review by the cooperative's accounting owner.

## Posting requirements

- Every posted financial transaction produces balanced double-entry journals.
- Journal headers, lines, account effects, and posting status commit atomically.
- Use fixed-precision decimal values or integer minor units with an agreed currency scale. Do not use binary floating point for money.
- Derive balances from posted entries or maintain reconcilable balance projections within the same transaction boundary.
- Keep journal and subledger reconciliation available by account, KCP, and cooperative.
- Correct posted transactions with linked reversals, subject to authorization and period rules.
- Record both transaction time and business date; define backdating and closed-period behavior.

## Illustrative entries only

| Event | Debit | Credit |
| --- | --- | --- |
| Cash savings deposit | Cash at servicing KCP | Member savings liability |
| Cash savings withdrawal | Member savings liability | Cash at servicing KCP |
| Cash loan disbursement | Loan principal receivable | Cash at disbursing KCP |
| Principal-only cash repayment | Cash at servicing KCP | Loan principal receivable |

These examples do not define product account mappings or account classification for membership capital, compulsory savings, fees, interest, penalties, or loss allowances. Actual installment allocation must follow an approved policy.

## Cross-KCP accounting

A transaction must preserve the servicing KCP and the relevant account or loan ownership dimensions. Cash movement belongs to the KCP handling that cash. Member home KCP and responsible KCP must not be overwritten to force reports to balance.

Decide whether KCPs maintain separately balanced books and use inter-KCP clearing accounts, or whether KCP reporting uses dimensions on one centralized ledger. The decision affects journal generation and must be resolved before cross-KCP posting is implemented.

## Required sign-off before financial posting

Approve the chart of accounts, product-to-account mappings, branch balancing approach, currency and rounding, opening balances, cash custody, reversal rules, business-day closing, period locks, and required reports. Include worked examples with expected journal lines and balances.

## Savings interest calculation and crediting

The confirmed business cadence is daily calculation with monthly balance crediting; this does not by itself decide when expense is recognized in the general ledger. See [Savings interest policy](savings-interest.md).

Two illustrative approaches require accounting review: recognize daily expense against an interest-payable account and settle that payable into member savings monthly, or keep daily calculation records and recognize expense when monthly interest is credited if the approved accounting policy permits. Select one approach; do not recognize the same expense twice.

Monthly credit is non-cash. It increases the member savings liability and does not increase teller cash. Define gross interest, deductions, any withholding liabilities, branch dimensions, and reconciliation to the sum of daily records. Daily records and monthly postings require distinct identifiers and an auditable link.

## Phase 2 local interest decision (2026-10-23)

ADR-0012 accepts calculation-only daily records and monthly expense recognition for local implementation: Dr savings_interest_expense / Cr member_savings. System credits use home KCP, no cash line, and unique account/month settlement. For the current restricted model savings equals cash plus posted interest expense. Real-money deduction treatment and broader operational accounting readiness remain separate gates.
