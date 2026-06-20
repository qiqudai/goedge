package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"
)

func startHTTPSProbeTestServer(t *testing.T, dnsNames []string, status int) (string, func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: dnsNames[0]},
		DNSNames:              dnsNames,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("key pair: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	})}
	go func() {
		_ = server.Serve(ln)
	}()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	return port, func() {
		_ = server.Close()
	}
}

func TestRunHTTPSProbeTaskSuccess(t *testing.T) {
	port, cleanup := startHTTPSProbeTestServer(t, []string{"www.example.com"}, http.StatusOK)
	defer cleanup()
	payload, _ := json.Marshal(httpsProbePayload{
		Domains:        []string{"www.example.com"},
		Ports:          []string{port},
		TimeoutSeconds: 2,
	})
	ret, err := runHTTPSProbeTask(string(payload))
	if err != nil {
		t.Fatalf("expected probe success, got err=%v ret=%s", err, ret)
	}
	var results []httpsProbeResult
	if err := json.Unmarshal([]byte(ret), &results); err != nil {
		t.Fatalf("invalid ret json: %v", err)
	}
	if len(results) != 1 || !results[0].OK || results[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected probe results: %+v", results)
	}
}

func TestRunHTTPSProbeTaskCertMismatchFails(t *testing.T) {
	port, cleanup := startHTTPSProbeTestServer(t, []string{"www.example.com"}, http.StatusOK)
	defer cleanup()
	payload, _ := json.Marshal(httpsProbePayload{
		Domains:        []string{"api.example.com"},
		Ports:          []string{port},
		TimeoutSeconds: 2,
	})
	ret, err := runHTTPSProbeTask(string(payload))
	if err == nil {
		t.Fatalf("expected probe failure, got ret=%s", ret)
	}
	var results []httpsProbeResult
	if json.Unmarshal([]byte(ret), &results) != nil || len(results) != 1 || results[0].OK {
		t.Fatalf("expected failed result, got %s", ret)
	}
}

func TestRunHTTPSProbeTaskAllowsOriginServerErrorAfterTLSValidation(t *testing.T) {
	port, cleanup := startHTTPSProbeTestServer(t, []string{"www.example.com"}, http.StatusBadGateway)
	defer cleanup()
	payload, _ := json.Marshal(httpsProbePayload{
		Domains:        []string{"www.example.com"},
		Ports:          []string{port},
		TimeoutSeconds: 2,
	})
	ret, err := runHTTPSProbeTask(string(payload))
	if err != nil {
		t.Fatalf("origin status must not fail TLS probe, err=%v ret=%s", err, ret)
	}
	var results []httpsProbeResult
	if err := json.Unmarshal([]byte(ret), &results); err != nil {
		t.Fatalf("invalid ret json: %v", err)
	}
	if len(results) != 1 || !results[0].OK || results[0].StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected probe results: %+v", results)
	}
}
