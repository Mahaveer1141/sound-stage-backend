-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE rooms ADD COLUMN is_chat_enabled BOOLEAN NOT NULL DEFAULT TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE rooms DROP COLUMN is_chat_enabled;
-- +goose StatementEnd
