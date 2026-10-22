# Frontend Design Reference

Status: Confirmed visual direction  
Date: 2026-10-21

Use shadcn/ui in the Next.js frontend with the project-owner-selected [Zen Inspired Theme on tweakcn](https://tweakcn.com/themes/cmlm03etv000204lh15608kec).

Theme ID: `cmlm03etv000204lh15608kec`.

## Implementation guidance

- Import the selected theme's actual exported CSS variables and font settings when setting up the frontend. Do not approximate its palette or substitute another tweakcn theme.
- Reuse semantic tokens for backgrounds, text, borders, primary actions, focus states, radius, and shadows. Keep theme definitions in one shared stylesheet.
- Use shadcn/ui components for forms, buttons, tables, dialogs, and feedback; keep customization small.
- Apply the theme to practical Teller and Supervisor screens for registration, deposits, withdrawals, and lending. The theme reference does not add marketing pages, charts, or dashboards to the MVP.
- Ensure monetary amounts, balances, errors, keyboard focus, and confirmation details remain readable. Status meaning must not depend on color alone.

## Verification at implementation time

The linked page identifies the theme as Zen Inspired Theme. The exact light/dark theme tokens are now captured in `frontend/theme/zen-inspired.json` and applied in `frontend/app/globals.css`; fonts are bundled via Fontsource packages. The frontend has been implemented for local review. Verify the implemented screens against the theme preview and check responsive forms and tables.

## Language and appearance preferences

Confirmed requirement: Phase 1 supports Bahasa Indonesia (`id`) and English (`en`), with light and dark modes using the existing exact Zen Inspired tokens.

- Language and theme controls are available before sign-in and in the staff header. They remain keyboard accessible and expose their names/current state to assistive technology.
- Bahasa Indonesia is the initial language. The initial theme follows the browser's system preference until the user explicitly selects light or dark.
- Preferences are stored per browser in `credit-union.locale` and `credit-union.theme`. They persist across reloads and sign-out, and synchronize between tabs. If browser storage is unavailable, controls still work for the current page.
- `frontend/lib/messages.ts` contains typed messages and interpolation for labels, empty states, statuses, and success notices. `frontend/lib/api-errors.ts` adapts known public API errors without changing the backend contract.
- Date and currency formatting follows the selected locale; currency remains IDR and financial payloads remain whole rupiah. Member names, KCP names, notes, and other entered data are not translated.
- Changing language does not reset a draft form or submit a financial operation. Existing feedback is rendered in the newly selected language.
- The document language is updated for screen readers. A trusted pre-paint script applies the saved/system theme, with hydration suppression limited to the root attributes that it changes.

Implementation uses React context and the existing CSS tokens; no additional runtime i18n/theme dependency or localized URL routing is introduced. See [frontend preference verification](testing/frontend-preferences.md).

## Cash-service navigation

Cash Deposit (Setor Tunai) and Cash Withdrawal (Tarik Tunai) are separate navigation entries and member-detail shortcuts. Both render the same cash form and use the existing cash API. The active menu determines the transaction kind; there is no second mode selector inside the form. Switching services clears the amount, note, confirmation and notices while retaining the selected account. Navigation is disabled while an operation is pending. Labels use the EN/ID dictionary and existing light/dark tokens.
