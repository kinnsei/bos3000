CREATE TABLE system_configs (
    id           SERIAL PRIMARY KEY,
    config_key   VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT NOT NULL,
    description  TEXT,
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO system_configs (config_key, config_value, description) VALUES
    ('bridge_broken_early_threshold_sec', '10', 'Minimum bridge duration (seconds) below which call is classified as bridge_broken_early wastage'),
    ('default_max_duration_sec', '3600', 'Default maximum call duration in seconds when not specified by API caller'),
    ('park_timeout_sec', '60', 'A-leg park timeout in seconds before auto-hangup');
