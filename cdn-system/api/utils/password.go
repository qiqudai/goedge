package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func IsSHA256Hex(input string) bool {
	if len(input) != 64 {
		return false
	}
	for _, r := range input {
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'a' && r <= 'f' {
			continue
		}
		if r >= 'A' && r <= 'F' {
			continue
		}
		return false
	}
	return true
}

func NormalizePasswordInput(input string) string {
	trimmed := strings.TrimSpace(input)
	if IsSHA256Hex(trimmed) {
		return strings.ToLower(trimmed)
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func HashPasswordForStorage(input string) (string, error) {
	normalized := NormalizePasswordInput(input)
	hashed, err := bcrypt.GenerateFromPassword([]byte(normalized), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
