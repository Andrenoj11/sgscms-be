package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func GenerateSecureToken(
	length int,
) (string, error) {
	if length < 16 {
		return "", fmt.Errorf(
			"secure token length must be at least 16 bytes",
		)
	}

	buffer := make([]byte, length)

	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf(
			"generate secure random token: %w",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(
		buffer,
	), nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}