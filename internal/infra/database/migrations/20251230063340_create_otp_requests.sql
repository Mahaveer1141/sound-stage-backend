-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE otp_requests (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR,
    user_id BIGINT,
    otp VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '10 minutes'),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT otp_requests_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT only_one_identifier CHECK ((email IS NOT NULL AND user_id IS NULL) OR (email IS NULL AND user_id IS NOT NULL)),
    CONSTRAINT fk_otp_requests_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_otp_requests_email ON otp_requests(LOWER(email));
CREATE INDEX idx_otp_requests_user_id ON otp_requests(user_id);
CREATE INDEX idx_otp_requests_expires_at ON otp_requests(expires_at);
CREATE INDEX idx_otp_requests_is_active ON otp_requests(is_active);
CREATE INDEX idx_otp_requests_created_at ON otp_requests(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS otp_requests;
-- +goose StatementEnd
