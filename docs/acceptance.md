# Acceptance Scenarios

> Phase 1 applies the scenarios relevant to registration, cash service, permissions, and basic lending. Savings-interest scenarios AC-021 through AC-032 are Phase 2 acceptance criteria. Repayment scenarios apply if basic repayments are included; otherwise a reconciled servicing procedure is required before live lending. See [scope](scope.md).


These scenarios become implementation tests after the associated proposed policies are approved. Confirmed business scenarios may be refined without weakening the agreed rules.

| ID | Scenario | Expected outcome | Basis |
| --- | --- | --- | --- |
| AC-001 | An existing member tries to register at another KCP | No second membership is created; authorized staff can find the existing membership | BR-002 |
| AC-002 | The same eligible person is registered concurrently at two KCPs | Exactly one membership is created; the other request resolves to a duplicate outcome | BR-002; identity policy open |
| AC-003 | A KCP A member deposits at KCP B | The same member account is credited; servicing KCP is B; home KCP remains A | BR-004; PR-002/003 |
| AC-004 | A KCP A member makes an eligible withdrawal at KCP B | The same account is debited and KCP B cash is reduced for cash service | BR-004; accounting proposal |
| AC-005 | Two concurrent withdrawals exceed available funds in total | Disallowed overdrawing is prevented without partial journals | Proposed product and consistency rules |
| AC-006 | A KCP A member applies at KCP B | One application is created with receiving KCP B and one explicitly assigned responsible KCP | BR-005/006 |
| AC-007 | A staff member has intake permission but no decision authority | They can submit an eligible application but cannot approve it | BR-009; access proposal |
| AC-008 | An approver exceeds the applicable amount, product, risk, or scope authority | The approval is blocked and no disbursement becomes available | BR-007; access proposal |
| AC-009 | No authority rule matches an application | Decision is blocked pending policy resolution | PR-010 |
| AC-010 | An installment is paid at KCP C for a loan managed by KCP B | The same loan is updated; servicing KCP is C; responsible KCP remains B | BR-008 |
| AC-011 | A committed deposit is retried with the same key and payload | The original outcome is returned with no second posting | PR-005 |
| AC-012 | An existing key is reused with a different amount | The request is rejected without altering the original transaction | Proposed idempotency rule |
| AC-013 | Posting fails while journal lines are being written | Domain changes and financial entries roll back together | PR-004 |
| AC-014 | Disbursement is requested twice for a single-disbursement loan | At most one permitted disbursement posts | Proposed lending policy |
| AC-015 | An authorized correction reverses a transaction | Original entries remain; linked reversing entries offset the permitted effects | PR-004; reversal policy open |
| AC-016 | A loan is approved but disbursement conditions are outstanding | Disbursement is blocked | Proposed workflow |
| AC-017 | Staff request a case outside their access scope | Backend denies access even if the frontend is bypassed | Proposed authorization |
| AC-018 | A backup is restored into an isolated environment | Agreed recovery objectives and ledger reconciliations are met | Proposed operational readiness |
| AC-019 | A Teller enters member information and attempts to approve a loan without Supervisor authority | Permitted data entry succeeds; loan approval is denied | BR-010; detailed permission mapping proposed |
| AC-020 | A Supervisor attempts to approve a loan outside the applicable authority matrix | Approval is denied despite the Supervisor role | BR-007/010 |

## Savings interest acceptance scenarios

The following use the illustrative rate bands in [Savings interest policy](savings-interest.md); they are test fixtures, not approval of production tariffs. Monetary expectations require agreed day-count and rounding rules.

| ID | Scenario | Expected outcome | Basis |
| --- | --- | --- | --- |
| AC-021 | Eligible balance is IDR 15,000,000 under the example bands | 4.5% p.a. applies to the entire balance; no progressive slicing | BR-011 |
| AC-022 | Balance equals IDR 1,000,000, 10,000,000, or 100,000,000 | Example bands select 4%, 4.5%, or 5% respectively | Proposed boundary convention |
| AC-023 | Daily balances cross a band during a month | Each day's eligible balance selects that day's single applicable rate; do not price the whole month from the final balance | BR-011/012 |
| AC-024 | Daily interest is calculated before monthly credit | Track calculation separately; do not increase spendable savings balance yet | Proposed accrual handling |
| AC-025 | A complete month is credited | Sum the daily calculations and apply approved rounding/deductions; post the resulting savings credit | BR-012; calculation details open |
| AC-026 | The same daily job or monthly credit job is retried or runs concurrently | One normal daily record per account/date and at most one normal monthly credit per account/period | Proposed job integrity |
| AC-027 | A dated rate version changes halfway through the month | Earlier dates retain their prior version; later dates use the new version | Proposed versioning |
| AC-028 | An account is missing one daily calculation when monthly processing starts | Block that account's credit, recover missing dates, then resume safely | Proposed recovery |
| AC-029 | Monthly journal posting fails | Savings credit and journal roll back together; retry cannot double-credit | Proposed posting integrity |
| AC-030 | Teller or Member attempts to edit rate configuration | Deny; an authorized Supervisor can manage configuration within the agreed scope | BR-013; access proposal |
| AC-031 | Global base applies with no explicit override under the agreed fallback policy | Use the dated base rate; never invent a rate for ambiguous configuration | Proposed precedence; policy open |
| AC-032 | Monthly interest is credited | Savings liability increases without a teller cash movement; accounting reconciliation holds | Accounting proposal |

Additional worked examples are required for rounding, interest, fees, partial and early repayments, late payments, reversals after subsequent activity, branch balancing, and period closing. Expected values must come from approved business and accounting policies.
