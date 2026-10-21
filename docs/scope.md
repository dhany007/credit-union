# Scope and Glossary

## Confirmed context

- One cooperative operates multiple KCPs.
- A person may have only one membership across the cooperative and may not register again at another KCP.
- Each member has one home KCP and one Phase 1 savings account with its own cooperative-wide account number.
- Deposits, withdrawals, loan applications, and installment payments are available at every KCP, subject to the applicable product rules and staff authority.
- Each loan application has one responsible KCP for file management and loan monitoring.
- Loan approval follows an authority matrix based on amount, product, and risk.
- Workflow responsibilities are assigned through staff roles.
- Initial roles are Member, Teller, and Supervisor. Teller includes member-data entry; there is no separate initial Admin role.
- The backend uses Go. The frontend uses Next.js.
- Savings interest uses an annual rate selected for the entire eligible balance, calculated daily and credited monthly. Global base-rate configuration and Supervisor-managed bands are deferred to Phase 2; final rates and detailed calculation conventions remain open.
- The initial deposit amount is IDR 50,000; its application to account opening or membership contributions requires confirmation.

## Glossary

| Term | Meaning |
| --- | --- |
| Cooperative | The single organization operating this system |
| KCP | A service branch within the cooperative; not a separate cooperative |
| Savings account number | A unique savings identifier assigned at registration and usable at every KCP; separate from membership number and NIK |
| Home KCP | The KCP associated with a member's registration |
| Servicing KCP | The KCP processing a particular transaction |
| Receiving KCP | The KCP receiving a loan application |
| Responsible KCP | The single KCP assigned to manage a loan application and monitor its resulting loan |
| Role | A set of permitted staff actions |
| Access scope | The KCPs or records on which a staff member may perform permitted actions |
| Authority matrix | Rules determining approval authority by amount, product, and risk |
| Ledger | The financial record of posted journal entries |
| Reversal | A new posting that offsets an earlier posting while preserving its history |

## Confirmed initial MVP

The project owner narrowed Phase 1 to four services:

- Member registration with cooperative-wide duplicate prevention using KTP/NIK.
- Cash deposits into the same member account at any KCP.
- Cash withdrawals at any KCP, subject to balance and permission checks.
- Basic lending: application and Supervisor approval/rejection only. Disbursement is deferred until lending policy is agreed.

Provide simple Teller/Supervisor screens and retain member home KCP, servicing KCP, and loan responsible KCP. Member self-service is not required. Detailed permissions and lending calculations still need targeted decisions.

Use one initial savings product and one initial loan product as the implementation proposal. Basic transaction history, receipts/references, exact money calculations, atomic journals, duplicate protection, and role checks support these services. Avoid advanced configuration engines and comprehensive dashboards.

Basic installment recording is proposed for live lending; confirm its inclusion or an approved external servicing process before loan rollout. The earlier cross-KCP installment rule remains valid whenever repayment service is provided.

## Deferred from Phase 1

- Automated savings interest, daily accrual jobs, monthly interest credits, and rate-band configuration screens. Preserve the accepted business policy for Phase 2.
- General-purpose configurable approval workflows, multiple lending products, advanced collections, and comprehensive analytics.
- Dedicated cash-session and closing dashboards; retain minimum cash records and reconciliation for the MVP.

The earlier detailed documents describe the target system. They are not all initial-release requirements. Follow this scope and the [simplified roadmap](roadmap.md) when selecting work. See [ADR-0008](adr/0008-core-mvp-first.md).

## Deferred unless explicitly added

- Multiple independent cooperatives or a multi-tenant SaaS offering.
- Public member self-service portal or mobile application.
- Offline financial posting and conflict synchronization.
- External payment integrations, automated credit scoring, and advanced collections.
- SHU distribution, payroll, and other non-core modules.

These are scope proposals, not permanent exclusions. Regulatory obligations, required reports, migration, and operational readiness must be assessed before real use.
