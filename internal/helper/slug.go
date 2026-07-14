package helper

import (
	"strings"

	sluglib "github.com/gosimple/slug"
)

func GenerateSlug(value string) string {
	normalized := strings.TrimSpace(value)

	return sluglib.Make(normalized)
}