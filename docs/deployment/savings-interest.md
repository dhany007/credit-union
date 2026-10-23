# Local savings interest operations

This runbook covers the accepted Phase 2 local feature, not real-money approval.

1. Back up the populated database before migration. Keep the main volume; never reset it.
2. Build and deploy the API/web together. Migration 0004 expands transaction/journal types and adds four interest tables. Existing migrations and historical money rows remain unchanged.
3. Keep SAVINGS_INTEREST_ENABLED=false until every writer/reader is compatible. The example environment defaults to false.
4. Sign in as Supervisor. Open Savings interest, review the global rate and bands, choose tomorrow or later, provide a reason, and save. The first version fixes activation. Default form values are not silently saved by deployment.
5. For local execution, set SAVINGS_INTEREST_ENABLED=true in the ignored .env.docker and run make up (or recreate the built API). There is no separate worker service.
6. Check the status screen and API logs. Closing starts after 00:05 Jakarta; the first calculation runs the day after activation. Monthly credit follows the first complete/partial eligible month. Restart catches up; do not insert historical transactions to test production behavior.
7. Before and after recovery, run testing/sql/phase-1-reconciliation.sql and testing/sql/savings-interest-reconciliation.sql with psql against the explicitly intended database. The first script includes full table/sequence fingerprints and now handles interest; its filename is retained for existing users.

Disabling the scheduler stops new calculations/credits, while compatible cash operations remain available. Re-enable to catch up. Preserve policy, completed-day, daily, monthly, journal, transaction and audit data together. Do not roll back to a pre-interest binary or delete credited entries.

The status failure class and structured logs distinguish timeout, lock timeout, incomplete/rejected work and dependency failures. Inspect operation/date-or-period and completed/failed counts. Reconcile and correct the underlying issue; retry uses the same account/month identity. Persistent failed accounts rotate through bounded batches rather than blocking healthy accounts.

Local approved rates are 3%, 4%, 4.5%, 5% on whole-balance bands below 1m, below 10m, below 100m and at least 100m; the global fallback is 3%. See ADR-0012 for all rules. Tax/deductions, closure/freeze/correction workflows, real-money cash funding and operating acceptance remain separate gates.
