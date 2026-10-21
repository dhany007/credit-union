-- Immutable migration 0002: one savings account per Phase 1 member.
ALTER TABLE members ADD COLUMN account_number text;

CREATE SEQUENCE savings_account_number_seq
    AS bigint
    MINVALUE 1000000001
    MAXVALUE 9999999999
    START WITH 1000000001
    NO CYCLE
    OWNED BY members.account_number;

ALTER TABLE members
    ALTER COLUMN account_number SET DEFAULT nextval('savings_account_number_seq')::text;

UPDATE members
SET account_number = nextval('savings_account_number_seq')::text
WHERE account_number IS NULL;

ALTER TABLE members
    ALTER COLUMN account_number SET NOT NULL,
    ADD CONSTRAINT members_account_number_key UNIQUE (account_number),
    ADD CONSTRAINT members_account_number_format CHECK (account_number ~ '^[1-9][0-9]{9}$');
