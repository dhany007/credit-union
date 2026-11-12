# Loan installment payments

- Add a Payments tab to the loan account for both staff roles. Supervisors can preview and record cash payments on active loans. Tellers can read payment history and receipts.
- Use the existing repayment endpoints and enforce Supervisor-only preview and posting. Continue posting cash to the servicing KCP and preserve the responsible KCP.
- Confirm the server-reviewed amount and revisions. Reuse the idempotency key if the response is uncertain, and refresh the account if the server rejects a stale preview.
- Show principal and interest allocation, payment history and printable receipts.
- Validate permission checks, cross-KCP posting, retry, concurrency, rollback and journal balance with PostgreSQL integration tests. Validate frontend types, lint and tests.
