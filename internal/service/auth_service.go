package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/security"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAdminInactive      = errors.New("admin account is inactive")
)

type AuthService struct {
	adminRepository repository.AdminRepository
	passwordHasher  *security.PasswordHasher
	jwtManager      *security.JWTManager
}

func NewAuthService(
	adminRepository repository.AdminRepository,
	passwordHasher *security.PasswordHasher,
	jwtManager *security.JWTManager,
) *AuthService {
	return &AuthService{
		adminRepository: adminRepository,
		passwordHasher:  passwordHasher,
		jwtManager:      jwtManager,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	request dto.LoginRequest,
) (*dto.LoginResponse, error) {
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

	passwordValid, err := s.passwordHasher.Verify(
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

	accessToken, expiresIn, err :=
		s.jwtManager.GenerateAccessToken(admin)
	if err != nil {
		return nil, fmt.Errorf(
			"generate admin access token: %w",
			err,
		)
	}

	loginResponse := &dto.LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Admin: dto.AdminResponse{
			ID:    admin.ID,
			Name:  admin.Name,
			Email: admin.Email,
			Role:  string(admin.Role),
		},
	}

	return loginResponse, nil
}