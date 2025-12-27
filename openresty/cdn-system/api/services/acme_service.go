package services

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-common/acme"
)

var AcmeTokens = acme.NewMemoryTokenStore()

func BuildCADirURL(certType string) string {
	switch strings.ToLower(strings.TrimSpace(certType)) {
	case "letsencrypt", "lets", "let's encrypt":
		return "https://acme-v02.api.letsencrypt.org/directory"
	case "zerossl":
		return "https://acme.zerossl.com/v2/DV90"
	case "buypass":
		return "https://api.buypass.com/acme/directory"
	case "google":
		return "https://dv.acme-v02.api.pki.goog/directory"
	default:
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}

func NewHTTP01Issuer(certType string) *acme.Issuer {
	_ = os.MkdirAll(config.App.AcmeWebroot, 0o755)
	_ = os.MkdirAll(config.App.AcmeAccountDir, 0o755)
	accountKey := filepath.Join(config.App.AcmeAccountDir, "account_"+strings.ToLower(certType)+".key")
	opts := acme.IssueOptions{
		Email:          config.App.AcmeEmail,
		CADirURL:       BuildCADirURL(certType),
		Webroot:        config.App.AcmeWebroot,
		AccountKeyPath: accountKey,
		Timeout:        60 * time.Second,
		TokenStore:     AcmeTokens,
	}
	return acme.NewIssuer(opts)
}

func IsRegisterRateLimited(err error) bool {
	return acme.IsRegisterRateLimited(err)
}
