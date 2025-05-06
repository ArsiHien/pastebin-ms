package shared

import (
	"github.com/google/uuid"
	"strings"
)

func GenerateURL(length int) (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	cleanUUID := strings.ReplaceAll(u.String(), "-", "")
	if length > len(cleanUUID) {
		length = len(cleanUUID)
	}
	return cleanUUID[:length], nil
}
