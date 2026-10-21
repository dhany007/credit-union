ALTER TABLE transactions
    ADD COLUMN balance_before bigint,
    ADD COLUMN balance_after bigint,
    ADD CONSTRAINT transactions_balance_snapshot_check CHECK (
        (balance_before IS NULL AND balance_after IS NULL)
        OR (
            balance_before IS NOT NULL AND balance_after IS NOT NULL
            AND balance_before >= 0 AND balance_after >= 0
            AND (
                (kind = 'deposit' AND balance_after - balance_before = amount)
                OR (kind = 'withdrawal' AND balance_before - balance_after = amount)
            )
        )
    );
