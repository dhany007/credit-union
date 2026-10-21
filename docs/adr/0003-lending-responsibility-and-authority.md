# ADR-0003: Loan Responsibility, Staff Roles, and Approval Authority

Status: Accepted  
Date: 2026-10-21  
Decision basis: Explicit project-owner acceptance of the lending model and role-based duties.

## Context

Loan applications may arrive at any KCP, but file management and loan monitoring require a clear responsible location. Receiving an application does not by itself determine who may approve it.

## Decision

- All KCPs may receive loan applications.
- Each application has one responsible KCP for file management and loan monitoring.
- Approval follows an authority matrix based on amount, product, and risk.
- Intake, analysis, decision, disbursement, and monitoring duties are distinguished by role.
- Installments may be paid at any KCP without changing the responsible KCP.

## Alternatives considered

- Automatically grant approval rights to the receiving KCP: insufficient to express the agreed authority matrix.
- Determine all workflow duties solely by staff location: conflicts with role-based responsibilities.
- Change responsibility whenever an installment is paid elsewhere: conflicts with responsibility continuity.

## Consequences

Store receiving KCP and responsible KCP separately. Distinguish role assignment from credit approval limits. A proposed access-scope layer will further restrict which cases each staff member may act on.

## Open details

Responsible KCP assignment and reassignment, role names, scope rules, duty conflicts, numeric thresholds, risk categories, committee approvals, and version handling remain open. Central-only approval is not an accepted requirement.
