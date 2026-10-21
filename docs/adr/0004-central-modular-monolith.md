# ADR-0004: Central Modular Monolith and PostgreSQL

Status: Proposed  
Date: 2026-10-21

## Context

Shared membership and cross-KCP money movement require consistent records. The project is starting from zero and has no established scale, hosting, or connectivity targets.

## Decision

Propose one Go backend organized into modules, a central PostgreSQL database, and online financial posting shared by all KCPs. Deploy the Next.js frontend alongside the API under an agreed hosting topology. Avoid separate databases per KCP for the initial release.

## Alternatives considered

- Microservices: allow independent scaling but add distributed workflow and consistency complexity.
- Branch-local databases with synchronization: support offline operation but complicate identity uniqueness and concurrent money movement.
- Another transactional relational database: viable, but not yet evaluated against operational constraints.

## Consequences

Local database transactions can protect multi-module financial operations. Central hosting requires reliable branch connectivity and a tested recovery plan. Module boundaries must remain explicit so one deployment does not become unstructured code.

## Open details

Approve the database choice, connectivity assumptions, availability targets, deployment environment, storage model, and recovery objectives. If offline posting is required, revisit this ADR before implementation.
