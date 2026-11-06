ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_username_key;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_key
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_username_active_key
    ON users (username)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS users_deleted_at_idx ON users (deleted_at);
