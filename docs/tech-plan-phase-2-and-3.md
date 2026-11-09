# Tech Plan - Phase 2 Acceptance and Phase 3 Lending Lifecycle

Status: Phase 2 locally accepted; Phase 3 backend implemented and verified locally on 2026-11-10. Frontend implementation remains pending. See testing/phase-3-backend.tdd.md and deployment/phase-3-loans.md.
Date: 2026-11-09
Updated: 2026-11-10 (annuity, schedule versions, and audit discussion).
Repository baseline: main 38b4be1 (clean at inspection).


## Final Local Implementation Decisions (2026-11-10)

Status: Accepted for local implementation by the project owner. This section and ADR-0013 take precedence over earlier Proposed/Open wording in this document. Historical discussion remains for context. No committee, further plan sign-off or approval workshop is required to start implementation.

### Fixed delivery scope

Implement backend first on the existing PR branch in commits separate from planning. Preserve existing application API contracts. Deliver amount-based pricing policies, annuity previews, final-term approval, one full cash payout, initial schedules, loan reads, and scheduled repayments. Frontend follows the documented three-view Loans workspace. Initial/restructure version types are supported in the model, but only initial issuance is executable; restructuring, penalties, write-offs and arbitrary posted-entry edits are excluded.

### Pricing and calculation contract

- One regular loan product; fixed annuity. Monthly nominal rate = annual_rate_bps / 10000 / 12. Store the nominal convention; do not label this input as an effective annual yield.
- Supervisor configures immutable loan_interest_policies: required base_rate_bps, complete bands, effective_date, reason and expected_latest_id. Rate bounds are 300..10000 bps (minimum 3% p.a., set by the owner) (technical ceiling, not a selected commercial tariff). Principal amounts obey config.MaxAmount; tenor is 1..120 months.
- Policy dates must be tomorrow or later in Asia/Jakarta. Select latest effective_date not after approval date, then greatest ID for that date; pending versions may be superseded with history retained.
- Inclusive lower/exclusive upper amount bands apply one rate to the entire approved principal; explicit fallback base handles gaps. Policy writes serialize and require idempotency.
- Actual tariffs and thresholds are operational configuration entered by the owner as Supervisor, not hardcoded product decisions. No bank rates or invented default policy are seeded. Missing effective configuration blocks priced approval/payout; it does not block implementation.
- Save policy ID/rate and approved amount/tenor with approval_revision. Subsequent policy updates or declining outstanding principal do not reprice the approved contract.
- New approvals must bind a reviewed preview policy ID/rate and proposed amount/tenor. Legacy approvals remain readable but cannot pay out without an explicit authorized pricing-completion/reapproval command. Extend the existing decision route to permit this only for an approved, undisbursed legacy case lacking complete pricing; preserve normal submitted-to-approved/rejected semantics and same-key replay.
- Reapproval of a fully priced approved application is not included in the initial delivery. Changed terms require a new application; no silent modification or cancellation of an existing approval.
- No automatic approval expiry in the initial local scope; payout always rechecks recipient, evidence, master eligibility and authority evidence.
- Calculate with exact rational arithmetic where practical, round monthly payment and interest half-up to integer IDR, principal = rounded payment - rounded interest, and reconcile final principal exactly. Reject pathological combinations producing nonpositive principal or invalid rows rather than hiding a write-off.
- Generate each monthly due date from the original payout day plus installment ordinal, clamping absent days to month-end. First installment is one month later. No holiday shifting or daily stub-interest adjustment in this monthly local model.
- Actual outstanding principal = paid-out principal minus posted principal allocations (and any future authorized reversals). It is separate from contractual scheduled remaining principal.

### Payout conditions and concurrency

For the local backend the condition-evidence object contains required recipient_verified, agreement_signed and conditions_verified booleans, all true, plus a required verification_note of 5..500 trimmed characters. These are authorized Teller attestations persisted with actor/time and payout audit. The backend validates the existing approved contract and prerequisites independently; it never treats these booleans as substitute approval authority. File upload/document management is outside scope.

Payout preview uses the current server Jakarta date and approved snapshot. Confirm using the expected approval revision, policy/rate/principal/tenor and payout date listed in the API inventory. Under locks, recompute and reject any mismatch. One application can yield one disbursed loan, enforced by database uniqueness across all record states. Cash handover follows original-reference lookup/replay on an uncertain response.

Approval and payment revisions are monotonic bigint values starting at 1 for a completed approval/new loan respectively. Advance payment_revision on each successful financial/state mutation. Clients cannot set revision counters. Use the existing runner for same-key replay before applying changed-state eligibility checks.

### Repayment contract

Apply a receipt to installment_number ascending in the current schedule, unpaid interest then unpaid principal within each installment. Partial receipts are supported and one receipt may cover multiple installments, including future scheduled installments; the preview explicitly shows allocations before confirmation.

Reject nonpositive receipts and amounts greater than total remaining scheduled principal plus interest. There is no automatic early-settlement interest rebate, fee or penalty in this local implementation; paying future installments pays their contractual scheduled interest. A special early-settlement/waiver workflow remains excluded.

A loan becomes paid_off only when all scheduled principal and interest are satisfied. Actual principal may reach zero before all contractual interest is paid in edge cases; do not infer full closure from principal alone. Repayment preview is read-only, and confirmation rejects stale payment revision/current version rather than silently reallocating.

### Final local journal model

Use one centralized cooperative ledger with servicing-KCP cash dimensions and the loan's responsible KCP retained on the loan. Separately balanced KCP books and inter-KCP clearing are excluded from this local delivery.

| Event | Debit | Credit |
| --- | --- | --- |
| Full cash disbursement | loan_principal_receivable: principal | cash at servicing KCP: principal |
| Cash repayment | cash at servicing KCP: full receipt | loan_principal_receivable: principal allocated; loan_interest_income: interest allocated |

Recognize loan interest income at receipt for this limited local model; schedules are contractual calculations and create no daily GL accrual. A repayment with both components has three journal lines; omit zero components and reconcile every line to its allocations. This is a local product/accounting decision, not a bank reporting implementation.

Loan cash flows do not change member savings. Update transaction kinds and financial projections explicitly. Savings history must exclude loan postings; KCP cash includes all real cash flows. Preserve loan/member/actor references without fabricating savings before/after snapshots for loan movements.

Global invariants after introducing loans:
- Every financial transaction balances; expected account/component lines match its kind.
- Loan principal receivable equals aggregate actual principal outstanding.
- Loan interest income equals posted interest allocations.
- Total cash + loan principal receivable + savings interest expense = member savings liability + loan interest income, for the restricted set of supported events.
- Existing savings-interest daily eligibility and monthly reconciliation remain unchanged.

### Audit and migration decisions

Keep all seven audit columns on every table. New staff rows use server identity/timestamps; legacy unknown creators and existing constrained system actors remain nullable without invented users. Enforce existing system-actor rules and audited command writes; do not retrofit false historical attribution.

Operational master/draft eligibility filters active/not-deleted records. Auth checks active sessions, users and KCPs; revoke inactive sessions. Financial history, immutable policy versions, replay, job-completion and technical bookkeeping reads retain their documented exceptions. No deletion API for posted financial facts. NIK/account uniqueness persists across deletion.

Use new ordered migrations; existing migrations remain immutable. For the application-shaped loans table split, use a documented local maintenance rollout: verified backup, stop old API writers, migrate, start compatible API. Preserve application IDs/decisions/replay and validate an isolated populated restore first. This is not a rolling schema rename under old binaries.

All endpoint paths, guards, bounded pagination and error semantics in the API inventory are accepted as the local implementation contract. Error assertions and conflict handling must be tested. Retain the existing approval limit fail-closed when not configured; the configuration value is not silently raised by this plan.

### Execution and verification

Implement/test backend in focused commits after this planning commit. Prove calculator, policy resolution, audit retrofit, legacy API compatibility, transactional payout/repayment, concurrency/replay and restored-data reconciliation. Rebuild/restart affected containers after runtime changes. Add a backend API/runbook/evidence update before completing implementation.

Remaining operational inputs are the owner's configured tariffs, policy effective date and approval limit; they are required configuration, not additional design-approval gates. Excluded workflows can be designed later without blocking this local backend scope.


## Memory Context

- No project `*-memory-system` or project-owned `AGENTS.md` was found.
- Read the roadmap, savings-interest tech plan, ADR index and lending ADR, open questions, accounting/business-process docs, implementation notes, acceptance criteria, and current loan module.
- Savings-interest implementation and ADR-0012 are complete. Local technical reconciliation passed and the project owner accepted the local result on 2026-11-09; live-money operational readiness remains open.
- Loan intake and Supervisor approval exist. Approval changes workflow status only; it creates no loan balance or payout.
- Installments may be paid at any KCP while the responsible KCP remains unchanged.

## Phase 2 Verification (2026-11-09)

- Docker API, web, and database containers were healthy; scheduler enable flag was true.
- Policy version 1 is effective 2026-10-24 with the accepted 3%, 4%, 4.5%, and 5% bands.
- Daily closing markers and 15 account rows per day are complete from 2026-10-24 through 2026-11-08 (16 dates).
- October has 15 monthly credits totaling IDR 106,522 across 120 eligible account-days.
- Monthly arithmetic, transaction links, journals, actor/snapshot attribution, and whole-ledger invariants all reconciled with zero mismatches. Interest added no cash journal lines; total cash remained IDR 111,975,000.
- Aggregate identity: IDR 843,685,000 deposits - IDR 731,710,000 withdrawals + IDR 106,522 interest = IDR 112,081,522 total savings. Journal debits and credits each total IDR 1,575,501,522.
- Existing isolated restore evidence remains in docs/testing/savings-interest.tdd.md; restore was not repeated during this read-only check. The owner accepted the local Phase 2 result on 2026-11-09.

## Requirement

1. **Phase 2 acceptance:** technical reconciliation and local owner acceptance are complete; operational readiness for live funds remains a separate gate.
2. **Phase 3 lending lifecycle:** extend the application/decision flow to support authorized disbursement and basic in-system repayment, with atomic financial posting and cross-KCP servicing.

Non-goals: new savings-interest features, configurable loan products, member self-service, automated collections, approval workflow engine, partial/staged disbursement, rescheduling, write-off, penalties, tax handling, or a general-ledger/worker redesign.

## Backend Scope

### Phase 2 acceptance

No application implementation is planned unless read-only checks find a defect; any defect requires a separate scoped fix. Do not manually alter financial data to make checks pass.

- Read interest status and completed business-date markers from the local deployment.
- Confirm the effective policy and daily calculations cover eligible dates beginning 2026-10-24.
- Confirm October settlement is complete after its due time (2026-11-01, 00:05 Asia/Jakarta). Reconcile each account's exact daily numerator total, rounded credit, balance snapshots, and journal.
- Confirm no teller-cash movement, duplicate account/month credit, or credit with missing eligible dates.
- Record scheduler lag/failure status; if local data is acceptance evidence, compare an isolated restore/re-run.
- Record owner acceptance separately from technical reconciliation. If settlement is absent or reconciliation differs, stop acceptance and investigate before manual recovery or real-money use.

### Phase 3 lending lifecycle

Keep the feature in `backend/internal/loan` and the current PostgreSQL transaction model. Reuse Gin, staff auth, durable command replay, audit, and journal patterns. Keep `cmd/api/main.go` limited to wiring/lifecycle.

- Add one disbursement action for an approved application after required conditions are recorded and verified.
- Create a loan lifecycle record and post cash disbursement at the servicing KCP atomically.
- Record repayments at any KCP while retaining the loan's responsible KCP and recording servicing KCP separately.
- Atomically update loan state/balances with transaction, balanced journal, audit, and idempotent command result.
- Enforce roles, case scope, and state transitions in the backend. Same-key retries return the original result; changed payload conflicts.
- Add focused loan status/payment-history projections only as needed by staff screens.
- Explicitly block unsupported cases. Do not invent pricing, allocation, fees, arrears, or accounting rules.

## Codebase Context

- Existing loan module: `backend/internal/loan/{domain.go,service.go,postgres_store.go,http.go}`.
- Current `loans` rows store application facts and decision fields; statuses are `submitted`, `approved`, `rejected`, and product is constrained to `regular`.
- Cash posting hardcodes savings deposit/withdrawal effects. Reusing it for loans would incorrectly mutate savings. Use a loan-owned posting path or a small reviewed shared primitive.
- Financial posting uses a caller-owned PostgreSQL transaction, double-entry journal, audit, and durable replay.
- `docs/accounting.md` leaves loan mappings, interest/payment allocation, KCP balancing, reversals, and period rules open. ADR-0005 remains Proposed.
- Schema changes require a new forward migration. Do not edit baseline schema or applied migrations.

## API Design

Final routes and payloads depend on approved product rules. Preserve staff session/role guards, strict decoding, same-origin protection, safe errors, and idempotency conventions. Expected command boundaries (route names illustrative): disburse one approved application after conditions; record a repayment using amount and authenticated servicing KCP; read loan status and bounded payment history.

Reject non-approved/incomplete applications, duplicate disbursement, invalid or unsupported repayment, and requests outside staff scope. Never accept actor, KCP, business timestamp, or accounting account values from clients.

These are additive endpoints/projections coordinated with the frontend. Preserve existing application/decision contracts. Apply `api-contract-review` before implementation; define pagination if history exceeds existing bounds.

## Decisions Required Before Phase 3 Implementation

Resolve the following with product/accounting owners and prepare worked examples:

- **OQ-009:** product pricing (annuity and amount-band policies selected; monthly conversion, rounding and initial allocation rules recorded below). Actual policy tariffs and exceptional servicing/accounting rules remain open.
- **OQ-011:** disbursement evidence/conditions, single versus staged payout, approval expiry, and payout KCP.
- **OQ-012:** supported late/partial/early payment, settlement, arrears, rescheduling, write-off, and closure. Explicitly block unsupported cases.
- **OQ-006/OQ-007:** chart of accounts, loan receivable/cash mappings, and central versus KCP-balanced books/inter-KCP clearing.
- **OQ-003/OQ-010:** responsible-KCP reassignment after payout and authority limits/risk for final terms.
- Reversal/correction authorization, business dates, closed-period behavior, and cash receipt/uncertain-payment procedure.

Do not reuse savings-interest rates or infer loan rules from them. Disbursement-only is not a complete lending pilot; in-system repayment and reconciliation (or an owner-approved reconciled external process) are required.

## Proposed Workflow

```mermaid
flowchart TD
  A[Approved application] --> B{Conditions complete?}
  B -->|No| C[Keep disbursement blocked]
  C --> B
  B -->|Yes| D[Authorized Teller disburses at a KCP]
  D --> E[Atomic loan, cash, journal, audit and replay commit]
  E --> F[Active loan under responsible KCP]
  F --> G[Member pays at any KCP]
  G --> H{Payment valid under approved allocation?}
  H -->|No| I[Reject without posting]
  H -->|Yes| J[Atomic repayment, journal, audit and replay commit]
  J --> K[Updated balance and loan status]
  K --> F
```

This outline is not authorization to pay a loan. Exact transitions and journal entries depend on approved rules.

## Implementation Plan

1. **Complete (2026-11-09):** verified status, completed dates, October settlement, account-level arithmetic and aggregate ledger reconciliation read-only. No rates or historical data were changed.
2. Resolve Phase 3 product, accounting, authority, disbursement-condition, and repayment decisions above; revise the plan if the agreed scope differs.
3. Draft and obtain acceptance for a new ADR covering loan product, posting model, KCP dimensions, and supported exceptions. Keep ADR-0005 historical and supersede it only if the new decision replaces it.
4. Add acceptance examples and forward-migration design for lifecycle, disbursement, repayment, journals, and state/uniqueness constraints. Preserve existing applications and APIs.
5. Implement allocation/calculation and state rules with tests first; add authorized idempotent commands and atomic persistence in `internal/loan`.
6. Add reconciliation for servicing-KCP cash, loan receivable, approved principal/interest allocation, and remaining balance. Verify populated restore/replay in an isolated database.
7. Coordinate staff frontend implementation using the Frontend Technical Plan section below; preserve EN/ID, light/dark, and role-based actions.
8. Release only after business/accounting acceptance, backend/frontend checks, reconciliation, restore rehearsal, and operating procedures are complete.

## Test Plan

- Phase 2 read-only checks: policy/date coverage, eligible account count, exact arithmetic/rounding, October account/month uniqueness, balanced journals, balance snapshots, unchanged cash, safe retry/restart.
- Phase 3 unit tests: approved state/conditions, limits, allocation, schedule boundaries, approved late/early/partial cases, payoff/closure, overflow/rounding.
- PostgreSQL integration: atomic disbursement/repayment, same-key replay, changed-payload conflict, concurrent duplicate payout/repayments, overpayment handling, journal/audit rollback, cross-KCP servicing with stable responsible KCP, permissions, transition races.
- Reconciliation: outstanding principal equals payouts less principal repayments/authorized reversals; cash matches approved loan accounts; interest/fees match policy; savings do not change.
- Migration/recovery: upgrade populated Phase 1/2 DB, restore isolated DB, compare schema/data/accounting digests, replay without duplicate postings.
- After implementation: `cd backend && go test -race ./...`, `go vet ./...`, Go lint, and frontend lint/typecheck/tests/build. Do not run tests during this planning-only change.

## Risks & Mitigations

- Unapproved financial policy: block implementation/posting until written decisions and expected journal examples are accepted.
- Cross-KCP imbalance: distinguish servicing cash from responsible KCP and resolve central-versus-branch accounting before rollout.
- Duplicate movement/races: row locks, uniqueness, caller-owned transaction, durable idempotency, and concurrency tests.
- Compatibility: forward-only migration/additive API; preserve application records and contracts.
- Unsupported repayment exception: reject explicitly; never silently allocate fees, interest, or arrears to principal.
- Recovery: verify backups; never delete/rewrite postings; correct only through an approved reversal policy.

## Open Questions

Phase 3 decisions above remain unresolved and block financial implementation. Phase 2 local technical reconciliation and owner acceptance are complete; live-money operational readiness remains open.

## Implementation Handoff

- This update changes only this technical plan; no implementation, schema, tests, configuration, roadmap, or ADR is changed by the 2026-11-10 planning revision.
- For implementation use `tdd-workflow`, `coding-standard`, `go-patterns`, and `go-lint-workflow`; keep changes scoped.
- Apply `error-handling`, `api-contract-review`, `observability-check`, `database-migrations`, and `postgres-patterns` at relevant boundaries.
- Do not implement lending financial posting until open product/accounting decisions and a proposed ADR are accepted.

## Phase 3 Discussion Record (2026-11-10)

This section records the owner's latest design direction and replaces conflicting earlier assumptions in this plan. It is a planning update only.

### Recorded decisions and proposal boundaries

- Use the loan-specific prefix `loan_*`, rather than `lender_*`: the cooperative is the lender; the member is the borrower.
- Separate applications from disbursed loans. Application intake and approval already exist in Phase 1; Phase 3 adds disbursement, schedules, repayments, allocation, and outstanding-principal tracking.
- Keep one responsible KCP on the loan; receiving, disbursing, and repayment-servicing KCPs are distinct facts. Repayments are accepted at any KCP.
- Use versioned schedules with `version_number` and boolean `is_current_version`. Remove the earlier proposed `supersedes_version_id`.
- Retain old versions, installments, and payment allocations when a new version becomes current.
- Record annuity as the selected calculation method. No loan interest rate has been selected; savings rates are unrelated.
- Record the universal audit-column standard and operational soft-delete predicate below. Financial history exceptions were accepted in discussion.
- The following six-table layout and individual column details are a proposed implementation design, not an already applied schema.
- The first delivery creates the initial schedule version. Actual restructuring remains deferred until carryover and approval rules are agreed; version support alone does not enable a restructuring command.

### Common audit columns on every table

The owner requested these columns on all tables, including the six lending tables. Extending existing tables requires a separately sequenced audit retrofit within this plan.

| Column | PostgreSQL type | Meaning |
| --- | --- | --- |
| `is_active` | boolean NOT NULL DEFAULT true | Record availability; separate from business lifecycle status |
| `created_by` | bigint | Creator identity; FK/nullable rules depend on truthful historical/system attribution |
| `created_at` | timestamptz NOT NULL | Creation time assigned by the database |
| `updated_by` | bigint NULL | Last modifying actor |
| `updated_at` | timestamptz NULL | Last modification time |
| `deleted_by` | bigint NULL | Soft-deleting actor |
| `deleted_at` | timestamptz NULL | Soft-deletion time |

For new staff commands, creator/updater/deleter identity comes from the authenticated actor, never client input. Use database timestamps. Update audit fields atomically with the domain change and audit event; a retry does not rewrite the original event.

Use a constraint requiring `deleted_by` and `deleted_at` to be both present or both absent, and require deleted records to have `is_active = false`. A merely inactive record may have no deletion timestamp. Update-field pairing and actor FK rules must be finalized consistently before DDL.

Existing rows may lack a known creator. Preserve known timestamps/actors; retain documented unknown or system attribution instead of inventing a human creator. Existing interest transactions/system audit already allow a constrained NULL actor. Decide how audit actor columns represent system jobs and legacy unknowns before enforcing NOT NULL or a system identity migration.

### Query and deletion policy

Operational master/draft eligibility queries use:

```sql
WHERE is_active = true
  AND deleted_at IS NULL
```

Apply the predicate explicitly to the relevant table aliases, including command eligibility checks. Deactivation and posting must serialize on the affected domain row to avoid checking eligibility before a concurrent deactivation.

The agreed financial exception is mandatory:
- Posted transactions, journals, disbursements, repayments, allocations, accepted schedule versions, and historical calculation/policy records remain available to audit and reconciliation. Corrections use authorized linked reversal entries.
- Do not soft-delete posted money or historical financial facts. Retain audit columns even where deletion is prohibited.
- Historical joins must still resolve an inactive/deleted member, user, or KCP; do not silently drop money rows through filtered INNER JOINs.
- Audit logs, idempotency/replay lookup, migration bookkeeping, and background-job completeness queries require explicit treatment. Hiding replay or completed-job records can duplicate postings. Do not introduce a generic filter that silently applies to these queries.
- A non-current schedule remains `is_active = true`, `deleted_at = NULL`. Only `is_current_version` changes.
- Preserve cooperative-wide NIK and savings-account uniqueness across inactive/deleted members. Do not replace these with partial active-only uniqueness.
- Member deactivation must not silently change savings-interest eligibility; confirm such behavior separately before changing the existing calculation.
- Master/draft soft deletion requires authorization and domain checks. Do not use cascading financial deletes. Unsupported cancellation, restoration, and identity-correction flows remain blocked until defined.

### Proposed lending columns

All tables below also contain the seven shared audit columns. Monetary values are whole-IDR bigint, never binary floating point. Lifecycle strings and numeric policy limits remain subject to the final product decision.

**1. `loan_applications`: requests and decisions**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Application identity |
| `application_number` | varchar UNIQUE | Staff-facing reference |
| `member_id` | bigint FK | Applicant |
| `receiving_branch_id` | integer FK | Intake KCP |
| `responsible_branch_id` | integer FK | Case owner |
| `product_code` | varchar | One initial product |
| `requested_amount` | bigint | Requested principal |
| `requested_tenor_months` | integer | Requested tenor |
| `purpose` | text | Purpose |
| `risk` | varchar | Risk category |
| `status` | varchar | Proposed submitted/approved/rejected/cancelled lifecycle |
| `approved_amount` | bigint NULL | Approved principal |
| `approved_tenor_months` | integer NULL | Approved tenor |
| `decision_by` | bigint FK NULL | Deciding Supervisor |
| `decision_at` | timestamptz NULL | Decision time |
| `decision_reason` | text NULL | Decision explanation |
| `authority_limit` | bigint NULL | Preserve existing recorded approval authority |

Preserve existing approval facts. Approval binds the final principal, tenor, dated loan-interest-policy ID and resolved rate snapshot before payout; changing material terms requires reapproval. See the Amount-Based Loan Pricing Decision below.

**2. `loans`: disbursed contracts**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Loan identity |
| `loan_number` | varchar UNIQUE | Staff-facing loan number |
| `application_id` | bigint FK UNIQUE | Original application; proposed one full payout |
| `member_id` | bigint FK | Borrower |
| `responsible_branch_id` | integer FK | Loan owner KCP |
| `disbursing_branch_id` | integer FK | Payout KCP |
| `principal_amount` | bigint | Original disbursed principal |
| `disbursement_transaction_id` | text FK UNIQUE | Financial posting reference |
| `disbursed_at` | timestamptz | Actual payout time |
| `status` | varchar | Proposed active/paid_off lifecycle |
| `closed_at` | timestamptz NULL | Loan completion time |

Create the loan, initial version/installments, disbursement, journal, audit, and replay result atomically. Version rows carry immutable pricing/tenor snapshots; the original version preserves original terms. Actual outstanding principal is reconciled from posted disbursements, principal allocations, and approved correcting entries. Add a cached projection only if needed, updated in the same locked transaction.

**3. `loan_schedule_versions`: terms and schedule revisions**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Version identity |
| `loan_id` | bigint FK | Loan |
| `version_number` | integer | Sequence within loan |
| `is_current_version` | boolean NOT NULL DEFAULT false | Current contractual schedule |
| `version_type` | varchar | Why the version exists: initial or restructure; separate from version number |
| `effective_date` | date | Start of contractual effect |
| `starting_principal` | bigint | Principal scheduled by this version |
| `interest_rate_bps` | integer | Annual rate; 300 represents 3%, illustrative only |
| `interest_method` | varchar | Annuity for the initial implementation |
| `tenor_months` | integer | Version tenor |
| `first_due_date` | date | First installment due date |
| `status` | varchar | Proposed draft/active/superseded approval lifecycle |
| `reason` | text | Initial issue or revision explanation |
| `approved_by` | bigint FK NULL | Version approver |
| `approved_at` | timestamptz NULL | Approval time |

`status` tracks approval/lifecycle, `is_current_version` tracks schedule selection, and `is_active` tracks record availability; enforce valid combinations. Do not derive current version solely from maximum version number.

**4. `loan_schedules`: installments in one version**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Installment identity |
| `schedule_version_id` | bigint FK | Parent version |
| `installment_number` | integer | Sequence starting at 1 |
| `due_date` | date | Due date |
| `principal_due` | bigint | Scheduled principal |
| `interest_due` | bigint | Scheduled interest |

Total installment is principal_due + interest_due. Derive paid/unpaid amounts from allocations and approved reversals rather than duplicating mutable totals. One installment per tenor month assumes monthly payments; frequency, first/last partial periods, and month-end handling remain to be confirmed.

**5. `loan_repayments`: actual receipts**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Receipt identity |
| `loan_id` | bigint FK | Loan paid |
| `repayment_number` | varchar UNIQUE | Receipt number |
| `transaction_id` | text FK UNIQUE | Journal-linked financial transaction |
| `servicing_branch_id` | integer FK | Receiving KCP |
| `amount` | bigint | Actual money received |
| `paid_at` | timestamptz | Actual receipt time |
| `note` | text NULL | Optional explanation |

The receiving Teller is captured by created_by and the financial actor. The final supported payment channels and correction representation require approval; no payment channel engine is planned.

**6. `loan_repayment_allocations`: receipts applied to installments**

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | bigint PK | Allocation identity |
| `repayment_id` | bigint FK | Actual receipt |
| `schedule_id` | bigint FK | Installment paid |
| `principal_amount` | bigint | Principal component paid |
| `interest_amount` | bigint | Interest component paid |

This permits one receipt to cover multiple installments and one installment to receive partial payments. Exact support and allocation order remain product decisions; the schema does not authorize arbitrary partial/advance payments.

Enforce same-loan membership for repayment and schedule (through the schedule version), nonnegative components with positive total, and aggregate allocations equal to the receipt. Use explicit service validation under a loan lock plus appropriate database constraints; row CHECK constraints alone cannot enforce cross-row sums. Consider unique (repayment_id, schedule_id) for one combined allocation per pair.

### Schedule version types

The owner requested an explicit category describing why a version was created. Use version_type (renamed from the earlier change_type proposal), independently of version_number and is_current_version.

| version_type | Meaning | Scope |
| --- | --- | --- |
| initial | First contractual schedule created at disbursement | Initial Phase 3 delivery |
| restructure | Replacement schedule following approved restructuring | Planned; financial carryover and approval rules must be agreed before enabling |

Version 1 is initial. Later versions are categorized by the actual cause, not their ordinal: versions 2 and 3 may both be restructure. Keep reason as required human-readable context, with the approving actor and timestamp retained separately.

Potential future categories include rate_change for an approved contractual rate revision and schedule_correction for an approved correction to future unpaid installments. These are examples only, not enabled types or approved workflows. A routine rate reset already covered by a contract needs its own policy before deciding whether it creates a version. Corrections must never rewrite posted payments or historical allocations.

Use a bounded database CHECK/service allowlist for supported types, extended through a forward migration when a new workflow is approved. Do not add a configurable version-type master table for this MVP. Validate initial as the first version and require the appropriate approval for any replacement; version_type does not itself authorize activation.

### Current-version activation and restructuring

Required uniqueness:

```sql
CREATE UNIQUE INDEX loan_schedule_version_number
ON loan_schedule_versions (loan_id, version_number);

CREATE UNIQUE INDEX loan_schedule_one_current_version
ON loan_schedule_versions (loan_id)
WHERE is_current_version = true;

CREATE UNIQUE INDEX loan_schedule_installment_number
ON loan_schedules (schedule_version_id, installment_number);
```

The current-version uniqueness predicate intentionally excludes is_active/deleted_at. A soft-delete flag must not permit a second current financial version. The index guarantees at most one; service transaction rules ensure an active loan has exactly one current version.

Under a lock on the loan row, validate the approved change, prepare the complete new version/installments with current=false, clear the old current flag, activate the new version, and commit atomically with audit/replay. On failure, roll back all changes. Repayment allocation and version activation take the same loan lock, with one documented ordering for shared cash/KCP locks.

Do not activate a future-effective version immediately; retain the old version until the effective date. Automated future activation is not included without a separate decision. Reject unsupported scheduling rather than silently switching early.

Never edit accepted installment amounts, erase old schedules, or move historical allocation foreign keys onto the new version. Carryover must explicitly reconcile outstanding principal, unpaid/accrued interest, arrears, and rounding adjustments. Restructuring does not automatically capitalize unpaid interest or reset a borrower's debt.


### Application through disbursement flow (proposed)

The member is the borrower; the cooperative is the lender. Phase 1 already supports staff-assisted application and approval. This describes the proposed Phase 3 extension, not an existing payout feature. The initial proposal is one full cash disbursement; final product and payout conditions remain subject to the open decisions.

```mermaid
flowchart TD
    A["Member requests a loan at any KCP"] --> B["Teller resolves existing member and checks eligibility"]
    B --> C{"Eligible and information complete?"}
    C -->|No| D["Correct information or stop without submitting"]
    C -->|Yes| E["Save loan_application: submitted; record receiving and responsible KCP"]
    E --> F["Supervisor reviews amount, tenor, annuity pricing and risk"]
    F --> G{"Authority and final terms valid?"}
    G -->|No| H["Block approval until authority or terms are resolved"]
    G -->|Yes| I{"Supervisor decision"}
    I -->|Reject| J["Record rejected decision, actor and reason"]
    I -->|Approve| K["Record approved terms and decision; no money moves"]
    K --> L["Teller verifies recipient, required evidence and payout conditions"]
    L --> M{"Conditions satisfied?"}
    M -->|No| N["Keep payout blocked"]
    N --> L
    M -->|Yes| O["Review payout amount, KCP cash and schedule preview"]
    O --> P["Confirm payout with an idempotency key"]
    P --> Q["Lock application; recheck approval, terms, eligibility, cash and duplicate payout"]
    Q --> R{"Checks still pass?"}
    R -->|No| S["Reject with no partial financial posting"]
    R -->|Yes| T["Atomic commit: loan + initial version + installments + payout + journal + audit + replay"]
    T --> U["Loan active; initial version current; return payout reference and receipt"]
```

| Step | Role | Main stored result |
| --- | --- | --- |
| Intake | Teller | Application, member, receiving KCP, responsible KCP, requested amount/tenor/purpose/risk |
| Analysis and decision | Supervisor within authority | Approved final terms or rejected decision, reason and actor |
| Payout preparation | Teller | Verification of approved conditions and recipient; final evidence fields remain to be defined |
| Disbursement | Authorized Teller | Active loan, version 1 of type initial/current, all installments, financial posting, audit and replay result |

Approved final terms must include principal, annual loan rate, annuity method, tenor, and the agreed due-date/rounding rules. Approval binds these terms; changing material terms requires reapproval. The storage representation and exact product values are still open. A schedule preview before payout does not create principal or a current financial version.

At successful payout, create the loan and its entire initial schedule in the same database transaction as the financial posting:
- loans references the approved application and actual disbursing KCP.
- loan_schedule_versions has version_number=1, version_type=initial, is_current_version=true, and a snapshot of the approved terms.
- loan_schedules contains the agreed number of installments; their principal total equals the actual disbursed principal.
- The proposed journal is Dr loan principal receivable / Cr cash at the disbursing KCP, pending approval of the account mappings and branch accounting.
- The member's savings balance is unaffected for the proposed cash payout.
- The application retains its approval evidence. The linked unique loan/application relation proves it has been disbursed; API representation of this fact must preserve existing application contracts.

The application row lock and unique loans.application_id prevent concurrent double disbursement. Cash validation and posting use the same servicing-KCP cash lock as other cash writers, with a documented consistent lock order. Commit all loan/schedule/financial/audit/replay records together or roll back all of them. Repeating the same command returns the original reference; a new key cannot bypass the unique application payout constraint.

Physical cash handover is governed by a separate approved Teller SOP; a database commit cannot atomically hand over banknotes. If the response is uncertain, retrieve/replay the original result before any new payout or cash handover. Reprinting a receipt must not post another transaction.

Receiving KCP, responsible KCP, and disbursing KCP remain distinct. Operational eligibility checks use active/not-deleted masters, while historic approval/financial attribution remains readable. This flow adds no general-purpose workflow engine or separate analysis role.

### Annuity calculation

Annuity was selected for the planned loan method. For equal regular periods and an unchanged periodic rate:

```text
P = principal being scheduled
r = interest rate per payment period
n = number of installments

A = P * r / (1 - (1 + r)^(-n))
interest[k] = opening_principal[k] * r
principal[k] = A - interest[k]
closing_principal[k] = opening_principal[k] - principal[k]
```

For r=0, use A=P/n to avoid division by zero. With timely regular payments, scheduled principal portions increase and interest portions decrease while the unrounded regular payment stays constant. Outstanding principal decreases. Fixed-principal effective payments instead have constant principal and declining totals; flat interest uses original principal. These alternatives were discussed but are not the selected direction.

Accepted for local implementation: fixed nominal annual pricing with monthly conversion r = annual_rate_bps / 10000 / 12 and monthly installments. The actual annual rates are selected from configured nominal bands. See the later pricing decision for accepted rounding/due-date/payment rules; savings Actual/365 does not apply to this loan convention.

Store contractual terms as a version snapshot. Calculate using exact rational or justified high-precision decimal arithmetic, never float64 money calculations. Define rounding mode/stage and bounds before implementation. Whole-IDR schedules may require a slightly different last payment: allocate the remaining principal to the final installment so sum(principal_due)=starting_principal and the closing scheduled principal is exactly zero. Do not silently write off principal, generate negative principal, or independently round components in a way that breaks installment totals.

The earlier 12-million/12-month/12%-p.a. example was illustrative, not an approved tariff. Add owner-approved examples with expected rounded rows before coding.

Contractual schedules and actual balances are distinct. Do not rewrite historical schedules based on payment timing. The treatment of overdue/early payments, additional interest, and payoff remains undecided.

### Migration and implementation sequence

1. Finalize remaining monetary/operating decisions and the corresponding ADR before loan financial implementation.
2. Inventory every existing table and classify queries as master eligibility, financial posting, immutable history, replay/job bookkeeping, or technical maintenance. Plan the all-table audit rollout explicitly, not as an implicit generic ORM scope.
3. Add audit fields through forward migrations. Preserve existing known created_by/created_at fields. Backfill known values only; document unknown/system attribution. Deploy compatible writers before enforcing stronger constraints. Assess table-lock time and batch large backfills; verify on an isolated populated restore.
4. Separate the current application-shaped loans table from disbursed loans using a coordinated forward migration and application deployment. Preserve application IDs, authority/decision history, audit/replay data, and existing /loans array/request/response contracts. Do not blindly rename the live table while old binaries still query it. Document maintenance/compatibility strategy before implementation.
5. Add new loan/version/installment/payment/allocation tables, FK/unique/check constraints, and financial posting account/kind support. Migrate no application to a disbursed loan without an actual authorized payout.
6. Implement the pure annuity calculator, initial schedule, and approved state transitions with focused tests. Keep Gin handlers, service rules, and pgx persistence in the existing loan module.
7. Implement idempotent disbursement and repayment with safe errors, sanitized logs, transaction rollback, lock ordering, and cross-KCP cash/loan reconciliation. Extend financial history projections without altering savings balances.
8. Add current-version replacement only when restructuring carryover/approval/effective-date rules have been agreed. Until then, expose only initial version creation.
9. Preserve frontend EN/ID and light/dark when the backend feature is integrated. This update does not implement frontend screens.
10. Roll out only with reconciled backup/restore evidence and the explicitly approved operational scope. Do not roll back to an incompatible binary after new financial postings; prefer a forward fix.

### Additional validation and remaining decisions

Future validation must cover:
- Active/deleted master filtering, truthful legacy/system actors, authorization, deletion-field consistency, immutable financial history, inactive-parent joins, NIK/account non-reuse, and replay after deactivation without duplicate money.
- Zero-rate annuity, approved monetary examples, rate/tenor bounds, precision and monthly/final rounding, exact principal sum, no negative installment, month-end/leap-year dates, and schedule-versus-actual balance separation.
- Concurrent new versions, unique version/ordinal constraints, exactly one current schedule for active loans, activation rollback, repayment versus version-change races, and retained old allocations.
- Same-loan allocation validation, exact receipt/component totals, partial/advance/overpayment behavior only as approved, journal failure rollback, and immutable original payment references after correction.
- Populated migration/restore parity for current applications, savings interest, audit history, sessions, command replay, loans and installments. Confirm all prior savings/cash reconciliation still holds.

The subsequent Amount-Based Loan Pricing Decision records the latest accepted local rules and supersedes the earlier open calculation/servicing assumptions. Actual tariffs, accounting/operating controls and exceptional servicing decisions still need resolution.


## Amount-Based Loan Pricing Decision (2026-11-10)

This section supersedes earlier open assumptions about manually choosing a rate during approval. The owner selected the amount-band policy approach after reviewing bank/BPR examples. Published bank figures remain references, not an approved tariff for this cooperative: the examples include different products and flat/effective methods.

### Selected pricing behavior

- Supervisor maintains a separate loan interest policy with a global base rate and principal-amount bands, following the existing savings-policy versioning pattern.
- Choose one annual rate for the entire approved principal. Do not apply progressive slices and do not use the requested amount when the approved amount differs.
- The system resolves the policy valid at approval, selects the band, and binds the approved principal, tenor, policy ID and rate snapshot to that decision.
- Both Teller and Supervisor see the resolved rate as read-only during approval/payout; no per-application manual override in the initial implementation. A pricing-policy change is a separate authorized action.
- Changing approved principal or other material contractual terms invalidates the old preview and requires fresh approval and policy resolution.
- Copy the approved snapshot into the initial schedule version at disbursement. Do not silently reprice an already approved case from a newer policy at payout; approval-expiry rules remain a separate decision.
- The rate is fixed for that contractual version. Repayment reducing outstanding principal does not move the loan to a new pricing band.
- New policy versions affect subsequent approvals. Restructuring/rate changes require a separately approved workflow and do not rewrite historic terms.
- No initial bank rates, thresholds or savings rates are silently seeded. If no applicable configured policy exists, block financial approval/payout with a clear policy-required result.
- Empty/unmatched bands use the explicitly configured base rate. Reject overlapping/invalid bands. Use inclusive lower and exclusive upper bounds; max_amount=NULL denotes no upper bound.

### Additional table: loan_interest_policies

The six lending tables above remain, with this seventh table added. Use complete band configuration in one JSONB policy version, as in savings interest; no separate band master or general-purpose rule engine is needed.

| Column | Type | Meaning |
| --- | --- | --- |
| id | bigint PK | Immutable policy version |
| effective_date | date | Business date from which new approvals may use this version |
| base_rate_bps | integer | Explicit fallback annual nominal rate in basis points |
| bands | jsonb | Array of min_amount, max_amount (nullable), rate_bps |
| reason | text | Explanation for policy creation |
| shared audit columns | As defined above | Supervisor/system attribution and timestamps |

Rate integers require explicit supported bounds, amount bounds must fit the existing technical ceiling, and complete policy validation precedes persistence. No negative rate/amount, reversed/zero-width band, overlap, or ambiguous boundary is permitted.

Add loan_interest_policy_id and approved_interest_rate_bps to application approval storage. Add loan_interest_policy_id to schedule versions alongside the existing interest_rate_bps snapshot. Keep the policy FK and snapshot permanently readable for financial history; preserve legacy approvals without fabricating terms, and require pricing completion/reapproval before payout.

### Policy lifecycle and API/frontend integration

- Proposed policy editing convention: immutable versions with effective dates tomorrow or later in Asia/Jakarta, matching savings policy practice. A prior version remains resolvable for its historical approval dates.
- Use a deterministic dated version selection and serialize concurrent policy writes; include an expected latest version and durable idempotency for policy creation.
- Do not deactivate or soft-delete a used policy to rewrite history. Keep all seven audit columns, with financial-history query exceptions as already specified.
- Add Supervisor-only list/create policy endpoints and a staff-authorized preview/resolution projection. Final route/request/response contracts belong to backend implementation and API review.
- Resolve approval policy using database time and persist the snapshot atomically with the decision/audit/replay result. Guard against stale previews at approval; require review of changed terms rather than silently accepting them.
- Add a small Supervisor pricing settings panel within the existing Loans workspace, not another top-level menu. It shows global rate, nominal bands, effective dates and version history with EN/ID and light/dark support.
- Application review/annuity previews show approved principal, selected band, rate p.a., policy version and calculated payment. The backend remains authoritative.

### Previously accepted local calculation and servicing rules

The owner accepted these minimal local rules before refining the pricing source to amount bands. The amount-band decision replaces manual rate entry; the remaining accepted rules are retained:
- Fixed annuity, annual nominal rate divided by 12 for monthly interest.
- Whole-IDR half-up rounding with final-principal adjustment to reconcile the principal exactly.
- Monthly due dates, first installment one month after actual disbursement; clamp missing calendar days to month-end. Anchor each installment to the original disbursement day to avoid cumulative date drift.
- Repayment follows oldest installments first, interest then principal; partial payments are allowed.
- No penalties or extra late-interest calculation in this initial local scope; overdue scheduled amounts remain payable.
- Reject money exceeding the remaining supported scheduled payable amount. Early-settlement discounts/interest waivers and other special treatments remain unimplemented until separately approved.
- One full cash disbursement per approved application.
- Prepare the versioned data model but defer restructuring commands until carryover/approval rules are agreed.

Here annual nominal rate/12 is the contractual convention, not a claim that a bank's quoted effective annual rate has the same periodic conversion. Label/store the convention consistently and do not import flat or effective bank tariffs without an explicit product decision.

### Pricing validation and unresolved decisions

Add focused tests for each exact nominal boundary, whole-principal rate selection, fallback, malformed/overlapping bands, changing approved amount, future/dated policy selection, simultaneous policy writes, stale previews, approval snapshots, idempotent policy/approval commands, and unchanged rates after repayments or later policy changes.

The approved design requires configuration, but the actual nominal thresholds, annual rates and base rate for this cooperative are still unspecified. Keep payout fail-closed until an explicit policy is saved. Account mappings, cross-KCP accounting, payout evidence/approval expiry, system/legacy audit attribution, restructuring carryover and exceptional early-settlement treatments remain separate open decisions.

## Frontend Technical Plan - Phase 3 (2026-11-10)

Status: Proposed frontend design, explicitly requested by the owner. This extends the backend-focused plan above; it does not approve outstanding financial rules.

### Navigation and menu count

Retain the existing single Pinjaman / Loans sidebar entry. Add three internal views:

| View | Purpose | Staff actions |
| --- | --- | --- |
| Pengajuan / Applications | Submitted, approved and rejected applications; intake and decisions | Teller submits; authorized Supervisor reviews and decides |
| Pencairan / Disbursement | Approved applications awaiting payout | Teller verifies conditions, reviews approved terms and confirms payout |
| Daftar Pinjaman / Loan Accounts | Disbursed loans, active by default with access to paid-off history | Staff inspect details; Teller records eligible repayments |

This introduces no additional sidebar entry: Teller continues to see five main menus (Members, Cash Deposit, Cash Withdrawal, Loans, Transactions); Supervisor sees six including Savings Interest. Payment is an action in loan details, not an extra top-level menu. Do not create menus corresponding to database tables.

Application decision belongs in application details, not a separate approval menu. Schedule versions and installment schedules belong in loan details. No restructuring editor/action in the first delivery.

### Screen flow

```mermaid
flowchart TD
    Menu["Loans menu"] --> Applications["Applications view"]
    Menu --> Payouts["Disbursement view"]
    Menu --> Accounts["Loan Accounts view"]
    Applications --> New["Teller: submit application for an existing member"]
    Applications --> Review["Supervisor: inspect and approve/reject final terms"]
    Review -->|Approved| Payouts
    Payouts --> Verify["Teller: verify recipient, conditions and approved terms"]
    Verify --> Preview["Review server-generated annuity schedule"]
    Preview --> Confirm["Confirm payout; resolve uncertain responses using original request"]
    Confirm -->|Committed| Detail["Loan details and payout receipt"]
    Accounts --> Detail
    Detail --> Summary["Summary and actual outstanding principal"]
    Detail --> Schedule["Installment schedule"]
    Detail --> Payments["Payment history and authorized repayment action"]
    Detail --> Versions["Version history and read-only prior schedules"]
    Payments --> PaymentReview["Review server-generated allocation"]
    PaymentReview -->|Confirmed and committed| Receipt["Payment receipt and refreshed balances"]
```

### Applications view

- Reuse the existing application intake/decision behavior and /loans contract while the backend separates application and active-loan storage.
- Search by application number, member/account number, and agreed bounded filters/status. Use an exact account lookup for selection; do not depend exclusively on the current latest-100 member dropdown.
- Show application reference, member/account, requested/approved principal and tenor, receiving/responsible KCP, status and decision.
- Teller form: member, requested amount, tenor, purpose, responsible KCP, and risk according to existing permissions.
- Supervisor review: application facts, analysis/reason, applicable authority, final principal/rate/annuity/tenor/due-date terms, and authoritative schedule preview.
- Resolve the rate from the dated amount-band loan policy and show it read-only with its policy/version reference. Supervisor manages the policy separately; no per-application rate override is planned. Do not derive approval authority solely from the Supervisor role.
- Approval/rejection includes confirmation, reason and submitting state. Preserve entered values after recoverable failures; display field errors near their inputs.
- After approval, offer navigation to the payout case; do not enable payout solely from a browser status label.

### Disbursement view

- List approved applications that have no committed payout; support exact application/account lookup.
- Detail shows member identity appropriate to staff permissions, account/reference, responsible KCP, approved principal, annual loan rate, annuity method, tenor, first due date, receiving KCP and actual servicing KCP.
- Show required condition checks/evidence defined by the final policy. A browser checkbox alone cannot establish backend eligibility.
- Show server-authoritative cash availability, payout eligibility and blocked reasons. Recheck on submission.
- Display a read-only annuity preview: installment number, due date, opening principal, principal due, interest due, installment total and closing scheduled principal. Opening/closing values can be projections; they need not be extra persisted columns.
- Label the preview as a schedule, distinguish it from actual outstanding balances, and disclose any approved final-installment rounding adjustment.
- Confirmation identifies the member, payout amount, KCP and agreed terms. Disable repeated submission while pending.
- On success, open the active loan detail/receipt and refresh the affected payout queue and cash state.
- A committed payout cannot be submitted again from a new request key. On an uncertain response, retain the original request identity and show recovery/retry for that same command; do not invite another cash payout. Follow the agreed physical-cash SOP.
- Receipt includes loan/application references, member, amount, timestamp, Teller, servicing KCP and financial reference. Printing/reprinting reads the committed result.

### Loan details and repayments

Loan Accounts defaults to active loans, but historical/paid-off loans remain reachable. Provide these four detail sections:

| Detail section | Content |
| --- | --- |
| Ringkasan / Summary | Original disbursed principal, actual remaining principal, current terms/version, status and responsible KCP |
| Jadwal Angsuran / Installments | Current contractual schedule, due/allocated/unpaid components and derived payment state |
| Pembayaran / Payments | Actual receipts, principal/interest allocations, servicing KCP/Teller, timestamps and references |
| Riwayat Versi / Versions | Version number/type, effective date, current marker, reason, approver and read-only installment details |

- Keep actual remaining principal distinct from the scheduled remaining principal at a selected installment.
- Version 1 displays type initial and a current badge. Old versions remain readable when a future approved workflow replaces them; is_current_version=false is not deletion.
- Repayment starts from the selected loan: server determines allowed amount/allocation under the approved policy; Teller reviews before confirmation.
- Show per-installment principal/interest allocation and total. The client does not choose arbitrary allocation rules or recalculate posting amounts.
- Preserve request identity on ambiguous failures and refresh actual loan balance, schedule allocation and payment history after success.
- The data model supports multiple allocations, but partial/advance/late/overpayments and reversals stay disabled until their rules and corresponding backend actions exist.
- Payment UI is a later increment of this same Phase 3 plan, dependent on approved allocation rules; application-through-payout UI can be implemented earlier for local review. This sequencing does not make a payout-only flow a complete live lending service.

### Components and data ownership

Follow the current Next.js Client Component, api helper, preferences provider, shadcn/ui primitives and tweakcn theme. No new UI library, global state manager, or workflow engine is needed.

- Keep page.tsx as navigation/wiring for this feature. Add a small loan workspace component with focused intake/review, payout and loan-detail components when their behavior warrants separation.
- Prefer a small explicit view state for applications/disbursement/accounts and selected application/loan IDs. Do not store duplicate derived balances or permissions.
- Use the existing Button, Input, Card, Badge and appropriate accessible primitives. Keep one-off layout local; extract shared pagination only if actual reuse keeps it simpler.
- Fetch bounded list/detail/preview data through existing api conventions. Abort obsolete lookups and bind displayed results to the selected record/query; hide old member details immediately when lookup input changes.
- Server preview, approval and payout must agree on the same terms/version. Invalidate preview/confirmation when terms change, and handle stale-term conflicts by reloading/reviewing rather than silently accepting changed terms.
- Backend endpoints/response projections for eligibility, previews, loan detail, installments and allocations must be finalized with this UI. Route names remain unspecified until API design; preserve existing application callers.
- Add additive response fields/capabilities as needed; the backend remains authoritative for permissions, status, active/deleted eligibility, accounting and calculations.
- Proposed 20 rows per list/detail page, following the owner's existing table preference. Coordinate server pagination for new endpoints; do not silently present the first 100 rows as the full loan/application history. Pagination tokens/filters/count semantics must be agreed in the API contract.

### Roles, language, theme and accessibility

- Teller: submit applications, perform authorized payout and eligible payment commands.
- Supervisor: inspect/analyze, approve/reject within authority, and monitor. Do not grant Teller cash commands to Supervisor implicitly.
- Member remains a domain role; member self-service login/screens are not in this delivery.
- Translate all new labels, validation errors, status/version-type badges, confirmation text, loading/empty/error messages and receipts into EN/ID using the existing message dictionary.
- Show money with the existing locale formatter and dates in Asia/Jakarta. Display p.a. explicitly for annual loan rates.
- Use semantic theme tokens for light/dark; ensure schedules, dialogs, receipts, disabled controls and error states remain legible in both.
- Provide keyboard-reachable actions, input labels, table captions/column headers, visible focus, confirmation focus handling and status announcements. Show status in text, not color alone.
- Keep forms usable on narrow screens and wide schedule tables horizontally scrollable without compressing monetary columns.

### Frontend delivery and validation

1. Confirm navigation, role matrix, product/rate/rounding/due-date/allocation rules and required API projections with the backend plan.
2. Extend application intake/review using existing contract compatibility, including exact member lookup and approved-term preview.
3. Build payout queue, eligibility/detail/confirmation, replay recovery and receipt around the new backend payout command.
4. Build loan account detail, current schedule and read-only initial version.
5. After payment rules are accepted, add payment preview/confirmation/history and allocation display. Keep restructuring read-only until separately enabled.
6. Run repository frontend lint, typecheck, meaningful behavior tests and production build; verify EN/ID, both themes, keyboard operation, responsive schedules, stale lookup/preview, permission denial, duplicate clicks, ambiguous responses and refreshed balances/queues.
7. Rebuild and restart the affected Docker service after implementation changes, as requested by the owner. Verify health and homepage, then inspect the actual application flows. This documentation-only update has no runtime image changes.

No frontend code or tests are changed or executed by this planning update. Navigation and screen organization above are proposed; unresolved financial policies are still implementation blockers.

## Application Schedule Preview API (2026-11-10)

The application form must offer an authenticated, read-only-in-effect schedule preview before submitting an application.

### Endpoint and request

POST /api/loans/preview; permitted to Teller and Supervisor with the existing session, same-origin and strict request-body guards. POST is used for structured calculation inputs; it does not create domain or financial records and requires no financial Idempotency-Key.

Example request (illustrative amount/tenor/date, not an approved tariff):

```json
{
  "amount": 15000000,
  "tenor_months": 12,
  "expected_disbursement_date": "2026-11-15"
}
```

The client does not provide rate, policy ID, schedule amounts or actor/KCP. Resolve the currently effective policy using the database approval/preview business date in Asia/Jakarta and the requested principal. The expected payout date supplies the due-date anchor; it does not select a future policy. Validate positive integer IDR amount within technical/product/authority-independent preview bounds, supported monthly tenor, strict YYYY-MM-DD date, and no past payout date. Product date limits must be explicit in implementation. A preview does not grant approval authority.

### Response

Return 200 with:
- amount, tenor_months, interest_method=annuity, annual_rate_bps, policy_id, matched band or explicit base-rate fallback, expected_disbursement_date, time_zone=Asia/Jakarta, and calculated_at.
- regular_installment_amount, total_principal, total_interest, total_payable. The final installment may differ because of the accepted final-principal rounding adjustment.
- installments array: installment_number, due_date, opening_principal, principal_due, interest_due, total_due, closing_principal.
- Explicit estimate semantics so consumers do not present a preview as an approved contract or actual payment receipt.

All money fields are integer rupiah. Totals must equal the returned rows, sum(principal_due)=amount, and the final closing_principal must be zero. Use the same backend annuity/due-date/rounding calculator for preview and final schedule creation; no independent frontend calculation.

### Lifecycle, errors and frontend behavior

- Resolve policy and read its complete immutable configuration in one consistent operation. Missing eligible policy or disallowed calculation returns a safe 422; malformed/range/date input returns 400; missing session 401; forbidden role/origin 403; transient infrastructure failure 503 and unexpected failure sanitized 500, following existing fault/httpapi mapping.
- Preview is bounded by supported tenor and request size. Honor request cancellation and database timeouts; log safe operation/error context without raw member/payload data.
- Store no application, version, installment, financial posting, audit mutation or command-replay result for preview. Normal request logs remain permitted.
- Recalculate when amount, tenor or expected payout date changes; abort/ignore obsolete requests and invalidate previous preview/confirmation immediately.
- Render summary and full schedule with EN/ID, light/dark, loading/empty/error/retry states, and explicit estimate labeling.
- At approval, resolve policy again and bind the approved terms/snapshot. A stale preview cannot silently authorize changed pricing; use an expected-policy/terms reference in the future approval contract and return 409 for stale terms that require review.
- At disbursement, use the approved rate snapshot and actual payout date, revalidate conditions, and generate/persist the final initial schedule atomically with payout. If the date differs from the estimate, require Teller review of the refreshed final preview before confirming; its exact confirmation-token/expected-date binding must be specified with the payout endpoint.
- Additive API: existing POST /api/loans and list/decision behavior remain compatible.

### Validation targets

Test role/origin/session guards, strict body decoding, unavailable policy, every nominal band boundary/fallback, policy changes between preview and approval, zero rate, rounding/final installment reconciliation, month-end/leap-year due dates, date validation, cancellation, and no financial/domain/replay writes. Prove preview and final schedule match when approved terms and payout date are identical.

## Phase 3 API Inventory and Contracts (proposed, 2026-11-10)

All paths below include /api. This inventory supplements the detailed application preview contract. Preserve existing session cookies, origin checks, strict decoding, fault/error responses and caller-owned command transactions.

### Existing application compatibility

| Method and path | Role | Contract |
| --- | --- | --- |
| GET /api/loans | Staff | Preserve existing latest-100 application array contract for the current frontend |
| POST /api/loans | Teller | Preserve intake fields and idempotent application creation; no financial posting |
| POST /api/loans/:id/decision | Supervisor within authority | Preserve approve/reject and reason behavior. Extend approval with explicit final principal/tenor and expected pricing/approval revision; resolved policy/rate are server-owned. Legacy approvals lacking full pricing evidence remain non-disbursable until completed/reapproved |
| POST /api/loans/preview | Teller, Supervisor | Detailed estimate contract in the preceding section |

A completed approval needs a monotonically increasing approval_revision and a complete approved-terms snapshot. Add the required persistence fields in the forward migration; updated_at alone is not the financial concurrency token. Reapproval must remain blocked once a linked disbursement exists. Design/test the explicit completion/reapproval transition without relaxing existing approval authority.

### Policy and application endpoints

| Method and path | Role | Request and result |
| --- | --- | --- |
| GET /api/loan-interest/policies | Supervisor | Bounded/paginated policy history: id, effective_date, base_rate_bps, bands, reason and safe audit attribution |
| POST /api/loan-interest/policies | Supervisor | Idempotency-Key; effective_date, base_rate_bps, bands[{min_amount,max_amount,rate_bps}], reason, expected_latest_id. Return persisted immutable policy version |
| GET /api/loan-applications | Staff | New paginated collection: optional exact account/application reference, status, responsible_branch_id and payout_pending; return application summaries and next_cursor |
| GET /api/loan-applications/:id | Staff | Detail with requested/approved terms, approval_revision, policy/rate snapshot, decision/authority attribution, receiving/responsible KCP, linked loan reference and authoritative payout eligibility/reasons |

Policy creation follows the prospective version/validation rules above. Same-date supersession, if supported like savings policies, must retain all rows and resolve greatest effective date then greatest ID. Reject stale expected_latest_id with 409. No arbitrary historical policy editing/deletion route.

### Disbursement endpoints

| Method and path | Role | Request and result |
| --- | --- | --- |
| POST /api/loan-applications/:id/disbursement-preview | Teller | Read-only-in-effect; expected_approval_revision. Server uses current Jakarta payout date, approved rate snapshot and approved terms; return final schedule, amount, servicing KCP, cash/condition eligibility, approval revision and payout_date |
| POST /api/loan-applications/:id/disburse | Teller | Idempotency-Key; expected_approval_revision, expected_policy_id, expected_rate_bps, expected_principal, expected_tenor_months, expected_payout_date, verified-condition evidence (final shape pending SOP), optional bounded note. Recompute and compare all expected terms/date under locks; return loan_id, loan_number, application_id, reference, amount, disbursed_at and initial_schedule_version_id |

Expected fields are concurrency/review assertions, never client authority to choose pricing/KCP/timestamps. Missing/changed approval, a date boundary, changed payout terms, or an already-linked payout produces a safe conflict or eligibility rejection. Read/replay the original committed outcome for the same key before checking current business state. Cross-key duplicate payout is blocked by unique application linkage.

The final condition-evidence fields and account mappings remain open; do not implement payout using a generic client boolean as proof of all prerequisites. No separate receipt-writing command: receipt/reprint reads the committed detail and financial reference.

### Loan accounts, versions and payment endpoints

Use loan-accounts to distinguish actual disbursed loans from the legacy /loans application contract.

| Method and path | Role | Request and result |
| --- | --- | --- |
| GET /api/loan-accounts | Staff | Paginated active/paid_off loans; optional exact account_number, loan_number, status, responsible_branch_id. Return actual principal outstanding, original principal, KCP and current version summary |
| GET /api/loan-accounts/:id | Staff | Loan summary, actual principal outstanding, approved/version terms, payout reference/actor/KCP/time, payment eligibility and safe audit attribution |
| GET /api/loan-accounts/:id/schedule-versions | Staff | Paginated versions with version_number, version_type, is_current_version, effective_date, terms, reason and approval attribution, including superseded historical versions |
| GET /api/loan-accounts/:id/installments | Staff | Paginated schedule rows; optional schedule_version_id defaults to current. Validate version ownership; return contractual amounts and separate actual paid/unpaid allocations. Preserve historical-version access |
| GET /api/loan-accounts/:id/repayments | Staff | Paginated committed receipts with amount, principal/interest totals, allocations, actor, servicing KCP, timestamp and financial reference |
| POST /api/loan-accounts/:id/repayment-preview | Teller | Read-only-in-effect; amount. Return oldest-first interest/principal allocations, total, outstanding payable, proposed principal after receipt and expected_payment_revision/current_schedule_version_id |
| POST /api/loan-accounts/:id/repayments | Teller | Idempotency-Key; amount, expected_payment_revision, expected_schedule_version_id, optional bounded note. Under loan lock, revalidate eligibility and recompute allocation; return repayment_id, repayment_number, reference, allocations, principal_before/after, paid_at and resulting loan status |

Add a monotonic payment_revision (or equivalent explicit loan revision) to the loan row and advance it atomically on every relevant financial/state change. A stale payment preview returns 409 and requires review; it must not silently accept a different allocation. Current schedule changes invalidate the preview. Inputs cannot set actor, servicing KCP, posting time, principal/interest components or allocation schedule IDs.

A payment may generate more than two journal lines when cash is allocated between principal and interest; do not apply the savings-only two-line invariant blindly. Final loan accounting recognition and account mappings remain approval gates. Reconciliation must match journal components, allocations, receipt amount and principal projections.

### Shared API guarantees

- Existing mutation commands retain 200/replay behavior. New successful commands/previews return 200; read collections return {items,next_cursor}. No writes on GET or preview.
- Every new list uses a fixed bounded page size of 20 and stable deterministic ordering with an opaque validated scope-bound cursor. Do not inherit the legacy latest-100 cap for new collections. Unknown valid filters return an empty collection; malformed filters/cursors return 400. Each page rechecks authorization.
- Cursor order for time-based lists is descending created_at/id; installments use installment_number/id within one version, versions use version_number/id within one loan. Freeze filter scope across cursor requests.
- Apply operational active/not-deleted filtering to eligibility/master reads; preserve financial/history/replay exceptions. Historical account lookups must not lose prior loans because a member is inactive. Loan details do not expose full NIK, secrets or raw database errors.
- 400 malformed/invalid fields, 401 no session, 403 role/origin/authority denial, 404 missing resource or version not belonging to the loan, 409 stale revision/idempotency conflict/already-completed action, 422 valid request blocked by policy/conditions/cash/overpayment, 503 temporary database/lock/dependency failure, sanitized 500 unexpected invariant failure. Follow existing fault mappings; make any required additions explicit and contract-tested.
- Commands use the existing durable request runner and bounded lock/statement timeouts. Preserve cancellation; no unbounded retries. Financial effects, revisions, journal, audit and replay result commit together.
- Query/service authorization rechecks staff identity, KCP and relevant active master eligibility. Shared cross-KCP visibility follows current rules; role labels alone do not grant approval limits.
- Log safe operation, reference/revision and failure classification without identity documents, raw bodies or credential data. New endpoints reuse the existing logger/error boundary.
- No restructure, arbitrary schedule edit, physical delete, posted repayment edit/delete, fee, penalty, write-off or manual reversal endpoint is included until its workflow/accounting policy is approved.
- Implement contract/auth/validation tests for these endpoints and compatibility tests for legacy /loans. Add database integration coverage for policy changes, stale revisions, same-key replay, cross-key payout duplicates, concurrent payments, rollback and historical reads.

This is an explicit proposed API contract inventory. Payout-evidence shape, final account mappings and configured tariffs must be finalized before the affected financial endpoints can be enabled.

## Minimum loan tariff decision (2026-11-10)

Owner selected a minimum annual nominal loan rate of 3% (300 bps) for both the global fallback and every amount band. Reject missing/null/zero/below-minimum rates with HTTP 400. Require an explicit bands array (empty is permitted for base-only policies). Enforce the floor on stored policies and approved/version snapshots. This is a lower bound, not an automatically seeded tariff. The generic calculator retains zero-rate arithmetic coverage, but loan policies cannot select it. Savings rates are unaffected.
