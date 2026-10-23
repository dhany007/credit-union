CREATE TABLE interest_policies (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 effective_date date NOT NULL,
 base_rate_bps integer NOT NULL CHECK(base_rate_bps BETWEEN 0 AND 10000),
 bands jsonb NOT NULL CHECK(jsonb_typeof(bands)='array'),
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 500),
 created_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX interest_policies_effective_idx ON interest_policies(effective_date DESC,id DESC);
CREATE TABLE interest_days (
 business_date date PRIMARY KEY,
 eligible_account_count integer NOT NULL CHECK(eligible_account_count>=0),
 policy_id bigint NOT NULL REFERENCES interest_policies(id),
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE interest_daily (
 member_id bigint NOT NULL REFERENCES members(id),
 business_date date NOT NULL,
 policy_id bigint NOT NULL REFERENCES interest_policies(id),
 closing_balance bigint NOT NULL CHECK(closing_balance>=0),
 rate_bps integer NOT NULL CHECK(rate_bps BETWEEN 0 AND 10000),
 numerator numeric(40,0) NOT NULL CHECK(numerator>=0 AND numerator=closing_balance::numeric*rate_bps),
 calculated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(member_id,business_date),
 FOREIGN KEY(business_date) REFERENCES interest_days(business_date) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE interest_monthly (
 member_id bigint NOT NULL REFERENCES members(id),
 period_start date NOT NULL CHECK(extract(day FROM period_start)=1),
 eligible_days integer NOT NULL CHECK(eligible_days BETWEEN 1 AND 31),
 numerator_total numeric(40,0) NOT NULL CHECK(numerator_total>=0),
 credit_amount bigint NOT NULL CHECK(credit_amount>=0),
 transaction_id text UNIQUE REFERENCES transactions(id),
 posted_at timestamptz NOT NULL,
 PRIMARY KEY(member_id,period_start),
 CHECK((credit_amount=0 AND transaction_id IS NULL) OR (credit_amount>0 AND transaction_id IS NOT NULL))
);
ALTER TABLE transactions DROP CONSTRAINT transactions_kind_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_kind_check CHECK(kind IN ('deposit','withdrawal','interest_credit'));
ALTER TABLE transactions ALTER COLUMN actor_id DROP NOT NULL;
ALTER TABLE transactions ADD CONSTRAINT transactions_actor_check CHECK(
 (kind='interest_credit' AND actor_id IS NULL) OR (kind IN ('deposit','withdrawal') AND actor_id IS NOT NULL));
ALTER TABLE transactions DROP CONSTRAINT transactions_balance_snapshot_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_balance_snapshot_check CHECK (
 (balance_before IS NULL AND balance_after IS NULL AND kind<>'interest_credit')
 OR (balance_before IS NOT NULL AND balance_after IS NOT NULL AND balance_before>=0 AND balance_after>=0
 AND ((kind IN ('deposit','interest_credit') AND balance_after-balance_before=amount)
 OR (kind='withdrawal' AND balance_before-balance_after=amount)))
);
ALTER TABLE journal_lines DROP CONSTRAINT journal_lines_account_check;
ALTER TABLE journal_lines ADD CONSTRAINT journal_lines_account_check
 CHECK(account IN ('cash','member_savings','savings_interest_expense'));
ALTER TABLE audit_events ALTER COLUMN actor_id DROP NOT NULL;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_actor_check
 CHECK(actor_id IS NOT NULL OR action IN ('interest_credit','interest_settlement'));
