-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE role_permissions (
    id SERIAL PRIMARY KEY,
    role_id       INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (role_id, permission_id)
);

-- admin: everything
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin';

-- moderator: promote, kick, speak (no update/delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'moderator'
  AND p.name IN ('room:promote', 'room:kick', 'room:speak');

-- speaker: speak
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'speaker'
  AND p.name = 'room:speak';

-- listener: no permissions — membership alone gives them listen access

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE role_permissions;
-- +goose StatementEnd