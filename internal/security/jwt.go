package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess = "access"
)

var (
	ErrInvalidToken = errors.New(
		"invalid token",
	)

	ErrExpiredToken = errors.New(
		"token has expired",
	)
)

type AccessTokenClaims struct {
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	TokenType string `json:"token_type"`

	jwt.RegisteredClaims
}

type JWTManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	now       func() time.Time
}

func NewJWTManager(
	cfg config.JWTConfig,
) *JWTManager {
	return &JWTManager{
		secret:    []byte(cfg.Secret),
		issuer:    cfg.Issuer,
		accessTTL: cfg.AccessTTL,
		now:       time.Now,
	}
}

func (m *JWTManager) GenerateAccessToken(
	admin *domain.Admin,
	sessionID string,
) (
	string,
	int64,
	error,
) {
	now := m.now().UTC()
	expiresAt := now.Add(m.accessTTL)

	claims := AccessTokenClaims{
		Email:     admin.Email,
		Role:      string(admin.Role),
		SessionID: sessionID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   admin.ID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		m.secret,
	)
	if err != nil {
		return "", 0, fmt.Errorf(
			"sign access token: %w",
			err,
		)
	}

	return signedToken,
		int64(m.accessTTL.Seconds()),
		nil
}

func (m *JWTManager) ParseAccessToken(
	tokenString string,
) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method !=
				jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return m.secret, nil
		},
		jwt.WithValidMethods(
			[]string{
				jwt.SigningMethodHS256.Alg(),
			},
		),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)

	if err != nil {
		if errors.Is(
			err,
			jwt.ErrTokenExpired,
		) {
			return nil, ErrExpiredToken
		}

		return nil, ErrInvalidToken
	}

	if token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != TokenTypeAccess ||
		claims.Subject == "" ||
		claims.SessionID == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}