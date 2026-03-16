package main

import (
	fsutil "cdn-common/io"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func resolveSiteTemplate(siteType string, defaults *edgeDefaultConfig) edgeSiteTemplate {
	if defaults == nil {
		return edgeSiteTemplate{}
	}
	switch strings.ToLower(strings.TrimSpace(siteType)) {
	case "api":
		return defaults.API
	case "download":
		return defaults.Download
	default:
		return defaults.Website
	}
}

func applyDefaultConfigToDomain(domain *edgeDomain, defaults *edgeDefaultConfig) {
	if domain == nil || defaults == nil {
		return
	}
	template := resolveSiteTemplate(domain.SiteType, defaults)
	if domain.Cache == nil {
		if template.CacheEnable || template.CacheTTL > 0 {
			domain.Cache = &edgeCacheConfig{
				Enable:     template.CacheEnable,
				DefaultTTL: template.CacheTTL,
			}
		}
	} else {
		if domain.Cache.DefaultTTL == 0 && template.CacheTTL > 0 {
			domain.Cache.DefaultTTL = template.CacheTTL
		}
		if !domain.Cache.Enable && template.CacheEnable {
			if domain.Cache.DefaultTTL == 0 && len(domain.Cache.Rules) == 0 {
				domain.Cache.Enable = true
			}
		}
	}
	if !domain.EnableGzip && template.Gzip {
		if strings.TrimSpace(domain.GzipTypes) == "" {
			domain.EnableGzip = true
		}
	}
	if strings.TrimSpace(domain.HTTPSSSLCiphers) == "" && strings.TrimSpace(template.SSLCiphers) != "" {
		domain.HTTPSSSLCiphers = template.SSLCiphers
	}
	if domain.WAFEnable == nil && template.WAFEnable {
		enabled := true
		domain.WAFEnable = &enabled
	}
}

func applyListenPortRules(domain *edgeDomain, resources *edgeResources) {
	if domain == nil || resources == nil {
		return
	}
	allowed := strings.TrimSpace(resources.Public.AllowedCustomPorts)
	disabled := strings.TrimSpace(resources.Public.DisabledCustomPorts)
	if allowed == "" && disabled == "" {
		return
	}
	domain.HttpListen = filterCustomPorts(domain.HttpListen, allowed, disabled)
	domain.HttpsListen = filterCustomPorts(domain.HttpsListen, allowed, disabled)
}

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

	defaults := cfg.DefaultConfig
	if defaults == nil {
		defaults = LocalDefaultConf
	}
	processedDomains := make([]edgeDomain, 0, len(cfg.Domains))
	for _, domain := range cfg.Domains {
		applyDefaultConfigToDomain(&domain, defaults)
		applyListenPortRules(&domain, cfg.Resources)
		processedDomains = append(processedDomains, domain)
	}

	defaultListen80 := true
	if cfg.Resources != nil {
		defaultListen80 = cfg.Resources.Website.DefaultListen80
	}
	blockUnbound := cfg.WAF != nil && cfg.WAF.BlockUnboundDomain
	defaultDomain := pickDefaultDomain(processedDomains, blockUnbound)
	ipv6Enabled := hasIPv6Enabled(processedDomains)

	upstreamKeepalive := map[string]edgeDomain{}
	for _, domain := range processedDomains {
		if domain.UpstreamKey != "" && domain.UpstreamKeepalive {
			upstreamKeepalive[domain.UpstreamKey] = domain
		}
	}

	var b strings.Builder
	seenUpstreams := map[string]struct{}{}
	for _, upstream := range cfg.Upstreams {
		id := strings.TrimSpace(upstream.ID)
		if id == "" || len(upstream.Targets) == 0 {
			continue
		}
		if _, exists := seenUpstreams[id]; exists {
			log.Printf("[Warn] Duplicate upstream skipped: id=%s", id)
			continue
		}
		seenUpstreams[id] = struct{}{}
		b.WriteString("upstream " + id + " {\n")
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
			timeout := keep.UpstreamKeepaliveTimeout
			if timeout <= 0 {
				timeout = 30
			}
			b.WriteString(fmt.Sprintf("    keepalive_timeout %ds;\n", timeout))
		}
		b.WriteString("}\n")
	}

	for _, domain := range processedDomains {
		if domain.Name == "" || domain.UpstreamKey == "" {
			continue
		}
		isDefault := defaultDomain != nil && domain.Name == defaultDomain.Name
		writeDomainServers(&b, domain, errorPages, errorPageDir, defaultListen80, isDefault)
	}

	if blockUnbound {
		blockedStatus := errorPageStatusForKey("ip")
		if blockedStatus == 0 {
			blockedStatus = 418
		}

		httpPorts := collectHTTPPorts(processedDomains, defaultListen80)
		if defaultListen80 {
			httpPorts = appendUniquePort(httpPorts, "80")
		}
		for _, port := range httpPorts {
			writeDefaultServer(&b, port, false, errorPages, errorPageDir, blockedStatus, ipv6Enabled)
		}

		for _, port := range collectHTTPSPorts(processedDomains) {
			writeDefaultServer(&b, port, true, errorPages, errorPageDir, blockedStatus, ipv6Enabled)
		}
	} else if defaultDomain == nil && shouldBindDefaultHTTP(processedDomains, defaultListen80) {
		httpPorts := collectHTTPPorts(processedDomains, defaultListen80)
		for _, port := range httpPorts {
			writeDefaultServer(&b, port, false, errorPages, errorPageDir, 404, ipv6Enabled)
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

func pickDefaultDomain(domains []edgeDomain, blockUnbound bool) *edgeDomain {
	if blockUnbound {
		return nil
	}
	for i := range domains {
		if domains[i].DefaultSite {
			return &domains[i]
		}
	}
	return nil
}

func hasIPv6Enabled(domains []edgeDomain) bool {
	for _, domain := range domains {
		if domain.IPv6Enable {
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

func writeDefaultServer(b *strings.Builder, port string, tls bool, errorPages map[string]string, errorPageDir string, status int, ipv6Enable bool) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	if tls {
		fallbackCert := fallbackCertPath()
		fallbackKey := fallbackKeyPath()
		b.WriteString("    listen " + port + " ssl default_server;\n")
		if ipv6Enable {
			b.WriteString("    listen [::]:" + port + " ssl default_server;\n")
		}
		b.WriteString("    ssl_certificate " + fallbackCert + ";\n")
		b.WriteString("    ssl_certificate_key " + fallbackKey + ";\n")
	} else {
		b.WriteString("    listen " + port + " default_server;\n")
		if ipv6Enable {
			b.WriteString("    listen [::]:" + port + " default_server;\n")
		}
	}
	b.WriteString("    server_name _;\n")
	writeErrorPageServerDirectives(b, errorPages)
	writeErrorPageDirectives(b, errorPages, errorPageDir)
	b.WriteString("    location / {\n")
	b.WriteString(fmt.Sprintf("        return %d;\n", status))
	b.WriteString("    }\n")
	b.WriteString("}\n")
}

func writeDomainServers(b *strings.Builder, domain edgeDomain, errorPages map[string]string, errorPageDir string, defaultListen80 bool, defaultServer bool) {
	httpPorts := domain.HttpListen
	if len(httpPorts) == 0 && defaultListen80 {
		httpPorts = []string{"80"}
	}
	httpsPorts := domain.HttpsListen

	blockedCode := blockedStatusCode(domain, errorPages)
	if blockedCode > 0 {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, blockedCode, defaultServer)
		}
		for _, port := range httpsPorts {
			writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, blockedCode, defaultServer)
		}
		return
	}

	if domain.HTTPSForce && len(httpsPorts) > 0 {
		writeHTTPSRedirectServer(b, domain, httpPorts, httpsPorts, errorPages, errorPageDir, defaultServer)
	} else {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, 0, defaultServer)
		}
	}

	for _, port := range httpsPorts {
		writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, 0, defaultServer)
	}
}

func writeHTTPSRedirectServer(b *strings.Builder, domain edgeDomain, httpPorts []string, httpsPorts []string, errorPages map[string]string, errorPageDir string, defaultServer bool) {
	redirectPort := domain.HTTPSRedirectPort
	if redirectPort == "" {
		redirectPort = "443"
	}
	listenSuffix := ""
	if defaultServer {
		listenSuffix = " default_server"
	}
	for _, port := range httpPorts {
		if strings.TrimSpace(port) == "" {
			continue
		}
		b.WriteString("server {\n")
		b.WriteString("    listen " + port + listenSuffix + ";\n")
		if domain.IPv6Enable {
			b.WriteString("    listen [::]:" + port + listenSuffix + ";\n")
		}
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

func writeHTTPServer(b *strings.Builder, domain edgeDomain, port string, tls bool, errorPages map[string]string, errorPageDir string, blockedCode int, defaultServer bool) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	listenSuffix := ""
	if tls {
		listenSuffix = " ssl"
	}
	if defaultServer {
		listenSuffix += " default_server"
	}
	if tls {
		fallbackCert := fallbackCertPath()
		fallbackKey := fallbackKeyPath()
		b.WriteString("    listen " + port + listenSuffix + ";\n")
		if domain.IPv6Enable {
			b.WriteString("    listen [::]:" + port + listenSuffix + ";\n")
		}
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
		b.WriteString("    listen " + port + listenSuffix + ";\n")
		if domain.IPv6Enable {
			b.WriteString("    listen [::]:" + port + listenSuffix + ";\n")
		}
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
		b.WriteString(fmt.Sprintf("    client_max_body_size %dk;\n", domain.BodyLimit))
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
	b.WriteString("    sub_filter_types *;\n")
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
	seenLocations := map[string]string{}
	seedLocation := func(location string, reason string) {
		if key := normalizeLocationKey(location); key != "" {
			seenLocations[key] = reason
		}
	}
	seedLocation("= /_guard/captcha.png", "reserved:guard_captcha")
	seedLocation("= /_guard/rotate_image", "reserved:guard_rotate")
	seedLocation("^~ /_guard/", "reserved:guard_dir")
	seedLocation("^~ /.well-known/acme-challenge/", "reserved:acme_challenge")
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
			if hasRuleSpecifier(rule) {
				log.Printf("[Warn] Invalid cache rule skipped: domain=%s rule=%q uri=%q prefix=%q ext=%q", domain.Name, rule.Rule, rule.URI, rule.Prefix, rule.Ext)
			}
			continue
		}
		key := normalizeLocationKey(location)
		if key != "" {
			if reason, exists := seenLocations[key]; exists {
				log.Printf("[Warn] Duplicate cache location skipped: domain=%s location=%q (conflict with %s)", domain.Name, location, reason)
				continue
			}
			seenLocations[key] = "cache_rule"
		}
		b.WriteString("    location " + location + " {\n")
		writeProxyBlock(b, domain, tls, cacheCfg, &rule)
		b.WriteString("    }\n")
	}

	defaultKey := normalizeLocationKey("/")
	if reason, exists := seenLocations[defaultKey]; exists {
		log.Printf("[Warn] Default cache location skipped: domain=%s location=%q (conflict with %s)", domain.Name, "/", reason)
	} else {
		b.WriteString("    location / {\n")
		writeProxyBlock(b, domain, tls, cacheCfg, nil)
		b.WriteString("    }\n")
		seenLocations[defaultKey] = "cache_default"
	}
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
		uri := strings.TrimSpace(rule.URI)
		if !strings.HasPrefix(uri, "/") {
			return ""
		}
		return "= " + uri
	}
	if rule.Prefix != "" {
		prefix := strings.TrimSpace(rule.Prefix)
		if !strings.HasPrefix(prefix, "/") {
			return ""
		}
		return "^~ " + prefix
	}
	if rule.Ext != "" {
		ext := strings.TrimSpace(rule.Ext)
		if ext == "" {
			return ""
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		return "~* \\" + ext + "$"
	}
	return ""
}

func hasRuleSpecifier(rule edgeCacheRule) bool {
	return strings.TrimSpace(rule.Rule) != "" ||
		strings.TrimSpace(rule.URI) != "" ||
		strings.TrimSpace(rule.Prefix) != "" ||
		strings.TrimSpace(rule.Ext) != ""
}

func normalizeLocationKey(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	parts := strings.Fields(location)
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "=":
		if len(parts) < 2 {
			return "exact"
		}
		return "exact " + strings.Join(parts[1:], " ")
	case "^~":
		if len(parts) < 2 {
			return "prefix"
		}
		return "prefix " + strings.Join(parts[1:], " ")
	default:
		if strings.HasPrefix(parts[0], "~") {
			if len(parts) < 2 {
				return "regex " + parts[0]
			}
			return "regex " + parts[0] + " " + strings.Join(parts[1:], " ")
		}
	}
	return "prefix " + strings.Join(parts, " ")
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
	writeProxyLogVars(b)
	writeProxyProtocol(b, domain)
	writeProxyTimeouts(b, domain)
	writeProxyBuffering(b, domain)
	writeProxyRanges(b, domain, rule)
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

func writeProxyLogVars(b *strings.Builder) {
	b.WriteString("        set $cdn_req_headers \"\";\n")
	b.WriteString("        set $cdn_resp_headers \"\";\n")
	b.WriteString("        set $cdn_req_body \"\";\n")
	b.WriteString("        set $cache_bypass 0;\n")
	b.WriteString("        set $cache_ttl 0;\n")
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
		if version == "1.1" && domain.UpstreamKeepalive {
			b.WriteString("        proxy_set_header Connection \"\";\n")
		}
		return
	}
	if domain.UpstreamKeepalive {
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Connection \"\";\n")
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

func writeProxyBuffering(b *strings.Builder, domain edgeDomain) {
	if domain.RealtimeSend {
		b.WriteString("        proxy_request_buffering off;\n")
	}
	if domain.RealtimeReturn || domain.RealtimeIdentify {
		b.WriteString("        proxy_buffering off;\n")
	}
}

func writeProxyRanges(b *strings.Builder, domain edgeDomain, rule *edgeCacheRule) {
	if domain.EnableRange || (rule != nil && rule.EnableRange) {
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
	if domain.OriginCert {
		if caPath := resolveOriginTrustedCA(); caPath != "" {
			b.WriteString("        proxy_ssl_verify on;\n")
			b.WriteString("        proxy_ssl_trusted_certificate " + caPath + ";\n")
			b.WriteString("        proxy_ssl_verify_depth 2;\n")
		} else {
			log.Printf("[Warn] Origin cert verify enabled but no trusted CA bundle found")
		}
	}
	if protocols := sanitizeNginxValue(domain.ProxySSLProtocols); protocols != "" {
		b.WriteString("        proxy_ssl_protocols " + protocols + ";\n")
	}
}

func resolveOriginTrustedCA() string {
	paths := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
	}
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
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
	if rule != nil && rule.IgnoreVary {
		b.WriteString("        proxy_ignore_headers Vary;\n")
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
	b.WriteString("        proxy_cache_bypass $cache_bypass;\n")
	b.WriteString("        proxy_no_cache $cache_bypass;\n")
}

func writeHTTPGlobalConfig(cfg *edgeNginxConfig, cacheEnabled bool) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "http_global.conf")
	var b strings.Builder
	if cacheEnabled {
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
	}
	writeHTTPPerformanceDefaults(&b, cfg, cacheEnabled)

	if cfg != nil && cfg.HTTP != nil {
		writeHTTPDirectives(&b, cfg.HTTP)
		if v := toString(cfg.HTTP["proxy_cache_methods"]); v != "" {
			b.WriteString("proxy_cache_methods " + v + ";\n")
		}
		if v := toString(cfg.HTTP["custom_snippet"]); v != "" {
			if cleaned, removed := sanitizeCustomHTTPSnippet(v); removed {
				log.Printf("[Warn] custom_snippet contains types directives; stripped to avoid duplicate MIME types")
				v = cleaned
			} else {
				v = cleaned
			}
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
	}
	logs := ""
	if cfg != nil {
		logs = strings.TrimSpace(cfg.LogsDir)
	}
	if logs == "" {
		logs = filepath.ToSlash(filepath.Join(rootDir, "logs"))
	}
	if logs != "" {
		logs = strings.TrimRight(logs, "/")
		if err := fsutil.EnsureDir(filepath.FromSlash(logs)); err != nil {
			log.Printf("[Warn] Ensure log dir failed: %s: %v", logs, err)
		}
		b.WriteString("access_log " + logs + "/access.json json_analytics if=$cdn_realtime_send;\n")
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func sanitizeCustomHTTPSnippet(snippet string) (string, bool) {
	if snippet == "" {
		return "", false
	}
	lines := strings.Split(snippet, "\n")
	out := make([]string, 0, len(lines))
	removed := false
	inTypes := false
	braceDepth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		lower := strings.ToLower(trimmed)
		if inTypes {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 {
				inTypes = false
				braceDepth = 0
			}
			removed = true
			continue
		}
		if strings.HasPrefix(lower, "include") && strings.Contains(lower, "mime.types") {
			removed = true
			continue
		}
		if lower == "types" || strings.HasPrefix(lower, "types ") || strings.HasPrefix(lower, "types{") {
			removed = true
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 {
				braceDepth = 1
			}
			inTypes = true
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), removed
}
