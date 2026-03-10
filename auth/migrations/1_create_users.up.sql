CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'client' CHECK (role IN ('admin', 'client')),
    balance BIGINT NOT NULL DEFAULT 0,
    credit_limit BIGINT NOT NULL DEFAULT 0,
    max_concurrent INT NOT NULL DEFAULT 10,
    daily_limit INT NOT NULL DEFAULT 1000,
    rate_plan_id BIGINT,
    a_leg_rate BIGINT,
    b_leg_rate BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'frozen', 'suspended')),
    webhook_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
