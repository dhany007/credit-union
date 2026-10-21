# Business Rules

## Confirmed rules

| ID | Rule |
| --- | --- |
| BR-001 | The system serves one cooperative with multiple KCPs. |
| BR-002 | A person has at most one membership across the cooperative. |
| BR-003 | Each member has one home KCP. |
| BR-004 | Members may deposit and withdraw at any KCP. |
| BR-005 | Members may submit loan applications at any KCP. |
| BR-006 | Each loan application has exactly one responsible KCP for file management and loan monitoring. |
| BR-007 | Approval authority depends on amount, product, and risk. |
| BR-008 | Installments may be paid at any KCP without changing the responsible KCP. |
| BR-009 | Staff roles distinguish application intake, analysis, decision, disbursement, and monitoring responsibilities. |
| BR-010 | Initial roles are Member, Teller, and Supervisor. Teller includes member-data input; Admin is not a separate initial business role. Supervisor handles monitoring and loan decisions within the authority matrix. |
| BR-011 | Savings interest rates are annual (p.a.); one balance-band rate applies to the entire eligible balance, not progressively by layer. |
| BR-012 | Savings interest is calculated daily and credited to the savings balance monthly. |
| BR-013 | Support a global savings base rate and Supervisor-managed balance bands. Example rates are not final production policy. |
| BR-014 | The initial deposit amount is IDR 50,000; the exact account-opening or contribution allocation remains open. |

See [Savings interest policy](savings-interest.md) for the example bands, calculation contract, and remaining decisions. Loan interest methods remain undecided.

## Delivery priority

Phase 1 implements registration, cash deposits, cash withdrawals, and basic lending. BR-011 through BR-013 remain accepted savings-interest policy for Phase 2. Retain cross-KCP and approval-authority rules, implemented with the simplest sufficient checks. See [ADR-0008](adr/0008-core-mvp-first.md).

## Proposed implementation rules

| ID | Proposal |
| --- | --- |
| PR-001 | Generate a cooperative-wide unique member number. Enforce person uniqueness through an agreed identity policy; member numbers alone cannot prevent duplicate people. |
| PR-002 | Record the servicing KCP and acting staff member on each transaction. Cross-KCP service must not create a new membership. |
| PR-003 | Servicing a transaction does not automatically change the member's home KCP or the loan's responsible KCP. |
| PR-004 | Posted financial transactions produce balanced journals atomically and cannot be edited or deleted through ordinary operations. Corrections use authorized reversals. |
| PR-005 | Financial commands support idempotent retries and protect against concurrent overdraw or duplicate disbursement. |
| PR-006 | Each submitted loan application must have an assigned responsible KCP. The assignment policy remains open. |
| PR-007 | Every lending decision stores its actor, outcome, timestamp, reasons, and the authority policy version used. |
| PR-008 | Permission checks combine role, access scope, workflow state, and applicable approval limits. |
| PR-009 | The resulting loan inherits the application's responsible KCP; later reassignment requires an explicit, audited action. |
| PR-010 | Missing or ambiguous approval rules block approval and require policy resolution; they do not grant unlimited authority. |

## Business rules still required

See [open questions](open-questions.md) for identity evidence, account eligibility, withdrawal limits, approval thresholds, committee decisions, interest formulas, rounding, payment allocation, and period closing.

Cross-KCP availability does not waive balance checks, member verification, product restrictions, approval requirements, or staff access controls. The exact rules for those checks remain to be agreed.
