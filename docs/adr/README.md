# Architecture Decision Records

ADRs preserve the context, decision, alternatives, and consequences of significant design choices. Accepted records reflect explicit project-owner decisions. Proposed records require review before they become architectural commitments.

| ADR | Decision | Status |
| --- | --- | --- |
| [0001](0001-cooperative-and-kcp-model.md) | One cooperative, unique membership, cross-KCP services | Accepted |
| [0002](0002-go-and-nextjs.md) | Go backend and Next.js frontend | Accepted |
| [0003](0003-lending-responsibility-and-authority.md) | Responsible KCP, role-based duties, and authority matrix | Accepted |
| [0004](0004-central-modular-monolith.md) | Central modular monolith with PostgreSQL | Proposed |
| [0005](0005-ledger-and-transaction-integrity.md) | Atomic ledger posting, reversals, and retry protection | Proposed |
| [0006](0006-initial-business-roles.md) | Initial Member, Teller, and Supervisor roles | Accepted |
| [0007](0007-daily-savings-interest-monthly-credit.md) | Annual whole-balance savings interest, daily calculation, monthly credit | Accepted |
| [0008](0008-core-mvp-first.md) | Four core services in Phase 1; savings interest deferred to Phase 2 | Accepted |

## Lifecycle

Use Proposed, Accepted, Rejected, or Superseded. Record acceptance dates when approval occurs. If an accepted decision changes materially, create a new ADR and mark the earlier record Superseded with a link. Do not rewrite historical decisions as if they were always different.

## New ADR template

```text
# ADR-NNNN: Title
Status: Proposed
Date: YYYY-MM-DD

## Context
## Decision
## Alternatives considered
## Consequences
## Open details
```
