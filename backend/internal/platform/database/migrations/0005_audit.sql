-- Nullable legacy/system actors preserve truthful history. No identity or money backfill.
CREATE FUNCTION cu_row_audit() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE actor bigint := NULLIF(current_setting('cu.actor_id',true),'')::bigint;
BEGIN
 IF TG_OP='INSERT' THEN NEW.created_by:=COALESCE(NEW.created_by,actor);
 ELSE NEW.updated_by:=actor; NEW.updated_at:=clock_timestamp();
 END IF;
 IF NEW.deleted_at IS NOT NULL AND NEW.deleted_by IS NULL THEN NEW.deleted_by:=actor; END IF;
 RETURN NEW;
END $$;

CREATE FUNCTION cu_install_audit() RETURNS void LANGUAGE plpgsql AS $$
DECLARE item record;
BEGIN
 FOR item IN SELECT tablename FROM pg_tables WHERE schemaname=current_schema() LOOP
  EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS is_active boolean NOT NULL DEFAULT true,
   ADD COLUMN IF NOT EXISTS created_by bigint REFERENCES users(id),
   ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
   ADD COLUMN IF NOT EXISTS updated_by bigint REFERENCES users(id),
   ADD COLUMN IF NOT EXISTS updated_at timestamptz,
   ADD COLUMN IF NOT EXISTS deleted_by bigint REFERENCES users(id),
   ADD COLUMN IF NOT EXISTS deleted_at timestamptz',item.tablename);
  IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(item.tablename)
   AND conname='cu_deletion_audit') THEN
   EXECUTE format('ALTER TABLE %I ADD CONSTRAINT cu_deletion_audit CHECK(
    (deleted_by IS NULL)=(deleted_at IS NULL) AND (deleted_at IS NULL OR NOT is_active))',item.tablename);
  END IF;
 END LOOP;
END $$;
SELECT cu_install_audit();

-- Backfill only facts already recorded, before attaching update triggers.
UPDATE transactions SET created_by=actor_id;
UPDATE audit_events SET created_by=actor_id;
UPDATE requests SET created_by=user_id;
UPDATE journal_lines j SET created_by=t.actor_id,created_at=t.created_at
 FROM transactions t WHERE t.id=j.transaction_id;
UPDATE sessions SET created_by=user_id;
UPDATE interest_daily SET created_at=calculated_at;
UPDATE interest_days SET created_at=completed_at;
UPDATE interest_monthly SET created_at=posted_at;
UPDATE schema_migrations SET created_at=applied_at;
UPDATE loans SET updated_by=decision_by,updated_at=decided_at WHERE decision_by IS NOT NULL;

-- Validate deferred pre-existing interest FKs after metadata backfill before later ALTER TABLE statements.
SET CONSTRAINTS ALL IMMEDIATE;

CREATE FUNCTION cu_attach_audit() RETURNS void LANGUAGE plpgsql AS $$
DECLARE item record;
BEGIN
 FOR item IN SELECT tablename FROM pg_tables WHERE schemaname=current_schema() LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid=to_regclass(item.tablename) AND tgname='cu_row_audit') THEN
   EXECUTE format('CREATE TRIGGER cu_row_audit BEFORE INSERT OR UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION cu_row_audit()',item.tablename);
  END IF;
 END LOOP;
END $$;
SELECT cu_attach_audit();
