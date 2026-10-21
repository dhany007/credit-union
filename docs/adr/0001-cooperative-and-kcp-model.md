# ADR-0001: One Cooperative with Shared Membership and Cross-KCP Services

Status: Accepted  
Date: 2026-10-21  
Decision basis: Explicit project-owner statements in the planning conversation.

## Context

The cooperative has multiple KCPs. A person must not register at another KCP after becoming a member. Members may deposit, withdraw, apply for loans, and pay installments at any KCP.

## Decision

Model one cooperative with multiple KCPs and one membership per person across the cooperative. Each member has one home KCP. Service location is separate from membership registration location.

## Alternatives considered

- Separate memberships per KCP: conflicts with the agreed uniqueness rule.
- Restrict all services to the home KCP: conflicts with the agreed service availability.
- Multiple independent cooperatives: outside the initial scope.

## Consequences

Member lookup and duplicate prevention must work across KCPs. Transactions must preserve servicing location without creating additional memberships. Identity policy and access controls are necessary for accurate and appropriate cross-KCP lookup.

## Open details

Identity evidence, home KCP transfers, account product restrictions, and cash accounting are not decided by this ADR. It establishes shared business membership without selecting database topology.
