# ADR-0010: Business dates and transaction pagination

Status: Accepted
Date: 2026-10-21

## Context
Staff need Today, Last 7 days, Last 30 days, and inclusive custom transaction dates, combined with an account number. Browser timezone and latest-100 truncation would make history misleading.

## Decision
Resolve dates on the backend in Asia/Jakarta and display transaction timestamps in that same zone. Today is the default. Seven/thirty days include today and the preceding six/twenty-nine dates. A custom end date includes the full local day using an exclusive following-midnight query boundary.

Add a separate authenticated `/api/transactions/search` envelope with fixed 50-row keyset pages ordered timestamp/reference descending. Preserve the earlier array endpoint. Bind cursors to normalized date/account scope; send resolved explicit dates on following pages to avoid midnight rollover. Reset starts Today/all accounts on page one.

## Alternatives considered
Browser-local dates were rejected because staff should see the same business day across devices. Changing the existing array response was rejected for compatibility. Offset pagination was rejected because new postings displace offsets. A snapshot/reporting system is unnecessary for the current live history view.

## Consequences
No schema or dependency change is required. Cursors are navigation values, not credentials. Every page checks staff permissions. The listing is live and not a point-in-time export; refresh for newer records. Global timestamp indexing remains a measured future optimization before a large dataset rollout.

## Open details
None for local Phase 1; production business-timezone configuration and large-data performance require separate operational decisions.
