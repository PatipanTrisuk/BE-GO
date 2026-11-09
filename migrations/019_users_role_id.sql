-- Switch users.role (name) to users.role_id (FK → roles)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id UUID;

UPDATE users u
SET role_id = r.id
FROM roles r
WHERE u.role_id IS NULL
  AND LOWER(u.role) = LOWER(r.name);

UPDATE users
SET role_id = (SELECT id FROM roles WHERE name = 'user' LIMIT 1)
WHERE role_id IS NULL;

ALTER TABLE users
    ALTER COLUMN role_id SET NOT NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_id_fkey;

ALTER TABLE users
    ADD CONSTRAINT users_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles(id);

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);

ALTER TABLE users DROP COLUMN IF EXISTS role;
