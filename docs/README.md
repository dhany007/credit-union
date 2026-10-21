# Credit Union System Documentation

Status: Initial planning draft  
Last updated: 2026-10-21

This documentation describes a single cooperative operating through multiple KCPs (service branches). It is written in English and records the decisions discussed with the project owner. The Phase 1 application is now implemented for local review; see [implementation notes](implementation.md). Operational readiness is not complete.

## Current delivery scope

Phase 1 is deliberately small: member registration, cash deposits, cash withdrawals, and basic lending through staff. Start with [scope](scope.md) and the [simplified roadmap](roadmap.md). Automated savings interest and advanced workflows are deferred to Phase 2. Detailed target-state documents are references, not a requirement to build everything now.

## Reading order

1. [Scope and glossary](scope.md)
2. [Business rules](business-rules.md)
3. [Business processes and flow diagrams](business-processes.md)
4. [Access control](access-control.md)
5. [Accounting foundations](accounting.md)
6. [Architecture and conceptual data model](architecture.md)
7. [Acceptance scenarios](acceptance.md)
8. [Architecture decision records](adr/README.md)
9. [Roadmap](roadmap.md)
10. [Open questions](open-questions.md)
11. [Frontend design reference](frontend-design.md)

## Savings product update

See [Savings interest policy](savings-interest.md) for annual whole-balance rates, daily calculation, monthly credit, and unresolved calculation details. These are savings rates, not loan rates. Their implementation is deferred to Phase 2.

## Decision status

- **Confirmed:** explicitly agreed with the project owner.
- **Proposed:** a recommendation for review; not an approved business policy.
- **Open:** requires a decision before the affected implementation begins.

The baseline and its diagrams describe intended behavior, not existing software. Proposed workflow details must not silently become confirmed requirements. Update rules, diagrams, acceptance scenarios, and ADRs together when decisions change.

## Next planning milestone

Begin the small implementation steps in the [roadmap](roadmap.md). Resolve only the decisions needed for the next step. Lending rules are needed before lending implementation; savings-interest calculation details are not Phase 1 blockers.

## Backend implementation references

- [Gin/domain backend tech plan](tech-plan-phase-1-backend.md)
- [TDD and validation evidence](testing/gin-refactor.tdd.md)

- [Local Docker deployment runbook](deployment/docker-local.md)

- [Savings account numbers tech plan](tech-plan-savings-account-numbers.md)
- [Savings account numbers TDD evidence](testing/savings-account-numbers.tdd.md)

- [Account-filtered transaction history tech plan](tech-plan-transaction-history.md)
- [Transaction history TDD evidence](testing/transaction-history.tdd.md)

- [Transaction date filters tech plan](tech-plan-transaction-date-filters.md)
- [Transaction date filters TDD evidence](testing/transaction-date-filters.tdd.md)

- [CI quality tech plan](tech-plan-ci-quality.md)
- [Quality checks and coverage](testing/quality-checks.md)
- [CI quality validation evidence](testing/ci-quality.tdd.md)

- [Member balance ledger tech plan](tech-plan-member-balance-ledger.md)
- [Member balance ledger TDD evidence](testing/member-balance-ledger.tdd.md)

- [Reconciled local demo data](development/demo-data.md)
- [Demo data tech plan](tech-plan-demo-data.md)
