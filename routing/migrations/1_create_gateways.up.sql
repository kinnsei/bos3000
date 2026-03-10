CREATE TABLE gateways (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('a_leg', 'b_leg')),
    sip_address VARCHAR(255) NOT NULL,
    weight INT NOT NULL DEFAULT 1,
    healthy BOOLEAN NOT NULL DEFAULT true,
    enabled BOOLEAN NOT NULL DEFAULT true,
    failover_gateway_id BIGINT REFERENCES gateways(id),
    carrier VARCHAR(100),
    health_check_failures INT NOT NULL DEFAULT 0,
    max_health_failures INT NOT NULL DEFAULT 3,
    last_health_check TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE gateway_prefixes (
    id BIGSERIAL PRIMARY KEY,
    gateway_id BIGINT NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    prefix VARCHAR(10) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    UNIQUE(gateway_id, prefix)
);
CREATE INDEX idx_gateway_prefixes_prefix ON gateway_prefixes(prefix);
