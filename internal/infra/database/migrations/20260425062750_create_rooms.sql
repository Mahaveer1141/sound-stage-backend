-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS rooms (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR NOT NULL,
	description TEXT,
	creator_id BIGINT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP,

	CONSTRAINT fk_rooms_creator_id FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_rooms_deleted_at ON rooms (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rooms;
-- +goose StatementEnd
