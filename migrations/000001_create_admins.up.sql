CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'admin',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT admins_email_unique UNIQUE (email),
    CONSTRAINT admins_role_check
        CHECK (role IN ('super_admin', 'admin', 'editor'))
);

CREATE INDEX idx_admins_is_active
    ON admins (is_active);