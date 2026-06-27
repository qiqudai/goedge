package acme

import "testing"

const sampleLegacyChain = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIQLegacy
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIQR12
-----END CERTIFICATE-----
`

const sampleYRChain = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIQLeaf
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIQYR2
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIQRootYR
-----END CERTIFICATE-----
`

func TestUsesIncompatibleLetsEncryptChain(t *testing.T) {
	t.Parallel()

	if UsesIncompatibleLetsEncryptChain(sampleLegacyChain) {
		t.Fatalf("legacy sample should not be rejected")
	}
	if UsesIncompatibleLetsEncryptChain(sampleYRChain) {
		t.Fatalf("YR2/Root YR chain should not be rejected")
	}
}

func TestResolvePreferredChain(t *testing.T) {
	t.Parallel()

	if got := ResolvePreferredChain("https://acme-v02.api.letsencrypt.org/directory", ""); got != PreferredLetsEncryptChain {
		t.Fatalf("ResolvePreferredChain() = %q, want %q", got, PreferredLetsEncryptChain)
	}
	if got := ResolvePreferredChain("https://acme.zerossl.com/v2/DV90", ""); got != "" {
		t.Fatalf("ResolvePreferredChain() = %q, want empty", got)
	}
	if got := ResolvePreferredChain("https://acme-v02.api.letsencrypt.org/directory", "Custom Root"); got != "Custom Root" {
		t.Fatalf("ResolvePreferredChain() = %q, want Custom Root", got)
	}
}

func TestIsLetsEncryptDirectory(t *testing.T) {
	t.Parallel()

	if !IsLetsEncryptDirectory("https://acme-v02.api.letsencrypt.org/directory") {
		t.Fatal("expected letsencrypt directory to match")
	}
	if IsLetsEncryptDirectory("https://acme.zerossl.com/v2/DV90") {
		t.Fatal("expected non-letsencrypt directory not to match")
	}
}
