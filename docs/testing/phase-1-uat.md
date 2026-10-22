# Phase 1 technical UAT and populated recovery drill

Date: 2026-10-22 (Asia/Jakarta). Application baseline: `v0.1.0`, main commit `f09f39462a037a3eaf14c4ac6b55656fc8b09e41`.

**Result: technical acceptance and local restore correctness PASS. Ready for owner acceptance; this is not approval to operate with real money.** No application implementation was changed for this exercise.

## Isolation and procedure

Docker Desktop was started. Source API/web writers were briefly paused while a populated custom-format PostgreSQL backup and comparison manifest were captured, then resumed in a finally block. The source was verified unchanged during backup and again after all UAT activity.

The backup was restored into a new Docker project, `credit-union-uat-20261022`, with its own network, generated private database password, and new `restore_data` volume. Browser UAT used `http://127.0.0.1:3001`; the main application remained `http://localhost:3000`. API scripts addressed the explicitly named UAT containers. An initial execution was rejected by automatic review over an ambiguous loopback address before any writes; the revised destination was verified on the isolated Docker network.

The UAT API used an explicitly synthetic IDR 1,000,000 approval ceiling; a second isolated API used zero to test policy rejection. The main `APPROVAL_LIMIT_IDR=0` and staff credentials were not changed. All newly registered identities, cash movements, and loan decisions were synthetic and confined to UAT.

After UAT, a second populated backup, now including loan decisions and command-retry outcomes, was restored into a fresh `credit_union_recovery_20261022` database inside the temporary project. Both restores required exact comparison before success was reported.

## Restore and accounting evidence

| Measure | Main source / first restore | Completed UAT / second restore |
| --- | ---: | ---: |
| Members | 15 | 17 |
| Cash transactions | 7,500 | 7,511 |
| Journal lines | 15,000 | 15,022 |
| Loan applications | 0 | 4 |
| Total deposits | IDR 843,685,000 | IDR 943,970,500 |
| Total withdrawals | IDR 731,710,000 | IDR 831,840,000 |
| Total savings | IDR 111,975,000 | IDR 112,130,500 |
| Journal debits | IDR 1,575,395,000 | IDR 1,775,810,500 |
| Journal credits | IDR 1,575,395,000 | IDR 1,775,810,500 |
| KCP Pusat cash | IDR 50,700,000 | IDR 50,800,500 |
| KCP Utara cash | IDR 61,275,000 | IDR 61,330,000 |
| Content mismatches | 0 | 0 |

The source and restored manifests matched all **10 public tables and five sequences**, including the full row contents of members, transactions, journals, users/sessions, loans/decisions, audit events, migration records, and retry responses. Table digests use ordered JSON rows; sequence checks include `last_value` and `is_called`. Schema-only dumps also matched across source, UAT, and recovered databases, including constraints and indexes; only pg_dump's random restrict/unrestrict guards were excluded from that schema comparison.

Independent assertions in [phase-1-reconciliation.sql](sql/phase-1-reconciliation.sql) check:

- Exactly two correctly directed and valued journal lines per transaction, with debit equal to credit.
- Each member's balance equals signed cash movements and savings-liability journal totals.
- Snapshot pairs, nonnegative values, arithmetic, and teller/servicing-KCP attribution.
- Nonnegative closing cash per KCP and global cash equal to total member savings.

The main source's full manifest remained identical after UAT: no new members, financial movements, decisions, session changes, or sequence consumption in the main database. PostgreSQL schema and financial reconciliation passed again after UAT.

The source archive SHA-256 was `54e125a4a6ed02cbbb6fce959c65076278e3e2f5df05355f2df579f82fbf1253`. Normalized schema SHA-256 was `572feed753ff34be475734e087a5ed7e390b35daf96fc8c314c3891f06503bc2`.

Observed archive creation was 1.02 seconds and first pg_restore was 1.03 seconds; the second backup/database-create/restore segment was 3.08 seconds. These are measurements of one small local dataset, excluding total service recovery time. No production RPO/RTO target or capacity claim is established.

## API acceptance — 29 groups passed

| Check | Result |
| --- | --- |
| Anonymous access denied | PASS |
| Incorrect credentials rejected | PASS |
| All staff logins and cookie protections | PASS |
| Registration issues account and posts opening 50000 | PASS |
| Registration retry returns same result | PASS |
| Duplicate NIK at another KCP rejected | PASS |
| Cross-KCP lookup and masked NIK | PASS |
| Cross-KCP deposit and retry without duplicate | PASS |
| Reused key with different payload rejected | PASS |
| Cross-KCP withdrawal preserves home KCP | PASS |
| Insufficient member balance blocked | PASS |
| Invalid/negative/fractional amounts and kind rejected | PASS |
| Supervisor cannot perform teller cash or registration | PASS |
| Invalid write origin rejected | PASS |
| Concurrent withdrawals cannot overdraw | PASS |
| Concurrent duplicate retry posts once | PASS |
| Insufficient KCP cash blocked without member changes | PASS |
| Teller loan approval denied | PASS |
| Zero approval limit blocks approval | PASS |
| Authorized approval and decision retry | PASS |
| Already decided loan cannot be decided again | PASS |
| Amount/risk authority and rejection | PASS |
| Responsible KCP preserved; decisions do not move money | PASS |
| Disbursement unavailable in Phase 1 | PASS |
| Seven movements have complete 0-to-55000 snapshot chain | PASS |
| Account/date filter, cursor binding, ten pages without missing/duplicate rows | PASS |
| Today/last 7/last 30 inclusive dates and timezone | PASS |
| Invalid filters rejected; valid unknown account empty | PASS |
| Logout invalidates all tested sessions | PASS |

## Browser acceptance

| Check | Result |
| --- | --- |
| Teller login and registration with generated account and IDR 50,000 opening receipt | PASS |
| Another KCP finds the same account without changing home KCP | PASS |
| Deposit IDR 75,000; confirmation, receipt, and 50,000 → 125,000 history | PASS |
| Withdraw IDR 25,000; confirmation, receipt, and 125,000 → 100,000 history | PASS |
| Attempt IDR 150,000 withdrawal; error shown, balance remains 100,000 | PASS |
| Loan intake at KCP Utara with KCP Pusat responsible | PASS |
| Supervisor approval with reason under isolated test authority; no payout UI | PASS |
| Combined account/custom dates show exactly the three UI cash movements, including opening 0 → 50,000 | PASS |
| English/Indonesian and light/dark display; switching language preserves applied filter | PASS |
| English/light preferences and authenticated session survive reload | PASS |
| Next/Previous navigate distinct 50-row pages and restore page one | PASS |
| Unknown account shows empty result without stale rows | PASS |
| Reset restores Today/all accounts | PASS |
| Logout returns to login with saved language/theme | PASS |

No browser error-level logs were captured. Screenshots `phase-1-uat-ledger-light.png` and `phase-1-uat-ledger-dark.png` were saved in the task's visualization directory; they are not embedded as machine-specific paths in this repository.

## Atomic rollback and retry recovery

A temporary BEFORE INSERT journal trigger raised an intentional database exception in the isolated UAT database. The cash API returned a sanitized HTTP 500. Every table digest, row count, sequence state, and financial total remained exactly unchanged. The trigger/function were removed in a finally block.

The same command key was then retried after removing the injected fault. Exactly one IDR 500 deposit posted, and a subsequent retry returned its original reference. The API test member closed at IDR 55,500; the UI member closed at IDR 100,000. These values reconcile to the IDR 155,500 increase over the restored starting dataset. The second recovery preserved both the resulting posting and its stored retry response.

## Retained evidence and cleanup

Private archives, source/restored/completed/recovered manifests, API assertions/report, fault before/after manifests, timing records, and the temporary Compose configuration remain ignored under `.local/phase1-uat-20261022/`. Backup files have mode 600; the directory has mode 700. No password, token, raw database dump, or member identity export is committed.

The temporary browser tab, containers, network, and specifically owned UAT volume were removed after evidence capture. Docker engine stopped during the final follow-up check and was restarted; main DB/API/web returned healthy, `http://localhost:3000/api/health` returned `ok`, and the complete main manifest was still identical to the source. No GitHub release or deployment configuration was changed.

## Acceptance boundary and remaining owner decisions

Phase 1's implemented registration, cash service, loan intake/decisions, history, and local recovery paths passed technical UAT. Owner/business sign-off is still outstanding. AC-018's local restore and reconciliation portion is verified; production recovery objectives remain undecided. Current-main automated backend/frontend checks had passed before this documentation-only exercise; full application suites were not rerun, and no application code changed.

Before a real-money pilot: agree the actual approval ceiling, product/accounting assumptions, member identity corrections, cash opening/transfers/closing and shortage handling, an audited correction/reversal procedure, production access/TLS setup, and backup retention/RPO/RTO. Physical cash handover and staff operating procedures require cooperative acceptance. No interest, disbursement, installment service, large-load benchmark, or production security audit was claimed.
