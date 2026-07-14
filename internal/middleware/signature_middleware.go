package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/gin-gonic/gin"
)

const signatureBodyLimit int64 = 4 * 1024 * 1024

type SignatureMiddleware struct {
	sessionRepository repository.AdminSessionRepository

	secretCipher *security.SecretCipher

	maxAge time.Duration
}

func NewSignatureMiddleware(
	sessionRepository repository.AdminSessionRepository,
	secretCipher *security.SecretCipher,
	maxAge time.Duration,
) *SignatureMiddleware {
	return &SignatureMiddleware{
		sessionRepository: sessionRepository,
		secretCipher:      secretCipher,
		maxAge:            maxAge,
	}
}

func (m *SignatureMiddleware) Verify() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err :=
			security.GetCurrentSessionID(c)
		if err != nil {
			signatureError(
				c,
				"Authentication session is unavailable",
			)
			return
		}

		timestampHeader := strings.TrimSpace(
			c.GetHeader("X-Timestamp"),
		)

		nonce := strings.TrimSpace(
			c.GetHeader("X-Nonce"),
		)

		receivedSignature := strings.TrimSpace(
			c.GetHeader("X-Signature"),
		)

		if timestampHeader == "" ||
			nonce == "" ||
			receivedSignature == "" {
			signatureError(
				c,
				"Signature headers are required",
			)
			return
		}

		if len(nonce) < 16 || len(nonce) > 100 {
			signatureError(
				c,
				"Request nonce is invalid",
			)
			return
		}

		timestamp, err := strconv.ParseInt(
			timestampHeader,
			10,
			64,
		)
		if err != nil {
			signatureError(
				c,
				"Request timestamp is invalid",
			)
			return
		}

		requestTime := time.Unix(
			timestamp,
			0,
		).UTC()

		now := time.Now().UTC()

		if requestTime.Before(
			now.Add(-m.maxAge),
		) ||
			requestTime.After(
				now.Add(m.maxAge),
			) {
			signatureError(
				c,
				"Request timestamp has expired",
			)
			return
		}

		body, err := readAndRestoreBody(
			c.Request,
		)
		if err != nil {
			response.Error(
				c,
				http.StatusBadRequest,
				"Unable to read request body",
				nil,
			)
			c.Abort()
			return
		}

		session, err :=
			m.sessionRepository.FindByID(
				c.Request.Context(),
				sessionID,
			)
		if err != nil {
			signatureError(
				c,
				"Authentication session is invalid",
			)
			return
		}

		signingSecret, err :=
			m.secretCipher.Decrypt(
				session.SigningSecretCiphertext,
			)
		if err != nil {
			response.Error(
				c,
				http.StatusInternalServerError,
				"Internal server error",
				nil,
			)
			c.Abort()
			return
		}

		expectedSignature :=
			buildRequestSignature(
				signingSecret,
				c.Request.Method,
				c.Request.URL.RequestURI(),
				timestampHeader,
				nonce,
				body,
			)

		if !hmac.Equal(
			[]byte(
				strings.ToLower(
					receivedSignature,
				),
			),
			[]byte(expectedSignature),
		) {
			signatureError(
				c,
				"Request signature is invalid",
			)
			return
		}

		err = m.sessionRepository.UseNonce(
			c.Request.Context(),
			sessionID,
			nonce,
			now.Add(m.maxAge),
		)

		if errors.Is(
			err,
			repository.ErrNonceAlreadyUsed,
		) {
			signatureError(
				c,
				"Request nonce has already been used",
			)
			return
		}

		if err != nil {
			response.Error(
				c,
				http.StatusInternalServerError,
				"Internal server error",
				nil,
			)
			c.Abort()
			return
		}

		c.Next()
	}
}

func readAndRestoreBody(
	request *http.Request,
) ([]byte, error) {
	if request.Body == nil {
		return []byte{}, nil
	}

	limitedReader := io.LimitReader(
		request.Body,
		signatureBodyLimit+1,
	)

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) >
		signatureBodyLimit {
		return nil, fmt.Errorf(
			"request body exceeds signature limit",
		)
	}

	request.Body = io.NopCloser(
		bytes.NewReader(body),
	)

	return body, nil
}

func buildRequestSignature(
	secret string,
	method string,
	requestURI string,
	timestamp string,
	nonce string,
	body []byte,
) string {
	bodyHash := sha256.Sum256(body)

	canonicalPayload := strings.Join(
		[]string{
			strings.ToUpper(method),
			requestURI,
			timestamp,
			nonce,
			hex.EncodeToString(bodyHash[:]),
		},
		"\n",
	)

	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)

	_, _ = mac.Write(
		[]byte(canonicalPayload),
	)

	return hex.EncodeToString(
		mac.Sum(nil),
	)
}

func signatureError(
	c *gin.Context,
	message string,
) {
	response.Error(
		c,
		http.StatusUnauthorized,
		message,
		nil,
	)

	c.Abort()
}