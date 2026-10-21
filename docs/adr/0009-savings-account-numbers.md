# ADR-0009: Separate savings account numbers

Status: Accepted
Date: 2026-10-21

## Context

Members must be able to deposit and withdraw at any KCP using an account number. Membership identity (NIK and `CU-xxxxxx`) is distinct from the savings account used for cash service. Phase 1 has one savings account per member and one savings product.

## Decision

Allocate a globally unique 10-digit numeric string automatically during registration, starting at `1000000001`. Do not encode NIK, date of birth, or KCP. The account number stays the same across KCPs. Continue using the existing internal member ID for foreign keys and cash posting.

Store `members.account_number` with a bounded PostgreSQL sequence, default, NOT NULL, format check, and UNIQUE constraint. The sequence is separate from member IDs, does not cycle, and may have gaps after failed transactions. The application exposes no renumbering operation and never accepts a caller-selected account number.

Use a tracked, serialized, transactional migration to assign numbers to existing members without modifying their balances, member numbers, home KCP, transaction journals, or history. Keep the existing initial schema unchanged. Do not downgrade by deleting allocated account numbers.

Provide exact authenticated cooperative-wide account lookup. Teller must see the resolved member name before confirming cash service. Changing or failing lookup clears the prior transaction target. Keep existing transaction authorization and confirmation controls.

## Alternatives considered

- Reuse the membership number: rejected because membership and savings identifiers serve different purposes.
- Derive the account from internal member ID: simpler, but permanently couples the business identifier to an internal key.
- Separate account tables and multiple savings accounts: deferred; not required by the current one-account Phase 1 scope.
- KCP-prefixed numbers: rejected because accounts are cooperative-wide and should survive organizational changes.

## Consequences

New member responses include `account_number`. Legacy stored registration replay results remain unchanged and may omit it; clients can retrieve the current projection. Account numbers are strings even though their content is numeric. Exact lookup uses the unique index and permits existing authenticated staff roles. Numbers are not authentication credentials; the existing staff authorization still applies.

This migration locks and backfills the small local MVP members table during a maintenance window. A larger deployment requires a separately assessed phased backfill. Number format is not an assertion about external banking standards.
