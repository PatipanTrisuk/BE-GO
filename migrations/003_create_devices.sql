CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    device_key VARCHAR(128) NOT NULL,
    name VARCHAR(100),
    user_agent TEXT,
    ip_address VARCHAR(45),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, device_key)
);

CREATE INDEX IF NOT EXISTS idx_devices_user_id
ON devices(user_id);

ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS device_id UUID
        REFERENCES devices(id)
        ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_device_id
ON refresh_tokens(device_id);
