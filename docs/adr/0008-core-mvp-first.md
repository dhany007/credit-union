# ADR-0008: Deliver the Four Core Services First

Status: Accepted  
Date: 2026-10-21  
Decision basis: The project owner requested a simpler initial phase: registration, cash deposits, withdrawals, and loans.

## Context

The planning documents describe a broad target system. Implementing savings interest automation and advanced workflows immediately would exceed the requested initial scope.

## Decision

Limit Phase 1 to member registration, cash deposits, cash withdrawals, and basic lending through Teller/Supervisor screens. Retain one cooperative with multiple KCPs, unique membership, cross-KCP service, loan responsibility, and approval limits. Prefer simple explicit permissions and status checks over configurable workflow engines.

Defer automated savings interest and its configuration interface to Phase 2. ADR-0007 remains the accepted future interest model; this record changes delivery timing, not its calculation principles.

## Alternatives considered

- Build the full previously documented roadmap first: exceeds the requested initial scope.
- Remove transaction consistency or approval controls to reduce work: does not support correct handling of the four services.

## Consequences

Keep basic transaction records, exact amounts, atomic balanced postings, duplicate protection, and backend permissions. Use minimal setup and operational screens. Resolve only the product rules needed for each implementation step.

The simplified scope and roadmap take precedence over earlier target-state delivery expectations. No implementation is claimed complete by this scope change.

## Open details

Confirm one savings product and one loan product, the deposit/withdrawal constraints, loan pricing and tenor, and simple approval authority. Basic repayment recording is proposed; either implement it or approve a reconciled external servicing process before real loan disbursement. Member self-service remains deferred.

## Phase 1 lending clarification (2026-10-21)

The project owner explicitly selected loan application and approval only. Disbursement, repayment scheduling, and repayments are deferred until lending rules are agreed. Approval is a workflow decision and does not create loan principal, move cash, or imply disbursement eligibility. Earlier references to a live lending pilot apply only when disbursement is later introduced.