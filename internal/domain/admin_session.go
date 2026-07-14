package domain

import "time"

type AdminSession struct {
	ID                      string
	AdminID                 string
	RefreshTokenHash        string
	SigningSecretCiphertext string
	UserAgent               string
	IPAddress               string
	ExpiresAt               time.Time
	RevokedAt               *time.Time
	ReplacedBySessionID     *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (s *AdminSession) IsActive(now time.Time) bool {
	return s.RevokedAt == nil &&
		s.ExpiresAt.After(now)
}