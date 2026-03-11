ALTER TABLE callback_calls
    ADD COLUMN recording_key TEXT,
    ADD COLUMN recording_a_key TEXT,
    ADD COLUMN recording_b_key TEXT,
    ADD COLUMN recording_status VARCHAR(20) CHECK (recording_status IN ('recording', 'merging', 'ready', 'failed'));
