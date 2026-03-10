CREATE TABLE rate_plans (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'uniform' CHECK (mode IN ('uniform', 'prefix')),
    uniform_a_rate BIGINT,
    uniform_b_rate BIGINT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rate_plan_prefixes (
    id BIGSERIAL PRIMARY KEY,
    rate_plan_id BIGINT NOT NULL REFERENCES rate_plans(id) ON DELETE CASCADE,
    prefix VARCHAR(10) NOT NULL,
    a_rate BIGINT NOT NULL,
    b_rate BIGINT NOT NULL,
    UNIQUE(rate_plan_id, prefix)
);
CREATE INDEX idx_rate_plan_prefixes_plan ON rate_plan_prefixes(rate_plan_id);
