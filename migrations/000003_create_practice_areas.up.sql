CREATE TABLE practice_areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(180) NOT NULL,
    description TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT practice_areas_display_order_check
        CHECK (display_order >= 0)
);

CREATE UNIQUE INDEX practice_areas_name_lower_unique
    ON practice_areas (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX practice_areas_slug_lower_unique
    ON practice_areas (LOWER(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_practice_areas_active_order
    ON practice_areas (is_active, display_order)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_practice_areas_deleted_at
    ON practice_areas (deleted_at);