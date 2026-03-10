CREATE TABLE did_numbers (
    id BIGSERIAL PRIMARY KEY,
    number VARCHAR(20) NOT NULL UNIQUE,
    user_id BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'in_use', 'reserved')),
    region VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_did_user ON did_numbers(user_id);
CREATE INDEX idx_did_status ON did_numbers(status);
