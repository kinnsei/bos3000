CREATE TABLE callback_calls (
    id              BIGSERIAL PRIMARY KEY,
    call_id         VARCHAR(64) UNIQUE NOT NULL,
    user_id         BIGINT NOT NULL,

    -- Request params
    a_number        VARCHAR(32) NOT NULL,
    b_number        VARCHAR(32) NOT NULL,
    caller_id       VARCHAR(32),
    max_duration    INT,
    custom_data     JSONB,
    callback_url    TEXT,

    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'initiating'
                    CHECK (status IN ('initiating','a_dialing','a_connected','b_dialing','b_connected','bridged','finished','failed')),

    -- A-leg
    a_fs_uuid       VARCHAR(64),
    a_gateway_name  VARCHAR(100),
    a_gateway_id    BIGINT,
    a_dial_at       TIMESTAMPTZ,
    a_answer_at     TIMESTAMPTZ,
    a_hangup_at     TIMESTAMPTZ,
    a_hangup_cause  VARCHAR(64),
    a_duration_ms   BIGINT NOT NULL DEFAULT 0,

    -- B-leg
    b_fs_uuid       VARCHAR(64),
    b_gateway_name  VARCHAR(100),
    b_gateway_id    BIGINT,
    b_dial_at       TIMESTAMPTZ,
    b_answer_at     TIMESTAMPTZ,
    b_hangup_at     TIMESTAMPTZ,
    b_hangup_cause  VARCHAR(64),
    b_duration_ms   BIGINT NOT NULL DEFAULT 0,

    -- Bridge
    bridge_at           TIMESTAMPTZ,
    bridge_end_at       TIMESTAMPTZ,
    bridge_duration_ms  BIGINT NOT NULL DEFAULT 0,

    -- Billing (all monetary values in fen)
    a_leg_rate          BIGINT NOT NULL DEFAULT 0,
    b_leg_rate          BIGINT NOT NULL DEFAULT 0,
    pre_deduct_amount   BIGINT NOT NULL DEFAULT 0,
    a_leg_cost          BIGINT NOT NULL DEFAULT 0,
    b_leg_cost          BIGINT NOT NULL DEFAULT 0,
    total_cost          BIGINT NOT NULL DEFAULT 0,

    -- Wastage
    wastage_type        VARCHAR(30)
                        CHECK (wastage_type IS NULL OR wastage_type IN ('a_connected_b_failed','bridge_broken_early')),
    wastage_cost        BIGINT,
    wastage_duration_ms BIGINT,

    -- Termination
    hangup_by       VARCHAR(20),
    failure_reason  TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_callback_calls_user_id ON callback_calls (user_id);
CREATE INDEX idx_callback_calls_status ON callback_calls (status);
CREATE INDEX idx_callback_calls_created_at ON callback_calls (created_at);
CREATE INDEX idx_callback_calls_user_status ON callback_calls (user_id, status);
CREATE INDEX idx_callback_calls_wastage ON callback_calls (wastage_type) WHERE wastage_type IS NOT NULL;
