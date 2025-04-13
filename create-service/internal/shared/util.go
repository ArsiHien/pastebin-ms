package shared

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateURL(length int) (string, error) {
	byteLen := (length + 1) / 2
	bytes := make([]byte, byteLen)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}
