\set ON_ERROR_STOP on
BEGIN;
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM interest_days c WHERE c.eligible_account_count<>
 (SELECT count(*) FROM interest_daily d WHERE d.business_date=c.business_date AND d.policy_id=c.policy_id)) THEN
 RAISE EXCEPTION 'Daily interest marker/count mismatch';END IF;
 IF EXISTS(SELECT 1 FROM interest_monthly m WHERE m.numerator_total<>
 (SELECT COALESCE(sum(d.numerator),0) FROM interest_daily d WHERE d.member_id=m.member_id
 AND d.business_date>=m.period_start AND d.business_date<m.period_start+interval '1 month')
 OR m.eligible_days<>(SELECT count(*) FROM interest_daily d WHERE d.member_id=m.member_id
 AND d.business_date>=m.period_start AND d.business_date<m.period_start+interval '1 month')
 OR m.credit_amount<>floor((m.numerator_total+1825000)/3650000)) THEN
 RAISE EXCEPTION 'Monthly interest does not match exact daily amounts';END IF;
 IF EXISTS(SELECT 1 FROM interest_monthly m LEFT JOIN transactions t ON t.id=m.transaction_id
 WHERE m.credit_amount>0 AND (t.id IS NULL OR t.kind<>'interest_credit' OR t.member_id<>m.member_id
 OR t.amount<>m.credit_amount OR t.created_at<>m.posted_at)) THEN
 RAISE EXCEPTION 'Monthly credit/posting mismatch';END IF;
 IF EXISTS(SELECT 1 FROM transactions t WHERE t.kind='interest_credit'
 AND NOT EXISTS(SELECT 1 FROM interest_monthly m WHERE m.transaction_id=t.id)) THEN
 RAISE EXCEPTION 'Interest transaction without settlement';END IF;
END $$;
SELECT 'Interest daily, monthly and posting reconciliation PASS' result;
ROLLBACK;
