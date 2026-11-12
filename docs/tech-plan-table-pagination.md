# Consistent table paging and loading

Status: implementation requested by the owner, 2026-11-11.

- Default displayed page size: 10. Choices: 10, 50, 100, All. All follows the current server filters; fetch server cursor pages sequentially and cancel on filter changes.
- Retain bounded server endpoints and cursor formats. Add optional positive before_id to GET /members, preserving its array response and default latest-100 behavior. This makes the full filtered directory reachable without an unbounded SQL query.
- Reuse shared controls for local arrays and server collections. Fixed table viewport reserves ten data rows, including short/empty pages; header stays visible while the body scrolls. Larger display pages scroll inside that viewport.
- Keep the last successful loan schedule visible, dimmed under a loading overlay during recomputation. Approval and confirmation use only a result matching the current terms; retained rows are display-only. Initial loads and failures remain explicit.
- Cover members, cash history (within its explicitly labelled latest-100 scope), transaction search, loan/application/installment lists, previews, allocations and savings band tables.
- Preserve EN/ID, light/dark, accessible labels, keyboard scrolling, cancellation, filters and printing.
- Validate additive member API pagination via RED/GREEN integration test; test cursor accumulation, frozen transaction dates, cancellation and display page boundaries. Run Go tests/lint and frontend typecheck/lint/tests/build, rebuild/restart affected services, and inspect the UI.
