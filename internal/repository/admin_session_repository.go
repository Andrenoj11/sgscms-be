package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrAdminSessionNotFound = errors.New(
		"admin session not found",
	)

	ErrAdminSessionRevoked = errors.New(
		"admin session revoked",
	)

	ErrNonceAlreadyUsed = errors.New(
		"request nonce already used",
	)
)

type AdminSessionRepository interface {
	Create(
		ctx context.Context,
		session *domain.AdminSession,
	) error

	FindByID(
		ctx context.Context,
		id string,
	) (*domain.AdminSession, error)

	FindByRefreshTokenHash(
		ctx context.Context,
		hash string,
	) (*domain.AdminSession, error)

	Rotate(
		ctx context.Context,
		oldSessionID string,
		newSession *domain.AdminSession,
	) error

	RevokeByRefreshTokenHash(
		ctx context.Context,
		hash string,
	) error

	UseNonce(
		ctx context.Context,
		sessionID string,
		nonce string,
		expiresAt time.Time,
	) error
}

type PostgresAdminSessionRepository struct {
	db *sql.DB
}

var _ AdminSessionRepository =
	(*PostgresAdminSessionRepository)(nil)

func NewPostgresAdminSessionRepository(
	db *sql.DB,
) *PostgresAdminSessionRepository {
	return &PostgresAdminSessionRepository{
		db: db,
	}
}

func (r *PostgresAdminSessionRepository) Create(
	ctx context.Context,
	session *domain.AdminSession,
) error {
	const query = `
		INSERT INTO admin_sessions (
			id,
			admin_id,
			refresh_token_hash,
			signing_secret_ciphertext,
			user_agent,
			ip_address,
			expires_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			NULLIF($5, ''),
			NULLIF($6, ''),
			$7
		)
		RETURNING
			created_at,
			updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		session.ID,
		session.AdminID,
		session.RefreshTokenHash,
		session.SigningSecretCiphertext,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
	).Scan(
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"create admin session: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresAdminSessionRepository) FindByID(
	ctx context.Context,
	id string,
) (*domain.AdminSession, error) {
	const query = `
		SELECT
			id,
			admin_id,
			refresh_token_hash,
			signing_secret_ciphertext,
			COALESCE(user_agent, ''),
			COALESCE(ip_address, ''),
			expires_at,
			revoked_at,
			replaced_by_session_id,
			created_at,
			updated_at
		FROM admin_sessions
		WHERE id = $1
		LIMIT 1
	`

	return r.findOne(ctx, query, id)
}

func (r *PostgresAdminSessionRepository) FindByRefreshTokenHash(
	ctx context.Context,
	hash string,
) (*domain.AdminSession, error) {
	const query = `
		SELECT
			id,
			admin_id,
			refresh_token_hash,
			signing_secret_ciphertext,
			COALESCE(user_agent, ''),
			COALESCE(ip_address, ''),
			expires_at,
			revoked_at,
			replaced_by_session_id,
			created_at,
			updated_at
		FROM admin_sessions
		WHERE refresh_token_hash = $1
		LIMIT 1
	`

	return r.findOne(ctx, query, hash)
}

func (r *PostgresAdminSessionRepository) findOne(
	ctx context.Context,
	query string,
	value string,
) (*domain.AdminSession, error) {
	session := &domain.AdminSession{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		value,
	).Scan(
		&session.ID,
		&session.AdminID,
		&session.RefreshTokenHash,
		&session.SigningSecretCiphertext,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.ReplacedBySessionID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAdminSessionNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find admin session: %w",
			err,
		)
	}

	return session, nil
}

func (r *PostgresAdminSessionRepository) Rotate(
	ctx context.Context,
	oldSessionID string,
	newSession *domain.AdminSession,
) error {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin session rotation: %w",
			err,
		)
	}

	defer func() {
		_ = transaction.Rollback()
	}()

	const revokeQuery = `
		UPDATE admin_sessions
		SET
			revoked_at = NOW(),
			replaced_by_session_id = $2,
			updated_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	result, err := transaction.ExecContext(
		ctx,
		revokeQuery,
		oldSessionID,
		newSession.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"revoke old admin session: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read rotated session rows: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrAdminSessionRevoked
	}

	const createQuery = `
		INSERT INTO admin_sessions (
			id,
			admin_id,
			refresh_token_hash,
			signing_secret_ciphertext,
			user_agent,
			ip_address,
			expires_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			NULLIF($5, ''),
			NULLIF($6, ''),
			$7
		)
		RETURNING
			created_at,
			updated_at
	`

	err = transaction.QueryRowContext(
		ctx,
		createQuery,
		newSession.ID,
		newSession.AdminID,
		newSession.RefreshTokenHash,
		newSession.SigningSecretCiphertext,
		newSession.UserAgent,
		newSession.IPAddress,
		newSession.ExpiresAt,
	).Scan(
		&newSession.CreatedAt,
		&newSession.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"create rotated admin session: %w",
			err,
		)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"commit session rotation: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresAdminSessionRepository) RevokeByRefreshTokenHash(
	ctx context.Context,
	hash string,
) error {
	const query = `
		UPDATE admin_sessions
		SET
			revoked_at = COALESCE(
				revoked_at,
				NOW()
			),
			updated_at = NOW()
		WHERE refresh_token_hash = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		hash,
	)
	if err != nil {
		return fmt.Errorf(
			"revoke admin session: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read revoked session rows: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrAdminSessionNotFound
	}

	return nil
}

func (r *PostgresAdminSessionRepository) UseNonce(
	ctx context.Context,
	sessionID string,
	nonce string,
	expiresAt time.Time,
) error {
	_, _ = r.db.ExecContext(
		ctx,
		`
			DELETE FROM request_nonces
			WHERE expires_at <= NOW()
		`,
	)

	const query = `
		INSERT INTO request_nonces (
			nonce,
			session_id,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		nonce,
		sessionID,
		expiresAt,
	)
	if err == nil {
		return nil
	}

	var postgresError *pgconn.PgError

	if errors.As(err, &postgresError) &&
		postgresError.Code == "23505" {
		return ErrNonceAlreadyUsed
	}

	return fmt.Errorf(
		"save request nonce: %w",
		err,
	)
}