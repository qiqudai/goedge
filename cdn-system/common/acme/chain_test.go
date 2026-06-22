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

	legacy := sampleLegacyChain
	legacy = legacy[:len(legacy)-1] // keep invalid PEM out of parse path
	if UsesIncompatibleLetsEncryptChain(legacy) {
		t.Fatalf("expected invalid/legacy sample not to trigger incompatible detection via parse failure")
	}

	cases := []struct {
		name string
		cn   string
		want bool
	}{
		{name: "yr2 intermediate", cn: "YR2", want: true},
		{name: "root yr", cn: "Root YR", want: true},
		{name: "legacy r12", cn: "R12", want: false},
		{name: "leaf domain", cn: "h5.example.com", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := hasIncompatibleLetsEncryptCN(tc.cn); got != tc.want {
				t.Fatalf("hasIncompatibleLetsEncryptCN(%q) = %v, want %v", tc.cn, got, tc.want)
			}
		})
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
