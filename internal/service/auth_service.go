package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New(
		"invalid email or password",
	)

	ErrAdminInactive = errors.New(
		"admin account is inactive",
	)

	ErrInvalidRefreshToken = errors.New(
		"invalid refresh token",
	)

	ErrExpiredRefreshToken = errors.New(
		"refresh token has expired",
	)
)

type AuthService struct {
	adminRepository   repository.AdminRepository
	sessionRepository repository.AdminSessionRepository
	passwordHasher    *security.PasswordHasher
	jwtManager        *security.JWTManager
	secretCipher      *security.SecretCipher
	refreshTTL        time.Duration
}

func NewAuthService(
	adminRepository repository.AdminRepository,
	sessionRepository repository.AdminSessionRepository,
	passwordHasher *security.PasswordHasher,
	jwtManager *security.JWTManager,
	secretCipher *security.SecretCipher,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		adminRepository:   adminRepository,
		sessionRepository: sessionRepository,
		passwordHasher:    passwordHasher,
		jwtManager:        jwtManager,
		secretCipher:      secretCipher,
		refreshTTL:        refreshTTL,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	request dto.LoginRequest,
	userAgent string,
	ipAddress string,
) (*dto.AuthResult, error) {
	email := strings.ToLower(
		strings.TrimSpace(request.Email),
	)

	admin, err := s.adminRepository.FindByEmail(
		ctx,
		email,
	)
	if errors.Is(err, repository.ErrAdminNotFound) {
		return nil, ErrInvalidCredentials
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find admin for login: %w",
			err,
		)
	}

	if !admin.IsActive {
		return nil, ErrAdminInactive
	}

	passwordValid, err :=
		s.passwordHasher.Verify(
			request.Password,
			admin.PasswordHash,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"verify admin password: %w",
			err,
		)
	}

	if !passwordValid {
		return nil, ErrInvalidCredentials
	}

	if err := s.adminRepository.UpdateLastLogin(
		ctx,
		admin.ID,
	); err != nil {
		return nil, fmt.Errorf(
			"update admin last login: %w",
			err,
		)
	}

	session, refreshToken, signingKey, err :=
		s.createSession(
			admin.ID,
			userAgent,
			ipAddress,
		)
	if err != nil {
		return nil, err
	}

	if err := s.sessionRepository.Create(
		ctx,
		session,
	); err != nil {
		return nil, err
	}

	accessToken, expiresIn, err :=
		s.jwtManager.GenerateAccessToken(
			admin,
			session.ID,
		)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResult{
		RefreshToken: refreshToken,
		Response: dto.LoginResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			SigningKey:  signingKey,
			Admin: dto.AdminResponse{
				ID:    admin.ID,
				Name:  admin.Name,
				Email: admin.Email,
				Role:  string(admin.Role),
			},
		},
	}, nil
}

func (s *AuthService) Refresh(
	ctx context.Context,
	refreshToken string,
	userAgent string,
	ipAddress string,
) (*dto.RefreshResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}

	refreshHash := security.HashToken(
		refreshToken,
	)

	oldSession, err :=
		s.sessionRepository.FindByRefreshTokenHash(
			ctx,
			refreshHash,
		)
	if errors.Is(
		err,
		repository.ErrAdminSessionNotFound,
	) {
		return nil, ErrInvalidRefreshToken
	}

	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	if oldSession.RevokedAt != nil {
		return nil, ErrInvalidRefreshToken
	}

	if !oldSession.ExpiresAt.After(now) {
		return nil, ErrExpiredRefreshToken
	}

	admin, err := s.adminRepository.FindByID(
		ctx,
		oldSession.AdminID,
	)
	if errors.Is(err, repository.ErrAdminNotFound) {
		return nil, ErrInvalidRefreshToken
	}

	if err != nil {
		return nil, err
	}

	if !admin.IsActive {
		return nil, ErrAdminInactive
	}

	newSession, newRefreshToken, signingKey, err :=
		s.createSession(
			admin.ID,
			userAgent,
			ipAddress,
		)
	if err != nil {
		return nil, err
	}

	if err := s.sessionRepository.Rotate(
		ctx,
		oldSession.ID,
		newSession,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrAdminSessionRevoked,
		) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, err
	}

	accessToken, expiresIn, err :=
		s.jwtManager.GenerateAccessToken(
			admin,
			newSession.ID,
		)
	if err != nil {
		return nil, err
	}

	return &dto.RefreshResult{
		RefreshToken: newRefreshToken,
		Response: dto.RefreshResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			SigningKey:  signingKey,
		},
	}, nil
}

func (s *AuthService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}

	hash := security.HashToken(refreshToken)

	err := s.sessionRepository.
		RevokeByRefreshTokenHash(
			ctx,
			hash,
		)

	if errors.Is(
		err,
		repository.ErrAdminSessionNotFound,
	) {
		return nil
	}

	return err
}

func (s *AuthService) createSession(
	adminID string,
	userAgent string,
	ipAddress string,
) (
	*domain.AdminSession,
	string,
	string,
	error,
) {
	refreshToken, err :=
		security.GenerateSecureToken(48)
	if err != nil {
		return nil, "", "", err
	}

	signingKey, err :=
		security.GenerateSecureToken(32)
	if err != nil {
		return nil, "", "", err
	}

	encryptedSigningKey, err :=
		s.secretCipher.Encrypt(signingKey)
	if err != nil {
		return nil, "", "", err
	}

	session := &domain.AdminSession{
		ID: uuid.NewString(),

		AdminID: adminID,

		RefreshTokenHash: security.HashToken(
			refreshToken,
		),

		SigningSecretCiphertext: encryptedSigningKey,

		UserAgent: strings.TrimSpace(
			userAgent,
		),

		IPAddress: strings.TrimSpace(
			ipAddress,
		),

		ExpiresAt: time.Now().
			UTC().
			Add(s.refreshTTL),
	}

	return session,
		refreshToken,
		signingKey,
		nil
}