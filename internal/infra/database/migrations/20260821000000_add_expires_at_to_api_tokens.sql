-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
TRUNCATE TABLE api_tokens;

ALTER TABLE api_tokens ADD COLUMN expires_at TIMESTAMPTZ NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE api_tokens DROP COLUMN expires_at;
-- +goose StatementEnd
