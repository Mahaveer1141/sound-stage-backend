-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE INDEX idx_room_users_room_id_last_joined_at ON room_users (room_id, last_joined_at ASC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP INDEX idx_room_users_room_id_last_joined_at;

-- +goose StatementEnd
