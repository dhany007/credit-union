# Tech Plan - Frontend lint and automated quality checks

## Requirement and context
Extend the existing Go lint CI with frontend lint and backend/frontend tests on every push and PR. Backend has race-enabled PostgreSQL integration tests; frontend currently has only TypeScript/build checks. Keep the MVP simple. No memory-system directory or AGENTS.md was found in prior project inspection.

## Backend scope
Use the existing Go suite with a disposable PostgreSQL 16 GitHub Actions service, explicit TEST_DATABASE_URL, race detector, uncached execution and vet. Do not reuse application data or secrets. Do not replace meaningful integration tests with mocks or permit silent database-test skipping in CI. No backend API/schema changes.

## Frontend integration
Add ESLint 9 and eslint-config-next matching installed Next16.4.2, flat core-web-vitals + TypeScript configuration, and npm lint/test scripts. Use Node24 built-in node:test, no new component/E2E framework. Test production API client success/failed/malformed responses, headers/abort behavior, localized error privacy and EN/ID formatting/interpolation. Fix newly exposed runtime bugs with failing regressions first. Do not label these unit tests as automated browser coverage.

## Implementation plan
1. Record current conventions and consult the read-only architect (completed; reviewer confirmed PostgreSQL isolation, stable checks and the native Node utility-test scope).
2. Add lint/test setup and behavior tests, inspect the initial failures.
3. Resolve genuine new lint findings with scoped changes, preserve user behavior and hook correctness; no blanket rule suppression.
4. Add CI Backend tests and Frontend lint and tests jobs, retain Go lint and formatting. Use read-only contents permissions, no path-filter gaps, stable job names, npm ci lockfile installs, finite timeouts, push/PR triggers.
5. Validate locally: lint, TypeScript, unit tests, production build, Go suite/vet, workflow structure. Document coverage limits and checks.
6. Required main checks may only be configured after the check names exist and pass remotely; preserve the existing PR-only protection and bypass settings. Commit/push remains subject to user authorization. Do not alter the pending owner-approval proposal.

## Risks and validation
Test service is disposable with synthetic credentials, never a real environment. Use explicit CI DB variable to avoid skipped integration tests. Preserve external API errors and stable status shapes; malformed JSON must fail rather than look successful. Node type stripping runs the existing erasable TypeScript modules; npm typecheck independently checks types. Native unit tests do not cover DOM behavior, forms, authentication or themes end-to-end. Keep existing documented browser checks and state that automated browser coverage remains future work. No arbitrary coverage target or extra framework.

## Open questions and handoff
None blocking. Apply coding-standard, go-lint-workflow, react-testing, react-patterns, typescript-patterns, frontend-build-resolver and env-config-management as relevant. Preserve existing financial tests, runtime dependencies and main PR rules. Record actual validation; do not claim CI status before a remote run.

## Path routing update

CI now runs only for the affected stack using native workflow path filters. Backend lint/tests and frontend checks are separate workflows; shared Makefile/Compose/environment example changes run both stacks, documentation-only changes run neither, and manual dispatch remains available. Job bodies and stable check names are preserved. See [quality checks](testing/quality-checks.md#changed-path-routing) for the mapping and required-check semantics.
