-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS taggables (
	id BIGSERIAL PRIMARY KEY,
	tag_id BIGINT NOT NULL,
	taggable_type VARCHAR NOT NULL,
	taggable_id BIGINT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT fk_taggables_tag_id FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
	CONSTRAINT taggables_taggable_type_not_empty CHECK (LENGTH(TRIM(taggable_type)) > 0)
);

CREATE INDEX idx_taggables_tag_id ON taggables (tag_id);
CREATE INDEX idx_taggables_taggable ON taggables (taggable_type, taggable_id);
CREATE UNIQUE INDEX idx_taggables_tag_taggable ON taggables (tag_id, taggable_type, taggable_id);

CREATE OR REPLACE FUNCTION check_max_tags_per_taggable()
RETURNS TRIGGER AS $$
BEGIN
    IF (SELECT COUNT(*) FROM taggables WHERE taggable_type = NEW.taggable_type AND taggable_id = NEW.taggable_id) > 5 THEN
        RAISE EXCEPTION 'A taggable cannot have more than 5 tags assigned.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_limit_tags_per_taggable
AFTER INSERT ON taggables
FOR EACH ROW
EXECUTE FUNCTION check_max_tags_per_taggable();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS taggables;
-- +goose StatementEnd
