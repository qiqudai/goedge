package services

import (
	"testing"
	"time"
)

func TestParseACMERateLimitRetryAt(t *testing.T) {
	msg := "acme: error: 429 :: POST :: https://acme-v02.api.letsencrypt.org/acme/new-order :: urn:ietf:params:acme:error:rateLimited :: too many certificates (5) already issued for this exact set of identifiers in the last 168h0m0s, retry after 2026-06-26 18:48:37 UTC"

	got, ok := parseACMERateLimitRetryAt(msg)
	if !ok {
		t.Fatal("expected rate limit retry_at to parse")
	}
	want := time.Date(2026, 6, 26, 18, 48, 37, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("retry_at = %s, want %s", got, want)
	}
}

func TestParseACMERateLimitRetryAtIgnoresNonRateLimit(t *testing.T) {
	msg := "acme: error: 400 :: urn:ietf:params:acme:error:externalAccountRequired :: The request must include externalAccountBinding"
	if got, ok := parseACMERateLimitRetryAt(msg); ok {
		t.Fatalf("unexpected retry_at parsed: %s", got)
	}
}
