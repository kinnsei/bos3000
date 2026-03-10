CREATE TABLE blacklisted_numbers (
    id BIGSERIAL PRIMARY KEY,
    number VARCHAR(20) NOT NULL,
    user_id BIGINT,
    reason TEXT,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE NULLS NOT DISTINCT (number, user_id)
);
CREATE INDEX idx_blacklist_number ON blacklisted_numbers(number);
CREATE INDEX idx_blacklist_user ON blacklisted_numbers(user_id);
