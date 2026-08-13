-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE CHECK (name = lower(name)),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO roles (name, description) VALUES
    ('owner', 'Full control over the room, including editing and deleting it'),
    ('admin', 'Full control over the room, including editing and deleting it'),
    ('moderator', 'Can promote and kick users, and can speak'),
    ('speaker', 'Can speak and listen'),
    ('listener', 'Can only listen');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE roles;
-- +goose StatementEnd
