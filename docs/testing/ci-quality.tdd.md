# TDD Evidence - Frontend utilities and CI quality checks

## Source
[Tech plan](../tech-plan-ci-quality.md), [quality-check documentation](quality-checks.md).

## Frontend regression evidence
Initial node:test execution selected 21 tests: 13 passed and 8 failed for intended missing behavior:
- Headers and tuple-array caller headers were lost (2 failures).
- Null/object-shaped HTTP error payloads did not use the safe fallback (2 failures).
- HTTP200 with non-JSON content resolved as successful data (1 failure).
- Unknown error names inherited from Object.prototype returned non-message values (3 failures).

After the scoped API/error-map fixes, all initial 21 passed, none skipped. A further response-body cancellation regression failed because AbortError was rewritten as a network error, then passed after preserving cancellation. The final suite contains 22 tests. Tests import production .ts modules and mock fetch only; no separate test implementation is exercised. Initial `node --test tests/` discovery failed before selecting the suite; the command was corrected to explicit test file/glob selection and this setup failure is not RED evidence.

## Lint evidence
First ESLint run reported four errors and two warnings in existing code: effect-driven state clearing/refresh, missing effect dependency, plain internal anchor, browser-preference hydration, and anonymous PostCSS export. Scoped fixes use cancellable fetches, identity-associated history, Next Link, and a named config. One targeted documented exemption retains SSR-safe browser preference hydration. Final npm run lint passed with zero warnings.

## Validation
- Full backend race-enabled uncached suite: PASS; PostgreSQL integration tests executed (36.172s), not skipped.
- go vet ./...: PASS.
- Frontend lint and TypeScript checks: PASS.
- Frontend unit tests: 22 PASS, zero failures/skips.
- Next.js production build: PASS.
- Clean npm ci lockfile installation: PASS.
- Workflow YAML parsed and expected triggers, read-only permissions, checkout credentials, job names/timeouts, PostgreSQL service and race flags verified.
- npm audit --omit=dev: zero runtime vulnerabilities. Development-only upstream lint dependency advisory and ESLint compatibility limits are recorded in quality-check documentation.

## Limits and delivery
No new automated component/browser suite or coverage threshold. No new backend behavior/schema. Browser-only preferences retain their previous behavior. No real database or secret is referenced by the CI workflow. Local validation does not claim a remote CI run. At local validation time, the new workflows had not yet been published. The user subsequently authorized commit/push to the existing feature branch. Required-check activation is separate; existing PR-only main rules and the pending owner-approval proposal remain unchanged.
