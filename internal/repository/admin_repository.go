package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
)

var (
	ErrAdminNotFound = errors.New("admin not found")
)

type AdminRepository interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*domain.Admin, error)

	Create(
		ctx context.Context,
		admin *domain.Admin,
	) error

	UpdateLastLogin(
		ctx context.Context,
		adminID string,
	) error
}

type PostgresAdminRepository struct {
	db *sql.DB
}

func NewPostgresAdminRepository(
	db *sql.DB,
) *PostgresAdminRepository {
	return &PostgresAdminRepository{
		db: db,
	}
}

func (r *PostgresAdminRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*domain.Admin, error) {
	const query = `
		SELECT
			id,
			name,
			email,
			password_hash,
			role,
			is_active,
			last_login_at,
			created_at,
			updated_at
		FROM admins
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`

	admin := &domain.Admin{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(email),
	).Scan(
		&admin.ID,
		&admin.Name,
		&admin.Email,
		&admin.PasswordHash,
		&admin.Role,
		&admin.IsActive,
		&admin.LastLoginAt,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAdminNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find admin by email: %w", err)
	}

	return admin, nil
}

func (r *PostgresAdminRepository) Create(
	ctx context.Context,
	admin *domain.Admin,
) error {
	const query = `
		INSERT INTO admins (
			name,
			email,
			password_hash,
			role,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		admin.Name,
		strings.ToLower(strings.TrimSpace(admin.Email)),
		admin.PasswordHash,
		admin.Role,
		admin.IsActive,
	).Scan(
		&admin.ID,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	return nil
}

func (r *PostgresAdminRepository) UpdateLastLogin(
	ctx context.Context,
	adminID string,
) error {
	const query = `
		UPDATE admins
		SET
			last_login_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		adminID,
	)
	if err != nil {
		return fmt.Errorf("update admin last login: %w", err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}

	if affectedRows == 0 {
		return ErrAdminNotFound
	}

	return nil
}