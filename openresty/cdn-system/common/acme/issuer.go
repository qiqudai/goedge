package acme

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type TokenStore interface {
	Put(token, value string, ttl time.Duration)
	Get(token string) (string, bool)
	Delete(token string)
}

type tokenEntry struct {
	value     string
	expiresAt time.Time
}

type MemoryTokenStore struct {
	mu    sync.RWMutex
	items map[string]tokenEntry
}

func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		items: map[string]tokenEntry{},
	}
}

func (s *MemoryTokenStore) Put(token, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[token] = tokenEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (s *MemoryTokenStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[token]
	if !ok {
		return "", false
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return "", false
	}
	return item.value, true
}

func (s *MemoryTokenStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
}

type HTTP01Provider struct {
	Webroot    string
	TokenStore TokenStore
	TTL        time.Duration
}

func NewHTTP01Provider(webroot string, store TokenStore) *HTTP01Provider {
	return &HTTP01Provider{
		Webroot:    webroot,
		TokenStore: store,
		TTL:        15 * time.Minute,
	}
}

func (p *HTTP01Provider) Present(_ string, token string, keyAuth string) error {
	if err := p.writeTokenFile(token, keyAuth); err != nil {
		return err
	}
	if p.TokenStore != nil {
		p.TokenStore.Put(token, keyAuth, p.TTL)
	}
	return nil
}

func (p *HTTP01Provider) CleanUp(_ string, token string, _ string) error {
	_ = p.removeTokenFile(token)
	if p.TokenStore != nil {
		p.TokenStore.Delete(token)
	}
	return nil
}

func (p *HTTP01Provider) writeTokenFile(token string, keyAuth string) error {
	if strings.TrimSpace(p.Webroot) == "" {
		return nil
	}
	target := filepath.Join(p.Webroot, ".well-known", "acme-challenge", token)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(keyAuth), 0o644)
}

func (p *HTTP01Provider) removeTokenFile(token string) error {
	if strings.TrimSpace(p.Webroot) == "" {
		return nil
	}
	target := filepath.Join(p.Webroot, ".well-known", "acme-challenge", token)
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type IssueOptions struct {
	Email          string
	CADirURL       string
	Webroot        string
	AccountKeyPath string
	KeyType        certcrypto.KeyType
	Timeout        time.Duration
	TokenStore     TokenStore
}

type IssueResult struct {
	CertPEM   string
	KeyPEM    string
	NotBefore time.Time
	NotAfter  time.Time
}

type acmeUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string {
	return u.Email
}

func (u *acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type Issuer struct {
	options IssueOptions
}

func NewIssuer(options IssueOptions) *Issuer {
	if options.KeyType == "" {
		options.KeyType = certcrypto.RSA2048
	}
	if options.Timeout == 0 {
		options.Timeout = 60 * time.Second
	}
	return &Issuer{options: options}
}

func (i *Issuer) Issue(domains []string) (*IssueResult, error) {
	if len(domains) == 0 {
		return nil, errors.New("domains is required")
	}
	user, err := i.loadOrCreateUser()
	if err != nil {
		return nil, err
	}

	cfg := lego.NewConfig(user)
	cfg.CADirURL = i.options.CADirURL
	cfg.Certificate.KeyType = i.options.KeyType
	cfg.HTTPClient.Timeout = i.options.Timeout

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	provider := NewHTTP01Provider(i.options.Webroot, i.options.TokenStore)
	if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
		return nil, err
	}

	if err := i.ensureRegistration(client, user); err != nil {
		return nil, err
	}

	req := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}
	res, err := client.Certificate.Obtain(req)
	if err != nil {
		return nil, err
	}

	certPEM := string(res.Certificate)
	keyPEM := string(res.PrivateKey)
	notBefore, notAfter, _ := ParseCertTimes(certPEM)

	return &IssueResult{
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}, nil
}

func (i *Issuer) ensureRegistration(client *lego.Client, user *acmeUser) error {
	if user.Registration != nil {
		return nil
	}
	if reg, err := client.Registration.ResolveAccountByKey(); err == nil {
		user.Registration = reg
		return nil
	}
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	user.Registration = reg
	return nil
}

func (i *Issuer) loadOrCreateUser() (*acmeUser, error) {
	if strings.TrimSpace(i.options.Email) == "" {
		return nil, errors.New("acme email is required")
	}
	key, err := loadOrCreatePrivateKey(i.options.AccountKeyPath)
	if err != nil {
		return nil, err
	}
	return &acmeUser{
		Email: i.options.Email,
		key:   key,
	}, nil
}

func loadOrCreatePrivateKey(path string) (crypto.PrivateKey, error) {
	if strings.TrimSpace(path) == "" {
		return rsa.GenerateKey(rand.Reader, 2048)
	}
	if data, err := os.ReadFile(path); err == nil {
		if key, err := parsePrivateKey(data); err == nil {
			return key, nil
		}
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, encodePrivateKey(key), 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func parsePrivateKey(pemBytes []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("unsupported private key format")
}

func encodePrivateKey(key *rsa.PrivateKey) []byte {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(block)
}

func ParseCertTimes(certPEM string) (time.Time, time.Time, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return time.Time{}, time.Time{}, errors.New("invalid certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return cert.NotBefore, cert.NotAfter, nil
}

func IsRegisterRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "register account") && strings.Contains(msg, "429")
}
