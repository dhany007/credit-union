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

 IF EXISTS(
 WITH components AS (
 SELECT r.transaction_id,COALESCE(sum(a.principal_amount),0) principal,COALESCE(sum(a.interest_amount),0) interest
 FROM loan_repayments r LEFT JOIN loan_repayment_allocations a ON a.repayment_id=r.id GROUP BY r.transaction_id
 ), expected AS (
 SELECT id transaction_id,1 line_no,CASE kind
 WHEN 'deposit' THEN 'cash' WHEN 'withdrawal' THEN 'member_savings'
 WHEN 'interest_credit' THEN 'savings_interest_expense'
 WHEN 'loan_disbursement' THEN 'loan_principal_receivable' WHEN 'loan_repayment' THEN 'cash' END account,
 amount debit,0::bigint credit FROM transactions
 UNION ALL
 SELECT id,2,CASE kind WHEN 'withdrawal' THEN 'cash' WHEN 'loan_disbursement' THEN 'cash'
 ELSE 'member_savings' END,0,amount FROM transactions WHERE kind<>'loan_repayment'
 UNION ALL SELECT transaction_id,2,'loan_principal_receivable',0,principal FROM components WHERE principal>0
 UNION ALL SELECT transaction_id,CASE WHEN principal>0 THEN 3 ELSE 2 END,'loan_interest_income',0,interest
 FROM components WHERE interest>0
 )
 SELECT 1 FROM expected e FULL JOIN journal_lines j USING(transaction_id,line_no)
 WHERE e.transaction_id IS NULL OR j.transaction_id IS NULL
 OR e.account<>j.account OR e.debit<>j.debit OR e.credit<>j.credit
 ) THEN RAISE EXCEPTION 'Kind-specific transaction journal mismatch';END IF;

 IF EXISTS(SELECT m.id FROM members m WHERE m.balance<0 OR m.balance<>COALESCE((
 SELECT sum(CASE kind WHEN 'withdrawal' THEN -amount ELSE amount END) FROM transactions t
 WHERE t.member_id=m.id AND t.kind IN ('deposit','withdrawal','interest_credit')),0)
 OR m.balance<>COALESCE((SELECT sum(j.credit-j.debit) FROM journal_lines j
 JOIN transactions t ON t.id=j.transaction_id WHERE j.account='member_savings' AND t.member_id=m.id),0))
 THEN RAISE EXCEPTION 'Member savings balance mismatch';END IF;

 IF EXISTS(SELECT 1 FROM transactions t LEFT JOIN users u ON u.id=t.actor_id JOIN members m ON m.id=t.member_id
 WHERE (t.kind IN ('deposit','withdrawal','loan_disbursement','loan_repayment')
 AND (u.id IS NULL OR u.branch_id<>t.branch_id OR (t.kind IN ('deposit','withdrawal') AND u.role<>'teller')
 OR (t.kind IN ('loan_disbursement','loan_repayment') AND u.role NOT IN ('teller','supervisor'))))
 OR (t.kind='interest_credit' AND (t.actor_id IS NOT NULL OR t.branch_id<>m.home_branch_id))
 OR (t.balance_before IS NULL)<>(t.balance_after IS NULL)
 OR (t.kind IN ('loan_disbursement','loan_repayment') AND t.balance_before IS NOT NULL)
 OR (t.balance_before IS NOT NULL AND (t.balance_before<0 OR t.balance_after<0 OR
 t.balance_after-t.balance_before<>CASE t.kind WHEN 'withdrawal' THEN -t.amount ELSE t.amount END)))
 THEN RAISE EXCEPTION 'Snapshot/actor attribution mismatch';END IF;

 IF EXISTS(SELECT 1 FROM loans l JOIN transactions t ON t.id=l.disbursement_transaction_id
 WHERE t.kind<>'loan_disbursement' OR t.amount<>l.principal_amount OR t.member_id<>l.member_id
 OR t.branch_id<>l.disbursing_branch_id OR t.created_at<>l.disbursed_at)
 OR EXISTS(SELECT 1 FROM loan_repayments r JOIN transactions t ON t.id=r.transaction_id JOIN loans l ON l.id=r.loan_id
 WHERE t.kind<>'loan_repayment' OR t.amount<>r.amount OR t.member_id<>l.member_id
 OR t.branch_id<>r.servicing_branch_id OR t.created_at<>r.paid_at)
 OR EXISTS(SELECT 1 FROM transactions t WHERE
 (t.kind='loan_disbursement' AND NOT EXISTS(SELECT 1 FROM loans l WHERE l.disbursement_transaction_id=t.id))
 OR (t.kind='loan_repayment' AND NOT EXISTS(SELECT 1 FROM loan_repayments r WHERE r.transaction_id=t.id)))
 THEN RAISE EXCEPTION 'Loan financial linkage mismatch';END IF;

 IF EXISTS(SELECT 1 FROM loan_repayments r WHERE r.amount<>COALESCE((
 SELECT sum(a.principal_amount+a.interest_amount) FROM loan_repayment_allocations a WHERE a.repayment_id=r.id),0))
 OR EXISTS(SELECT 1 FROM loan_schedules s WHERE s.principal_due<COALESCE((
 SELECT sum(a.principal_amount) FROM loan_repayment_allocations a WHERE a.schedule_id=s.id),0)
 OR s.interest_due<COALESCE((SELECT sum(a.interest_amount) FROM loan_repayment_allocations a WHERE a.schedule_id=s.id),0))
 OR EXISTS(SELECT 1 FROM loans l WHERE l.principal_amount<COALESCE((
 SELECT sum(a.principal_amount) FROM loan_repayment_allocations a WHERE a.loan_id=l.id),0))
 THEN RAISE EXCEPTION 'Loan allocation mismatch';END IF;

 IF EXISTS(SELECT 1 FROM loans l WHERE
 (SELECT count(*) FROM loan_schedule_versions v WHERE v.loan_id=l.id AND v.is_current_version)<>1)
 OR EXISTS(SELECT 1 FROM loan_schedule_versions v WHERE v.starting_principal<>COALESCE((
 SELECT sum(s.principal_due) FROM loan_schedules s WHERE s.schedule_version_id=v.id),0)
 OR v.tenor_months<>(SELECT count(*) FROM loan_schedules s WHERE s.schedule_version_id=v.id))
 THEN RAISE EXCEPTION 'Loan schedule/version mismatch';END IF;

 IF (SELECT COALESCE(sum(debit-credit),0) FROM journal_lines WHERE account='loan_principal_receivable')
 <>(SELECT COALESCE(sum(principal_amount),0) FROM loans)-
 (SELECT COALESCE(sum(principal_amount),0) FROM loan_repayment_allocations)
 OR (SELECT COALESCE(sum(credit-debit),0) FROM journal_lines WHERE account='loan_interest_income')
 <>(SELECT COALESCE(sum(interest_amount),0) FROM loan_repayment_allocations)
 THEN RAISE EXCEPTION 'Loan subledger/GL mismatch';END IF;

 IF EXISTS(SELECT b.id FROM branches b WHERE COALESCE((SELECT sum(j.debit-j.credit) FROM journal_lines j
 JOIN transactions t ON t.id=j.transaction_id WHERE j.account='cash' AND t.branch_id=b.id),0)<0)
 THEN RAISE EXCEPTION 'Negative KCP cash';END IF;
 IF (SELECT COALESCE(sum(debit-credit),0) FROM journal_lines
 WHERE account IN ('cash','loan_principal_receivable','savings_interest_expense'))
 <>(SELECT COALESCE(sum(balance),0) FROM members)+
 (SELECT COALESCE(sum(credit-debit),0) FROM journal_lines WHERE account='loan_interest_income')
 THEN RAISE EXCEPTION 'Global supported-event ledger mismatch';END IF;
END $$;
SELECT jsonb_build_object(
 'tables',(SELECT jsonb_object_agg(name,jsonb_build_object('rows',rows,'digest',digest)) FROM evidence_tables),
 'sequences',(SELECT jsonb_object_agg(name,jsonb_build_object('last_value',last_value,'is_called',is_called)) FROM evidence_sequences),
 'financial',jsonb_build_object(
 'members',(SELECT count(*) FROM members),'transactions',(SELECT count(*) FROM transactions),
 'journal_lines',(SELECT count(*) FROM journal_lines),'loan_applications',(SELECT count(*) FROM loan_applications),
 'loans',(SELECT count(*) FROM loans),'repayments',(SELECT count(*) FROM loan_repayments),
 'total_savings',(SELECT COALESCE(sum(balance),0) FROM members),
 'deposits',(SELECT COALESCE(sum(amount),0) FROM transactions WHERE kind='deposit'),
 'withdrawals',(SELECT COALESCE(sum(amount),0) FROM transactions WHERE kind='withdrawal'),
 'debits',(SELECT COALESCE(sum(debit),0) FROM journal_lines),
 'credits',(SELECT COALESCE(sum(credit),0) FROM journal_lines),
 'unknown_snapshots',(SELECT count(*) FROM transactions
 WHERE balance_before IS NULL AND kind IN ('deposit','withdrawal','interest_credit')),
 'loan_principal_outstanding',(SELECT COALESCE(sum(debit-credit),0) FROM journal_lines
 WHERE account='loan_principal_receivable'),
 'loan_interest_income',(SELECT COALESCE(sum(credit-debit),0) FROM journal_lines WHERE account='loan_interest_income'),
 'per_member',(SELECT jsonb_agg(jsonb_build_object('id',m.id,'account',m.account_number,'balance',m.balance,
 'transactions',(SELECT count(*) FROM transactions t WHERE t.member_id=m.id)) ORDER BY m.id) FROM members m),
 'per_branch',(SELECT jsonb_agg(jsonb_build_object('id',b.id,'cash',COALESCE((
 SELECT sum(j.debit-j.credit) FROM journal_lines j JOIN transactions t ON t.id=j.transaction_id
 WHERE j.account='cash' AND t.branch_id=b.id),0)) ORDER BY b.id) FROM branches b)
 ));
ROLLBACK;
