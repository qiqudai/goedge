package utils

import (
	"cdn-api/config"
	"testing"
	"time"
)

func TestGenerateAndParseTokenWithSecureSecret(t *testing.T) {
	oldSecret := config.App.SecretKey
	oldJWTSecret := config.App.JWTSecret
	config.App.SecretKey = ""
	config.App.JWTSecret = ""
	t.Cleanup(func() {
		config.App.SecretKey = oldSecret
		config.App.JWTSecret = oldJWTSecret
	})
	t.Setenv("JWT_SECRET", "secure-test-secret-32-bytes-minimum-value")

	token, err := GenerateTokenWithExpiry(42, "admin", time.Minute)
	if err != nil {
		t.Fatalf("GenerateTokenWithExpiry() error = %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.Role != "admin" {
		t.Fatalf("unexpected claims: uid=%d role=%q", claims.UserID, claims.Role)
	}
}

func TestJWTUsesDedicatedConfigSecretBeforeAppSecret(t *testing.T) {
	oldSecret := config.App.SecretKey
	oldJWTSecret := config.App.JWTSecret
	t.Cleanup(func() {
		config.App.SecretKey = oldSecret
		config.App.JWTSecret = oldJWTSecret
	})
	t.Setenv("JWT_SECRET", "")
	config.App.SecretKey = "0123456789abcdef0123456789abcdef"
	config.App.JWTSecret = "secure-config-jwt-secret-32-bytes-value"

	token, err := GenerateTokenWithExpiry(7, "admin", time.Minute)
	if err != nil {
		t.Fatalf("GenerateTokenWithExpiry() error = %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 7 || claims.Role != "admin" {
		t.Fatalf("unexpected claims: uid=%d role=%q", claims.UserID, claims.Role)
	}
}

func TestJWTRejectsInsecureDefaultSecrets(t *testing.T) {
	oldSecret := config.App.SecretKey
	oldJWTSecret := config.App.JWTSecret
	t.Cleanup(func() {
		config.App.SecretKey = oldSecret
		config.App.JWTSecret = oldJWTSecret
	})
	t.Setenv("JWT_SECRET", "")
	config.App.JWTSecret = ""

	for _, secret := range []string{
		"",
		"YOUR_SECRET_KEY_SHOULD_BE_IN_ENV",
		"0123456789abcdef0123456789abcdef",
		"short-secret",
	} {
		config.App.SecretKey = secret
		if _, err := GenerateTokenWithExpiry(1, "admin", time.Minute); err == nil {
			t.Fatalf("expected error for insecure secret %q", secret)
		}
	}
}
