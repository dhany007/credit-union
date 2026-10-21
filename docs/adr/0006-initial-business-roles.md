# ADR-0006: Initial Member, Teller, and Supervisor Roles

Status: Accepted  
Date: 2026-10-21  
Decision basis: The project owner specified Member, Teller/Admin, and Supervisor, then clarified that Admin is consolidated into Teller.

## Context

Earlier planning proposed separate role names for each lending stage. The project owner wants a smaller initial role model with Teller performing member-data entry and Supervisor handling monitoring and loan approval.

## Decision

Use three initial business roles: Member, Teller, and Supervisor. Do not introduce a separate Admin role for member-data entry. Supervisor approval remains constrained by the previously agreed amount, product, and risk authority matrix.

This refines ADR-0003 without changing its KCP responsibility or approval principles. Workflow stages remain distinct actions even when one role covers several stages.

## Alternatives considered

- Separate Admin and Teller roles: not selected for the initial scope.
- A separate role for every lending stage: unnecessary for the requested initial role model.

## Consequences

Maintain granular permissions beneath the three roles. The role name alone must not grant every action. Data-entry permissions do not inherently include approval permissions. Member login provisioning remains separate from membership registration.

## Open details

Exact Teller transaction and disbursement permissions, Supervisor analysis duties, KCP access scope, role combinations, self-approval restrictions, and ownership of accounting, audit, and technical administration require review. Member portal inclusion remains open.
