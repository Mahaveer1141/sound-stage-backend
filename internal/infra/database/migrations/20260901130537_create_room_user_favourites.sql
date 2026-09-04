-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS room_user_favourites (
	id BIGSERIAL PRIMARY KEY,
	room_id BIGINT NOT NULL,
	user_id BIGINT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT fk_room_user_favourites_room_id FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
	CONSTRAINT fk_room_user_favourites_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_room_user_favourites_room_id ON room_user_favourites (room_id);
CREATE INDEX idx_room_user_favourites_user_id ON room_user_favourites (user_id);
CREATE UNIQUE INDEX idx_room_user_favourites_room_user_unique ON room_user_favourites (room_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS room_user_favourites;
-- +goose StatementEnd
