-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
TRUNCATE TABLE room_users;

ALTER TABLE room_users ADD COLUMN role_id BIGINT REFERENCES roles(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE room_users DROP COLUMN role_id;
-- +goose StatementEnd
