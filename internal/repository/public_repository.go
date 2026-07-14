package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/dto"
)

type PublicRepository interface {
	ListPracticeAreas(
		ctx context.Context,
	) ([]domain.PracticeArea, error)

	ListTeams(
		ctx context.Context,
		filter dto.PublicListQuery,
	) ([]domain.Team, int64, error)

	FindTeamBySlug(
		ctx context.Context,
		slug string,
	) (*domain.Team, error)

	ListNews(
		ctx context.Context,
		filter dto.PublicNewsListQuery,
	) ([]domain.News, int64, error)

	FindNewsBySlug(
		ctx context.Context,
		slug string,
	) (*domain.News, error)
}

type PostgresPublicRepository struct {
	db *sql.DB
}

var _ PublicRepository = (*PostgresPublicRepository)(nil)

func NewPostgresPublicRepository(
	db *sql.DB,
) *PostgresPublicRepository {
	return &PostgresPublicRepository{
		db: db,
	}
}

func (r *PostgresPublicRepository) ListPracticeAreas(
	ctx context.Context,
) ([]domain.PracticeArea, error) {
	const query = `
		SELECT
			id,
			name,
			slug,
			COALESCE(description, ''),
			is_active,
			display_order,
			created_by,
			updated_by,
			created_at,
			updated_at,
			deleted_at
		FROM practice_areas
		WHERE deleted_at IS NULL
		  AND is_active = TRUE
		ORDER BY
			display_order ASC,
			name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"list public practice areas: %w",
			err,
		)
	}
	defer rows.Close()

	practiceAreas := make(
		[]domain.PracticeArea,
		0,
	)

	for rows.Next() {
		var practiceArea domain.PracticeArea

		err := rows.Scan(
			&practiceArea.ID,
			&practiceArea.Name,
			&practiceArea.Slug,
			&practiceArea.Description,
			&practiceArea.IsActive,
			&practiceArea.DisplayOrder,
			&practiceArea.CreatedBy,
			&practiceArea.UpdatedBy,
			&practiceArea.CreatedAt,
			&practiceArea.UpdatedAt,
			&practiceArea.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan public practice area: %w",
				err,
			)
		}

		practiceAreas = append(
			practiceAreas,
			practiceArea,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate public practice areas: %w",
			err,
		)
	}

	return practiceAreas, nil
}

func (r *PostgresPublicRepository) ListTeams(
	ctx context.Context,
	filter dto.PublicListQuery,
) ([]domain.Team, int64, error) {
	whereQuery := `
		WHERE deleted_at IS NULL
		  AND is_published = TRUE
		  AND published_at IS NOT NULL
		  AND published_at <= NOW()
		  AND (
				$1 = ''
				OR name ILIKE '%' || $1 || '%'
				OR position ILIKE '%' || $1 || '%'
				OR short_description ILIKE '%' || $1 || '%'
		  )
	`

	const countPrefix = `
		SELECT COUNT(*)
		FROM teams
	`

	var total int64

	err := r.db.QueryRowContext(
		ctx,
		countPrefix+whereQuery,
		filter.Search,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count public teams: %w",
			err,
		)
	}

	offset := (filter.Page - 1) * filter.Limit

	listQuery := `
		SELECT
			id,
			name,
			slug,
			COALESCE(degree, ''),
			position,
			COALESCE(short_description, ''),
			COALESCE(biography, ''),
			COALESCE(photo_url, ''),
			COALESCE(email, ''),
			COALESCE(linkedin_url, ''),
			display_order,
			is_published,
			published_at,
			created_by,
			updated_by,
			created_at,
			updated_at,
			deleted_at
		FROM teams
	` + whereQuery + `
		ORDER BY
			display_order ASC,
			name ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		filter.Search,
		filter.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list public teams: %w",
			err,
		)
	}
	defer rows.Close()

	teams := make([]domain.Team, 0)

	for rows.Next() {
		var team domain.Team

		err := rows.Scan(
			&team.ID,
			&team.Name,
			&team.Slug,
			&team.Degree,
			&team.Position,
			&team.ShortDescription,
			&team.Biography,
			&team.PhotoURL,
			&team.Email,
			&team.LinkedInURL,
			&team.DisplayOrder,
			&team.IsPublished,
			&team.PublishedAt,
			&team.CreatedBy,
			&team.UpdatedBy,
			&team.CreatedAt,
			&team.UpdatedAt,
			&team.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"scan public team: %w",
				err,
			)
		}

		practiceAreas, err :=
			r.findPublicPracticeAreasByTeamID(
				ctx,
				team.ID,
			)
		if err != nil {
			return nil, 0, err
		}

		team.PracticeAreas = practiceAreas

		teams = append(teams, team)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"iterate public teams: %w",
			err,
		)
	}

	return teams, total, nil
}

func (r *PostgresPublicRepository) FindTeamBySlug(
	ctx context.Context,
	slugValue string,
) (*domain.Team, error) {
	const query = `
		SELECT
			id,
			name,
			slug,
			COALESCE(degree, ''),
			position,
			COALESCE(short_description, ''),
			COALESCE(biography, ''),
			COALESCE(photo_url, ''),
			COALESCE(email, ''),
			COALESCE(linkedin_url, ''),
			display_order,
			is_published,
			published_at,
			created_by,
			updated_by,
			created_at,
			updated_at,
			deleted_at
		FROM teams
		WHERE LOWER(slug) = LOWER($1)
		  AND deleted_at IS NULL
		  AND is_published = TRUE
		  AND published_at IS NOT NULL
		  AND published_at <= NOW()
		LIMIT 1
	`

	team := &domain.Team{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(slugValue),
	).Scan(
		&team.ID,
		&team.Name,
		&team.Slug,
		&team.Degree,
		&team.Position,
		&team.ShortDescription,
		&team.Biography,
		&team.PhotoURL,
		&team.Email,
		&team.LinkedInURL,
		&team.DisplayOrder,
		&team.IsPublished,
		&team.PublishedAt,
		&team.CreatedBy,
		&team.UpdatedBy,
		&team.CreatedAt,
		&team.UpdatedAt,
		&team.DeletedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTeamNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find public team by slug: %w",
			err,
		)
	}

	practiceAreas, err :=
		r.findPublicPracticeAreasByTeamID(
			ctx,
			team.ID,
		)
	if err != nil {
		return nil, err
	}

	team.PracticeAreas = practiceAreas

	return team, nil
}

func (r *PostgresPublicRepository) ListNews(
	ctx context.Context,
	filter dto.PublicNewsListQuery,
) ([]domain.News, int64, error) {
	whereQuery := `
		WHERE deleted_at IS NULL
		  AND status = 'published'
		  AND published_at IS NOT NULL
		  AND published_at <= NOW()
		  AND (
				$1 = ''
				OR title ILIKE '%' || $1 || '%'
				OR excerpt ILIKE '%' || $1 || '%'
		  )
		  AND (
				$2::BOOLEAN IS NULL
				OR is_featured = $2
		  )
	`

	const countPrefix = `
		SELECT COUNT(*)
		FROM news
	`

	var total int64

	err := r.db.QueryRowContext(
		ctx,
		countPrefix+whereQuery,
		filter.Search,
		filter.IsFeatured,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count public news: %w",
			err,
		)
	}

	offset := (filter.Page - 1) * filter.Limit

	listQuery := `
		SELECT
			id,
			title,
			slug,
			COALESCE(excerpt, ''),
			content,
			COALESCE(featured_image_url, ''),
			status,
			is_featured,
			published_at,
			created_by,
			updated_by,
			created_at,
			updated_at,
			deleted_at
		FROM news
	` + whereQuery + `
		ORDER BY
			is_featured DESC,
			published_at DESC,
			created_at DESC
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		filter.Search,
		filter.IsFeatured,
		filter.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list public news: %w",
			err,
		)
	}
	defer rows.Close()

	newsList := make([]domain.News, 0)

	for rows.Next() {
		var news domain.News

		err := rows.Scan(
			&news.ID,
			&news.Title,
			&news.Slug,
			&news.Excerpt,
			&news.Content,
			&news.FeaturedImageURL,
			&news.Status,
			&news.IsFeatured,
			&news.PublishedAt,
			&news.CreatedBy,
			&news.UpdatedBy,
			&news.CreatedAt,
			&news.UpdatedAt,
			&news.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"scan public news: %w",
				err,
			)
		}

		newsList = append(
			newsList,
			news,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"iterate public news: %w",
			err,
		)
	}

	return newsList, total, nil
}

func (r *PostgresPublicRepository) FindNewsBySlug(
	ctx context.Context,
	slugValue string,
) (*domain.News, error) {
	const query = `
		SELECT
			id,
			title,
			slug,
			COALESCE(excerpt, ''),
			content,
			COALESCE(featured_image_url, ''),
			status,
			is_featured,
			published_at,
			created_by,
			updated_by,
			created_at,
			updated_at,
			deleted_at
		FROM news
		WHERE LOWER(slug) = LOWER($1)
		  AND deleted_at IS NULL
		  AND status = 'published'
		  AND published_at IS NOT NULL
		  AND published_at <= NOW()
		LIMIT 1
	`

	news := &domain.News{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(slugValue),
	).Scan(
		&news.ID,
		&news.Title,
		&news.Slug,
		&news.Excerpt,
		&news.Content,
		&news.FeaturedImageURL,
		&news.Status,
		&news.IsFeatured,
		&news.PublishedAt,
		&news.CreatedBy,
		&news.UpdatedBy,
		&news.CreatedAt,
		&news.UpdatedAt,
		&news.DeletedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNewsNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find public news by slug: %w",
			err,
		)
	}

	return news, nil
}

func (r *PostgresPublicRepository) findPublicPracticeAreasByTeamID(
	ctx context.Context,
	teamID string,
) ([]domain.PracticeArea, error) {
	const query = `
		SELECT
			pa.id,
			pa.name,
			pa.slug,
			COALESCE(pa.description, ''),
			pa.is_active,
			pa.display_order,
			pa.created_by,
			pa.updated_by,
			pa.created_at,
			pa.updated_at,
			pa.deleted_at
		FROM team_practice_areas tpa
		INNER JOIN practice_areas pa
			ON pa.id = tpa.practice_area_id
		WHERE tpa.team_id = $1
		  AND pa.deleted_at IS NULL
		  AND pa.is_active = TRUE
		ORDER BY
			pa.display_order ASC,
			pa.name ASC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		teamID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find public team practice areas: %w",
			err,
		)
	}
	defer rows.Close()

	practiceAreas := make(
		[]domain.PracticeArea,
		0,
	)

	for rows.Next() {
		var practiceArea domain.PracticeArea

		err := rows.Scan(
			&practiceArea.ID,
			&practiceArea.Name,
			&practiceArea.Slug,
			&practiceArea.Description,
			&practiceArea.IsActive,
			&practiceArea.DisplayOrder,
			&practiceArea.CreatedBy,
			&practiceArea.UpdatedBy,
			&practiceArea.CreatedAt,
			&practiceArea.UpdatedAt,
			&practiceArea.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan public team practice area: %w",
				err,
			)
		}

		practiceAreas = append(
			practiceAreas,
			practiceArea,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate public team practice areas: %w",
			err,
		)
	}

	return practiceAreas, nil
}