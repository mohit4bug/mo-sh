package shared

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomString(length int) string {
	byteLength := (length + 1) / 2
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}
