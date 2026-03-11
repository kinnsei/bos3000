CREATE TABLE webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    call_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivering', 'delivered', 'retrying', 'failed', 'dlq')),
    webhook_url TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    last_error TEXT,
    delivered_at TIMESTAMPTZ,
    dlq_at TIMESTAMPTZ,
    dlq_retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_user_created ON webhook_deliveries(user_id, created_at DESC);
CREATE INDEX idx_webhook_status ON webhook_deliveries(status) WHERE status IN ('pending', 'retrying', 'dlq');
CREATE INDEX idx_webhook_retry_poll ON webhook_deliveries(next_retry_at) WHERE status = 'retrying';
CREATE INDEX idx_webhook_call ON webhook_deliveries(call_id);
