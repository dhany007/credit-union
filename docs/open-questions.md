# Open Questions and Decision Register

> Prioritize only questions needed for the next MVP step. OQ-018 through OQ-022 concern Phase 2 savings interest and do not block initial development. Advanced product and workflow cases can be deferred with explicit scope limits. See [roadmap](roadmap.md).


Open questions are not approved defaults. Owner roles below are suggested decision owners, not assigned individuals.

| ID | Decision required | Suggested owner | Blocking milestone |
| --- | --- | --- | --- |
| OQ-001 | What establishes unique person identity? How are missing NIK, minors, foreign members, former members, and corrected identities handled? | Membership operations | Membership implementation |
| OQ-002 | Is home KCP always the registration KCP? Who can transfer it, and must history be retained? | Operations | Membership implementation |
| OQ-003 | How is the responsible KCP selected? Who can reassign it, including after disbursement? | Credit operations | Lending implementation |
| OQ-004 | Initial roles are Member, Teller, and Supervisor (confirmed 2026-10-21). What exact action permissions, KCP scopes, and incompatible duties apply? Who handles accounting, audit, and technical administration? | Management and control | Access implementation |
| OQ-005 | Initial deposit is IDR 50,000; clarify whether it funds account opening or membership contributions. Which savings products, eligibility, minimum balance, withdrawal, fee, and closure rules apply? | Product and accounting | Savings implementation |
| OQ-006 | Which currency, amount scale, rounding rules, chart of accounts, and opening-balance policies apply? | Accounting | Financial posting |
| OQ-007 | Do KCPs maintain separately balanced books? What inter-KCP clearing and reconciliation entries are needed? | Accounting | Cross-KCP financial posting |
| OQ-008 | How do teller cash sessions, cash transfers, cash shortages, reversals, business dates, and closing work? | Operations and accounting | Teller pilot |
| OQ-009 | Which lending products, interest methods, repayment schedules, fees, penalties, and allocation rules apply? | Credit and accounting | Lending calculations |
| OQ-010 | What are the authority thresholds, risk categories, committee quorum, escalation paths, and policy-version rules? | Credit management | Approval implementation |
| OQ-011 | What evidence and conditions authorize disbursement? Are staged disbursements allowed? | Credit operations | Disbursement implementation |
| OQ-012 | How do arrears, early settlement, rescheduling, write-offs, and loan closure work? | Credit and accounting | Lending pilot; unsupported cases need an explicit operating procedure |
| OQ-013 | What member or payer verification is required for each service, including third-party payments? | Operations and control | Teller pilot |
| OQ-014 | What existing data and balances must be migrated, and how will reconciliation and cutover be approved? | Operations and accounting | Production pilot |
| OQ-015 | What deployment, connectivity, recovery objectives, retention, privacy, applicable regulatory obligations, and reporting requirements apply? | Management, IT, and relevant advisers | Design where affected; production readiness |
| OQ-016 | How many members, KCPs, staff, concurrent sessions, and daily transactions should the system support? | Project owner | Infrastructure sizing |
| OQ-017 | Is a member portal needed in the initial release? | Project owner | Scope finalization |

## Savings interest decisions

Confirmed: whole-balance annual rate by band, daily calculation, and monthly credit. Owner acceptance on 2026-10-23 is recorded in ADR-0012; original questions are retained below for history. OQ-018: per-account Jakarta closing balance. OQ-019: Actual/365 Fixed and monthly half-up whole-IDR rounding without carry. OQ-020: approved rates, explicit global fallback and prospective audited Supervisor versions. OQ-021: first-day credit and actual-day late compounding; real-money deductions and unsupported closure/correction workflows remain gated. OQ-022: monthly expense/savings posting at home KCP for local implementation; real-money readiness remains separate.

| ID | Decision required | Suggested owner | Blocking milestone |
| --- | --- | --- | --- |
| OQ-018 | Which balance is eligible: daily closing balance or another basis; per account or aggregated per member; what timezone and cutoff? | Product and accounting | Savings interest implementation |
| OQ-019 | Which annual day-count denominator, leap-year handling, precision, rounding stage, and residual policy apply? | Accounting | Savings interest implementation |
| OQ-020 | What are the final rates, exact band boundaries, global/product/band precedence, gap behavior, and Supervisor configuration scope or approval controls? | Product and management | Rate configuration implementation |
| OQ-021 | What monthly credit date, credited-interest eligibility timing, deduction rules, closure/inactive-account treatment, and historical correction rules apply? | Operations and accounting | Monthly interest credit implementation |
| OQ-022 | Are interest expenses accrued in the ledger daily or recognized at monthly credit? What journal accounts and branch attribution apply? | Accounting | Interest financial posting |

## Suggested next workshop

Start with OQ-001, OQ-002, OQ-004, and OQ-005. Bring anonymized examples of a registration form, a savings receipt, a daily cash report, and current product rules. Next, resolve accounting and cash rules before implementing money movement. Lending policy can be developed alongside membership and savings work.

When a question is resolved, update the relevant rule, ADR if architectural, process diagram, and acceptance scenario. Record the decision date and decision owner; do not delete the historical reasoning.
