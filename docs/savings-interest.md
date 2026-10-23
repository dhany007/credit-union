# Savings Interest Policy

Status: Accepted by the project owner on 2026-10-23.
Implementation: Phase 2. See [ADR-0012 and flow diagram](adr/0012-savings-interest-calculation-and-posting.md), the [tech plan](tech-plan-savings-interest.md), and [validation evidence](testing/savings-interest.tdd.md).

## Approved rules

- One annual rate applies to the entire eligible daily closing balance of each savings account.
- Use Asia/Jakarta calendar days and Actual/365 Fixed, including leap years.
- Include the opening deposit and interest already posted to savings. Uncredited calculations earn no interest.
- Calculate exact daily numerators using integer basis points. Sum monthly, round half up once to whole IDR, and discard the fractional residual.
- Credit a complete month on the first day of the following month, after daily closing normally begins at 00:05 Jakarta.
- Late jobs catch up without backdating. Credited interest earns interest from its actual posting day's closing balance.
- Supervisor creates complete audited versions, effective tomorrow or later. Pending same-date versions may be superseded; historical calculations retain their original version.
- A matching band takes precedence over the global base rate. Bands are lower-inclusive and upper-exclusive; explicit gaps use the global rate. No product override.
- One Supervisor may configure rates; reason and expected latest version are required. Teller cannot configure rates.
- The first version fixes the activation date. No pre-activation interest or historical monetary backfill.
- Daily records are calculation records. Monthly credit recognizes Dr savings_interest_expense / Cr member_savings, attributed to home KCP and System. It does not increase teller cash.
- Local validation is gross-only. Applicable deductions and real-money operating/accounting readiness remain a separate approval gate. Closure, freezes, historical adjustments and reversals are outside this increment.

## Approved starting rates

Global base: 3% p.a.

| Daily closing balance B (IDR) | Rate on the whole B |
| --- | --- |
| 0 <= B < 1,000,000 | 3% p.a. |
| 1,000,000 <= B < 10,000,000 | 4% p.a. |
| 10,000,000 <= B < 100,000,000 | 4.5% p.a. |
| B >= 100,000,000 | 5% p.a. |

Exactly IDR 1,000,000 receives 4%; IDR 10,000,000 receives 4.5%; IDR 100,000,000 receives 5%. Select again each day. IDR 15,000,000 at 4.5% for 30 days produces IDR 55,479 after monthly rounding.

## Processing and recovery

An API-owned loop closes completed dates and catches up on startup and once per minute when SAVINGS_INTEREST_ENABLED=true. It shares PostgreSQL locks with savings writers. Daily calculations and completion markers commit together; unique account/month settlements prevent duplicate payment. A zero credit is recorded as completed without a zero-value transaction.

Supervisor's Savings interest screen displays progress and supports prospective policy versions. The transaction screens display interest as a positive non-cash credit and localize System attribution. EN/ID and light/dark use the existing preference controls.

Loan interest remains separate and is not configured by this feature.
