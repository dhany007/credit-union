# Quality checks

## Pull-request checks

| Check | Verification |
|---|---|
| Go lint and formatting | Pinned golangci-lint, configuration verification, Go formatting |
| Backend tests | Go vet and uncached race-enabled unit/PostgreSQL integration tests |
| Frontend lint and tests | ESLint Next/Core Web Vitals + TypeScript rules, TypeScript checking, native Node unit tests, production build |

Three workflows use native GitHub Actions path filters on branch pushes and pull requests. Backend changes trigger Go lint and backend tests; frontend changes trigger frontend checks. Changes to both stacks trigger both sets. Documentation-only changes trigger neither. Manual dispatch runs the selected workflow regardless of paths. Tag pushes do not run these branch checks. Permissions are read-only contents; checkout does not persist credentials. Jobs have finite timeouts and cancel obsolete runs for the same ref. The existing Go lint check name is preserved.

The backend job owns a disposable PostgreSQL 16 service with a runner-local synthetic password and explicit TEST_DATABASE_URL. Existing test fixtures create/drop isolated schemas. Integration tests must run, not silently skip because the database variable is absent. No repository secrets, application database, or Docker local database are used.


### Changed-path routing

| Changed path | Checks |
| --- | --- |
| `backend/**` | Go lint and formatting; Backend tests |
| `frontend/**` | Frontend lint and tests |
| `scripts/lint.sh` | Go lint and formatting |
| `scripts/test.sh`, `scripts/backend.sh`, `scripts/seed-demo.sh` | Backend tests |
| `Makefile`, `compose.yml`, `.env.docker.example` | All three checks (shared local setup) |
| A workflow file | That workflow's checks |
| Documentation only | None |

Backend tests remain in `.github/workflows/quality.yml`; frontend checks move to `.github/workflows/frontend.yml`. Existing job/check names are preserved. Pull request filtering uses the complete PR diff; a mixed-stack PR continues running both stacks even if its latest commit touches only one stack. Workflow filters follow the [GitHub Actions path-filter documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onpushpull_requestpull_request_targetpathspaths-ignore).

Path-filtered workflows do not report checks for unaffected stacks. Do not configure all three contexts as unconditional required checks for every PR; that would leave skipped workflows pending. Existing repository protection is unchanged by this workflow change.

## Local commands

Backend, from repository root with the existing local test database configured:

```sh
bash scripts/lint.sh
bash scripts/test.sh -count=1
cd backend
go vet ./...
```

Frontend, with Node24 and npm from frontend/:

```sh
npm ci
npm run lint
npm run typecheck
npm test
npm run build
```

The frontend uses Node24's native test runner and TypeScript stripping to import the production utility modules. No new test framework, DOM emulator, loader, or runtime dependency is introduced. TypeScript checking runs separately because stripping does not typecheck. The existing package intentionally retains its module configuration; Node may emit a harmless MODULE_TYPELESS_PACKAGE_JSON detection warning for imported .ts files. This is not a test failure.

## Frontend coverage and fixes

- Successful/failed API responses, non-JSON responses, malformed error bodies, caller headers in all RequestInit forms, cache disabling, cancellation propagation, and no automatic mutation retry.
- EN/ID controls, numeric interpolation, account-number string preservation, translation placeholder parity, public errors, and safe fallback for unknown/inherited-property error names.
- Tests exercise production code and stub fetch only; no real requests occur in unit tests.
- Initial failing regressions exposed malformed JSON being treated as success, lost Headers/tuple headers, invalid HTTP error-body handling, and inherited error-map properties. Small fixes preserve the existing public messages.
- ESLint fixes use Next Link, cancellable workspace/history fetching, and history associated with member/staff identity so results from another selection are not rendered. A single documented hydration exception remains for reading browser-only preferences after SSR; other hook rules stay enabled.

These are utility unit tests, not automated browser or component tests. Login/registration/cash/loan forms, preference persistence, and end-to-end navigation still rely on the documented browser checks. Expand automated UI coverage separately when its tooling/scope is approved; do not claim complete frontend coverage or impose an arbitrary percentage threshold.

## Dependency note

eslint-config-next is pinned to the installed Next16.4.2. ESLint9 is retained because the config's React plugin does not support ESLint10. npm audit reports a development-only braces dependency chain through the Next lint plugin, without a compatible published fix in this installation. Do not force a downgrade to Next14 as suggested by audit. npm audit --omit=dev reports zero runtime vulnerabilities. Track an upstream compatible update separately.

## Required-check rollout

Stable contexts are Go lint and formatting, Backend tests, and Frontend lint and tests. Only require the new contexts after this branch is pushed and those exact checks have succeeded remotely. Keep the existing PR-only main protection and bypass policy unchanged. Local success is not proof that GitHub Actions has run; remote check enforcement remains a separate activation step. This work does not implement the pending author-specific owner approval policy.
