package utils

import (
	"cdn-api/config"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var insecureJWTSecrets = map[string]struct{}{
	"":                                 {},
	"YOUR_SECRET_KEY_SHOULD_BE_IN_ENV": {},
	"0123456789abcdef0123456789abcdef": {},
}

func jwtSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(config.App.JWTSecret)
	}
	if secret == "" {
		secret = strings.TrimSpace(config.App.SecretKey)
	}
	if _, insecure := insecureJWTSecrets[secret]; insecure || len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret is not securely configured")
	}
	return []byte(secret), nil
}

type Claims struct {
	UserID int64  `json:"uid"`
	Role   string `json:"role"` // "admin" or "user"
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for a user
func GenerateToken(userID int64, role string) (string, error) {
	return GenerateTokenWithExpiry(userID, role, 24*time.Hour)
}

// GenerateTokenWithExpiry creates a JWT token for a user with a custom TTL.
func GenerateTokenWithExpiry(userID int64, role string, ttl time.Duration) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expirationTime := time.Now().Add(ttl)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "cdn-core-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken validates the token and returns the claims
func ParseToken(tokenString string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
