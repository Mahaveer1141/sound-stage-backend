-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS room_user_blocks (
	id BIGSERIAL PRIMARY KEY,
	room_id BIGINT NOT NULL,
	user_id BIGINT NOT NULL,
  blocked_by_id BIGINT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT fk_room_user_blocks_room_id FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
	CONSTRAINT fk_room_user_blocks_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT fk_room_user_blocks_blocked_by_id FOREIGN KEY (blocked_by_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT chk_room_user_blocks_different_user CHECK (user_id != blocked_by_id)
);

CREATE INDEX idx_room_user_blocks_room_id ON room_user_blocks (room_id);
CREATE INDEX idx_room_user_blocks_user_id ON room_user_blocks (user_id);
CREATE INDEX idx_room_user_blocks_blocked_by_id ON room_user_blocks (blocked_by_id);
CREATE UNIQUE INDEX idx_room_user_blocks_room_user_unique ON room_user_blocks (room_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS room_user_blocks;
-- +goose StatementEnd
