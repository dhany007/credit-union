# Loan installment payments

- Add a Payments tab to the loan account for both staff roles. Tellers can preview and record cash payments on active loans. Supervisors can read payment history and receipts.
- Use the existing repayment endpoints and enforce Teller-only preview and posting. Continue posting cash to the servicing KCP and preserve the responsible KCP.
- Confirm the server-reviewed amount and revisions. Reuse the idempotency key if the response is uncertain, and refresh the account if the server rejects a stale preview.
- Show principal and interest allocation, payment history and printable receipts.
- Validate permission checks, cross-KCP posting, retry, concurrency, rollback and journal balance with PostgreSQL integration tests. Validate frontend types, lint and tests.

The Teller workspace requests a preview with `{"next_installment":true}`. The API returns the remaining interest and principal of the oldest unpaid installment, bound to the reviewed loan revision and schedule version. The Teller confirms that amount without editing it. Explicit-amount requests remain supported by the API.
