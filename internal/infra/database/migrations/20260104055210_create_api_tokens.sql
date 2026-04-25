-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE api_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR NOT NULL UNIQUE,
    type VARCHAR NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT fk_api_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT api_tokens_type_check CHECK (type IN ('access', 'refresh'))
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_is_active ON api_tokens(is_active);
CREATE INDEX idx_api_tokens_created_at ON api_tokens(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS api_tokens;
-- +goose StatementEnd
