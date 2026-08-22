-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS tags (
	id BIGSERIAL PRIMARY KEY,
	name CITEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT tags_name_not_empty CHECK (LENGTH(TRIM(name)) > 0)
);

CREATE INDEX idx_tags_created_at ON tags(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS tags;
-- +goose StatementEnd
