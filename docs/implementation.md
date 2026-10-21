# Phase 1 Implementation Notes

Date: 2026-10-21

## Available for local review

- Gin HTTP API with domain-local handlers/services/PostgreSQL stores, plus small shared platform packages.
- Next.js frontend, official shadcn/ui primitives, and the exact shared tweakcn theme tokens. Fonts are bundled through Fontsource; theme snapshot is tracked at `frontend/theme/zen-inspired.json`.
- Staff login, unique-NIK member registration, opening deposit, cross-KCP cash service, and transaction history.
- Loan intake and Supervisor decisions only; no cash disbursement or loan servicing.

See the [root README](../README.md) for setup, accounts, policy settings, assumptions, and limits.

## Implementation choices for review

| Item | Implemented choice |
| --- | --- |
| Identity | NIK unique in PostgreSQL, separate internal/member numbers, masked NIK on member list |
| Opening deposit | IDR 50,000 ordinary savings, atomic with registration |
| Money | Whole-rupiah integers; no binary floating point calculations |
| Transaction boundary | Balance, cash transaction, two journal lines, audit, and idempotency outcome commit together |
| Cross-KCP cash | Servicing KCP follows signed-in staff; home KCP does not change |
| Cash availability | Derived from cash journal lines; zero opening branch cash |
| Approval | Explicit environment limit; regular product and low risk only; no authority by default |
| KCP permissions | Shared member/case lookup; transactions use own KCP; Supervisor can decide across KCPs |
| Loan outcome | Approval/rejection records workflow status and reason only; does not establish final pricing or authorize payout |
| Authentication | Bcrypt passwords; random sessions stored as SHA-256 hashes; strict same-site, HTTP-only cookies; write origin checks |
| Database lifecycle | Separate API/migration/development-seed commands; existing wrapper interface retained |

These implementation choices do not mark all proposed business policies as approved. No zero-interest loan or arbitrary live approval ceiling is assumed. No new infrastructure, microservices, or workflow engine is introduced.

## Verification and remaining work

PostgreSQL integration tests with Go race detector cover the critical financial and access cases; frontend production build includes TypeScript checking. The local login page has been visually checked in the in-app browser. No production load, security audit, or restore drill has been completed.

Still pending before operational rollout: agreed approval limits, member identity correction, product/accounting sign-off, cash transfer/opening procedures, reversal support, backup restore verification, production access setup, and operational acceptance. Phase 2 savings interest remains deferred.

## Gin refactor

`backend/cmd/api/main.go` owns process lifecycle; `internal/platform/app` wires four domains: `auth`, `member`, `cash`, and `loan`. Registration consumes cash posting through a narrow capability and the same caller-owned transaction. Persistent legacy fingerprints are projected explicitly. The SQL schema remains unchanged.

Authentication dependency failures deliberately return 503 instead of 401; missing/wrong credentials and expired sessions still return 401. Log fields exclude identity searches, request bodies, and session material. See [TDD evidence](testing/gin-refactor.tdd.md) for validation and remaining scope.
