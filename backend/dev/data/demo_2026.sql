\set ON_ERROR_STOP on
\if :{?local_demo}
\else
  \set local_demo false
\endif
\if :local_demo
\else
  \echo 'Refusing: pass local_demo=true only for the local disposable/demo application database.'
  DO $$ BEGIN RAISE EXCEPTION 'Required local demo flags missing'; END $$;
\endif
\if :{?apply}
\else
  \set apply false
\endif
\if :{?cutoff}
\else
  \echo 'Refusing: pass one captured real cutoff timestamp for both preview and apply.'
  DO $$ BEGIN RAISE EXCEPTION 'Required local demo flags missing'; END $$;
\endif

BEGIN;
SET LOCAL TIME ZONE 'Asia/Jakarta';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '120s';
SELECT pg_advisory_xact_lock(hashtextextended('credit-union.demo-2026-15x500-v1',0));
CREATE TEMP TABLE demo_config ON COMMIT DROP AS
 SELECT 'demo-2026-15x500-v1'::text AS batch, :'cutoff'::timestamptz AS cutoff;
SELECT EXISTS(SELECT 1 FROM audit_events WHERE action='demo_seed_completed'
 AND entity_id='demo-2026-15x500-v1') AS already_seeded \gset
\if :already_seeded
  \echo 'Batch already exists: validating existing data; no insert or sequence consumption.'
  \ir demo_2026_reconcile.sql
  ROLLBACK;
  \quit
\endif

DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM interest_policies) THEN
  RAISE EXCEPTION 'Seed before savings interest activation is configured';END IF;
 IF EXISTS(SELECT 1 FROM members) OR EXISTS(SELECT 1 FROM transactions)
 OR EXISTS(SELECT 1 FROM journal_lines) OR EXISTS(SELECT 1 FROM loans)
 OR EXISTS(SELECT 1 FROM requests) THEN
  RAISE EXCEPTION 'Refusing nonempty financial data; this local fixture never deletes, replaces, or mixes existing data';
 END IF;
 IF (SELECT extract(year FROM cutoff)::int FROM demo_config)<>2026
 OR (SELECT cutoff FROM demo_config)>clock_timestamp()
 OR (SELECT cutoff FROM demo_config)<timestamptz '2026-01-02 09:02:00+07' THEN
  RAISE EXCEPTION 'Invalid real cutoff for the 2026 local demonstration fixture';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM schema_migrations WHERE version=3) THEN
  RAISE EXCEPTION 'Apply member balance snapshot migration3 first';
 END IF;
END $$;

CREATE TEMP TABLE demo_staff ON COMMIT DROP AS
 SELECT CASE username WHEN 'teller.a' THEN 1 ELSE 2 END AS ordinal,
 id AS actor_id,branch_id,0::bigint AS cash
 FROM users WHERE username IN ('teller.a','teller.b') AND role='teller';
DO $$ BEGIN
 IF (SELECT count(*) FROM demo_staff)<>2 OR (SELECT count(DISTINCT branch_id) FROM demo_staff)<>2 THEN
  RAISE EXCEPTION 'Two configured tellers in distinct KCPs are required';
 END IF;
END $$;
CREATE TEMP TABLE demo_members_plan(
 ordinal integer PRIMARY KEY,nik text,name text,home_staff integer,
 balance bigint NOT NULL DEFAULT 0,created_at timestamptz,member_id bigint,account_number text
) ON COMMIT DROP;
INSERT INTO demo_members_plan(ordinal,nik,name,home_staff)
 SELECT n,lpad(n::text,16,'0'),'DEMO 2026 '||lpad(n::text,2,'0')||' - '||
 (ARRAY['Aditya Pratama','Bunga Lestari','Cahya Putra','Dewi Anggraini','Eka Saputra',
 'Farah Amelia','Gilang Maulana','Hana Permata','Indra Wijaya','Jihan Safitri',
 'Kirana Putri','Lukman Hakim','Maya Kartika','Nanda Prasetyo','Olivia Anindita'])[n],
 CASE WHEN n%2=1 THEN 1 ELSE 2 END FROM generate_series(1,15) n;

CREATE TEMP TABLE demo_days ON COMMIT DROP AS
 SELECT day::date AS day,row_number() OVER(ORDER BY day)::int AS ordinal
 FROM generate_series(date '2026-01-01',
 (SELECT (cutoff AT TIME ZONE 'Asia/Jakarta')::date FROM demo_config),interval '1 day') day
 WHERE extract(isodow FROM day)<>7
 AND (day::date+time '09:02') AT TIME ZONE 'Asia/Jakarta'<(SELECT cutoff FROM demo_config);
SELECT setseed(0.20261021);
CREATE TEMP TABLE demo_plan(
 reference text PRIMARY KEY,member_ordinal integer,raw_no integer,ordinal integer,
 created_at timestamptz,staff_ordinal integer,kind text,amount bigint,balance_before bigint,balance_after bigint
) ON COMMIT DROP;
INSERT INTO demo_plan(reference,member_ordinal,raw_no,created_at)
 SELECT 'TX-'||substr(md5(c.batch||':'||m.ordinal||':'||n),1,20),m.ordinal,n,
 CASE WHEN n=1 THEN timestamptz '2026-01-01 09:00:00+07'+m.ordinal*interval '1 second'
 ELSE (d.day+time '09:01') AT TIME ZONE 'Asia/Jakarta'
 +floor(random()*extract(epoch FROM
 (least((d.day+time '16:00') AT TIME ZONE 'Asia/Jakarta',c.cutoff-interval '1 second')
 -((d.day+time '09:01') AT TIME ZONE 'Asia/Jakarta'))))*interval '1 second'
 +n*interval '1 microsecond'+m.ordinal*interval '1 millisecond' END
 FROM demo_members_plan m CROSS JOIN generate_series(1,500) n CROSS JOIN demo_config c
 JOIN demo_days d ON d.ordinal=1+floor((n-1)*((SELECT count(*) FROM demo_days)-1)/499.0)::int
 ORDER BY m.ordinal,n;
WITH numbered AS (
 SELECT reference,row_number() OVER(PARTITION BY member_ordinal ORDER BY created_at,reference)::int n
 FROM demo_plan
) UPDATE demo_plan p SET ordinal=n.n FROM numbered n WHERE n.reference=p.reference;
DO $$
DECLARE r record; member_balance bigint;branch_cash bigint;home integer;
 chosen_staff integer;direction text;amount_value bigint;desired bigint;signed_amount bigint;
BEGIN
 FOR r IN SELECT * FROM demo_plan ORDER BY created_at,reference LOOP
  SELECT balance,home_staff INTO STRICT member_balance,home FROM demo_members_plan WHERE ordinal=r.member_ordinal;
  chosen_staff:=CASE WHEN r.ordinal=1 THEN home ELSE 1+floor(random()*2)::int END;
  SELECT cash INTO STRICT branch_cash FROM demo_staff WHERE ordinal=chosen_staff;
  direction:=CASE WHEN r.ordinal>1 AND random()<0.48 THEN 'withdrawal' ELSE 'deposit' END;
  desired:=CASE WHEN r.ordinal=1 THEN 50000 ELSE 5000*(1+floor(random()*(20+r.member_ordinal*8))::bigint) END;
  amount_value:=desired;
  IF direction='withdrawal' THEN
   amount_value:=least(desired,(member_balance/5000)*5000,(branch_cash/5000)*5000);
   IF amount_value=0 THEN direction:='deposit';amount_value:=desired; END IF;
  END IF;
  signed_amount:=CASE WHEN direction='deposit' THEN amount_value ELSE -amount_value END;
  UPDATE demo_plan SET staff_ordinal=chosen_staff,kind=direction,amount=amount_value,
   balance_before=member_balance,balance_after=member_balance+signed_amount WHERE reference=r.reference;
  UPDATE demo_members_plan SET balance=balance+signed_amount,created_at=coalesce(created_at,r.created_at)
   WHERE ordinal=r.member_ordinal;
  UPDATE demo_staff SET cash=cash+signed_amount WHERE ordinal=chosen_staff;
 END LOOP;
END $$;

DO $$ BEGIN
 IF (SELECT count(*) FROM demo_plan)<>7500 OR EXISTS(
  SELECT member_ordinal FROM demo_plan GROUP BY member_ordinal
  HAVING count(*)<>500 OR count(DISTINCT kind)<>2 OR count(DISTINCT date_trunc('month',created_at))<>
  (SELECT extract(month FROM cutoff)::int FROM demo_config)) THEN RAISE EXCEPTION 'Plan count/type/month mismatch';END IF;
 IF EXISTS(SELECT 1 FROM demo_plan WHERE balance_before<0 OR balance_after<0 OR amount<=0 OR amount%5000<>0
 OR balance_after>1000000000000 OR created_at>(SELECT cutoff FROM demo_config)) THEN
  RAISE EXCEPTION 'Invalid plan money or timestamp';END IF;
 IF EXISTS(SELECT 1 FROM demo_members_plan m WHERE m.balance<>(SELECT sum(
  CASE kind WHEN 'deposit' THEN amount ELSE -amount END) FROM demo_plan p WHERE p.member_ordinal=m.ordinal))
 OR EXISTS(SELECT 1 FROM demo_staff WHERE cash<0) THEN RAISE EXCEPTION 'Plan balance mismatch';END IF;
END $$;
SELECT count(DISTINCT member_ordinal) members,count(*) transactions,
 count(*) FILTER(WHERE kind='deposit') deposits,count(*) FILTER(WHERE kind='withdrawal') withdrawals,
 min(created_at) first_transaction,max(created_at) last_transaction,
 md5(string_agg(reference||':'||created_at||':'||kind||':'||amount||':'||balance_before||':'||balance_after||':'||member_ordinal||':'||staff_ordinal,
 '|' ORDER BY created_at,reference)) AS plan_checksum FROM demo_plan;
SELECT to_char(created_at,'YYYY-MM') AS period_month,count(*) transactions,
 count(*) FILTER(WHERE kind='deposit') deposits,count(*) FILTER(WHERE kind='withdrawal') withdrawals
 FROM demo_plan GROUP BY 1 ORDER BY 1;

\if :apply
 SELECT pg_advisory_xact_lock_shared(101,1);
 LOCK TABLE members,transactions,journal_lines,loans,requests,audit_events IN SHARE ROW EXCLUSIVE MODE;
 DO $$
 DECLARE r record;new_id bigint;new_account text;
 BEGIN
  IF EXISTS(SELECT 1 FROM interest_policies) THEN
   RAISE EXCEPTION 'Seed before savings interest activation is configured';END IF;
  IF EXISTS(SELECT 1 FROM members) OR EXISTS(SELECT 1 FROM transactions)
  OR EXISTS(SELECT 1 FROM journal_lines) OR EXISTS(SELECT 1 FROM loans) OR EXISTS(SELECT 1 FROM requests) THEN
   RAISE EXCEPTION 'Financial data changed after preview; refusing to mix data';END IF;
  FOR r IN SELECT p.*,s.branch_id,s.actor_id FROM demo_members_plan p
   JOIN demo_staff s ON s.ordinal=p.home_staff ORDER BY p.ordinal LOOP
   INSERT INTO members(nik,name,phone,address,home_branch_id,balance,created_at)
   VALUES(r.nik,r.name,'00000000'||lpad(r.ordinal::text,4,'0'),
    'DEMO: identitas sintetis, bukan data pribadi asli',r.branch_id,r.balance,r.created_at)
   RETURNING id,account_number INTO new_id,new_account;
   UPDATE demo_members_plan SET member_id=new_id,account_number=new_account WHERE ordinal=r.ordinal;
   INSERT INTO audit_events(actor_id,action,entity_id,details)
   VALUES(r.actor_id,'register_member',new_id::text,jsonb_build_object('demo_batch',
    'demo-2026-15x500-v1','synthetic',true));
  END LOOP;
 END $$;
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at,balance_before,balance_after)
 SELECT p.reference,m.member_id,s.branch_id,s.actor_id,p.kind,p.amount,
 'DEMO demo-2026-15x500-v1 #'||lpad(p.ordinal::text,3,'0')||
 CASE WHEN p.ordinal=1 THEN ' - Setoran awal' WHEN p.kind='deposit' THEN ' - Setor tunai' ELSE ' - Tarik tunai' END,
 p.created_at,p.balance_before,p.balance_after
 FROM demo_plan p JOIN demo_members_plan m ON m.ordinal=p.member_ordinal
 JOIN demo_staff s ON s.ordinal=p.staff_ordinal ORDER BY p.created_at,p.reference;
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
 SELECT reference,1,CASE kind WHEN 'deposit' THEN 'cash' ELSE 'member_savings' END,amount,0 FROM demo_plan
 UNION ALL SELECT reference,2,CASE kind WHEN 'deposit' THEN 'member_savings' ELSE 'cash' END,0,amount FROM demo_plan;
 INSERT INTO audit_events(actor_id,action,entity_id,details)
 SELECT s.actor_id,p.kind,p.reference,jsonb_build_object('demo_batch','demo-2026-15x500-v1','synthetic',true)
 FROM demo_plan p JOIN demo_staff s ON s.ordinal=p.staff_ordinal;
 INSERT INTO audit_events(actor_id,action,entity_id,details)
 SELECT s.actor_id,'demo_seed_completed',c.batch,jsonb_build_object('demo_batch',c.batch,'synthetic',true,
 'cutoff',c.cutoff,'seed',0.20261021,'member_ids',(SELECT jsonb_agg(member_id ORDER BY ordinal) FROM demo_members_plan),
 'member_count',15,'transactions_per_member',500,'opening_deposit_included',true,
 'plan_checksum',(SELECT md5(string_agg(reference||':'||created_at||':'||kind||':'||amount||':'||balance_before||':'||balance_after||':'||member_ordinal||':'||staff_ordinal,
 '|' ORDER BY created_at,reference)) FROM demo_plan)) FROM demo_config c JOIN demo_staff s ON s.ordinal=1;
 \ir demo_2026_reconcile.sql
 COMMIT;
\else
 \echo 'PREVIEW ONLY: rolling back temporary plan; no persistent rows or sequence values allocated.'
 ROLLBACK;
\endif
