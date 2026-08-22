-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS room_categories (
	id BIGSERIAL PRIMARY KEY,
	room_id BIGINT NOT NULL,
	category_id BIGINT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT fk_room_categories_room_id FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
	CONSTRAINT fk_room_categories_category_id FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE INDEX idx_room_categories_room_id ON room_categories (room_id);
CREATE INDEX idx_room_categories_category_id ON room_categories (category_id);
CREATE UNIQUE INDEX idx_room_categories_room_id_category_id ON room_categories (room_id, category_id);

CREATE OR REPLACE FUNCTION check_max_room_categories()
RETURNS TRIGGER AS $$
BEGIN
    IF (SELECT COUNT(*) FROM room_categories WHERE room_id = NEW.room_id) > 3 THEN
        RAISE EXCEPTION 'A room cannot have more than 3 categories assigned.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_limit_room_categories
AFTER INSERT ON room_categories
FOR EACH ROW
EXECUTE FUNCTION check_max_room_categories();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS room_categories;
-- +goose StatementEnd
