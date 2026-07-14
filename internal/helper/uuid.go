package helper

import "github.com/google/uuid"

func IsValidUUID(value string) bool {
	_, err := uuid.Parse(value)

	return err == nil
}