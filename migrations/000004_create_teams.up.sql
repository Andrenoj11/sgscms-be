CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(180) NOT NULL,
    degree VARCHAR(100) NULL,
    position VARCHAR(150) NOT NULL,
    short_description TEXT NULL,
    biography TEXT NULL,
    photo_url TEXT NULL,
    email VARCHAR(150) NULL,
    linkedin_url TEXT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES admins(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT teams_display_order_check
        CHECK (display_order >= 0)
);

CREATE UNIQUE INDEX teams_name_lower_unique
    ON teams (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX teams_slug_lower_unique
    ON teams (LOWER(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_teams_published_order
    ON teams (
        is_published,
        display_order,
        name
    )
    WHERE deleted_at IS NULL;

CREATE INDEX idx_teams_deleted_at
    ON teams (deleted_at);

CREATE TABLE team_practice_areas (
    team_id UUID NOT NULL
        REFERENCES teams(id)
        ON DELETE CASCADE,

    practice_area_id UUID NOT NULL
        REFERENCES practice_areas(id)
        ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (
        team_id,
        practice_area_id
    )
);

CREATE INDEX idx_team_practice_areas_practice_area_id
    ON team_practice_areas (practice_area_id);