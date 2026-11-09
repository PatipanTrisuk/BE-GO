CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

INSERT INTO permissions (code, description) VALUES
    ('users.read', 'List and view users'),
    ('users.create', 'Create users'),
    ('users.update', 'Update users'),
    ('users.delete', 'Soft-delete users'),
    ('users.restore', 'Restore deleted users'),
    ('users.kick', 'Revoke user sessions'),
    ('users.reset_password', 'Reset user passwords'),
    ('users.verify_email', 'Mark user email as verified'),
    ('users.export', 'Export users'),
    ('admin.stats', 'View admin stats'),
    ('roles.read', 'List and view roles'),
    ('roles.create', 'Create roles'),
    ('roles.update', 'Update roles and permissions'),
    ('roles.delete', 'Delete roles')
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (name, description, is_system) VALUES
    ('user', 'Default registered user', TRUE),
    ('moderator', 'Limited management access', TRUE),
    ('admin', 'Full administrative access', TRUE)
ON CONFLICT (name) DO NOTHING;

-- admin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- moderator: limited set
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'users.read',
    'users.kick',
    'users.verify_email',
    'admin.stats',
    'roles.read'
)
WHERE r.name = 'moderator'
ON CONFLICT DO NOTHING;
