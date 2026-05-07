# CDN Stability Hardening Implementation Plan

This document is written as an implementation handoff for an AI coding agent or engineer. Follow it in order. Do not skip tests. The goal is to prevent these production failures:

- HTTPS is shown as enabled, but the deployed edge node has no usable certificate.
- A certificate is selected, but it does not cover the site domain.
- DNS sync silently writes to the wrong zone, breaking CNAME and ACME validation.
- HTTPS origins work in some clients or environments but fail because origin Host/SNI behavior is ambiguous.

## Non-Negotiable Rules

1. Never mark customer HTTPS as active until certificate issuance, node config reload, and HTTPS probing all pass.
2. Never deploy a selected certificate to a domain unless the parsed certificate SAN/CN covers that exact domain.
3. Never split DNS zones by the last two labels. Use public suffix parsing.
4. Never treat nginx reload success as business success. Always probe the real edge endpoint after reload.
5. Fallback certificates may keep nginx alive, but they must not be reported as successful customer HTTPS.

## Phase 1: Certificate Domain Validation

### Goal

Prevent wrong certificate binding before config is generated.

### Files

- `api/services/cert_match.go` new file
- `api/services/config_service.go`
- `api/services/cert_match_test.go` new file
- `api/controllers/site_admin.go`
- `api/controllers/cert_controller.go`

### Implementation

Create `api/services/cert_match.go`:

```go
package services

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
)

type CertCoverageResult struct {
	OK     bool
	Reason string
	Names  []string
}

func CertificateCoversDomain(certPEM string, domain string) CertCoverageResult {
	domain = normalizeDomainHostForEdge(domain)
	if domain == "" {
		return CertCoverageResult{Reason: "domain is empty"}
	}
	block, _ := pem.Decode([]byte(strings.TrimSpace(certPEM)))
	if block == nil {
		return CertCoverageResult{Reason: "invalid PEM certificate"}
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertCoverageResult{Reason: "failed to parse certificate"}
	}
	names := make([]string, 0, len(cert.DNSNames)+1)
	names = append(names, cert.DNSNames...)
	if cert.Subject.CommonName != "" {
		names = append(names, cert.Subject.CommonName)
	}
	for _, name := range names {
		if certNameMatchesDomain(name, domain) {
			return CertCoverageResult{OK: true, Names: names}
		}
	}
	return CertCoverageResult{Reason: "certificate does not cover domain", Names: names}
}

func certNameMatchesDomain(certName string, domain string) bool {
	certName = normalizeDomainHostForEdge(certName)
	domain = normalizeDomainHostForEdge(domain)
	if certName == "" || domain == "" {
		return false
	}
	if certName == domain {
		return true
	}
	if strings.HasPrefix(certName, "*.") {
		suffix := strings.TrimPrefix(certName, "*.")
		if suffix == "" {
			return false
		}
		// Wildcard only matches one label: *.example.com covers a.example.com,
		// but not a.b.example.com and not example.com.
		if !strings.HasSuffix(domain, "."+suffix) {
			return false
		}
		left := strings.TrimSuffix(domain, "."+suffix)
		return left != "" && !strings.Contains(left, ".")
	}
	return false
}
```

Update `findCertForSiteDomain` in `api/services/config_service.go`:

```go
func findCertForSiteDomain(certID int64, domain string, certs []models.Cert) *models.Cert {
	if certID > 0 {
		for _, cert := range certs {
			if int64(cert.ID) != certID {
				continue
			}
			if CertificateCoversDomain(cert.Cert, domain).OK {
				return &cert
			}
			return nil
		}
	}
	return findCertForDomain(domain, certs)
}
```

Update `findCertForDomain` to parse `cert.Cert` first when certificate PEM exists. `cert.Domain` is only metadata and must not be the source of truth.

```go
if strings.TrimSpace(cert.Cert) != "" {
	if CertificateCoversDomain(cert.Cert, domain).OK {
		return &cert
	}
	continue
}
```

Add validation when users/admins set `cert_id` on a site:

- Load site domains.
- Load certificate.
- For every domain in site domains, call `CertificateCoversDomain`.
- If any domain is not covered, return HTTP 400 with a clear message:

```json
{
  "error": "certificate does not cover domain: www.example.com"
}
```

### Tests

Add `api/services/cert_match_test.go`:

- Exact SAN covers `www.example.com`.
- `*.example.com` covers `a.example.com`.
- `*.example.com` does not cover `a.b.example.com`.
- `*.example.com` does not cover `example.com`.
- Invalid PEM returns `OK=false`.
- Selected `cert_id` that does not cover domain returns nil from `findCertForSiteDomain`.

### Acceptance

- Wrong `certificate_id` can never be emitted into node config.
- A site with wrong cert binding remains HTTP-only or rejected by API.
- Tests pass: `cd api && go test ./...`.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 1 regression:
- api unit tests: PASS/FAIL, command: cd api && go test ./...
- cert exact-domain test: PASS/FAIL
- cert wildcard single-label test: PASS/FAIL
- cert wildcard multi-label reject test: PASS/FAIL
- wrong certificate_id API reject test: PASS/FAIL
- generated config excludes wrong cert test: PASS/FAIL
```

Manual recovery check:

1. Create or upload a certificate for `a.example.com`.
2. Try to bind it to site domain `b.example.com`.
3. Expected: API returns 400 and site config is unchanged.
4. Regenerate node config.
5. Expected: `b.example.com` has no SSL cert data from `a.example.com`.

## Phase 2: HTTPS Activation State Machine

### Goal

Separate "user requested HTTPS" from "HTTPS is active and safe".

### Data Model

Add these fields to `site` when the schema supports migrations:

```sql
ALTER TABLE site ADD COLUMN https_state varchar(32) NOT NULL DEFAULT 'off';
ALTER TABLE site ADD COLUMN https_pending_cert_id bigint DEFAULT NULL;
ALTER TABLE site ADD COLUMN https_active_cert_id bigint DEFAULT NULL;
ALTER TABLE site ADD COLUMN https_last_error text;
ALTER TABLE site ADD COLUMN https_activated_at datetime DEFAULT NULL;
ALTER TABLE site ADD COLUMN https_probe_at datetime DEFAULT NULL;
```

If this project avoids migrations, store the same keys inside `site.settings.https`:

```json
{
  "enable": false,
  "state": "pending",
  "pending_certificate_id": 123,
  "active_certificate_id": 0,
  "last_error": "",
  "activated_at": null,
  "probe_at": null
}
```

Allowed states:

- `off`: HTTPS disabled.
- `pending_issue`: cert created but not ready.
- `pending_deploy`: cert ready, waiting for node sync.
- `probing`: node reload ACK received, probing edge HTTPS.
- `active`: real HTTPS is usable.
- `failed`: issue/deploy/probe failed.

### Files

- `api/services/https_activation_service.go` new file
- `api/controllers/site_admin.go`
- `api/controllers/agent_ws_controller.go`
- `api/services/cert_issue_service.go`
- `api/services/config_service.go`
- `api/services/https_activation_service_test.go` new file

### Admin Apply Flow

Change `AdminApplyCert` behavior:

1. Create cert with state `waiting`.
2. Do not set `https.enable=true`.
3. Store `https.state=pending_issue`.
4. Store `https.pending_certificate_id=cert.ID`.
5. Keep existing `https_listen` if present, but config generation must ignore HTTPS until active.
6. Trigger `IssueCertsAsync`.

Pseudo-code:

```go
httpsCfg["enable"] = false
httpsCfg["state"] = "pending_issue"
httpsCfg["pending_certificate_id"] = cert.ID
httpsCfg["active_certificate_id"] = 0
httpsCfg["last_error"] = ""
```

### Certificate Ready Hook

In `UpdateIssuedCert`, after the cert is saved as ready:

1. Find sites where `settings.https.pending_certificate_id == certID`.
2. Validate the cert covers every site domain.
3. If validation fails, mark site `https.state=failed`, `last_error=...`.
4. If valid, set:

```json
{
  "https": {
    "state": "pending_deploy",
    "pending_certificate_id": 123,
    "active_certificate_id": 0,
    "enable": false
  }
}
```

5. Call `BumpConfigVersion("site", siteIDs)`.

### Config Generation

In `GenerateConfigForNode`, only emit HTTPS listen/cert when HTTPS is active:

```go
httpsState := extractHTTPSState(effectiveSite.Settings)
hasHTTPS := httpsState == "active"
selectedCertID := extractHTTPSActiveCertID(effectiveSite.Settings)
```

Temporary exception:

- If a site already has legacy `https.enable=true` and no `state`, treat it as `active` only if selected cert covers all domains.
- If no matching cert exists, set `hasHTTPS=false` and log/report a config warning.

### Deploy and Probe Flow

After cert becomes ready:

1. Create a `config_sync` task for affected sites.
2. When all target nodes ACK success for that config task, enqueue `https_probe` task.
3. Probe each active domain on each target node before activation.

New task type:

```go
const TaskTypeHTTPSProbe = "https_probe"
```

Payload:

```json
{
  "site_id": 88,
  "cert_id": 123,
  "domains": ["www.example.com"],
  "ports": ["443"],
  "timeout_seconds": 8
}
```

Agent behavior:

- Connect to local node address/port.
- Set TLS `ServerName` to the domain.
- Send HTTP request with `Host: domain`.
- Validate:
  - TLS handshake succeeds.
  - Peer certificate covers domain.
  - HTTP status is not a connection-level failure. Accept `200-499`; reject timeout, TLS error, 502 from no upstream, 525-like failures.

API behavior on successful probes:

```json
{
  "https": {
    "enable": true,
    "state": "active",
    "active_certificate_id": 123,
    "pending_certificate_id": 0,
    "last_error": "",
    "activated_at": "..."
  }
}
```

Then call `BumpConfigVersion("site", []siteID)` again so the node receives the final active HTTPS config.

Important: To probe HTTPS before activation, add a temporary deploy state:

- `pending_deploy` emits HTTPS server block only to selected target nodes but marks it `probe_only=true`.
- Or simpler: after certificate ready, set `state=probing`, emit HTTPS with cert, probe it, then set `active` if probe passes. UI must display "验证中", not "已开启".

### Failure Handling

If issue fails:

- `https.state=failed`
- `https.enable=false`
- `https.last_error=cert.ret`
- Do not emit HTTPS listen.

If node reload fails:

- keep `https.state=pending_deploy` or `failed`, depending on retry exhaustion.
- show node-specific failure in task progress.

If probe fails:

- `https.state=failed`
- `https.enable=false`
- keep site HTTP working.
- do not delete the cert.
- show the exact domain/node/error.

### Tests

Add tests:

- Apply cert does not enable HTTPS immediately.
- Issued cert that does not cover site marks HTTPS failed.
- Issued cert that covers site moves to `pending_deploy`.
- Config generation does not emit `https_listen` for `pending_issue`.
- Legacy enabled HTTPS with missing cert does not emit fallback as success.
- Probe success activates HTTPS.
- Probe failure keeps HTTPS disabled.

### Acceptance

- UI cannot show "HTTPS enabled" until state is `active`.
- Config cannot emit fallback cert as a successful customer cert.
- ACME failure never breaks existing HTTP service.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 2 regression:
- api unit tests: PASS/FAIL, command: cd api && go test ./...
- apply HTTPS keeps state pending_issue test: PASS/FAIL
- ACME success moves state to pending_deploy/probing test: PASS/FAIL
- ACME failure moves state to failed test: PASS/FAIL
- pending HTTPS does not emit active HTTPS config test: PASS/FAIL
- active HTTPS emits cert only after coverage validation test: PASS/FAIL
- legacy https.enable migration behavior test: PASS/FAIL
```

Manual recovery check:

1. Pick an HTTP-only staging site.
2. Click/apply HTTPS certificate.
3. Expected immediately after request:
   - UI/API state is `pending_issue`.
   - UI must not display `已开启`.
   - generated node config must not report successful customer HTTPS.
4. Force ACME failure using an invalid domain or blocked DNS.
5. Expected:
   - site state becomes `failed`;
   - HTTP access still works;
   - HTTPS is not advertised as active;
   - error message contains the ACME/DNS failure reason.
6. Repeat with a valid domain.
7. Expected:
   - state reaches `active` only after issue + node ACK + probe.

## Phase 3: DNS Zone Parsing and Verification

### Goal

Make DNS CNAME and DNS-01 reliable for domains like `example.com.cn`, `foo.co.uk`, and wildcard domains.

### Files

- `api/go.mod`
- `api/services/dns_name.go` new file
- `api/services/dns_sync.go`
- `api/services/site_task_service.go`
- `api/services/dns_name_test.go` new file

### Dependency

Add:

```bash
cd api
go get golang.org/x/net/publicsuffix
```

### Implementation

Create `api/services/dns_name.go`:

```go
package services

import (
	"net"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func SplitDNSZoneAndRecord(host string) (zone string, record string) {
	host = normalizeDomainHost(host)
	if host == "" || net.ParseIP(host) != nil {
		return "", ""
	}
	wildcard := false
	if strings.HasPrefix(host, "*.") {
		wildcard = true
		host = strings.TrimPrefix(host, "*.")
	}
	zone, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || zone == "" {
		return "", ""
	}
	prefix := strings.TrimSuffix(host, "."+zone)
	if prefix == host {
		prefix = ""
	}
	if wildcard {
		if prefix == "" {
			prefix = "*"
		} else {
			prefix = "*." + prefix
		}
	}
	if prefix == "" {
		return zone, "@"
	}
	return zone, prefix
}
```

Replace all calls to `splitRootDomain` with `SplitDNSZoneAndRecord`.

Keep the old function as a wrapper only if needed:

```go
func splitRootDomain(domain string) (string, string) {
	return SplitDNSZoneAndRecord(domain)
}
```

### DNS Post-Write Verification

After upsert:

1. Query provider records and confirm the expected record exists.
2. Query public DNS resolvers with timeout:
   - `1.1.1.1`
   - `8.8.8.8`
   - `223.5.5.5`
3. Mark status:
   - `provider_confirmed`
   - `public_resolving`
   - `pending_propagation`
   - `failed`

Add function:

```go
func VerifyCNAMERecord(fqdn string, expected string, timeout time.Duration) DNSVerifyResult
```

### Tests

- `www.example.com` => zone `example.com`, record `www`.
- `example.com` => zone `example.com`, record `@`.
- `www.example.com.cn` => zone `example.com.cn`, record `www`.
- `a.b.foo.co.uk` => zone `foo.co.uk`, record `a.b`.
- `*.example.com` => zone `example.com`, record `*`.
- IP input returns empty.

### Acceptance

- No DNS sync code guesses zone by last two labels.
- DNS task output includes provider verification and public propagation status.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 3 regression:
- api unit tests: PASS/FAIL, command: cd api && go test ./...
- publicsuffix split www.example.com test: PASS/FAIL
- publicsuffix split example.com apex test: PASS/FAIL
- publicsuffix split www.example.com.cn test: PASS/FAIL
- publicsuffix split a.b.foo.co.uk test: PASS/FAIL
- wildcard split *.example.com test: PASS/FAIL
- IP input reject test: PASS/FAIL
- DNS provider confirmation test: PASS/FAIL
- public resolver propagation pending/success test: PASS/FAIL
```

Manual recovery check:

1. Configure a staging DNS API account with a controllable zone.
2. Sync `www.example.com.cn` or another public-suffix domain available in the account.
3. Expected: provider API receives zone `example.com.cn`, record `www`.
4. Sync apex `example.com.cn`.
5. Expected: provider API receives record `@`.
6. Sync wildcard `*.example.com`.
7. Expected: provider API receives record `*`.
8. Check task output.
9. Expected: task shows provider confirmation and public DNS propagation state.

## Phase 4: Explicit Origin Host and SNI Policy

### Goal

Make HTTPS origin behavior predictable across IP origins, domain origins, load balancers, and strict certificate validation.

### Data Model

Add these site settings under `settings.origin`:

```json
{
  "host_header": "origin.example.com",
  "sni": "origin.example.com",
  "verify_tls": false,
  "tls_server_name": "origin.example.com"
}
```

Recommended semantics:

- `host_header`: Host header sent to origin. Empty means use client `$host`.
- `sni`: SNI sent to HTTPS origin. Empty means:
  - use `host_header` if set,
  - else use `$host`,
  - else use origin hostname if origin is a domain.
- `verify_tls`: whether to verify origin certificate.
- `tls_server_name`: optional alias for `sni`; keep one canonical field internally.

### Files

- `api/models/config_models.go`
- `api/services/config_service.go`
- `agent/config.go`
- `agent/http_config.go`
- `agent/http_config_origin_tls_test.go` new file
- `web/admin/src/components/manage/OriginConfig.vue`

### API Config DTO

Extend edge domain structs:

```go
OriginHostHeader string `json:"origin_host_header"`
OriginSNI        string `json:"origin_sni"`
OriginVerifyTLS  bool   `json:"origin_verify_tls"`
```

In `extractAdvancedConfig` or a new `extractOriginAdvancedConfig`, parse:

```go
origin := getMap(settings, "origin")
hostHeader := parseString(origin["host_header"])
sni := parseString(firstNonEmpty(origin["sni"], origin["tls_server_name"]))
verifyTLS := parseBoolValue(origin["verify_tls"], false)
```

### Agent Nginx Generation

In `writeProxyBase`, do not always force `Host $host` if `OriginHostHeader` is configured.

Pseudo-code:

```go
func writeProxyBase(b *strings.Builder, domain edgeDomain, customHeaderSet map[string]struct{}) {
	host := "$host"
	if v := sanitizeNginxValue(domain.OriginHostHeader); v != "" {
		host = v
	}
	writeProxyHeaderIfMissing(b, customHeaderSet, "Host", host)
}
```

Change call sites accordingly.

In `writeProxySSL`:

```go
if strings.ToLower(domain.OriginProtocol) == "http" {
	return
}
b.WriteString("        proxy_ssl_server_name on;\n")
if sni := sanitizeNginxValue(domain.OriginSNI); sni != "" {
	b.WriteString("        proxy_ssl_name " + sni + ";\n")
} else if host := sanitizeNginxValue(domain.OriginHostHeader); host != "" {
	b.WriteString("        proxy_ssl_name " + host + ";\n")
} else {
	b.WriteString("        proxy_ssl_name $host;\n")
}
if domain.OriginVerifyTLS {
	...
}
```

### UI

In origin settings:

- Add "回源 Host" input.
- Add "回源 SNI" input.
- Add "校验源站证书" switch.
- Add warning copy only in tooltip/help text:
  - If origin is IP and protocol is HTTPS, recommend setting SNI.
  - If verify TLS is enabled, SNI must match origin certificate.

### Tests

- HTTPS origin with no custom host emits `proxy_ssl_name $host`.
- HTTPS origin with `origin_host_header` emits `proxy_set_header Host origin.example.com` and `proxy_ssl_name origin.example.com`.
- HTTPS origin with explicit `origin_sni` uses explicit SNI even when Host differs.
- HTTP origin emits no proxy SSL directives.
- Verify TLS emits `proxy_ssl_verify on` and trusted CA.

### Acceptance

- Users can configure origin address, Host, and SNI independently.
- IP HTTPS origins no longer randomly fail due to missing SNI.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 4 regression:
- agent unit tests: PASS/FAIL, command: cd agent && go test ./...
- api unit tests: PASS/FAIL, command: cd api && go test ./...
- HTTPS origin default proxy_ssl_name $host test: PASS/FAIL
- origin_host_header changes Host header test: PASS/FAIL
- origin_host_header becomes default SNI test: PASS/FAIL
- explicit origin_sni overrides host header test: PASS/FAIL
- HTTP origin emits no proxy_ssl directives test: PASS/FAIL
- verify_tls emits proxy_ssl_verify and CA bundle test: PASS/FAIL
- UI saves and reloads origin Host/SNI fields test: PASS/FAIL
```

Manual recovery check:

1. Create a staging origin reachable by IP but serving TLS for `origin.example.com`.
2. Configure CDN origin address as the IP, protocol `https`, origin Host `origin.example.com`, SNI `origin.example.com`.
3. Expected: CDN access succeeds.
4. Remove SNI while keeping origin as IP.
5. Expected: if origin requires SNI, probe fails with clear TLS/SNI error.
6. Restore SNI.
7. Expected: probe and customer request recover without changing site domain.

## Phase 5: Config Deploy ACK and Probe System

### Goal

Close the loop between API intent and edge reality.

### Files

- `api/models/task.go`
- `api/controllers/agent_ws_controller.go`
- `api/services/probe_service.go` new file
- `agent/tasks.go`
- `agent/https_probe.go` new file
- `agent/https_probe_test.go` new file

### Probe Task

Agent task handler:

```go
case "https_probe":
	return runHTTPSProbeTask(data)
```

Probe result:

```json
{
  "domain": "www.example.com",
  "port": "443",
  "ok": true,
  "tls_version": "TLS1.3",
  "cert_subject": "...",
  "cert_not_after": "...",
  "status_code": 200,
  "error": ""
}
```

Agent implementation requirements:

- Use `tls.DialWithDialer`.
- Dial local node IP or `127.0.0.1`.
- Set `ServerName=domain`.
- Send HTTP request with `Host=domain`.
- Timeout every probe.
- Return one result per domain/port.

API requirements:

- Store probe result in task `ret`.
- Activate HTTPS only when every required probe passes.
- Failed probe must include exact domain, node, port, and TLS/HTTP error.

### Acceptance

- A config task success without probe success never activates HTTPS.
- Operators can see why activation failed.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 5 regression:
- agent unit tests: PASS/FAIL, command: cd agent && go test ./...
- api unit tests: PASS/FAIL, command: cd api && go test ./...
- https_probe TLS success test: PASS/FAIL
- https_probe certificate mismatch failure test: PASS/FAIL
- https_probe timeout failure test: PASS/FAIL
- API activates HTTPS only when all node probes pass test: PASS/FAIL
- API keeps HTTPS failed/pending when one node probe fails test: PASS/FAIL
- task ret includes domain/node/port/error test: PASS/FAIL
```

Manual recovery check:

1. Deploy a valid HTTPS config to one staging node.
2. Run `https_probe`.
3. Expected: task result includes TLS version, status code, certificate subject, and `ok=true`.
4. Break the certificate or SNI intentionally.
5. Run `https_probe`.
6. Expected: `ok=false`, exact error is visible, HTTPS state does not become `active`.
7. Restore config.
8. Expected: next probe passes and activation can continue.

## Phase 6: Compatibility Smoke Matrix

### Goal

Catch "PC works, mobile fails" before customers do.

### Add Test Script

Create `tests/compat/https_matrix.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

DOMAIN="$1"
IP="$2"
PORT="${3:-443}"

curl --resolve "$DOMAIN:$PORT:$IP" "https://$DOMAIN:$PORT/" -I --http1.1 --max-time 10
curl --resolve "$DOMAIN:$PORT:$IP" "https://$DOMAIN:$PORT/" -I --http2 --max-time 10
curl --resolve "$DOMAIN:$PORT:$IP" "https://$DOMAIN:$PORT/" -I \
  -A "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148" \
  --max-time 10
curl --resolve "$DOMAIN:$PORT:$IP" "https://$DOMAIN:$PORT/" -I \
  -A "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/120 Mobile Safari/537.36" \
  --max-time 10
openssl s_client -connect "$IP:$PORT" -servername "$DOMAIN" -verify_return_error </dev/null
```

### CI/Release Gate

Before release, run:

```bash
cd api && go test ./...
cd ../agent && go test ./...
cd ../tests/ui_e2e && npm test
```

Manual staging gate:

1. Create site with HTTP only.
2. Apply cert.
3. Confirm UI says "pending", not "enabled".
4. Wait for ACME success.
5. Confirm node sync task success.
6. Confirm probe success.
7. Confirm UI says "active".
8. Run compatibility matrix on the staging edge IP.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 6 regression:
- api tests: PASS/FAIL, command: cd api && go test ./...
- agent tests: PASS/FAIL, command: cd agent && go test ./...
- UI e2e tests: PASS/FAIL, command: cd tests/ui_e2e && npm test
- compatibility HTTP/1.1 curl: PASS/FAIL
- compatibility HTTP/2 curl: PASS/FAIL
- compatibility iOS UA curl: PASS/FAIL
- compatibility Android UA curl: PASS/FAIL
- compatibility openssl s_client: PASS/FAIL
```

Manual recovery check:

1. Run `tests/compat/https_matrix.sh <domain> <edge-ip> 443`.
2. Expected: all curl commands complete within 10 seconds.
3. Expected: `openssl s_client` verifies the certificate chain and name.
4. If any check fails, the release is blocked until the exact client/protocol failure is fixed or documented as intentionally unsupported.

## Phase 7: Observability and Operator UX

### Add Status Fields to API Responses

For each site list/detail response, include:

```json
{
  "https_state": "active",
  "https_error": "",
  "https_active_cert_id": 123,
  "https_pending_cert_id": 0,
  "https_probe_at": "..."
}
```

For each certificate response, include:

```json
{
  "coverage": {
    "valid_for_domains": ["www.example.com"],
    "invalid_for_domains": [],
    "not_after": "..."
  }
}
```

### UI Labels

- `off`: 未开启
- `pending_issue`: 申请证书中
- `pending_deploy`: 同步节点中
- `probing`: 验证 HTTPS 中
- `active`: 已开启
- `failed`: 开启失败

Never display "已开启" unless `state=active`.

### Regression Standard

The implementer must paste the following evidence into the final handoff:

```text
Phase 7 regression:
- api tests: PASS/FAIL, command: cd api && go test ./...
- UI e2e tests: PASS/FAIL, command: cd tests/ui_e2e && npm test
- site list exposes https_state test: PASS/FAIL
- cert list exposes coverage test: PASS/FAIL
- pending_issue UI label test: PASS/FAIL
- active UI label test: PASS/FAIL
- failed UI label and error display test: PASS/FAIL
```

Manual recovery check:

1. Create one site in each HTTPS state: `off`, `pending_issue`, `pending_deploy`, `probing`, `active`, `failed`.
2. Open site list and site detail.
3. Expected: every state is displayed with the correct label.
4. Expected: only `active` displays `已开启`.
5. Expected: `failed` displays actionable error text.

## Suggested Implementation Order

1. Implement Phase 1 certificate coverage validation.
2. Implement Phase 3 public suffix DNS parsing.
3. Implement Phase 4 origin Host/SNI.
4. Implement Phase 2 HTTPS state machine without probe activation first.
5. Implement Phase 5 probe tasks and activation.
6. Implement Phase 6 compatibility script.
7. Implement Phase 7 UI/status improvements.

## Final Definition of Done

The hardening is complete only when all are true:

- A wrong selected cert is rejected by API and never appears in generated node config.
- Applying HTTPS does not enable customer HTTPS until cert issue, node reload, and probe pass.
- ACME failure leaves HTTP service working and displays a clear failure reason.
- DNS sync handles `example.com.cn`, `foo.co.uk`, wildcard domains, and apex domains.
- HTTPS origin supports independent origin address, Host header, SNI, and TLS verification.
- Tests pass in `api` and `agent`.
- Staging compatibility matrix passes for HTTP/1.1, HTTP/2, iOS UA, Android UA, and `openssl s_client`.

## Global Test Recovery Standard

Every implementation PR or AI coding session must finish with this exact checklist filled out:

```text
Global regression report:
- Changed files:
- Database/schema changes:
- Backward compatibility notes:
- cd api && go test ./...: PASS/FAIL
- cd agent && go test ./...: PASS/FAIL
- cd tests/ui_e2e && npm test: PASS/FAIL or NOT RUN with reason
- HTTPS apply valid-domain staging test: PASS/FAIL or NOT RUN with reason
- HTTPS apply invalid-domain staging test: PASS/FAIL or NOT RUN with reason
- Wrong certificate binding rejection test: PASS/FAIL
- DNS public suffix test: PASS/FAIL
- Origin IP + HTTPS + SNI test: PASS/FAIL
- Node reload rollback test: PASS/FAIL
- Compatibility matrix test: PASS/FAIL or NOT RUN with reason
- Known residual risks:
```

Release is blocked when any of these are true:

- Wrong certificate binding is accepted.
- HTTPS is displayed as enabled before probe success.
- A failed ACME issuance changes a working HTTP-only site into a broken HTTPS site.
- DNS zone parsing fails for a public suffix domain.
- Origin HTTPS by IP cannot be fixed using explicit SNI.
- nginx reload failure leaves the node with a broken generated config.
- The implementer cannot explain a failed or skipped regression item.

## Implementation Mapping for This Repository

Use this section as the copy-paste checklist for AI execution in this codebase.

1. Certificate coverage guard
   - Add or maintain `api/services/cert_match.go`.
   - Enforce PEM SAN/CN matching in `api/services/config_service.go`.
   - Reject selected certificate IDs in `api/controllers/site_admin.go` when they do not cover every site domain.
   - Regression: `cd api && go test ./...` must include exact SAN, wildcard one-label, wildcard reject multi-label, invalid PEM, and wrong selected cert cases.

2. HTTPS state machine and activation gate
   - `AdminApplyCert` must only write `settings.https.state=pending_issue`, `pending_certificate_id`, and `enable=false`.
   - Cert issuance success must move sites to `probing`, generate config, then create HTTPS probe tasks.
   - Probe success is the only path to `state=active` and `active_certificate_id`.
   - Probe or ACME failure must leave `enable=false`, `state=failed`, and `last_error`.
   - Regression: list/detail UI must never show HTTPS as enabled unless `state=active`.

3. DNS public suffix parsing
   - `api/services/dns_sync.go` must call a public-suffix-aware helper.
   - Regression: `example.com.cn`, `foo.co.uk`, apex domains, wildcard domains, URL input, and IP reject tests must pass.

4. Origin HTTPS Host/SNI/TLS policy
   - API config must emit `origin_host_header`, `origin_sni`, and `origin_verify_tls`.
   - Agent nginx generation must set `Host`, `proxy_ssl_name`, and optional `proxy_ssl_verify`.
   - Admin UI must expose Host, SNI, and source certificate verification in single-site and batch settings.
   - Regression: generated nginx config must show custom Host, default SNI from Host, explicit SNI override, and no SSL directives for HTTP origin.

5. Operator visibility
   - Site list API must expose `https_state`, `https_error`, `active_cert_id`, and `pending_cert_id`.
   - Admin list must display `关闭`, `申请中`, `验证中`, `开启`, or `失败`.
   - Open-site URL helper must use HTTPS only for `state=active`.

Required local verification commands:

```bash
cd api && go test ./...
cd agent && go test ./...
cd web/admin && npm run build
git diff --check
```
