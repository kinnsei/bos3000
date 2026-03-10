CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(30) NOT NULL CHECK (type IN ('pre_deduct', 'finalize', 'refund', 'topup', 'deduction', 'adjustment')),
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reference_id VARCHAR(100),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_ref ON transactions(reference_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);
