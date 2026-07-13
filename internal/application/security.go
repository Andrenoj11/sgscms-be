package application

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strconv"
	"strings"
)

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", fmt.Errorf("password must contain at least 12 characters")
	}
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	h := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(h)), nil
}
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return false
	}
	var m, t uint64
	var p64 uint64
	for _, v := range strings.Split(parts[2], ",") {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) != 2 {
			return false
		}
		n, e := strconv.ParseUint(kv[1], 10, 32)
		if e != nil {
			return false
		}
		switch kv[0] {
		case "m":
			m = n
		case "t":
			t = n
		case "p":
			p64 = n
		}
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[3])
	if e != nil {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(t), uint32(m), uint8(p64), uint32(len(expected)))
	return hmac.Equal(actual, expected)
}
func RandomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func SHA256Hex(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
