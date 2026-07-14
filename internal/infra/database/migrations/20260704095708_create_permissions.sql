-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE CHECK (name = lower(name)),
    resource    VARCHAR(255) NOT NULL,
    action      VARCHAR(255) NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO permissions (name, resource, action) VALUES
    ('room:update',  'room', 'update'),
    ('room:delete',  'room', 'delete'),
    ('room:promote', 'room', 'promote'),
    ('room:kick',    'room', 'kick'),
    ('room:speak',   'room', 'speak');

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE permissions;
-- +goose StatementEnd