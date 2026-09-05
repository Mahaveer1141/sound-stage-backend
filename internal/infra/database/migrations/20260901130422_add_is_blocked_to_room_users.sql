-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE room_users ADD COLUMN is_blocked BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE room_users ADD COLUMN blocked_by_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE room_users DROP COLUMN blocked_by_id;
ALTER TABLE room_users DROP COLUMN is_blocked;
-- +goose StatementEnd
