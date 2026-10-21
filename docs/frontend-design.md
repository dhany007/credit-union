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
