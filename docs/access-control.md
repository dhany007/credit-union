# Access Control

## Confirmed principle

The initial roles are Member, Teller, and Supervisor. Teller includes member-data entry; there is no separate Admin role in the initial business role model. Supervisor monitors operations and approves loans within the authority matrix based on amount, product, and risk. Detailed permission bundles and staffing assignments still require review. See [ADR-0006](adr/0006-initial-business-roles.md).

## Initial roles and proposed permission details

| Role | Main permissions | Boundary |
| --- | --- | --- |
| Member | View own profile, balances, and loans if member access is included | Own records only; portal scope remains open |
| Teller | Enter member data; proposed permissions include application intake, deposits, withdrawals, installments, and approved disbursement processing | Input and execution do not grant loan approval; transaction and cash-session limits apply |
| Supervisor | Monitor operations and approve or reject loans within assigned authority; proposed duties include verification and analysis | Subject to amount, product, risk, and KCP scope; no automatic unlimited approval |

The three role names and consolidation of Admin into Teller are confirmed; the detailed action mapping above remains proposed where indicated. Whether one person can hold multiple roles, and which duties cannot be combined, remains open. Preventing a staff member from approving their own input is a recommendation pending policy approval. Accounting, audit, and technical administration responsibilities still need assignment; they are not additional confirmed roles and must not silently become unrestricted Supervisor permissions.

Membership registration does not automatically create a login account. Member portal inclusion and provisioning remain open.

## Proposed authorization model

An action is permitted only when all applicable checks pass:

1. The user is authenticated and active.
2. Their assigned role grants the action.
3. Their KCP scope covers the target operation or case.
4. The current workflow state allows the action.
5. Any transaction limits or authority-matrix conditions are satisfied.
6. Any separation-of-duty or required additional approvals are satisfied.

Cross-KCP member service requires a deliberate lookup permission available to authorized service roles. It does not imply unrestricted access to all identity documents, loan analysis, or records across the cooperative.

## Authority matrix design inputs

Each version should define the product, amount boundaries, risk category, authorized role or decision level, organizational scope, effective dates, and any required number of approvals. Define boundary inclusivity, overlapping rule resolution, currency, aggregation of existing exposure, and handling of policy changes before implementation.

No numeric approval limits or risk classifications have been agreed. Do not seed illustrative thresholds as production policy.

## Savings rate configuration

Supervisor manages savings-interest configuration; Teller and Member roles do not receive this permission by default. Rates are annual and apply to savings balances, independently of loan approval authority. See [Savings interest policy](savings-interest.md).

Proposed controls: explicit configuration permission, version history, effective dates, and audit reasons. The KCP/product scope of configuration rights and any second-person approval requirement remain open. No role may directly rewrite historical daily accruals or posted credits through a normal rate edit.

## Implemented Phase 1 transaction history

Teller and Supervisor may read cash transaction history across KCPs through the dedicated Transactions menu and `GET /api/transactions`. Anonymous access is rejected. A full 10-digit savings account filter selects only that account; no filter shows up to 100 latest postings across the cooperative. Opening deposits, ordinary deposits, and withdrawals are included. The response contains member name, membership/account numbers, cash direction/amount, reference/note, date, servicing KCP, and recording staff. It does not expose NIK, contact fields, session data, credit analysis, or other identity documents. Viewing history never grants posting or loan-decision authority. The member portal remains absent.
