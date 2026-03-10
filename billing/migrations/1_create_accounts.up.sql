-- Billing accounts mirror auth.users but owned by billing service
-- Balance updates happen here with row-level locking
CREATE TABLE billing_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    credit_limit BIGINT NOT NULL DEFAULT 0,
    max_concurrent INT NOT NULL DEFAULT 10,
    rate_plan_id BIGINT,
    a_leg_rate BIGINT,
    b_leg_rate BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
