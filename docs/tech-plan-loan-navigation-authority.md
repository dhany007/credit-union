# Loan navigation and operational authority

Owner-approved scope, 2026-11-11.

- Sidebar loan submenus: Applications, Loan accounts, and Supervisor-only Interest rates. Remove separate payout navigation.
- Teller creates applications on a full workspace page with Back. Successful creation returns to the refreshed list with an EN/ID success toast.
- Applications derive display status from persisted status and linked loan: submitted -> awaiting approval; approved without linked loan -> awaiting payout; linked loan -> disbursed; rejected -> rejected.
- Supervisor alone may preview/post payout. Teller retains application creation, estimates and read access. Apply role checks at HTTP and command service boundaries.
- Payout keeps complete approval, revision/pricing, verification, sufficient branch cash, locking, atomic journal and idempotency checks. Pending/rejected applications cannot pay out.
- Reuse existing list filters and ten-row table components. Preserve themes, translations, uncertainty recovery and printable financial receipts.
- Validate role restrictions and approved-only payout via integration tests before changing code, then existing concurrency/reconciliation suite, frontend checks and Docker rebuild/restart. No schema migration.

- Defer installment payment UI and authority changes to a separate PR.
