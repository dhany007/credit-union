# ADR-0007: Daily Whole-Balance Savings Interest with Monthly Credit

Status: Accepted  
Date: 2026-10-21  
Decision basis: The project owner clarified that rates concern savings balances, confirmed p.a., and approved one rate for the whole balance, daily calculation, and monthly crediting.

## Context

Earlier discussion mistakenly interpreted the rate bands as loan pricing. The intended feature pays savings interest to members. The cooperative needs configurable rates without confusing savings remuneration with loan amortization.

## Decision

Select one annual savings rate according to the eligible balance band and apply it to the entire eligible balance. Calculate interest daily and credit it to the savings balance monthly. Support global base-rate configuration and Supervisor-managed bands. Proposed rate values remain examples until approved for operation.

## Alternatives considered

- Progressive marginal bands: rejected by the owner in favor of one rate for the whole balance.
- Calculate from month-end balance only: does not meet the confirmed daily calculation requirement.
- Interpret the bands as loan interest: corrected; loan interest is a separate unresolved policy.

## Consequences

Daily balance and rate history must be reproducible. Monthly crediting needs duplicate protection, accounting reconciliation, and recovery for missed runs. Savings rate changes should use dated versions prospectively; fixed-rate loan assumptions must not be applied to this feature.

## Open details

Eligible balance basis, rate-change mechanics, day-count denominator, precision and rounding, deductions, posting date, account status handling, and accounting treatment remain open. Daily closing balance and the technical safeguards in the product specification are proposals. See [savings interest policy](../savings-interest.md).
