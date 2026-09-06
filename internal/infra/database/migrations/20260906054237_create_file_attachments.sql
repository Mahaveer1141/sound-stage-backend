-- +goose Up
-- +goose StatementBegin

CREATE TABLE file_attachments (
    id BIGSERIAL PRIMARY KEY,
    owner_type VARCHAR(255) NOT NULL,
    owner_id BIGINT NOT NULL,
    context VARCHAR(255) NOT NULL,
    public_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    format VARCHAR(50),
    url TEXT NOT NULL,
    bytes BIGINT NOT NULL,
    width INT,
    height INT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_file_attachment_owner ON file_attachments (owner_type, owner_id, context);
CREATE UNIQUE INDEX idx_file_attachment_unique ON file_attachments (public_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE file_attachments;
-- +goose StatementEnd
