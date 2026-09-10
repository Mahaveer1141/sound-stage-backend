-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE otp_requests
ADD COLUMN is_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE otp_requests
DROP COLUMN is_verified;

-- +goose StatementEnd
