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
	ErrTeamNotFound = errors.New("team not found")
)

type TeamRepository interface {
	Create(
		ctx context.Context,
		team *domain.Team,
		practiceAreaIDs []string,
	) error

	FindByID(
		ctx context.Context,
		id string,
	) (*domain.Team, error)

	FindByName(
		ctx context.Context,
		name string,
	) (*domain.Team, error)

	FindBySlug(
		ctx context.Context,
		slug string,
	) (*domain.Team, error)

	List(
		ctx context.Context,
		filter dto.TeamListQuery,
	) ([]domain.Team, int64, error)

	Update(
		ctx context.Context,
		team *domain.Team,
		practiceAreaIDs []string,
	) error

	SoftDelete(
		ctx context.Context,
		id string,
		adminID string,
	) error
}

type PostgresTeamRepository struct {
	db *sql.DB
}

var _ TeamRepository = (*PostgresTeamRepository)(nil)

func NewPostgresTeamRepository(
	db *sql.DB,
) *PostgresTeamRepository {
	return &PostgresTeamRepository{
		db: db,
	}
}

func (r *PostgresTeamRepository) Create(
	ctx context.Context,
	team *domain.Team,
	practiceAreaIDs []string,
) error {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin create team transaction: %w",
			err,
		)
	}

	defer func() {
		_ = transaction.Rollback()
	}()

	const query = `
		INSERT INTO teams (
			name,
			slug,
			degree,
			position,
			short_description,
			biography,
			photo_url,
			email,
			linkedin_url,
			display_order,
			is_published,
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
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			NULLIF($9, ''),
			$10,
			$11,
			CASE
				WHEN $11 = TRUE THEN NOW()
				ELSE NULL
			END,
			$12,
			$12
		)
		RETURNING
			id,
			published_at,
			created_at,
			updated_at
	`

	err = transaction.QueryRowContext(
		ctx,
		query,
		team.Name,
		team.Slug,
		team.Degree,
		team.Position,
		team.ShortDescription,
		team.Biography,
		team.PhotoURL,
		team.Email,
		team.LinkedInURL,
		team.DisplayOrder,
		team.IsPublished,
		team.CreatedBy,
	).Scan(
		&team.ID,
		&team.PublishedAt,
		&team.CreatedAt,
		&team.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"insert team: %w",
			err,
		)
	}

	if err := insertTeamPracticeAreas(
		ctx,
		transaction,
		team.ID,
		practiceAreaIDs,
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"commit create team transaction: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresTeamRepository) FindByID(
	ctx context.Context,
	id string,
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
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`

	team := &domain.Team{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
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
			"find team by ID: %w",
			err,
		)
	}

	practiceAreas, err := r.findPracticeAreasByTeamID(
		ctx,
		team.ID,
	)
	if err != nil {
		return nil, err
	}

	team.PracticeAreas = practiceAreas

	return team, nil
}

func (r *PostgresTeamRepository) FindByName(
	ctx context.Context,
	name string,
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
		WHERE LOWER(name) = LOWER($1)
		  AND deleted_at IS NULL
		LIMIT 1
	`

	return r.findOne(
		ctx,
		query,
		strings.TrimSpace(name),
	)
}

func (r *PostgresTeamRepository) FindBySlug(
	ctx context.Context,
	slug string,
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
		LIMIT 1
	`

	return r.findOne(
		ctx,
		query,
		strings.TrimSpace(slug),
	)
}

func (r *PostgresTeamRepository) findOne(
	ctx context.Context,
	query string,
	value string,
) (*domain.Team, error) {
	team := &domain.Team{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		value,
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
			"find team: %w",
			err,
		)
	}

	return team, nil
}

func (r *PostgresTeamRepository) List(
	ctx context.Context,
	filter dto.TeamListQuery,
) ([]domain.Team, int64, error) {
	whereQuery := `
		WHERE deleted_at IS NULL
		  AND (
				$1 = ''
				OR name ILIKE '%' || $1 || '%'
				OR position ILIKE '%' || $1 || '%'
		  )
		  AND (
				$2::BOOLEAN IS NULL
				OR is_published = $2
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
		filter.Published,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count teams: %w",
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
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		filter.Search,
		filter.Published,
		filter.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list teams: %w",
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
				"scan team row: %w",
				err,
			)
		}

		practiceAreas, err :=
			r.findPracticeAreasByTeamID(
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
			"iterate team rows: %w",
			err,
		)
	}

	return teams, total, nil
}

func (r *PostgresTeamRepository) Update(
	ctx context.Context,
	team *domain.Team,
	practiceAreaIDs []string,
) error {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin update team transaction: %w",
			err,
		)
	}

	defer func() {
		_ = transaction.Rollback()
	}()

	const query = `
		UPDATE teams
		SET
			name = $2,
			slug = $3,
			degree = NULLIF($4, ''),
			position = $5,
			short_description = NULLIF($6, ''),
			biography = NULLIF($7, ''),
			photo_url = NULLIF($8, ''),
			email = NULLIF($9, ''),
			linkedin_url = NULLIF($10, ''),
			display_order = $11,
			is_published = $12,
			published_at = CASE
				WHEN $12 = TRUE
					AND published_at IS NULL
					THEN NOW()
				WHEN $12 = FALSE
					THEN NULL
				ELSE published_at
			END,
			updated_by = $13,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		RETURNING
			published_at,
			updated_at
	`

	err = transaction.QueryRowContext(
		ctx,
		query,
		team.ID,
		team.Name,
		team.Slug,
		team.Degree,
		team.Position,
		team.ShortDescription,
		team.Biography,
		team.PhotoURL,
		team.Email,
		team.LinkedInURL,
		team.DisplayOrder,
		team.IsPublished,
		team.UpdatedBy,
	).Scan(
		&team.PublishedAt,
		&team.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrTeamNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"update team: %w",
			err,
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			DELETE FROM team_practice_areas
			WHERE team_id = $1
		`,
		team.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"delete existing team practice areas: %w",
			err,
		)
	}

	if err := insertTeamPracticeAreas(
		ctx,
		transaction,
		team.ID,
		practiceAreaIDs,
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"commit update team transaction: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresTeamRepository) SoftDelete(
	ctx context.Context,
	id string,
	adminID string,
) error {
	const query = `
		UPDATE teams
		SET
			deleted_at = NOW(),
			updated_at = NOW(),
			updated_by = $2,
			is_published = FALSE,
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
			"soft delete team: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read affected rows after deleting team: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrTeamNotFound
	}

	return nil
}

func (r *PostgresTeamRepository) findPracticeAreasByTeamID(
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
			"find team practice areas: %w",
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
				"scan team practice area: %w",
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
			"iterate team practice areas: %w",
			err,
		)
	}

	return practiceAreas, nil
}

func insertTeamPracticeAreas(
	ctx context.Context,
	transaction *sql.Tx,
	teamID string,
	practiceAreaIDs []string,
) error {
	const query = `
		INSERT INTO team_practice_areas (
			team_id,
			practice_area_id
		)
		VALUES ($1, $2)
	`

	for _, practiceAreaID := range practiceAreaIDs {
		_, err := transaction.ExecContext(
			ctx,
			query,
			teamID,
			practiceAreaID,
		)
		if err != nil {
			return fmt.Errorf(
				"insert team practice area: %w",
				err,
			)
		}
	}

	return nil
}