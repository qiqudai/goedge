package services

import (
	"cdn-api/config"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

type CryptService struct{}

var Crypto = &CryptService{}

func (s *CryptService) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	key := []byte(config.App.SecretKey)
	if len(key) != 32 {
		// Fallback or error? For now, if empty or invalid, try padding or error.
		// Detailed implementation: ensure key is 32 bytes.
		// Users might set it short.
		if len(key) == 0 {
			return plain, errors.New("secret key not configured")
		}
		// Simple pad or truncate for robustness (though configured key should be correct)
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *CryptService) Decrypt(cipherStr string) (string, error) {
	if cipherStr == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(cipherStr)
	if err != nil {
		return "", err
	}

	key := []byte(config.App.SecretKey)
	if len(key) != 32 {
		if len(key) == 0 {
			return "", errors.New("secret key not configured")
		}
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
