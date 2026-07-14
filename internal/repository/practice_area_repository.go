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
	ErrPracticeAreaNotFound = errors.New(
		"practice area not found",
	)
)

type PracticeAreaRepository interface {
	Create(
		ctx context.Context,
		practiceArea *domain.PracticeArea,
	) error

	FindByID(
		ctx context.Context,
		id string,
	) (*domain.PracticeArea, error)

	FindByName(
		ctx context.Context,
		name string,
	) (*domain.PracticeArea, error)

	FindBySlug(
		ctx context.Context,
		slug string,
	) (*domain.PracticeArea, error)

	List(
		ctx context.Context,
		query dto.PracticeAreaListQuery,
	) ([]domain.PracticeArea, int64, error)

	Update(
		ctx context.Context,
		practiceArea *domain.PracticeArea,
	) error

	SoftDelete(
		ctx context.Context,
		id string,
		adminID string,
	) error
}

type PostgresPracticeAreaRepository struct {
	db *sql.DB
}

func NewPostgresPracticeAreaRepository(
	db *sql.DB,
) *PostgresPracticeAreaRepository {
	return &PostgresPracticeAreaRepository{
		db: db,
	}
}

func (r *PostgresPracticeAreaRepository) Create(
	ctx context.Context,
	practiceArea *domain.PracticeArea,
) error {
	const query = `
		INSERT INTO practice_areas (
			name,
			slug,
			description,
			is_active,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		practiceArea.Name,
		practiceArea.Slug,
		practiceArea.Description,
		practiceArea.IsActive,
		practiceArea.DisplayOrder,
		practiceArea.CreatedBy,
	).Scan(
		&practiceArea.ID,
		&practiceArea.CreatedAt,
		&practiceArea.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf(
			"create practice area: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresPracticeAreaRepository) FindByID(
	ctx context.Context,
	id string,
) (*domain.PracticeArea, error) {
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
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`

	practiceArea := &domain.PracticeArea{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
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

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPracticeAreaNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find practice area by ID: %w",
			err,
		)
	}

	return practiceArea, nil
}

func (r *PostgresPracticeAreaRepository) FindByName(
	ctx context.Context,
	name string,
) (*domain.PracticeArea, error) {
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
		WHERE LOWER(name) = LOWER($1)
		  AND deleted_at IS NULL
		LIMIT 1
	`

	return r.findOne(ctx, query, strings.TrimSpace(name))
}

func (r *PostgresPracticeAreaRepository) FindBySlug(
	ctx context.Context,
	value string,
) (*domain.PracticeArea, error) {
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
		WHERE LOWER(slug) = LOWER($1)
		  AND deleted_at IS NULL
		LIMIT 1
	`

	return r.findOne(ctx, query, strings.TrimSpace(value))
}

func (r *PostgresPracticeAreaRepository) findOne(
	ctx context.Context,
	query string,
	value string,
) (*domain.PracticeArea, error) {
	practiceArea := &domain.PracticeArea{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		value,
	).Scan(
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

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPracticeAreaNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find practice area: %w",
			err,
		)
	}

	return practiceArea, nil
}

func (r *PostgresPracticeAreaRepository) List(
	ctx context.Context,
	filter dto.PracticeAreaListQuery,
) ([]domain.PracticeArea, int64, error) {
	whereQuery := `
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%')
		  AND ($2::BOOLEAN IS NULL OR is_active = $2)
	`

	const countPrefix = `
		SELECT COUNT(*)
		FROM practice_areas
	`

	var total int64

	err := r.db.QueryRowContext(
		ctx,
		countPrefix+whereQuery,
		filter.Search,
		filter.Active,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count practice areas: %w",
			err,
		)
	}

	offset := (filter.Page - 1) * filter.Limit

	listQuery := `
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
	` + whereQuery + `
		ORDER BY
			display_order ASC,
			name ASC
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		filter.Search,
		filter.Active,
		filter.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list practice areas: %w",
			err,
		)
	}
	defer rows.Close()

	practiceAreas := make([]domain.PracticeArea, 0)

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
			return nil, 0, fmt.Errorf(
				"scan practice area row: %w",
				err,
			)
		}

		practiceAreas = append(
			practiceAreas,
			practiceArea,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"iterate practice area rows: %w",
			err,
		)
	}

	return practiceAreas, total, nil
}