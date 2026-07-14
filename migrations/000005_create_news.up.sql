CREATE TABLE news (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(280) NOT NULL,
    excerpt TEXT NULL,
    content TEXT NOT NULL,
    featured_image_url TEXT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT news_status_check
        CHECK (
            status IN (
                'draft',
                'published',
                'archived'
            )
        )
);

CREATE UNIQUE INDEX news_slug_lower_unique
    ON news (LOWER(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_news_status_published_at
    ON news (
        status,
        published_at DESC
    )
    WHERE deleted_at IS NULL;

CREATE INDEX idx_news_featured
    ON news (
        is_featured,
        published_at DESC
    )
    WHERE deleted_at IS NULL;

CREATE INDEX idx_news_deleted_at
    ON news (deleted_at);