-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE rooms ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'public';
ALTER TABLE rooms ADD COLUMN private_code VARCHAR(8);
ALTER TABLE rooms ALTER COLUMN name TYPE CITEXT;

ALTER TABLE rooms ADD CONSTRAINT chk_rooms_type CHECK (type IN ('public', 'private'));
ALTER TABLE rooms ADD CONSTRAINT chk_private_code CHECK ((private_code IS NULL AND type = 'public') OR (private_code IS NOT NULL AND type = 'private'));
CREATE UNIQUE INDEX idx_rooms_private_code ON rooms (private_code) WHERE private_code IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE rooms DROP CONSTRAINT IF EXISTS chk_rooms_type;
ALTER TABLE rooms DROP COLUMN IF EXISTS private_code;
ALTER TABLE rooms DROP COLUMN IF EXISTS type;
ALTER TABLE rooms ALTER COLUMN name TYPE VARCHAR(255);
-- +goose StatementEnd
