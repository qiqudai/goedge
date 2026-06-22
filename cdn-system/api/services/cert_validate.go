package services

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

func NormalizePEMInput(pemData string) string {
	return strings.TrimSpace(pemData)
}

// SplitPEMBundle extracts certificate chain and private key from a combined PEM paste.
func SplitPEMBundle(pemData string) (certsPEM string, keyPEM string) {
	pemData = strings.TrimSpace(pemData)
	if pemData == "" {
		return "", ""
	}
	var certs []string
	var key string
	rest := []byte(pemData)
	for {
		block, remain := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remain
		encoded := string(pem.EncodeToMemory(block))
		switch block.Type {
		case "CERTIFICATE":
			certs = append(certs, encoded)
		case "RSA PRIVATE KEY", "PRIVATE KEY", "EC PRIVATE KEY":
			if key == "" {
				key = encoded
			}
		}
	}
	return strings.TrimSpace(strings.Join(certs, "")), strings.TrimSpace(key)
}

func NormalizeUploadCertKeyInputs(certPEM, keyPEM string) (string, string) {
	certPEM = NormalizePEMInput(certPEM)
	keyPEM = NormalizePEMInput(keyPEM)

	if certFromBundle, keyFromCert := SplitPEMBundle(certPEM); keyFromCert != "" {
		if keyPEM == "" {
			keyPEM = keyFromCert
		}
		if certFromBundle != "" {
			certPEM = certFromBundle
		}
	}
	if certFromKey, keyFromKey := SplitPEMBundle(keyPEM); keyFromKey != "" {
		keyPEM = keyFromKey
		if certPEM == "" && certFromKey != "" {
			certPEM = certFromKey
		}
	}
	return certPEM, keyPEM
}

func ExposeStoredPrivateKey(stored string) string {
	plain := strings.TrimSpace(stored)
	if plain == "" {
		return ""
	}
	if _, err := parsePrivateKeyPEM(plain); err == nil {
		return plain
	}
	if dec, err := Crypto.Decrypt(stored); err == nil {
		if _, err := parsePrivateKeyPEM(dec); err == nil {
			return dec
		}
	}
	return ""
}

func EncryptCertKeyForStore(plainKey string) (string, error) {
	plainKey = NormalizePEMInput(plainKey)
	if plainKey == "" {
		return "", errors.New("private key is required")
	}
	if _, err := parsePrivateKeyPEM(plainKey); err != nil {
		return "", err
	}
	encrypted, err := Crypto.Encrypt(plainKey)
	if err != nil {
		return plainKey, nil
	}
	return encrypted, nil
}

func ValidateUploadCertKeyPair(certPEM, keyPEM string) error {
	certPEM = NormalizePEMInput(certPEM)
	keyPEM = NormalizePEMInput(keyPEM)
	if certPEM == "" || keyPEM == "" {
		return errors.New("cert and key are required for upload")
	}
	if _, err := parseX509CertificatePEM(certPEM); err != nil {
		return fmt.Errorf("invalid certificate: %w", err)
	}
	if _, err := parsePrivateKeyPEM(keyPEM); err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		return fmt.Errorf("certificate and private key do not match")
	}
	return nil
}

func parseX509CertificatePEM(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("PEM block must contain a CERTIFICATE")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func ParsePrivateKeyPEM(keyPEM string) (any, error) {
	return parsePrivateKeyPEM(keyPEM)
}

func parsePrivateKeyPEM(keyPEM string) (any, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("PEM block must contain a private key")
	}
	if block.Type == "CERTIFICATE" {
		return nil, errors.New("key field must contain a private key, not a certificate")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if err := key.Validate(); err != nil {
			return nil, err
		}
		return key, nil
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		switch typed := key.(type) {
		case *rsa.PrivateKey:
			if err := typed.Validate(); err != nil {
				return nil, err
			}
		case *ecdsa.PrivateKey:
			if typed.Curve == nil {
				return nil, errors.New("invalid ECDSA private key")
			}
		default:
			return nil, errors.New("unsupported private key algorithm")
		}
		return key, nil
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if key.Curve == nil {
			return nil, errors.New("invalid ECDSA private key")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}
