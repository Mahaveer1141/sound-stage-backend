-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_rooms_deleted_at;
ALTER TABLE rooms DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_users_email_unique;
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_users_created_at;

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

CREATE UNIQUE INDEX idx_users_email_unique ON users(LOWER(email));
CREATE INDEX idx_users_last_login_at ON users(last_login_at DESC) WHERE last_login_at IS NOT NULL;
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_room_users_room_id_created_at ON room_users(room_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_rooms_deleted_at ON rooms (deleted_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

DROP INDEX IF EXISTS idx_users_email_unique;
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_room_users_room_id_created_at;

CREATE UNIQUE INDEX idx_users_email_unique ON users(LOWER(email)) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_last_login_at ON users(last_login_at DESC) WHERE last_login_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at DESC) WHERE deleted_at IS NULL;
-- +goose StatementEnd
