\set ON_ERROR_STOP on
BEGIN;
SET LOCAL TIME ZONE 'UTC';
CREATE TEMP TABLE evidence_tables(name text,rows bigint,digest text) ON COMMIT DROP;
CREATE TEMP TABLE evidence_sequences(name text,last_value bigint,is_called boolean) ON COMMIT DROP;
DO $$ DECLARE obj record; n bigint; digest text; lastval bigint; called boolean;
BEGIN
 FOR obj IN SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename LOOP
  EXECUTE format('SELECT count(*),md5(coalesce(string_agg(row_json::text,E''\n'' ORDER BY row_json::text),'''')) FROM (SELECT to_jsonb(t) row_json FROM public.%I t) r',obj.tablename) INTO n,digest;
  INSERT INTO evidence_tables VALUES(obj.tablename,n,digest);
 END LOOP;
 FOR obj IN SELECT sequencename FROM pg_sequences WHERE schemaname='public' ORDER BY sequencename LOOP
  EXECUTE format('SELECT last_value,is_called FROM public.%I',obj.sequencename) INTO lastval,called;
  INSERT INTO evidence_sequences VALUES(obj.sequencename,lastval,called);
 END LOOP;
 IF EXISTS(SELECT t.id FROM transactions t LEFT JOIN journal_lines j ON j.transaction_id=t.id
 GROUP BY t.id,t.kind,t.amount HAVING count(j.*)<>2 OR sum(j.debit)<>sum(j.credit)
 OR count(*) FILTER(WHERE j.line_no=1 AND j.debit=t.amount AND j.credit=0 AND j.account=CASE t.kind WHEN 'deposit' THEN 'cash' ELSE 'member_savings' END)<>1
 OR count(*) FILTER(WHERE j.line_no=2 AND j.credit=t.amount AND j.debit=0 AND j.account=CASE t.kind WHEN 'deposit' THEN 'member_savings' ELSE 'cash' END)<>1) THEN
 RAISE EXCEPTION 'Transaction journal mismatch';END IF;
 IF EXISTS(SELECT m.id FROM members m WHERE m.balance<0 OR m.balance<>coalesce((SELECT sum(CASE kind WHEN 'deposit' THEN amount ELSE -amount END) FROM transactions t WHERE t.member_id=m.id),0)
 OR m.balance<>coalesce((SELECT sum(j.credit-j.debit) FROM journal_lines j JOIN transactions t ON t.id=j.transaction_id WHERE j.account='member_savings' AND t.member_id=m.id),0)) THEN
 RAISE EXCEPTION 'Member balance/cashflow/savings liability mismatch';END IF;
 IF EXISTS(SELECT 1 FROM transactions t JOIN users u ON u.id=t.actor_id WHERE u.branch_id<>t.branch_id OR u.role<>'teller'
 OR (t.balance_before IS NULL)<>(t.balance_after IS NULL)
 OR (t.balance_before IS NOT NULL AND (t.balance_before<0 OR t.balance_after<0 OR t.balance_after-t.balance_before<>CASE t.kind WHEN 'deposit' THEN t.amount ELSE -t.amount END))) THEN
 RAISE EXCEPTION 'Snapshot/actor attribution mismatch';END IF;
 IF EXISTS(SELECT b.id FROM branches b WHERE coalesce((SELECT sum(j.debit-j.credit) FROM journal_lines j JOIN transactions t ON t.id=j.transaction_id WHERE j.account='cash' AND t.branch_id=b.id),0)<0) THEN
 RAISE EXCEPTION 'Negative KCP cash';END IF;
 IF (SELECT coalesce(sum(balance),0) FROM members)<>(SELECT coalesce(sum(debit-credit),0) FROM journal_lines WHERE account='cash') THEN
 RAISE EXCEPTION 'Global cash/savings mismatch';END IF;
END $$;
SELECT jsonb_build_object(
 'tables',(SELECT jsonb_object_agg(name,jsonb_build_object('rows',rows,'digest',digest)) FROM evidence_tables),
 'sequences',(SELECT jsonb_object_agg(name,jsonb_build_object('last_value',last_value,'is_called',is_called)) FROM evidence_sequences),
 'financial',jsonb_build_object(
 'members',(SELECT count(*) FROM members),'transactions',(SELECT count(*) FROM transactions),
 'journal_lines',(SELECT count(*) FROM journal_lines),'loans',(SELECT count(*) FROM loans),
 'total_savings',(SELECT coalesce(sum(balance),0) FROM members),
 'deposits',(SELECT coalesce(sum(amount),0) FROM transactions WHERE kind='deposit'),
 'withdrawals',(SELECT coalesce(sum(amount),0) FROM transactions WHERE kind='withdrawal'),
 'debits',(SELECT coalesce(sum(debit),0) FROM journal_lines),
 'credits',(SELECT coalesce(sum(credit),0) FROM journal_lines),
 'unknown_snapshots',(SELECT count(*) FROM transactions WHERE balance_before IS NULL),
 'per_member',(SELECT jsonb_agg(jsonb_build_object('id',m.id,'account',m.account_number,'balance',m.balance,'transactions',(SELECT count(*) FROM transactions t WHERE t.member_id=m.id)) ORDER BY m.id) FROM members m),
 'per_branch',(SELECT jsonb_agg(jsonb_build_object('id',b.id,'cash',coalesce((SELECT sum(j.debit-j.credit) FROM journal_lines j JOIN transactions t ON t.id=j.transaction_id WHERE j.account='cash' AND t.branch_id=b.id),0)) ORDER BY b.id) FROM branches b)
 ));
ROLLBACK;
