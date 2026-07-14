CREATE TABLE admin_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    admin_id UUID NOT NULL
        REFERENCES admins(id)
        ON DELETE CASCADE,

    refresh_token_hash CHAR(64) NOT NULL,

    signing_secret_ciphertext TEXT NOT NULL,

    user_agent TEXT NULL,
    ip_address VARCHAR(64) NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,

    replaced_by_session_id UUID NULL
        REFERENCES admin_sessions(id)
        ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX admin_sessions_refresh_token_hash_unique
    ON admin_sessions (refresh_token_hash);

CREATE INDEX idx_admin_sessions_admin_id
    ON admin_sessions (admin_id);

CREATE INDEX idx_admin_sessions_active
    ON admin_sessions (
        admin_id,
        expires_at
    )
    WHERE revoked_at IS NULL;

CREATE TABLE request_nonces (
    nonce VARCHAR(100) PRIMARY KEY,

    session_id UUID NOT NULL
        REFERENCES admin_sessions(id)
        ON DELETE CASCADE,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_nonces_expires_at
    ON request_nonces (expires_at);