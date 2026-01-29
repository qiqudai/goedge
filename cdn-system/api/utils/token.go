package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateNodeToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}
