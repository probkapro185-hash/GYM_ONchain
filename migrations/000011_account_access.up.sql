ALTER TABLE users
    ADD COLUMN password_setup_required BOOLEAN NOT NULL DEFAULT false;

CREATE TYPE account_token_purpose AS ENUM ('activation', 'password_reset');

CREATE TABLE account_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    account_token_purpose NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_account_token_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_account_tokens_user_purpose
    ON account_tokens(user_id, purpose, created_at DESC);
CREATE INDEX idx_account_tokens_active
    ON account_tokens(expires_at)
    WHERE used_at IS NULL;
