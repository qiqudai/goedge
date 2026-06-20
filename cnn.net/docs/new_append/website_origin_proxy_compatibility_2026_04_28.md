# Website Origin Proxy Compatibility (2026-04-28)

This document summarizes the final compatibility rules that `cnn.net` should replicate for website acceleration. The goal is not to patch one domain, but to make normal website origin requests behave as close to browser access as a reverse proxy reasonably can.

## 1. Scope

Apply these rules to `website` type sites by default.

Do not apply browser navigation compatibility headers to:
- `api` type sites
- `download` type sites

Site-specific custom request headers must keep higher priority. If a site explicitly configures a header, the default compatibility header must not duplicate or override it.

## 2. Default Reverse Proxy Headers

All proxied sites should include these base headers unless the site has configured a custom value:

- `Host: $host`
- `X-Real-IP: $remote_addr`
- `X-Forwarded-For: $remote_addr`
- `X-Forwarded-Proto: $scheme`
- `X-Forwarded-Host: $host`
- `X-Forwarded-Port: $server_port`

Important behavior:
- Use `$remote_addr` for `X-Forwarded-For`.
- Do not use `$proxy_add_x_forwarded_for` as the default, because some strict origins reject duplicated forwarded headers.

## 3. Website Browser Compatibility Headers

For `website` type sites, pass through these client browser headers when present:

- `User-Agent`
- `Accept`
- `Accept-Language`
- `Accept-Encoding`
- `Referer`
- `Cache-Control`
- `Upgrade-Insecure-Requests`
- `Sec-Fetch-Site`
- `Sec-Fetch-Mode`
- `Sec-Fetch-User`
- `Sec-Fetch-Dest`
- `Sec-CH-UA`
- `Sec-CH-UA-Mobile`
- `Sec-CH-UA-Platform`

Reason:
- Many web origins branch on browser navigation headers.
- Some origins return empty bodies, different redirects, or mobile/desktop variants when these headers are missing.
- Passing client-derived values is safer than hardcoding a fake browser profile.

## 4. Hop-by-hop and Invalid Origin Response Headers

For non-WebSocket website proxying, hide these upstream response headers:

- `Upgrade`
- `Connection`
- `Keep-Alive`
- `Proxy-Connection`

Reason:
- Some origins incorrectly emit `Upgrade: h2,h2c` on normal page responses.
- Passing these headers through HTTP/2 or HTTP/3 can trigger browser or client protocol errors.
- WebSocket sites are excluded because `Upgrade` and `Connection` are part of the expected protocol flow.

## 5. HTTPS Absolute Redirect Compatibility

For HTTPS edge sites, rewrite absolute `http://` upstream redirects to `https://`:

```nginx
proxy_redirect ~^http://([^/]+)(/.*)?$ https://$1$2;
```

Reason:
- Origins often generate absolute HTTP redirects even when users are already visiting the CDN over HTTPS.
- Browsers may have HSTS for the domain and upgrade back to HTTPS automatically.
- Without edge-side redirect normalization, this can become an endless browser redirect loop.

## 6. Multi-domain Website Host Rule

For multi-domain websites, prefer Host follow mode by default:

- Request to `m.example.com` should reach origin with `Host: m.example.com`.
- Request to `web.example.com` should reach origin with `Host: web.example.com`.

Avoid forcing one fixed origin Host across all domains unless explicitly configured by the user.

Reason:
- Many origins select desktop/mobile content and redirects using the combination of `Host` and `User-Agent`.
- Forcing `Host: web.example.com` for an `m.example.com` request can make mobile requests redirect back to `m.example.com`.
- Forcing `Host: m.example.com` for a `web.example.com` request can make desktop requests redirect back to `web.example.com`.
- The result is a redirect loop even when both domains are otherwise valid.

## 7. Case Study: gb8801

Observed behavior:

- `Host: m.gb8801.co` with mobile User-Agent returned `200`.
- `Host: m.gb8801.co` with desktop User-Agent redirected to `http://web.gb8801.co`.
- `Host: web.gb8801.co` with desktop User-Agent returned `200`.
- `Host: web.gb8801.co` with mobile User-Agent redirected to `http://m.gb8801.co`.

Final working behavior:

- Include both `m.gb8801.co` and `web.gb8801.co` as website domains.
- Use HTTPS origin protocol.
- Use Host follow mode instead of forcing one Host for both domains.
- Keep the HTTP-to-HTTPS absolute redirect rewrite enabled.
- Hide invalid hop-by-hop origin response headers.

Validation result:

- Desktop browser flow: `https://m.gb8801.co` redirects once to `https://web.gb8801.co`, then returns `200`.
- Mobile browser flow: `https://m.gb8801.co` returns `200` directly.

## 8. Implementation Reference

Go Agent implementation reference:

- `agent/http_config.go`
  - `writeProxyBase`
  - `writeBrowserCompatibilityHeaders`
  - `writeProxyHiddenResponseHeaders`
  - `writeProxyRedirectRules`
- `agent/http_config_headers_test.go`
  - website compatibility headers test
  - API/download skip behavior test
  - hop-by-hop response header hiding test
  - HTTPS redirect rewrite test

The merged Go commit on `main`:

- `ea7171a8 Merge website origin proxy compatibility`
