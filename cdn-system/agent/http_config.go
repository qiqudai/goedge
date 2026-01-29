package main

import (
	fsutil "cdn-common/io"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func writeHTTPConfig(cfg edgeConfig) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "http.conf")
	if len(cfg.Domains) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}

	errorPageDir := filepath.Join(rootDir, "conf", "error_pages")
	if absDir, err := filepath.Abs(errorPageDir); err == nil {
		errorPageDir = absDir
	}
	errorPages := normalizeErrorPages(cfg.ErrorPages)
	if len(errorPages) > 0 {
		if err := writeErrorPageFiles(errorPageDir, errorPages); err != nil {
			return err
		}
	}

	defaultListen80 := true
	if cfg.Resources != nil {
		defaultListen80 = cfg.Resources.Website.DefaultListen80
	}

	upstreamKeepalive := map[string]edgeDomain{}
	for _, domain := range cfg.Domains {
		if domain.UpstreamKey != "" && domain.UpstreamKeepalive {
			upstreamKeepalive[domain.UpstreamKey] = domain
		}
	}

	var b strings.Builder
	for _, upstream := range cfg.Upstreams {
		if upstream.ID == "" || len(upstream.Targets) == 0 {
			continue
		}
		b.WriteString("upstream " + upstream.ID + " {\n")
		for _, target := range upstream.Targets {
			if target.Addr == "" {
				continue
			}
			if target.Weight > 0 {
				b.WriteString(fmt.Sprintf("    server %s weight=%d;\n", target.Addr, target.Weight))
			} else {
				b.WriteString(fmt.Sprintf("    server %s;\n", target.Addr))
			}
		}
		if keep, ok := upstreamKeepalive[upstream.ID]; ok {
			conn := keep.UpstreamKeepaliveConn
			if conn <= 0 {
				conn = 32
			}
			b.WriteString(fmt.Sprintf("    keepalive %d;\n", conn))
		}
		b.WriteString("}\n")
	}

	for _, domain := range cfg.Domains {
		if domain.Name == "" || domain.UpstreamKey == "" {
			continue
		}
		writeDomainServers(&b, domain, errorPages, errorPageDir, defaultListen80)
	}

	blockUnbound := cfg.WAF != nil && cfg.WAF.BlockUnboundDomain
	if blockUnbound {
		blockedStatus := errorPageStatusForKey("ip")
		if blockedStatus == 0 {
			blockedStatus = 418
		}

		httpPorts := collectHTTPPorts(cfg.Domains, defaultListen80)
		if defaultListen80 {
			httpPorts = appendUniquePort(httpPorts, "80")
		}
		for _, port := range httpPorts {
			writeDefaultServer(&b, port, false, errorPages, errorPageDir, blockedStatus)
		}

		for _, port := range collectHTTPSPorts(cfg.Domains) {
			writeDefaultServer(&b, port, true, errorPages, errorPageDir, blockedStatus)
		}
	} else if shouldBindDefaultHTTP(cfg.Domains, defaultListen80) {
		httpPorts := collectHTTPPorts(cfg.Domains, defaultListen80)
		for _, port := range httpPorts {
			writeDefaultServer(&b, port, false, errorPages, errorPageDir, 404)
		}
	}

	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func shouldBindDefaultHTTP(domains []edgeDomain, defaultListen80 bool) bool {
	if defaultListen80 {
		return true
	}
	for _, domain := range domains {
		if len(domain.HttpListen) > 0 {
			return true
		}
	}
	return false
}

func collectHTTPPorts(domains []edgeDomain, defaultListen80 bool) []string {
	ports := map[string]struct{}{}
	if defaultListen80 {
		ports["80"] = struct{}{}
	}
	for _, domain := range domains {
		for _, port := range domain.HttpListen {
			port = strings.TrimSpace(port)
			if port != "" {
				ports[port] = struct{}{}
			}
		}
	}
	return sortedPorts(ports)
}

func collectHTTPSPorts(domains []edgeDomain) []string {
	ports := map[string]struct{}{}
	for _, domain := range domains {
		for _, port := range domain.HttpsListen {
			port = strings.TrimSpace(port)
			if port != "" {
				ports[port] = struct{}{}
			}
		}
	}
	return sortedPorts(ports)
}

func sortedPorts(ports map[string]struct{}) []string {
	if len(ports) == 0 {
		return nil
	}
	out := make([]string, 0, len(ports))
	for port := range ports {
		out = append(out, port)
	}
	sort.Strings(out)
	return out
}

func appendUniquePort(ports []string, port string) []string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ports
	}
	for _, existing := range ports {
		if existing == port {
			return ports
		}
	}
	return append(ports, port)
}

func fallbackCertPath() string {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return "cert/fallback.pem"
	}
	certPath := filepath.Join(rootDir, "cert", "fallback.pem")
	if abs, err := filepath.Abs(certPath); err == nil {
		certPath = abs
	}
	return filepath.ToSlash(certPath)
}

func fallbackKeyPath() string {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return "cert/fallback.key"
	}
	keyPath := filepath.Join(rootDir, "cert", "fallback.key")
	if abs, err := filepath.Abs(keyPath); err == nil {
		keyPath = abs
	}
	return filepath.ToSlash(keyPath)
}

func writeDefaultServer(b *strings.Builder, port string, tls bool, errorPages map[string]string, errorPageDir string, status int) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	if tls {
		fallbackCert := fallbackCertPath()
		fallbackKey := fallbackKeyPath()
		b.WriteString("    listen " + port + " ssl default_server;\n")
		b.WriteString("    ssl_certificate " + fallbackCert + ";\n")
		b.WriteString("    ssl_certificate_key " + fallbackKey + ";\n")
	} else {
		b.WriteString("    listen " + port + " default_server;\n")
	}
	b.WriteString("    server_name _;\n")
	writeErrorPageServerDirectives(b, errorPages)
	writeErrorPageDirectives(b, errorPages, errorPageDir)
	b.WriteString("    location / {\n")
	b.WriteString(fmt.Sprintf("        return %d;\n", status))
	b.WriteString("    }\n")
	b.WriteString("}\n")
}

func writeDomainServers(b *strings.Builder, domain edgeDomain, errorPages map[string]string, errorPageDir string, defaultListen80 bool) {
	httpPorts := domain.HttpListen
	if len(httpPorts) == 0 && defaultListen80 {
		httpPorts = []string{"80"}
	}
	httpsPorts := domain.HttpsListen

	blockedCode := blockedStatusCode(domain, errorPages)
	if blockedCode > 0 {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, blockedCode)
		}
		for _, port := range httpsPorts {
			writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, blockedCode)
		}
		return
	}

	if domain.HTTPSForce && len(httpsPorts) > 0 {
		writeHTTPSRedirectServer(b, domain, httpPorts, httpsPorts, errorPages, errorPageDir)
	} else {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, 0)
		}
	}

	for _, port := range httpsPorts {
		writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, 0)
	}
}

func writeHTTPSRedirectServer(b *strings.Builder, domain edgeDomain, httpPorts []string, httpsPorts []string, errorPages map[string]string, errorPageDir string) {
	redirectPort := domain.HTTPSRedirectPort
	if redirectPort == "" {
		redirectPort = "443"
	}
	for _, port := range httpPorts {
		if strings.TrimSpace(port) == "" {
			continue
		}
		b.WriteString("server {\n")
		b.WriteString("    listen " + port + ";\n")
		b.WriteString("    server_name " + domain.Name + ";\n")
		writeErrorPageServerDirectives(b, errorPages)
		writeErrorPageDirectives(b, errorPages, errorPageDir)
		writeAcmeLocation(b)
		b.WriteString("    location / {\n")
		b.WriteString("        return 301 https://$host:" + redirectPort + "$request_uri;\n")
		b.WriteString("    }\n")
		b.WriteString("}\n")
	}
}

func writeHTTPServer(b *strings.Builder, domain edgeDomain, port string, tls bool, errorPages map[string]string, errorPageDir string, blockedCode int) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	if tls {
		fallbackCert := fallbackCertPath()
		fallbackKey := fallbackKeyPath()
		b.WriteString("    listen " + port + " ssl;\n")
		if domain.HTTPSHTTP2 {
			b.WriteString("    http2 on;\n")
		}
		if domain.HTTPSHTTP3 {
			b.WriteString(fmt.Sprintf("    add_header Alt-Svc 'h3=\\\":%s\\\"; ma=86400' always;\n", port))
		}
		b.WriteString("    ssl_certificate " + fallbackCert + ";\n")
		b.WriteString("    ssl_certificate_key " + fallbackKey + ";\n")
		b.WriteString("    ssl_certificate_by_lua_block {\n")
		b.WriteString("        local ssl_mgr = require \"lua.ssl_manager\"\n")
		b.WriteString("        ssl_mgr.set_certificate()\n")
		b.WriteString("    }\n")
		if protocols := sanitizeNginxValue(domain.HTTPSSSLProtocols); protocols != "" {
			b.WriteString("    ssl_protocols " + protocols + ";\n")
		}
		if ciphers := sanitizeNginxValue(domain.HTTPSSSLCiphers); ciphers != "" {
			b.WriteString("    ssl_ciphers " + ciphers + ";\n")
		}
		if prefer := sanitizeNginxToken(domain.HTTPSSSLPreferServerCiphers); prefer != "" {
			b.WriteString("    ssl_prefer_server_ciphers " + prefer + ";\n")
		}
		if domain.HTTPSOCSP {
			b.WriteString("    ssl_stapling on;\n")
			b.WriteString("    ssl_stapling_verify on;\n")
		}
		if domain.HTTPSHSTS {
			b.WriteString("    add_header Strict-Transport-Security \"max-age=31536000\" always;\n")
		}
	} else {
		b.WriteString("    listen " + port + ";\n")
	}
	b.WriteString("    server_name " + domain.Name + ";\n")
	if blockedCode > 0 {
		writeErrorPageServerDirectives(b, errorPages)
		writeErrorPageDirectives(b, errorPages, errorPageDir)
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        return %d;\n", blockedCode))
		b.WriteString("    }\n")
		b.WriteString("}\n")
		return
	}
	if domain.BodyLimit > 0 {
		b.WriteString(fmt.Sprintf("    client_max_body_size %dm;\n", domain.BodyLimit))
	}
	if domain.EnableGzip {
		b.WriteString("    gzip on;\n")
		if types := sanitizeNginxValue(domain.GzipTypes); types != "" {
			b.WriteString("    gzip_types " + types + ";\n")
		}
	}
	if domain.LimitRate > 0 {
		b.WriteString(fmt.Sprintf("    limit_rate %d;\n", domain.LimitRate))
	}
	if domain.ConnLimit > 0 {
		b.WriteString(fmt.Sprintf("    limit_conn addr_conn %d;\n", domain.ConnLimit))
	}

	if _, ok := errorPages["conn_limit"]; ok {
		b.WriteString("    limit_conn_status 429;\n")
	}

	writeErrorPageServerDirectives(b, errorPages)
	writeErrorPageDirectives(b, errorPages, errorPageDir)

	b.WriteString("    set $cc_rule_id " + fmt.Sprintf("%d", domain.CCRuleID) + ";\n")

	writeCacheLocations(b, domain, tls)

	b.WriteString("}\n")
}

func normalizeErrorPages(pages map[string]string) map[string]string {
	if len(pages) == 0 {
		return nil
	}
	out := make(map[string]string, len(pages))
	for code, content := range pages {
		key := strings.TrimSpace(code)
		val := strings.TrimSpace(content)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeErrorPageFiles(dir string, pages map[string]string) error {
	if len(pages) == 0 {
		return nil
	}
	if err := fsutil.EnsureDir(dir); err != nil {
		return err
	}
	for _, code := range sortedStringKeys(pages) {
		content := pages[code]
		filename := filepath.Join(dir, code+".html")
		if err := fsutil.WriteFileAtomic(filename, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func isNumericStatus(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func writeErrorPageDirectives(b *strings.Builder, pages map[string]string, dir string) {
	if len(pages) == 0 {
		return
	}
	for _, key := range sortedStringKeys(pages) {
		status := errorPageStatusForKey(key)
		if status == 0 {
			continue
		}
		fileName := key + ".html"
		uri := "/__cdn_error/" + fileName
		filePath := filepath.ToSlash(filepath.Join(dir, fileName))
		b.WriteString(fmt.Sprintf("    error_page %d %s;\n", status, uri))
		b.WriteString("    location = " + uri + " {\n")
		b.WriteString("        internal;\n")
		b.WriteString("        default_type text/html;\n")
		b.WriteString("        set $cdn_client_ip $remote_addr;\n")
		b.WriteString("        if ($realip_remote_addr != \"\") {\n")
		b.WriteString("            set $cdn_client_ip $realip_remote_addr;\n")
		b.WriteString("        }\n")
		b.WriteString("        sub_filter '{client_ip}' '$cdn_client_ip';\n")
		b.WriteString("        sub_filter '{node_ip}' '$server_addr';\n")
		b.WriteString("        sub_filter_once off;\n")
		b.WriteString("        alias " + filePath + ";\n")
		b.WriteString("    }\n")
	}
}

func writeErrorPageServerDirectives(b *strings.Builder, pages map[string]string) {
	if len(pages) == 0 {
		return
	}
	b.WriteString("    sub_filter_types text/html;\n")
}

func errorPageStatusForKey(key string) int {
	if isNumericStatus(key) {
		if v, err := strconv.Atoi(key); err == nil {
			return v
		}
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "traffic_limit":
		return 509
	case "site_locked":
		return 451
	case "domain_invalid":
		return 404
	case "conn_limit":
		return 429
	case "timeout":
		return 410
	case "ip":
		return 418
	default:
		return 0
	}
}

func blockedStatusCode(domain edgeDomain, pages map[string]string) int {
	status := strings.ToLower(strings.TrimSpace(domain.Status))
	var key string
	switch status {
	case "locked":
		key = "site_locked"
	case "expired":
		key = "timeout"
	case "traffic_limit":
		key = "traffic_limit"
	case "conn_limit":
		key = "conn_limit"
	default:
		return 0
	}
	if _, ok := pages[key]; !ok {
		return 0
	}
	return errorPageStatusForKey(key)
}

func writeCacheLocations(b *strings.Builder, domain edgeDomain, tls bool) {
	writeGuardLocations(b)
	writeAcmeLocation(b)
	cacheCfg := domain.Cache
	rules := make([]edgeCacheRule, 0)
	if cacheCfg != nil && len(cacheCfg.Rules) > 0 {
		rules = append(rules, cacheCfg.Rules...)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	for _, rule := range rules {
		location := buildRuleLocation(rule)
		if location == "" {
			continue
		}
		b.WriteString("    location " + location + " {\n")
		writeProxyBlock(b, domain, tls, cacheCfg, &rule)
		b.WriteString("    }\n")
	}

	b.WriteString("    location / {\n")
	writeProxyBlock(b, domain, tls, cacheCfg, nil)
	b.WriteString("    }\n")
}

func writeGuardLocations(b *strings.Builder) {
	rootDir := runtimeRoot()
	guardDir := filepath.Join(rootDir, "conf", "guard")
	if abs, err := filepath.Abs(guardDir); err == nil {
		guardDir = abs
	}
	guardDir = filepath.ToSlash(guardDir) + "/"
	b.WriteString("    location = /_guard/captcha.png {\n")
	b.WriteString("        default_type image/png;\n")
	b.WriteString("        content_by_lua_block {\n")
	b.WriteString("            local guard = require \"lua.guard\"\n")
	b.WriteString("            guard.serve_captcha_png()\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")

	b.WriteString("    location = /_guard/rotate_image {\n")
	b.WriteString("        default_type image/jpeg;\n")
	b.WriteString("        content_by_lua_block {\n")
	b.WriteString("            local guard = require \"lua.guard\"\n")
	b.WriteString("            guard.serve_rotate_image()\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")

	b.WriteString("    location ^~ /_guard/ {\n")
	b.WriteString("        alias " + guardDir + ";\n")
	b.WriteString("    }\n")
}

func writeAcmeLocation(b *strings.Builder) {
	rootDir := runtimeRoot()
	acmeRoot := filepath.Join(rootDir, "cert", "acme")
	if abs, err := filepath.Abs(acmeRoot); err == nil {
		acmeRoot = abs
	}
	acmeRoot = filepath.ToSlash(acmeRoot)
	apiBase := strings.TrimRight(strings.TrimSpace(API_BaseURL), "/")
	if apiBase == "" {
		return
	}
	b.WriteString("    location ^~ /.well-known/acme-challenge/ {\n")
	b.WriteString("        root " + acmeRoot + ";\n")
	b.WriteString("        default_type text/plain;\n")
	b.WriteString("        try_files $uri @acme_master;\n")
	b.WriteString("    }\n")
	b.WriteString("    location @acme_master {\n")
	b.WriteString("        proxy_pass " + apiBase + ";\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("    }\n")
}

func buildRuleLocation(rule edgeCacheRule) string {
	if rule.Rule != "" {
		return normalizeRuleLocation(rule.Rule)
	}
	if rule.URI != "" {
		return "= " + rule.URI
	}
	if rule.Prefix != "" {
		return "^~ " + rule.Prefix
	}
	if rule.Ext != "" {
		ext := rule.Ext
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		return "~* \\" + ext + "$"
	}
	return ""
}

func normalizeRuleLocation(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	if strings.HasPrefix(rule, "=") {
		return rule
	}
	if strings.HasPrefix(rule, "^~") || strings.HasPrefix(rule, "~") {
		return rule
	}
	if strings.HasPrefix(rule, "/") {
		return "^~ " + rule
	}
	if strings.HasPrefix(rule, ".") {
		return "~* \\" + rule + "$"
	}
	return "~* " + rule
}

func writeProxyBlock(b *strings.Builder, domain edgeDomain, tls bool, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	writeProxyBase(b)
	writeProxyProtocol(b, domain)
	writeProxyTimeouts(b, domain)
	writeProxyRanges(b, domain)
	writeProxyCustomHeaders(b, domain.Headers, domain.ResponseHeaders)
	b.WriteString("        proxy_pass $backend_target;\n")
	writeProxySSL(b, domain)
	applyCacheDirectives(b, cacheCfg, rule)
}

func writeProxyBase(b *strings.Builder) {
	b.WriteString("        limit_req zone=cc_limit burst=20 nodelay;\n")
	b.WriteString("        limit_conn addr_conn 50;\n")
	b.WriteString("        set $backend_target \"\";\n")
	b.WriteString("        access_by_lua_file lua/access_guard.lua;\n")
	b.WriteString("        header_filter_by_lua_file lua/response_headers.lua;\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
}

func writeProxyProtocol(b *strings.Builder, domain edgeDomain) {
	if domain.EnableWebsocket {
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection $connection_upgrade;\n")
		return
	}
	if version := sanitizeProxyHTTPVersion(domain.ProxyHTTPVersion); version != "" {
		b.WriteString("        proxy_http_version " + version + ";\n")
	}
}

func writeProxyTimeouts(b *strings.Builder, domain edgeDomain) {
	if timeout := sanitizeNginxToken(domain.ProxyConnectTimeout); timeout != "" {
		b.WriteString("        proxy_connect_timeout " + timeout + ";\n")
	}
	if timeout := sanitizeNginxToken(domain.ProxyReadTimeout); timeout != "" {
		b.WriteString("        proxy_read_timeout " + timeout + ";\n")
	}
	if timeout := sanitizeNginxToken(domain.ProxySendTimeout); timeout != "" {
		b.WriteString("        proxy_send_timeout " + timeout + ";\n")
	}
}

func writeProxyRanges(b *strings.Builder, domain edgeDomain) {
	if domain.EnableRange {
		b.WriteString("        proxy_force_ranges on;\n")
	}
}

func writeProxyCustomHeaders(b *strings.Builder, headers map[string]string, responseHeaders map[string]string) {
	for _, key := range sortedStringKeys(headers) {
		value := headers[key]
		name := sanitizeHeaderName(key)
		value = sanitizeHeaderValue(value)
		if name == "" || value == "" {
			continue
		}
		b.WriteString("        proxy_set_header " + name + " " + quoteNginxValue(value) + ";\n")
	}
	for _, key := range sortedStringKeys(responseHeaders) {
		value := responseHeaders[key]
		name := sanitizeHeaderName(key)
		value = sanitizeHeaderValue(value)
		if name == "" || value == "" {
			continue
		}
		b.WriteString("        add_header " + name + " " + quoteNginxValue(value) + " always;\n")
	}
}

func writeProxySSL(b *strings.Builder, domain edgeDomain) {
	if strings.ToLower(domain.OriginProtocol) == "http" {
		return
	}
	b.WriteString("        proxy_ssl_server_name on;\n")
	if protocols := sanitizeNginxValue(domain.ProxySSLProtocols); protocols != "" {
		b.WriteString("        proxy_ssl_protocols " + protocols + ";\n")
	}
}

func applyCacheDirectives(b *strings.Builder, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	if cacheCfg == nil || !cacheCfg.Enable {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	enabled := true
	if rule != nil && rule.Enable != nil && !*rule.Enable {
		enabled = false
	}
	if rule != nil && rule.NoCache {
		enabled = false
	}
	if !enabled {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	b.WriteString("        proxy_cache my_cache;\n")
	b.WriteString("        proxy_cache_lock on;\n")
	b.WriteString("        proxy_cache_lock_timeout 5s;\n")
	b.WriteString("        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;\n")
	b.WriteString("        proxy_cache_background_update on;\n")
	if rule != nil && rule.ForceCache {
		b.WriteString("        proxy_ignore_headers Cache-Control Expires;\n")
	}
	ttl := cacheCfg.DefaultTTL
	if rule != nil && rule.TTL > 0 {
		ttl = rule.TTL
	}
	if ttl > 0 {
		b.WriteString(fmt.Sprintf("        proxy_cache_valid 200 302 %ds;\n", ttl))
	}
	cacheKey := ""
	if rule != nil && rule.CacheKey != "" {
		cacheKey = rule.CacheKey
	} else if rule != nil && rule.IgnoreArgs {
		cacheKey = "$host$uri"
	} else {
		cacheKey = "$host$uri$is_args$args"
	}
	b.WriteString("        proxy_cache_key " + cacheKey + ";\n")
}

func writeHTTPGlobalConfig(cfg *edgeNginxConfig) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "http_global.conf")
	var b strings.Builder
	defaultCacheDir := filepath.ToSlash(filepath.Join(rootDir, "cache"))
	cacheDir := defaultCacheDir
	cacheMaxSize := ""
	cacheZoneSize := ""
	if cfg != nil && cfg.HTTP != nil {
		if v := sanitizeNginxValue(toString(cfg.HTTP["proxy_cache_dir"])); v != "" {
			cacheDir = v
		}
		cacheMaxSize = toString(cfg.HTTP["proxy_cache_max_size"])
		cacheZoneSize = toString(cfg.HTTP["proxy_cache_keys_zone_size"])
	}
	if cacheDir == "" {
		cacheDir = defaultCacheDir
	}
	if err := fsutil.EnsureDir(filepath.FromSlash(cacheDir)); err != nil {
		if cacheDir != defaultCacheDir {
			if err2 := fsutil.EnsureDir(filepath.FromSlash(defaultCacheDir)); err2 != nil {
				log.Printf("[Warn] Ensure proxy cache dir failed: %s: %v; fallback failed: %v", cacheDir, err, err2)
			} else {
				log.Printf("[Warn] Ensure proxy cache dir failed: %s: %v; fallback to %s", cacheDir, err, defaultCacheDir)
				cacheDir = defaultCacheDir
			}
		} else {
			log.Printf("[Warn] Ensure proxy cache dir failed: %s: %v", cacheDir, err)
		}
	}
	zoneSize := "50m"
	if cacheZoneSize != "" {
		zoneSize = cacheZoneSize
	}
	cacheLine := "proxy_cache_path " + cacheDir + " levels=1:2 keys_zone=my_cache:" + zoneSize + " inactive=24h use_temp_path=off"
	if cacheMaxSize != "" {
		cacheLine = cacheLine + " max_size=" + cacheMaxSize
	}
	b.WriteString(cacheLine + ";\n")

	if cfg != nil && cfg.HTTP != nil {
		writeHTTPDirectives(&b, cfg.HTTP)
		if v := toString(cfg.HTTP["proxy_cache_methods"]); v != "" {
			b.WriteString("proxy_cache_methods " + v + ";\n")
		}
		if v := toString(cfg.HTTP["custom_snippet"]); v != "" {
			if !strings.HasSuffix(v, "\n") {
				v += "\n"
			}
			b.WriteString(v)
		}
	}
	if cfg != nil {
		if v := strings.TrimSpace(cfg.Resolver); v != "" {
			b.WriteString("resolver " + v + ";\n")
		}
		if v := strings.TrimSpace(cfg.ResolverTimeout); v != "" {
			b.WriteString("resolver_timeout " + v + ";\n")
		}
		if logs := strings.TrimSpace(cfg.LogsDir); logs != "" {
			logs = strings.TrimRight(logs, "/")
			b.WriteString("access_log " + logs + "/access.json json_analytics;\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}
