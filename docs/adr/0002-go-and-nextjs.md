# ADR-0002: Go Backend and Next.js Frontend

Status: Accepted  
Date: 2026-10-21  
Decision basis: Project owner's stated technology direction.

## Context

The project requires staff-facing web interfaces and a backend implementing cooperative business operations.

## Decision

Use Go for the backend and Next.js for the frontend.

## Alternatives considered

A single full-stack Next.js application or another backend language could support web workflows, but neither was selected for this project. No formal benchmark comparison has been performed.

## Consequences

Maintain an explicit API contract and two build environments. The proposed responsibility split places business rules and financial authorization in Go and user interaction in Next.js. Database, runtime versions, libraries, hosting, and authentication details remain separate decisions.

## Open details

Confirm TypeScript usage, API routing, session management, deployment topology, and supported dependency versions during technical foundation work.
