ALTER TABLE users
    ALTER COLUMN password DROP NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS auth_provider VARCHAR(20) NOT NULL DEFAULT 'local';

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS discord_id VARCHAR(32);

CREATE UNIQUE INDEX IF NOT EXISTS users_discord_id_key
    ON users (discord_id)
    WHERE discord_id IS NOT NULL;
