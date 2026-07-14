package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type SecretCipher struct {
	aead cipher.AEAD
}

func NewSecretCipher(
	key []byte,
) (*SecretCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf(
			"create AES cipher: %w",
			err,
		)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"create AES-GCM cipher: %w",
			err,
		)
	}

	return &SecretCipher{
		aead: aead,
	}, nil
}

func (c *SecretCipher) Encrypt(
	plaintext string,
) (string, error) {
	nonce := make(
		[]byte,
		c.aead.NonceSize(),
	)

	if _, err := io.ReadFull(
		rand.Reader,
		nonce,
	); err != nil {
		return "", fmt.Errorf(
			"generate encryption nonce: %w",
			err,
		)
	}

	ciphertext := c.aead.Seal(
		nonce,
		nonce,
		[]byte(plaintext),
		nil,
	)

	return base64.RawStdEncoding.EncodeToString(
		ciphertext,
	), nil
}

func (c *SecretCipher) Decrypt(
	encoded string,
) (string, error) {
	ciphertext, err :=
		base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf(
			"decode encrypted secret: %w",
			err,
		)
	}

	nonceSize := c.aead.NonceSize()

	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf(
			"encrypted secret is invalid",
		)
	}

	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	plaintext, err := c.aead.Open(
		nil,
		nonce,
		encryptedData,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf(
			"decrypt secret: %w",
			err,
		)
	}

	return string(plaintext), nil
}