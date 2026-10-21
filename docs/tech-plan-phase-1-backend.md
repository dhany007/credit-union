# Tech Plan - Phase 1 Backend Features and Gin Refactor

Status: Implemented for local review; see [TDD evidence](testing/gin-refactor.tdd.md).
Date: 2026-10-21
Delivery: Backend implementation completed after explicit owner authorization. The design and planning-only verification below are retained as historical planning context.
Baseline: Current WSL working tree; commit `432b199` contains only the initial documentation.

## Memory Context

No `*-memory-system` directory or `AGENTS.md` was found, excluding dependency/build/Git directories. References: [scope](scope.md), [rules](business-rules.md), [roadmap](roadmap.md), [implementation notes](implementation.md), and ADRs [0003](adr/0003-lending-responsibility-and-authority.md), [0006](adr/0006-initial-business-roles.md), [0008](adr/0008-core-mvp-first.md), including the later loan clarification.

Confirmed scope: one cooperative/multiple KCPs, unique membership, Teller input and cash service, Supervisor decisions, cross-KCP service. Loans cover applications and approval/rejection only. Disbursement, repayments, automated savings interest, and member portal are deferred. IDR 50,000 opening amount is confirmed; its current classification as ordinary savings is an implementation assumption awaiting business sign-off.

Skills applied: `tech-plan`, `coding-standard`, `go-patterns`, `go-lint-workflow`, `error-handling`, `api-contract-review`, `observability-check`, `postgres-patterns`, `database-migrations`, `security-review`, and `env-config-management`. A read-only architect consultation examined the actual source; recommendations below were checked against it.

## Requirement and Backend Scope

Replace the single-file backend with a maintainable Gin API organized by domain, adapted from the owner-provided 6hc reference, with an implementation plan for every Phase 1 feature. Preserve financial guarantees and frontend compatibility. This is one planning artifact with distinct feature plans, following the skill's single-file write contract.

Exclude frontend/UI changes, microservices, ORM replacement, generic CRUD repositories, DI containers, event buses, configurable workflow engines, new financial products, disbursement, and interest jobs. Gin is the only requested new framework; retain pgx and bcrypt.

## Codebase Context

| Current evidence | Responsibility |
| --- | --- |
| `backend/main.go:76-109` | Config, pool, migrate/seed dispatch, server |
| `backend/main.go:110-135` | Development seed |
| `backend/main.go:136-196` | Routes, headers, origin, JSON decoding/error mapping |
| `backend/main.go:197-287` | Sessions, roles, login throttling, login/logout |
| `backend/main.go:288-345` | Metadata and member lookup |
| `backend/main.go:346-405` | Audit and persistent idempotent transaction runner |
| `backend/main.go:406-449` | Shared cash posting and locks |
| `backend/main.go:450-534` | Registration, cash endpoint, history |
| `backend/main.go:535-637` | Loan intake/list/decision |
| `backend/schema.sql` | Existing SQL tables/constraints/indexes |
| `backend/main_test.go` | Eight integration subtests with some shared fixtures |
| `scripts/backend.sh`, `scripts/test.sh` | Root `go run .` commands and isolated-schema DB testing |
| `frontend/app/page.tsx:13-45` | Current JSON consumer and request-key handling |

Go module: Go 1.26.0, pgx v5.11.0, bcrypt through x/crypto; Gin not installed. Database observed read-only: PostgreSQL 16.15, extension `plpgsql` 1.0 only. No Supabase, RLS, or replica dependency. `golangci-lint` is absent; no existing lint config was found.

## Proposed Package Ownership

Updated after the owner's 6hc reference. Prefer a small domain-oriented layout over the earlier global `httpapi/service/postgres` feature-file grouping. The reference provides a concrete familiar pattern, while this MVP has only four domains.

```text
backend/
  cmd/
    api/main.go                    # HTTP process lifecycle only
    migrate/main.go                # Existing explicit initial-schema command
  dev/cmd/seed/main.go              # Local-only seed, never automatic startup
  internal/
    platform/
      app/app.go                   # One construction function and explicit wiring
      config/config.go             # Existing env settings, validated once
      database/
        pool.go                    # Shared pgx pool
        transaction.go             # Persistent idempotent command boundary
        audit.go                   # Caller-transaction audit persistence
        schema.go, schema.sql      # Existing initial schema, unchanged semantics
      httpapi/
        router.go, middleware.go   # Shared Gin setup, origin, recovery, safe logs
        decode.go, response.go     # Strict JSON and response/error helpers
    auth/
      http.go, service.go, postgres_store.go, domain.go
    member/
      http.go, service.go, postgres_store.go, domain.go
    cash/
      http.go, service.go, postgres_store.go, domain.go
    loan/
      http.go, service.go, postgres_store.go, domain.go
```

Create files only when they own actual behavior; combine small files if clearer. `cmd/api/main.go` loads config, creates the app, handles signals/server shutdown, and closes resources. `platform/app` explicitly constructs the pool, stores, services, and route registration. One `New`/`Build` function and one HTTP server suffice; do not replicate the reference's large stores/services registries or option factories.

Domain-local `http.go` owns Gin adapters and transport DTOs. `service.go` owns business validation and typed commands/results; it accepts `context.Context` and an authenticated actor, never a Gin context or HTTP writer. `postgres_store.go` owns SQL/row scanning; `domain.go` holds only that domain's necessary types/errors. Do not import store-generated row types directly into public response contracts.

Dependency rules:

- `platform/app` imports and wires all four domains. Shared platform mechanics do not import business domains.
- A domain's handlers may use shared `platform/httpapi` utilities; that shared package must not import the domain back.
- `auth.Actor` may be used by other services, provided auth does not import member/cash/loan. No generic shared domain dumping ground.
- Registration consumes a small member-defined opening-deposit capability, implemented by cash and injected by app. Cash must not import member.
- Metadata adapters receive explicit read capabilities for staff identity, branch choices/cash, and configured approval settings; avoid feature imports into shared router code or SQL inside app wiring.

Keep pgx and concrete PostgresStore constructors. Introduce narrow interfaces only where they represent a useful consumer boundary or test seam, such as registration calling cash posting. No generic CRUD repository or unit-of-work framework.

Registration's service starts one command through `platform/database`'s narrow idempotency runner. In its callback, member store inserts identity, cash's `DepositInTx` receives the SAME caller-owned `pgx.Tx`, then registration audit/result is recorded. Cash's normal deposit/withdrawal uses that same posting implementation. Inner stores/posters never Begin/Commit. Member identity operations own identity fields; cash owns balance/journal effects even while the existing balance column remains on `members`. No schema split is required for domain boundaries.

Use readable multiline SQL, named business constants, and explicit errors. Preserve the transaction and HTTP contracts below regardless of the changed package layout.

### 6hc Reference Evidence and Adaptation

Read-only source checkout:

```text
/home/dhany/workspace/former/citra-international-digital-former/work/github/2nterdigital/6hc/6hc
```

The supplied parent contains several checkouts; inspection used the nested main `6hc` checkout. Source evidence:

| Reference | Verified pattern | Credit union decision |
| --- | --- | --- |
| `backend/cmd/api/main.go:19-23` | Thin main/run, signal context, build/shutdown lifecycle | Adopt one API process lifecycle |
| `backend/internal/platform/app/api.go:48` | BuildAPI composition entry | Adapt one small construction function |
| `backend/internal/platform/app/composition.go:47` | Explicit pool/stores/services/route wiring | Adopt explicit wiring, omit large registries/fallbacks |
| `backend/internal/adminauth/http.go:74` | Domain-local Gin RegisterAuthRoutes and DTOs | Adopt domain-local HTTP adapters; preserve OUR routes/cookies |
| `backend/internal/adminauth/session_service.go:74-88` | Typed constructor-injected service with context and test seams | Adopt small explicit dependencies, not every option/interface |
| `backend/internal/adminauth/postgres_store.go:67` | Domain-owned PostgresStore | Adopt concrete domain stores using current pgx |
| `backend/sqlc.yaml` | sqlc-generated domain query packages with pgx/v5 | Tooling is separate; defer generation in this scoped refactor |
| `backend/go.mod` | Gin, pgx, migration tooling | Retain our pgx/bcrypt; add pinned Gin during implementation |
| `docs/rewrite/memory/README.md`, `CURRENT_STATUS.md` | Evidence-based docs and explicit readiness boundaries | Use as reference context, not requirements for this project |

Adopt structure and maintainability patterns, not source wholesale. Do not copy reference business logic, schemas, permissions, credential defaults, cookie names/TTL, API envelopes/prefixes, workers, extra servers, in-memory fallback, or telemetry infrastructure. Its history and domain rules are not credit-union policy.

Keep handwritten parameterized pgx SQL initially. `sqlc` is a possible separately agreed tooling improvement; it is not necessary to adopt domain packages. Likewise, adopting `golang-migrate` requires a separate safe baseline plan against the existing schema before future evolution. Never reset/recreate current data to install a migration tool.

Developer command layout will change to `go run ./cmd/api`, `go run ./cmd/migrate`, and `go run ./dev/cmd/seed`. Keep the public `scripts/backend.sh`, `scripts/backend.sh migrate`, and `scripts/backend.sh seed` commands working through explicit dispatch; update direct-command docs during implementation. This is a developer-command change, not an HTTP API change. Do not preserve a redundant root binary indefinitely just to mimic the old layout.

### Shared Transaction Contract

Preserve one boundary for every idempotent business command:

1. Authenticate/authorize and strictly decode; compute the compatible fingerprint.
2. Begin transaction; acquire the authenticated actor/request-key advisory lock.
3. Read persisted request: same fingerprint replays its stored result, different fingerprint returns 409.
4. Execute use case; registration calls the same transaction-scoped cash helper as deposits.
5. Persist all balance, transaction, journal, audit, and retry-result effects together.
6. Commit once, then return success. Inner helpers never begin/commit independently.

Keep cash lock order: member row `FOR UPDATE`, then servicing-KCP advisory lock; the request lock precedes both. Retain current lock namespaces and isolation behavior. No internal HTTP calls or external work inside transactions. No floating-point money. No automatic retry following ambiguous commit; caller retries the same key/payload. Cancel work promptly; use a short bounded cleanup context only for rollback if the original context is cancelled.

## API Design

Extraction/Gin migration classification: **no contract change**. Preserve paths, JSON names, integer IDs/amounts, validation/normalization, current 200 mutation success, `[]` empty lists, masked NIK, timestamp encoding, and `{"error":"message"}` errors. Do not add `/v1` or a response envelope.

| Route | Access | Input | Success shape |
| --- | --- | --- | --- |
| `GET /api/health` | Public | None | `status`; DB failure 503 |
| `POST /api/login` | Public, valid Origin | `username,password` | `ok` plus cookie |
| `POST /api/logout` | Staff, valid Origin | No business input | `ok` plus expired cookie |
| `GET /api/me` | Staff | None | `id,username,name,role,branch_id,branch_name` |
| `GET /api/meta` | Staff | None | `branches,branch_cash,approval_limit,approval_product,approval_risk,disbursement_enabled` |
| `GET /api/members` | Staff | Optional `search` | Existing member array |
| `POST /api/members` | Teller, request key | `nik,name,phone,address` | `id,number,reference,balance` |
| `POST /api/cash` | Teller, request key | `member_id,amount,kind,note` | `reference` |
| `GET /api/members/:id/transactions` | Staff | Path ID | Existing history array |
| `POST /api/loans` | Teller, request key | `member_id,amount,tenor_months,purpose,risk,responsible_branch_id` | `id,status` |
| `GET /api/loans` | Staff | None | Existing application array |
| `POST /api/loans/:id/decision` | Supervisor, request key | `decision,reason` | `id,status` |

`:id` is Gin notation only; public URLs remain unchanged. All business POST mutations require `Idempotency-Key` of 16-100 bytes, scoped per authenticated user. Actor/role/servicing KCP must come from the session, never from user input.

Preserve search trimming and 100-byte bound, exact NIK matching, name/member-number substring matching, existing 100-row caps, member/loan descending-ID order, and history descending creation time. New pagination, stable history tie-breaking, stricter validation, or changed error text require explicit compatible-change assessment.

### Gin Compatibility Guardrails

- Use `gin.New()` with explicit recovery, logging, headers/origin, authentication, and role middleware. Abort and return on denial.
- Pass `c.Request.Context()` downstream and use `c.Param("id")`; `Request.PathValue` is not populated by Gin. Never retain pooled Gin contexts.
- Keep the local strict JSON decoder: 16 KiB body limit, unknown-field rejection, one JSON object, trailing EOF check, exact integer decoding. Preserve existing byte-length validation and normalization. Do not rely on default binding or automatic binding error output. [Gin binding documentation](https://gin-gonic.com/en/docs/binding/binding-and-validation/) explains the difference between automatic and caller-controlled errors.
- Use Gin as the handler of the configured `http.Server`, retaining loopback address and timeouts. Add bounded graceful shutdown, not a bare default `router.Run()`. See [Gin custom HTTP configuration](https://gin-gonic.com/en/docs/server-config/custom-http-config/).
- Explicitly disable untrusted proxy interpretation with `SetTrustedProxies(nil)` unless approved proxy addresses are configured. Preserve `RemoteAddr` throttling semantics first; do not trust arbitrary forwarded IPs. See [Gin trusted proxies](https://gin-gonic.com/en/docs/server-config/trusted-proxies/).
- Characterize HEAD, unknown routes, wrong methods/Allow, trailing and repeated slashes, and invalid path IDs before migration. Configure equivalent handling; no accidental redirect of financial POST requests.
- Use route templates for logs, never raw URLs/query strings: supported member search can contain a full NIK. Recovery must not dump sensitive headers/body.
- Preserve typed response values, empty arrays, masked identity, and `time.Time` timestamp serialization. Replay persisted JSON as raw JSON, not an encoded string or base64 byte slice.

### Durable Fingerprint Compatibility

Current hashing uses `method + actual URL path + json.Marshal(input)`. Declaration order and trimming matter. Do not replace anonymous input structs with maps and assume old retries will work.

Use dedicated legacy-compatible fingerprint projections, separate from public DTO layout:

| Operation | Ordered JSON fields | Existing normalization |
| --- | --- | --- |
| Registration | `nik,name,phone,address` | Trim all four before hashing |
| Cash | `member_id,amount,kind,note` | No new trimming |
| Application | `member_id,amount,tenor_months,purpose,risk,responsible_branch_id` | Purpose trimmed for validation only |
| Decision | `decision,reason` | Reason trimmed for validation only |

Preserve zero/empty fields and actual path, not Gin's route template. Persisted request records must replay across old/new binaries. Test using fingerprints computed by the OLD representation, not expectations generated from the new code itself.

## Feature Plan F0 - Gin Foundation and Runtime

**Requirement / Backend scope:** Thin `cmd/api/main.go`, one `platform/app` composition function, domain-local Gin adapters/services/stores, and stable wrapper commands. No business SQL or HTTP validation in entry points.

**Implementation plan:**

1. Capture HTTP/JSON/cookie/replay behavior before extraction. Keep existing integration guarantees.
2. Extract services/store/adapters in small compiling steps, then migrate router wiring to Gin.
3. Pin a compatible stable Gin release during implementation, review transitive dependencies, and run `go mod tidy`. No dependency changes during planning.
4. Move unchanged schema embed to `platform/database` and split API/migration/development-seed entry points as above. Update wrapper dispatch and direct-command documentation; preserve wrapper behavior. No migration-tool adoption or schema redesign.
5. Parse environment into typed config once; retain existing variable names/defaults. Validate origin, cookie boolean, approval limit, and command selection with sanitized errors. Require seed password only for seed.
6. Keep existing server timeouts and loopback binding. Implement signal cancellation and bounded shutdown, then pool cleanup; do not terminate before cleanup completes.

**API / Test plan:** All routes in the API table remain. Add characterization tests for decoding, HEAD/404/405, redirects, cookies, raw replay, recovery, config, and command dispatch. Test initial schema/seed only in an isolated fixture.

**Risks / Open questions:** Framework defaults are a compatibility risk; tests must cover them. Select/pin the Gin version at implementation time. No new migration framework or deployment service is needed.

## Feature Plan F1 - Staff Authentication and Authorization

**Requirement / Backend scope:** Teller and Supervisor login, permitted actions, current-user lookup, and logout. Retain bcrypt, hashed session tokens, eight-hour expiry, HttpOnly/Strict cookies, and configurable Secure. No member login or staff administration.

**Implementation plan:**

1. Separate auth SQL, password/session service, and cookie/role adapters.
2. Preserve old-session invalidation on login and token invalidation on logout. Return token-generation errors rather than panicking.
3. Keep missing/expired session and wrong credentials at 401, wrong role/origin at 403. Abort before downstream handlers.
4. Plan a separate error correction: currently any session/user query failure becomes 401. Distinguish no-row/credential failure from DB outage/cancellation; map dependency failure safely to 503 with the same `error` property. Classification: **behavior fix with client impact**, separate from router compatibility work.
5. Initially preserve throttle behavior. Address shared proxy-IP lockout separately with tested bounded keys; do not switch to untrusted forwarded IPs or add Redis.

**API / Test plan:** `/login`, `/logout`, `/me`, protected-route middleware. Test valid/invalid/unknown login, expiry, cookie flags, role/origin denial, session replacement, logout, rate limits with a controllable clock, entropy failure, and DB outage. No sensitive values in logs.

**Risks / Open questions:** Next.js may make all users share one remote IP, so the current IP cap can lock out unrelated staff. Preserve protection while planning the separate keying fix. Production provisioning and password recovery remain out of scope; no permission expansion is assumed.

## Feature Plan F2 - KCP Metadata and Health

**Requirement / Backend scope:** Staff see KCP choices, cash at their own KCP, and small approval-policy metadata. Public health reflects database availability. No KCP management dashboard.

**Implementation plan:**

1. Extract branch queries and cash aggregates into PostgreSQL helpers and typed query results.
2. Derive branch cash from committed cash journal lines using authenticated staff KCP, not a client-supplied branch or cache.
3. Preserve all `/meta` fields, including regular/low approval policy and `disbursement_enabled:false`.
4. Preserve health 200/503 behavior; never return DSNs, database versions, or secrets.

**API / Test plan:** `/meta` and `/health`. Test both KCP identities, zero cash, totals after deposit/withdrawal, authentication, full response shape, and failed/cancelled health queries. No schema change.

**Risks / Open questions:** Do not substitute cooperative-wide totals for branch cash. Supervisor's existing decision scope is cooperative-wide. Opening cash and transfers remain unresolved operational features, not implicit balances.

## Feature Plan F3 - Member Registration and Search

**Requirement / Backend scope:** One member per KTP/NIK across KCPs, separate member number, home KCP, current opening deposit, and existing lookup with masked identity.

**Implementation plan:**

1. Introduce named commands/results and transport DTOs while retaining current trim/validation: 16 digits, reject all-zero NIK, bounded name/phone/address, existing errors.
2. Use one idempotent transaction for member insertion, shared IDR 50,000 cash posting, cash/registration audits, and saved result.
3. Keep unique database constraint as the final concurrency guard; never rely only on a duplicate pre-check.
4. Assign home KCP from the registering Teller. Service elsewhere must not create another membership.
5. Preserve exact NIK and existing substring search, masked results, descending ID, and limit 100. Keep SQL parameterized.

**API / Test plan:** `POST /members` and `GET /members`. Test simultaneous same-NIK registration in two KCPs, duplicate 409 without extra funds, opening-post rollback, old/new retry compatibility, validation/strict JSON, search cases, and masking. Assert member, balance, transaction, journal, audits, and retry record are all-or-nothing.

**Risks / Open questions:** Ordinary-savings classification of Rp50,000 remains an assumption, not new business approval. Identity corrections/eligibility require policy. Format validation is not Dukcapil verification. Full NIK search must be absent from access logs.

## Feature Plan F4 - Cash Deposit

**Requirement / Backend scope:** Teller accepts cash for an existing member at any KCP, preserving home KCP and recording servicing KCP, actor, reference, and accounting effects.

**Implementation plan:**

1. Keep `/cash` and `kind:"deposit"`; do not add a new endpoint.
2. Preserve positive integer rupiah, note/member validation, and technical cap of IDR 1 trillion. Use named constants for policy-significant values.
3. Call the shared posting helper with the caller's `pgx.Tx`; lock member then servicing KCP.
4. Increase member balance, debit cash, credit member savings, and write transaction/audit/replay outcome in one commit.
5. Reuse the same helper for opening deposit; do not duplicate accounting in Member service.

**API / Test plan:** `POST /cash`, existing `{reference}` response. Test cross-KCP service, boundaries, fractional rejection, concurrent same-key requests, changed-payload conflict, persisted legacy replay, missing-member 404, and forced failure in journal/audit/retry persistence.

**Risks / Open questions:** Response loss after commit must not trigger automatic fresh-key retry. Same-key retry is the recovery path. Business deposit limits are not inferred from the technical cap. No database redesign required.

## Feature Plan F5 - Cash Withdrawal

**Requirement / Backend scope:** Teller pays cash at any KCP only when member funds AND servicing-KCP cash are sufficient.

**Implementation plan:**

1. Keep `/cash` and `kind:"withdrawal"` with the same command format as deposits.
2. Within the shared transaction, lock member then KCP and check committed availability; preserve separate 422 shortage errors.
3. Reduce member balance, debit member savings, credit KCP cash, and write transaction/audit/replay outcome atomically.
4. Preserve current zero balance floor; do not invent a retained balance requirement or cash funding.

**API / Test plan:** Existing cash contract. Test simultaneous withdrawals on one account AND withdrawals on different members competing for the same KCP cash. Independently fund fixtures so member insufficiency cannot falsely prove branch locking. Test exact exhaustion, branch shortage with sufficient member balance, replay, and rollback.

**Risks / Open questions:** All cash writers must obey the same branch lock. Funding, transfers, and reversal procedures remain outside this refactor. Do not weaken cash checks to make an unfunded KCP pay out.

## Feature Plan F6 - History and Basic Reconciliation

**Requirement / Backend scope:** Staff view recent member transactions with reference, direction, amount, KCP, actor, note, and timestamp. No analytics service or public privileged reporting endpoint.

**Implementation plan:**

1. Extract history SQL and a typed read model; preserve ID parsing, response keys, timestamp encoding, cap 100, and current ordering.
2. Preserve `[]` for no history, including a valid nonexistent member ID. A change to 404 is separate API work.
3. Read committed transaction/journal records; do not add mutable cash caches.
4. Add test-side reconciliation: every transaction balances, member savings lines reconcile to balances, KCP cash reconciles to cash lines.

**API / Test plan:** `GET /members/:id/transactions`. Test empty/populated/malformed-ID responses, cross-KCP records, staff access, timestamps, and field shape. Reconcile after concurrency and forced failures, not only success paths.

**Risks / Open questions:** Current history has no tie-breaker for equal timestamps and no pagination. Preserve behavior first; cursor/tie-breaking and formal receipt formats are follow-up work.

## Feature Plan F7 - Loan Application and Listing

**Requirement / Backend scope:** Teller submits regular-product applications at any KCP, with receiving KCP and one selected responsible KCP. Staff list applications. No money movement or active loan principal.

**Implementation plan:**

1. Extract typed Loan commands and PostgreSQL queries; preserve legacy fingerprint order.
2. Preserve amount cap, positive integer values, tenor 1-120 months, purpose length, risk enum, existing member, and valid responsible KCP.
3. Derive receiving KCP/creator from session, validate chosen responsible KCP in the database.
4. Insert `submitted`, audit, and save retry outcome in one transaction.
5. Preserve listing keys, descending ID, cap 100, and empty-string decision reason/actor when not decided.

**API / Test plan:** `POST /loans` and `GET /loans`. Test cross-KCP membership, invalid member/KCP 400, validation boundaries, permissions, concurrent replay, rollback, and zero financial effects. All existing loan-list fields remain present.

**Risks / Open questions:** Teller risk is preliminary, not an automated or independent decision. Responsible-KCP assignment policy is unresolved; retain explicit selection. Rates and schedules remain deferred rather than assuming a zero-interest loan.

## Feature Plan F8 - Supervisor Loan Decision

**Requirement / Backend scope:** Approve/reject a submitted application with actor, reason, time, and applied authority limit. Decision updates workflow only.

**Implementation plan:**

1. Lock application in the idempotent transaction and recheck submitted status.
2. Require Supervisor role and block self-decision; preserve 404/409/403 outcomes.
3. Approval requires positive configured limit, amount within it, regular product, and low risk. Default zero denies approval. Rejection does not require positive approval authority.
4. Save decision, actor, reason, timestamp, authority limit, audit, and retry result atomically. Use one small explicit policy, not a rules engine.
5. Keep payout routes absent and metadata false; approval creates no principal, cash transaction, or savings movement.

**API / Test plan:** `POST /loans/:id/decision`. Test threshold boundaries, zero policy, medium/high risk, wrong role, self-decision, existing decision/new key 409, same-key replay, concurrent approve/reject, and persistence rollback. Compare financial rows/balances before and after.

**Risks / Open questions:** Production thresholds remain undecided; never seed them from test fixtures. Preserve applied-limit evidence when config later changes. Committees, reassignment, product expansion, and payout remain deferred.

## Error Handling and Observability

Services return transport-neutral typed errors and wrap causes with `%w`; HTTP maps them once. Do not swallow database errors or expose SQL/validator details.

| Failure | Boundary behavior | Retry/log expectation |
| --- | --- | --- |
| Invalid input/JSON/key | Existing 400 safe message | No automatic retry; no payload logging |
| Missing/expired session or invalid credentials | 401 | Separate from dependency failure |
| Forbidden role/origin/self-decision | 403 | Abort before mutation |
| Missing cash member/decision target | Existing 404 | No SQL/identity detail leakage |
| Duplicate NIK, changed-key payload, already decided | Existing 409 | Distinguish replay from a new command |
| Insufficient funds/cash or authority | Existing 422 | No partial writes |
| Infrastructure failure/deadline | Preserve 500 during extraction; explicit later correction to safe 503 | No blind write retry after ambiguous commit |
| Client cancellation | Stop and roll back | No second response or noisy false server-error log |
| Unexpected panic/invariant failure | Controlled safe 500 | Sanitized diagnostic; no header/body dump |

Use stdlib `slog` for operation, route template, method, status, duration, request ID, and stable error class. Safe SQLSTATE/allowlisted constraint name may help diagnosis; do not log PostgreSQL detail containing submitted values. Omit query strings, NIK, phone/address, bodies, passwords, cookies, tokens/hashes, and fingerprints. Recovery must not dump sensitive request headers. Record failures once at the responsible boundary.

No telemetry infrastructure is required. Request outcome/duration and durable business audit records are sufficient for this local MVP; do not add Prometheus, a collector, or dashboards without need. Human review of financial/identity access controls remains necessary before real use.

## Database, Rollout, and Rollback

Classification: source/package and router rollout; **no schema or data migration required**. Preserve tables, constraints, indexes, sessions, journal/balance records, audit history, and persisted request results.

Moving schema embed changes source ownership only. Compare old/new schema contents and apply the unchanged SQL only to a clean isolated fixture during tests. Do not rerun DDL against application data merely to move files. Preserve explicit migrate/seed commands; no automatic startup migrations, new extensions, indexes, RLS, or database configuration changes.

Retain primary-database reads/writes and current locking semantics. Bound connection acquisition/request work, honor cancellation, and avoid unnecessary pool tuning. If a query/index change becomes necessary, inspect its plan on safe fixtures before adding indexes; no expensive production diagnostics. Database privilege separation and restore verification remain deployment tasks, not claimed complete here.

Implementation order:

1. Characterize current behavior and persisted retry compatibility.
2. Extract packages and typed commands/results with all existing guarantees intact.
3. Wire Gin and verify router/middleware/JSON compatibility.
4. Make explicitly classified error fixes separately; do not smuggle policy changes into extraction.
5. Run focused tests and smoke checks, then update implementation/architecture docs during implementation.

The implementation is currently uncommitted; do not reset to docs-only `origin/main`, clean the working tree, or discard existing code. Record a recoverable source baseline if requested before refactoring.

Rollback restores prior source/build and environment against the same database; it never drops tables, clears request records, reseeds credentials, or rewrites transactions. Legacy retry compatibility must work in both directions. If a schema change is needed, amend this plan with an explicit migration/rollback strategy before proceeding.

## Test Plan and Validation Commands

Keep all eight current integration guarantees. Refactor new tests into independent fixtures rather than adding ordering-dependent subtests. Check setup/query/JSON errors; ignored errors must not make assertions falsely pass. Use table-driven validation/authority tests, `httptest` with the Gin engine, and real PostgreSQL integration tests for atomicity and locks.

Required additional regressions:

- Persist an old-format fingerprint/result and replay through the new router; do not compute expected bytes from the new implementation.
- Concurrent identical keys produce exactly one posting; altered payload conflicts.
- Cancel a request waiting for member/KCP locks; assert cleanup and no partial effects.
- Force audit/retry-result persistence failure as well as journal failure.
- Different members compete for the same last KCP cash; concurrent decisions yield one transition.
- Strict JSON, recovery, HEAD/404/405/Allow, redirects, cookies, non-nil arrays, and timestamp representation retain contracts.
- Database failure in authentication is the deliberately designed dependency failure, not a false expired-session result.

Commands below are for future implementation; they were not run during planning:

```bash
# From project root: configured PostgreSQL integration tests with race detection.
bash scripts/test.sh

# Backend checks after package extraction:
cd backend
gofmt -l cmd/ dev/cmd/ internal/   # Read-only check; format touched files during implementation.
go vet ./...
go build ./...
go test ./...              # Does not substitute for the configured DB suite above.
```

An integration test skipped for absent `TEST_DATABASE_URL` is not proof of correctness. Existing `scripts/test.sh` uses a randomly named isolated schema; in shared environments prefer a dedicated test database. Never run synthetic financial tests against real member records.

Use `go-lint-workflow` to adapt its lint-config asset only during implementation. Verify installed golangci-lint/config versions, configuration validity, and `lll` activation at 120 characters before reporting lint coverage. Run scoped/package-aware checks and distinguish baseline debt. No golangci-lint result is claimed today because the tool is absent. Docs-only `origin/main` is not a useful implementation lint baseline.

A later local smoke test may run existing API/frontend, sign in, locate a member, and use synthetic cash/loan cases with explicitly test-only authority settings. The frontend is an integration consumer, not a UI refactor target.

## Risks & Mitigations

- **Package proliferation:** four small domain packages and necessary shared platform mechanics only. Follow the owner-selected domain organization without empty scaffolding, generic repositories/DI, microservices, or reference-scale infrastructure.
- **Fragmented money writes:** one outer transaction, shared caller-owned cash helper, lock-order and forced-failure tests.
- **Retry incompatibility:** dedicated legacy projections, actual request path, raw stored response, cross-version fixtures.
- **Gin defaults:** explicit decoder/middleware/proxy/routing configuration and HTTP characterization tests.
- **Misleading authentication failures:** distinguish no-row from DB failure; deliver status correction explicitly.
- **Private data in logs:** omit query/body/session content; sanitized route/outcome tests.
- **Policy drift:** retain safe defaults; technical limits and test fixtures do not become approved lending policy.
- **False completion:** refactor/tests do not complete cash transfer/reversal processes or production readiness.

## Open Questions

None block the behavior-preserving package extraction or Gin migration.

Business decisions still needed before real operations: approval amount/scope, accounting classification of the opening Rp50,000, retained-balance/withdrawal rules, responsible-KCP assignment, identity correction, and cash funding/correction procedures. Preserve safe defaults and documented assumptions until settled. Do not expand into disbursement or savings interest.

Treat auth dependency-error correction as its own client-visible behavior fix. Pagination, rate-limit policy changes, and stronger validation are separate compatibility decisions, not automatic consequences of selecting Gin.

## Implementation Handoff

- Implement in a separate step after the owner asks to proceed; this plan does not approve new business policies.
- Automatically use `tdd-workflow`: meaningful RED for missing guarantees, minimal GREEN, then refactor. Existing behavior extraction starts with passing characterization tests; do not fabricate failing tests just to move files.
- Automatically follow `coding-standard` and `go-patterns`: readable Go, explicit errors/context, small ownership-focused packages, and concrete dependencies where sufficient.
- Use `go-lint-workflow` for scoped/baseline-aware formatting, vet, and lint. Use `go-build-resolver` for minimal fixes if validation fails.
- Apply `error-handling` and `api-contract-review` for validation/status/contract changes; apply `observability-check`, `security-review`, and `env-config-management` to relevant critical paths.
- Apply `postgres-patterns` for SQL/locking/pool behavior; use `database-migrations` and amend the plan before any schema/data change.
- Limit dependency edits to Gin/required transitive modules and correct module bookkeeping; retain pgx/bcrypt. No incidental ORM replacement or unrelated upgrade.
- Keep implementation backend-only. Startup/test scripts and docs may be updated during the later implementation as necessary, but frontend source stays unchanged if contracts are preserved.
- Run the focused commands above and report exact passes, failures, skipped tooling, and residual limits. Do not claim production sign-off.

## Planning Verification

Only this Markdown artifact is authorized for writing during this task. Compare existing-file hashes with the pre-planning baseline and confirm source, tests, configuration, schema, generated files, scripts, dependency files, frontend, and earlier documentation are unchanged. No builds, formatters, generators, package installs, or database mutations were performed in planning.
