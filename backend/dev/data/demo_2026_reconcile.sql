-- Run inside a transaction (psql --single-transaction for standalone verification).
\set ON_ERROR_STOP on
SET LOCAL TIME ZONE 'Asia/Jakarta';
DO $$ BEGIN
 IF (SELECT count(*) FROM audit_events WHERE action='demo_seed_completed'
 AND entity_id='demo-2026-15x500-v1')<>1 THEN RAISE EXCEPTION 'Missing or duplicate demo batch marker';END IF;
END $$;
CREATE TEMP TABLE demo_actual_config ON COMMIT DROP AS
 SELECT details,(details->>'cutoff')::timestamptz cutoff FROM audit_events
 WHERE action='demo_seed_completed' AND entity_id='demo-2026-15x500-v1';
CREATE TEMP TABLE demo_actual_members ON COMMIT DROP AS
 SELECT m.*,ids.ordinality::int demo_ordinal FROM demo_actual_config c
 CROSS JOIN LATERAL jsonb_array_elements_text(c.details->'member_ids') WITH ORDINALITY ids(id,ordinality)
 JOIN members m ON m.id=ids.id::bigint;
CREATE TEMP TABLE demo_actual_tx ON COMMIT DROP AS
 SELECT t.*,m.demo_ordinal,CASE u.username WHEN 'teller.a' THEN 1 WHEN 'teller.b' THEN 2 END staff_ordinal
 FROM transactions t JOIN demo_actual_members m ON m.id=t.member_id JOIN users u ON u.id=t.actor_id;

DO $$ BEGIN
 IF (SELECT count(*) FROM demo_actual_members)<>15 OR (SELECT count(DISTINCT id) FROM demo_actual_members)<>15
 OR (SELECT count(*) FROM demo_actual_tx)<>7500 OR EXISTS(
 SELECT member_id FROM demo_actual_tx GROUP BY member_id HAVING count(*)<>500 OR count(DISTINCT kind)<>2
 OR count(DISTINCT branch_id)<>2) THEN RAISE EXCEPTION 'Demo member/count/type/KCP mismatch';END IF;
 IF EXISTS(SELECT 1 FROM demo_actual_members WHERE nik<>lpad(demo_ordinal::text,16,'0')
 OR name NOT LIKE 'DEMO 2026 %' OR account_number !~ '^[1-9][0-9]{9}$') THEN
 RAISE EXCEPTION 'Synthetic identity/account mapping mismatch';END IF;
 IF EXISTS(SELECT 1 FROM demo_actual_tx t JOIN demo_actual_members m ON m.id=t.member_id
 JOIN users u ON u.id=t.actor_id WHERE t.balance_before IS NULL OR t.balance_after IS NULL
 OR t.balance_before<0 OR t.balance_after<0 OR t.balance_after>1000000000000
 OR t.amount<=0 OR t.amount%5000<>0 OR t.created_at<m.created_at
 OR t.created_at<timestamptz '2026-01-01 00:00:00+07'
 OR t.created_at>(SELECT cutoff FROM demo_actual_config)
 OR t.note NOT LIKE 'DEMO demo-2026-15x500-v1 %' OR u.role<>'teller' OR u.branch_id<>t.branch_id
 OR t.staff_ordinal IS NULL
 OR t.balance_after-t.balance_before<>CASE t.kind WHEN 'deposit' THEN t.amount ELSE -t.amount END) THEN
 RAISE EXCEPTION 'Invalid balance snapshot/money/time/actor/metadata';END IF;
 IF EXISTS(WITH ordered AS (
 SELECT t.*,lag(balance_after,1,0::bigint) OVER(PARTITION BY member_id ORDER BY created_at,id) previous_after,
 row_number() OVER(PARTITION BY member_id ORDER BY created_at,id) movement_no FROM demo_actual_tx t)
 SELECT 1 FROM ordered t JOIN demo_actual_members m ON m.id=t.member_id
 WHERE t.balance_before<>t.previous_after OR (t.movement_no=1 AND
 (t.kind<>'deposit' OR t.amount<>50000 OR t.balance_before<>0 OR t.balance_after<>50000
 OR t.branch_id<>m.home_branch_id OR t.created_at<>m.created_at))) THEN
 RAISE EXCEPTION 'Broken ledger chain or invalid opening deposit';END IF;
 IF EXISTS(SELECT 1 FROM demo_actual_members m WHERE m.balance<0 OR m.balance<>(SELECT sum(
 CASE kind WHEN 'deposit' THEN amount ELSE -amount END) FROM demo_actual_tx t WHERE t.member_id=m.id)
 OR m.balance<>(SELECT balance_after FROM demo_actual_tx t WHERE t.member_id=m.id ORDER BY created_at DESC,id DESC LIMIT 1)) THEN
 RAISE EXCEPTION 'Member closing balance does not reconcile';END IF;
 IF EXISTS(SELECT m.id FROM demo_actual_members m CROSS JOIN generate_series(timestamptz '2026-01-01 00:00+07',
 (SELECT date_trunc('month',cutoff) FROM demo_actual_config),interval '1 month') month_start
 WHERE NOT EXISTS(SELECT 1 FROM demo_actual_tx t WHERE t.member_id=m.id
 AND date_trunc('month',t.created_at)=month_start)) THEN RAISE EXCEPTION 'Missing member/month coverage';END IF;
 IF EXISTS(SELECT member_id FROM demo_actual_tx GROUP BY member_id
 HAVING max(created_at)::date<>(SELECT cutoff::date FROM demo_actual_config)) THEN
 RAISE EXCEPTION 'Each member must have a transaction on the cutoff date';END IF;
 IF EXISTS(SELECT t.id FROM demo_actual_tx t LEFT JOIN journal_lines j ON j.transaction_id=t.id
 GROUP BY t.id,t.kind,t.amount HAVING count(j.*)<>2
 OR count(*) FILTER(WHERE j.line_no=1 AND j.debit=t.amount AND j.credit=0
 AND j.account=CASE t.kind WHEN 'deposit' THEN 'cash' ELSE 'member_savings' END)<>1
 OR count(*) FILTER(WHERE j.line_no=2 AND j.debit=0 AND j.credit=t.amount
 AND j.account=CASE t.kind WHEN 'deposit' THEN 'member_savings' ELSE 'cash' END)<>1
 OR sum(j.debit)<>sum(j.credit)) THEN RAISE EXCEPTION 'Journal direction/count/amount mismatch';END IF;
 IF EXISTS(SELECT 1 FROM (SELECT sum(CASE kind WHEN 'deposit' THEN amount ELSE -amount END)
 OVER(PARTITION BY branch_id ORDER BY created_at,id ROWS UNBOUNDED PRECEDING) cash FROM demo_actual_tx) running
 WHERE cash<0) THEN RAISE EXCEPTION 'Historical KCP cash became negative';END IF;
 IF EXISTS(SELECT 1 FROM (SELECT t.branch_id,sum(CASE t.kind WHEN 'deposit' THEN t.amount ELSE -t.amount END) net
 FROM demo_actual_tx t GROUP BY t.branch_id) totals WHERE totals.net<>(SELECT sum(j.debit-j.credit)
 FROM journal_lines j JOIN demo_actual_tx t ON t.id=j.transaction_id
 WHERE j.account='cash' AND t.branch_id=totals.branch_id)) THEN RAISE EXCEPTION 'KCP closing cash mismatch';END IF;
 IF (SELECT count(*) FROM audit_events WHERE details->>'demo_batch'='demo-2026-15x500-v1')<>7516
 OR (SELECT count(*) FROM audit_events WHERE details->>'demo_batch'='demo-2026-15x500-v1'
 AND action IN ('deposit','withdrawal'))<>7500
 OR (SELECT count(*) FROM audit_events WHERE details->>'demo_batch'='demo-2026-15x500-v1'
 AND action='register_member')<>15 THEN RAISE EXCEPTION 'Audit count mismatch';END IF;
 IF EXISTS(SELECT t.id FROM demo_actual_tx t LEFT JOIN audit_events a ON a.entity_id=t.id
 AND a.action=t.kind AND a.actor_id=t.actor_id AND a.details->>'demo_batch'='demo-2026-15x500-v1'
 GROUP BY t.id HAVING count(a.*)<>1) THEN RAISE EXCEPTION 'Cash audit attribution mismatch';END IF;
 IF (SELECT md5(string_agg(id||':'||created_at||':'||kind||':'||amount||':'||balance_before||':'||balance_after
 ||':'||demo_ordinal||':'||staff_ordinal,'|' ORDER BY created_at,id)) FROM demo_actual_tx)
 IS DISTINCT FROM (SELECT details->>'plan_checksum' FROM demo_actual_config) THEN
 RAISE EXCEPTION 'Actual transactions do not match the generated plan checksum';END IF;
END $$;

SELECT 'ALL RECONCILIATION CHECKS PASSED' AS result;
SELECT m.demo_ordinal member,m.name,m.account_number,count(*) transactions,
 count(*) FILTER(WHERE t.kind='deposit') deposits,count(*) FILTER(WHERE t.kind='withdrawal') withdrawals,
 sum(t.amount) FILTER(WHERE t.kind='deposit') total_deposits,
 sum(t.amount) FILTER(WHERE t.kind='withdrawal') total_withdrawals,m.balance closing_balance,
 min(t.created_at)::date first_date,max(t.created_at)::date last_date
 FROM demo_actual_members m JOIN demo_actual_tx t ON t.member_id=m.id
 GROUP BY m.demo_ordinal,m.name,m.account_number,m.balance ORDER BY m.demo_ordinal;
SELECT to_char(created_at,'YYYY-MM') AS period_month,count(*) transactions,count(DISTINCT member_id) members,
 count(*) FILTER(WHERE kind='deposit') deposits,count(*) FILTER(WHERE kind='withdrawal') withdrawals
 FROM demo_actual_tx GROUP BY 1 ORDER BY 1;
SELECT b.name,sum(CASE t.kind WHEN 'deposit' THEN t.amount ELSE -t.amount END) closing_cash
 FROM demo_actual_tx t JOIN branches b ON b.id=t.branch_id GROUP BY b.id,b.name ORDER BY b.id;
SELECT (SELECT count(*) FROM demo_actual_members) members,(SELECT count(*) FROM demo_actual_tx) transactions,
 count(*) journal_lines,sum(debit) debits,sum(credit) credits,
 (SELECT sum(balance) FROM demo_actual_members) total_savings,
 (SELECT sum(amount) FROM demo_actual_tx WHERE kind='deposit') total_deposits,
 (SELECT sum(amount) FROM demo_actual_tx WHERE kind='withdrawal') total_withdrawals
 FROM journal_lines j JOIN demo_actual_tx t ON t.id=j.transaction_id;
