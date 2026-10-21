# Frontend language and theme verification

Date: 2026-10-21. Environment: local Docker Compose, Next.js production standalone frontend at `http://localhost:3000`.

## Completed checks

- `npm run typecheck`: passed.
- Docker frontend build: passed, including TypeScript and static page generation.
- Compose startup: database, API, and frontend healthy; initial migration exited 0.
- Browser login page: Indonesian/English copy and controls changed immediately.
- Browser light/dark appearance: both displayed using the existing selected theme tokens. The initial browser system preference selected dark mode.
- Reload: explicit English/light and Indonesian/dark preferences persisted.
- Invalid login: English error changed immediately to Indonesian when switching language, without resubmitting credentials.
- Authenticated Teller member, cash, and loan screens: translated navigation, form labels, risk options, currency, policy notices, and empty states verified.
- Unsaved member-form text remained present after changing language; the draft was canceled without submission.
- No member registration, cash posting, loan submission, or loan decision was performed for this frontend check.

The repository has no component/E2E test runner dependency; these checks used the actual local browser rather than adding a new testing stack. Financial behavior remains covered by the existing Go integration tests; they were not rerun for this frontend-only change.
