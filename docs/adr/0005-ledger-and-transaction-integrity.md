# ADR-0005: Atomic Ledger Posting and Financial Transaction Integrity

Status: Proposed  
Date: 2026-10-21

## Context

Cross-KCP deposits, withdrawals, disbursements, and repayments must remain explainable despite failures, repeated requests, and simultaneous operations.

## Decision

Propose balanced double-entry posting for every posted financial transaction. Commit the financial transaction, journal, account or loan effects, and durable audit event atomically. Use exact monetary representations, persistent idempotency records, and verified concurrency control. Correct posted financial records with authorized linked reversals.

## Alternatives considered

- Store only mutable balances: cannot adequately explain financial history or support reconciliation.
- Write domain changes and journals separately: permits partial financial outcomes.
- Rely only on frontend duplicate prevention: does not handle retries or concurrent requests reliably.

## Consequences

Financial features require approved accounting examples and meaningful failure/concurrency tests. Journals remain available for audit, while financial corrections require explicit workflows. External side effects must not create inconsistent database posting outcomes.

## Open details

Confirm chart of accounts, branch balancing, product mappings, precision, rounding, balance projections, reversals, closed periods, and retention. This technical proposal does not replace accounting policy approval.
