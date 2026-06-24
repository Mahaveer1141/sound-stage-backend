-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS room_users (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL,
	room_id BIGINT NOT NULL,
	last_joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_left_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_online BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT fk_room_users_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT fk_room_users_room_id FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX idx_room_users_user_id ON room_users (user_id);
CREATE INDEX idx_room_users_room_id ON room_users (room_id);
CREATE UNIQUE INDEX idx_room_users_user_id_room_id ON room_users (user_id, room_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS room_users;
-- +goose StatementEnd
