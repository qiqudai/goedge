package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SecretKey = []byte(getJWTSecret())

func getJWTSecret() string {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	return "YOUR_SECRET_KEY_SHOULD_BE_IN_ENV"
}

type Claims struct {
	UserID int64  `json:"uid"`
	Role   string `json:"role"` // "admin" or "user"
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for a user
func GenerateToken(userID int64, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "cdn-core-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

// ParseToken validates the token and returns the claims
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
