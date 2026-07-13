package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      uint32 = 64 * 1024
	argonIterations  uint32 = 3
	argonParallelism uint8  = 2
	argonSaltLength         = 16
	argonKeyLength   uint32 = 32
)

var (
	ErrInvalidPasswordHash = errors.New("invalid password hash format")
)

type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonIterations,
		argonMemory,
		argonParallelism,
		argonKeyLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	passwordHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		encodedSalt,
		encodedHash,
	)

	return passwordHash, nil
}

func (h *PasswordHasher) Verify(
	password string,
	encodedHash string,
) (bool, error) {
	parameters, salt, expectedHash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.iterations,
		parameters.memory,
		parameters.parallelism,
		uint32(len(expectedHash)),
	)

	isValid := subtle.ConstantTimeCompare(
		actualHash,
		expectedHash,
	) == 1

	return isValid, nil
}

type argonParameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decodeHash(
	encodedHash string,
) (
	argonParameters,
	[]byte,
	[]byte,
	error,
) {
	parts := strings.Split(encodedHash, "$")

	if len(parts) != 6 {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	if parts[1] != "argon2id" {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	var version int

	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	if version != argon2.Version {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	var parameters argonParameters

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&parameters.memory,
		&parameters.iterations,
		&parameters.parallelism,
	); err != nil {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	return parameters, salt, hash, nil
}