-- Maintenance deployment: old API writers must be stopped before this application-table split.
ALTER TABLE loans RENAME TO loan_applications;
ALTER TABLE loan_applications ADD COLUMN approved_amount bigint,
 ADD COLUMN approved_tenor_months integer, ADD COLUMN loan_interest_policy_id bigint,
 ADD COLUMN approved_interest_rate_bps integer, ADD COLUMN approval_revision bigint NOT NULL DEFAULT 0;
CREATE FUNCTION loan_valid_band_rates(bands jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE WHEN jsonb_typeof(bands) IS DISTINCT FROM 'array' THEN false ELSE NOT EXISTS(
 SELECT 1 FROM jsonb_array_elements(bands) b WHERE
 CASE WHEN jsonb_typeof(b->'rate_bps')='number' THEN
 (b->>'rate_bps')::numeric NOT BETWEEN 300 AND 10000 OR mod((b->>'rate_bps')::numeric,1)<>0
 ELSE true END) END
$$;
CREATE TABLE loan_interest_policies(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 effective_date date NOT NULL, base_rate_bps integer NOT NULL CHECK(base_rate_bps BETWEEN 300 AND 10000),
 bands jsonb NOT NULL CHECK(loan_valid_band_rates(bands)), reason text NOT NULL CHECK(length(reason) BETWEEN 5 AND 500)
);
CREATE INDEX loan_interest_policy_date ON loan_interest_policies(effective_date DESC,id DESC);
ALTER TABLE loan_applications ADD FOREIGN KEY(loan_interest_policy_id) REFERENCES loan_interest_policies(id),
 ADD CHECK((approved_amount IS NULL AND approved_tenor_months IS NULL AND loan_interest_policy_id IS NULL
 AND approved_interest_rate_bps IS NULL) OR (approved_amount IS NOT NULL AND approved_tenor_months IS NOT NULL
 AND approved_interest_rate_bps IS NOT NULL AND approved_amount>0 AND approved_tenor_months BETWEEN 1 AND 120
 AND loan_interest_policy_id IS NOT NULL AND approved_interest_rate_bps BETWEEN 300 AND 10000 AND approval_revision>0));

CREATE TABLE loans(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 loan_number text GENERATED ALWAYS AS ('LN-'||id::text) STORED UNIQUE,
 application_id bigint NOT NULL UNIQUE REFERENCES loan_applications(id),
 member_id bigint NOT NULL REFERENCES members(id),
 responsible_branch_id integer NOT NULL REFERENCES branches(id),
 disbursing_branch_id integer NOT NULL REFERENCES branches(id),
 principal_amount bigint NOT NULL CHECK(principal_amount>0),
 disbursement_transaction_id text NOT NULL UNIQUE REFERENCES transactions(id),
 disbursed_at timestamptz NOT NULL,
 status text NOT NULL DEFAULT 'active' CHECK(status IN ('active','paid_off')),
 closed_at timestamptz,
 payment_revision bigint NOT NULL DEFAULT 1 CHECK(payment_revision>0),
 verification_note text NOT NULL CHECK(length(verification_note) BETWEEN 5 AND 500),
 recipient_verified boolean NOT NULL CHECK(recipient_verified),
 agreement_signed boolean NOT NULL CHECK(agreement_signed),
 conditions_verified boolean NOT NULL CHECK(conditions_verified)
);
CREATE INDEX loans_account_member ON loans(member_id,id DESC);
CREATE TABLE loan_schedule_versions(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, loan_id bigint NOT NULL REFERENCES loans(id),
 version_number integer NOT NULL CHECK(version_number>0),
 is_current_version boolean NOT NULL DEFAULT false,
 version_type text NOT NULL CHECK(version_type IN ('initial','restructure')),
 effective_date date NOT NULL, starting_principal bigint NOT NULL CHECK(starting_principal>0),
 loan_interest_policy_id bigint NOT NULL REFERENCES loan_interest_policies(id),
 interest_rate_bps integer NOT NULL CHECK(interest_rate_bps BETWEEN 300 AND 10000),
 interest_method text NOT NULL DEFAULT 'annuity' CHECK(interest_method='annuity'),
 tenor_months integer NOT NULL CHECK(tenor_months BETWEEN 1 AND 120),
 first_due_date date NOT NULL,
 status text NOT NULL CHECK(status IN ('draft','active','superseded')),
 reason text NOT NULL, approved_by bigint NOT NULL REFERENCES users(id), approved_at timestamptz NOT NULL,
 UNIQUE(loan_id,version_number),UNIQUE(id,loan_id),
 CHECK(NOT is_current_version OR status='active')
);
CREATE UNIQUE INDEX loan_one_current_version ON loan_schedule_versions(loan_id) WHERE is_current_version;
CREATE TABLE loan_schedules(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 loan_id bigint NOT NULL REFERENCES loans(id),
 schedule_version_id bigint NOT NULL,
 installment_number integer NOT NULL CHECK(installment_number>0), due_date date NOT NULL,
 principal_due bigint NOT NULL CHECK(principal_due>0), interest_due bigint NOT NULL CHECK(interest_due>=0),
 UNIQUE(schedule_version_id,installment_number),UNIQUE(id,loan_id),
 FOREIGN KEY(schedule_version_id,loan_id) REFERENCES loan_schedule_versions(id,loan_id)
);
CREATE TABLE loan_repayments(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 repayment_number text GENERATED ALWAYS AS ('RP-'||id::text) STORED UNIQUE,
 loan_id bigint NOT NULL REFERENCES loans(id), transaction_id text NOT NULL UNIQUE REFERENCES transactions(id),
 servicing_branch_id integer NOT NULL REFERENCES branches(id),
 amount bigint NOT NULL CHECK(amount>0),paid_at timestamptz NOT NULL,note text NOT NULL DEFAULT '',
 UNIQUE(id,loan_id)
);
CREATE TABLE loan_repayment_allocations(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 loan_id bigint NOT NULL REFERENCES loans(id),
 repayment_id bigint NOT NULL, schedule_id bigint NOT NULL,
 principal_amount bigint NOT NULL CHECK(principal_amount>=0),
 interest_amount bigint NOT NULL CHECK(interest_amount>=0),
 CHECK(principal_amount+interest_amount>0),
 UNIQUE(repayment_id,schedule_id),
 FOREIGN KEY(repayment_id,loan_id) REFERENCES loan_repayments(id,loan_id),
 FOREIGN KEY(schedule_id,loan_id) REFERENCES loan_schedules(id,loan_id)
);
CREATE INDEX loan_allocation_schedule ON loan_repayment_allocations(schedule_id);
CREATE INDEX loan_repayment_history ON loan_repayments(loan_id,id DESC);
ALTER TABLE transactions DROP CONSTRAINT transactions_kind_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_kind_check
 CHECK(kind IN ('deposit','withdrawal','interest_credit','loan_disbursement','loan_repayment'));
ALTER TABLE transactions DROP CONSTRAINT transactions_actor_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_actor_check CHECK(
 (kind='interest_credit' AND actor_id IS NULL) OR (kind<>'interest_credit' AND actor_id IS NOT NULL));
ALTER TABLE transactions DROP CONSTRAINT transactions_balance_snapshot_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_balance_snapshot_check CHECK(
 (kind IN ('loan_disbursement','loan_repayment') AND balance_before IS NULL AND balance_after IS NULL)
 OR (kind IN ('deposit','withdrawal') AND balance_before IS NULL AND balance_after IS NULL)
 OR (kind IN ('deposit','withdrawal','interest_credit') AND balance_before>=0 AND balance_after>=0
 AND balance_before IS NOT NULL AND balance_after IS NOT NULL
 AND balance_after-balance_before=CASE kind WHEN 'withdrawal' THEN -amount ELSE amount END));
ALTER TABLE journal_lines DROP CONSTRAINT journal_lines_account_check;
ALTER TABLE journal_lines ADD CONSTRAINT journal_lines_account_check CHECK(account IN(
 'cash','member_savings','savings_interest_expense','loan_principal_receivable','loan_interest_income'));
SELECT cu_install_audit();
SELECT cu_attach_audit();
DO $$ DECLARE name text; BEGIN
 FOREACH name IN ARRAY ARRAY['transactions','journal_lines','requests','audit_events','interest_policies',
 'interest_days','interest_daily','interest_monthly','loan_interest_policies','loans',
 'loan_schedule_versions','loan_schedules','loan_repayments','loan_repayment_allocations'] LOOP
  EXECUTE format('ALTER TABLE %I ADD CONSTRAINT cu_financial_history CHECK(is_active AND deleted_at IS NULL)',name);
 END LOOP;
END $$;
