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

var (
	ErrNewsNotFound = errors.New("news not found")
)

type NewsRepository interface {
	Create(
		ctx context.Context,
		news *domain.News,
	) error

	FindByID(
		ctx context.Context,
		id string,
	) (*domain.News, error)

	FindBySlug(
		ctx context.Context,
		slug string,
	) (*domain.News, error)

	List(
		ctx context.Context,
		filter dto.NewsListQuery,
	) ([]domain.News, int64, error)

	Update(
		ctx context.Context,
		news *domain.News,
	) error

	SoftDelete(
		ctx context.Context,
		id string,
		adminID string,
	) error
}

type PostgresNewsRepository struct {
	db *sql.DB
}

var _ NewsRepository = (*PostgresNewsRepository)(nil)

func NewPostgresNewsRepository(
	db *sql.DB,
) *PostgresNewsRepository {
	return &PostgresNewsRepository{
		db: db,
	}
}

func (r *PostgresNewsRepository) Create(
	ctx context.Context,
	news *domain.News,
) error {
	const query = `
		INSERT INTO news (
			title,
			slug,
			excerpt,
			content,
			featured_image_url,
			status,
			is_featured,
			published_at,
			created_by,
			updated_by
		)
		VALUES (
			$1,
			$2,
			NULLIF($3, ''),
			$4,
			NULLIF($5, ''),
			$6,
			$7,
			CASE
				WHEN $6 = 'published'
					THEN NOW()
				ELSE NULL
			END,
			$8,
			$8
		)
		RETURNING
			id,
			published_at,
			created_at,
			updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		news.Title,
		news.Slug,
		news.Excerpt,
		news.Content,
		news.FeaturedImageURL,
		news.Status,
		news.IsFeatured,
		news.CreatedBy,
	).Scan(
		&news.ID,
		&news.PublishedAt,
		&news.CreatedAt,
		&news.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"create news: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresNewsRepository) FindByID(
	ctx context.Context,
	id string,
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
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`

	news := &domain.News{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
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
			"find news by ID: %w",
			err,
		)
	}

	return news, nil
}

func (r *PostgresNewsRepository) FindBySlug(
	ctx context.Context,
	slug string,
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
		LIMIT 1
	`

	news := &domain.News{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(slug),
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
			"find news by slug: %w",
			err,
		)
	}

	return news, nil
}

func (r *PostgresNewsRepository) List(
	ctx context.Context,
	filter dto.NewsListQuery,
) ([]domain.News, int64, error) {
	whereQuery := `
		WHERE deleted_at IS NULL
		  AND (
				$1 = ''
				OR title ILIKE '%' || $1 || '%'
				OR excerpt ILIKE '%' || $1 || '%'
		  )
		  AND (
				$2 = ''
				OR status = $2
		  )
		  AND (
				$3::BOOLEAN IS NULL
				OR is_featured = $3
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
		filter.Status,
		filter.IsFeatured,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count news: %w",
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
			CASE
				WHEN status = 'published' THEN 0
				WHEN status = 'draft' THEN 1
				ELSE 2
			END,
			published_at DESC NULLS LAST,
			created_at DESC
		LIMIT $4
		OFFSET $5
	`

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		filter.Search,
		filter.Status,
		filter.IsFeatured,
		filter.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list news: %w",
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
				"scan news row: %w",
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
			"iterate news rows: %w",
			err,
		)
	}

	return newsList, total, nil
}

func (r *PostgresNewsRepository) Update(
	ctx context.Context,
	news *domain.News,
) error {
	const query = `
		UPDATE news
		SET
			title = $2,
			slug = $3,
			excerpt = NULLIF($4, ''),
			content = $5,
			featured_image_url = NULLIF($6, ''),
			status = $7,
			is_featured = $8,
			published_at = CASE
				WHEN $7 = 'published'
					AND published_at IS NULL
					THEN NOW()
				WHEN $7 <> 'published'
					THEN NULL
				ELSE published_at
			END,
			updated_by = $9,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		RETURNING
			published_at,
			updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		news.ID,
		news.Title,
		news.Slug,
		news.Excerpt,
		news.Content,
		news.FeaturedImageURL,
		news.Status,
		news.IsFeatured,
		news.UpdatedBy,
	).Scan(
		&news.PublishedAt,
		&news.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNewsNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"update news: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresNewsRepository) SoftDelete(
	ctx context.Context,
	id string,
	adminID string,
) error {
	const query = `
		UPDATE news
		SET
			deleted_at = NOW(),
			updated_at = NOW(),
			updated_by = $2,
			status = 'archived',
			is_featured = FALSE,
			published_at = NULL
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		id,
		adminID,
	)
	if err != nil {
		return fmt.Errorf(
			"soft delete news: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read affected rows after deleting news: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrNewsNotFound
	}

	return nil
}