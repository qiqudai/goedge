package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
)

type ConfigService struct{}

var (
	ErrNodeNotFound  = errors.New("node not found")
	ErrNodeIDMissing = errors.New("node_id is required")
)

// NewConfigService creates a new instance
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// GenerateConfigForNode constructs the configuration for a specific node
func (s *ConfigService) GenerateConfigForNode(nodeID string) (*models.EdgeConfig, error) {
	node, err := findNode(nodeID)
	if err != nil {
		return nil, err
	}

	payload := &models.EdgeConfig{
		Version:            0,
		NodeID:             strconv.FormatInt(node.ID, 10),
		NodeLevel:          node.Level,
		NodeBandwidthLimit: strings.TrimSpace(node.BwLimit),
		AntiBlocking:       true,
		Domains:            make([]models.EdgeDomain, 0),
		Upstreams:          make([]models.EdgeUpstream, 0),
	}
	if val, err := GetNodeConfigValue(node.ID, "anti_blocking"); err == nil && strings.TrimSpace(val) != "" {
		payload.AntiBlocking = ParseBoolFlag(val)
	}

	if globalCfg := loadGlobalConfig(); globalCfg != nil {
		payload.WAF = &globalCfg.WAF
		payload.Resources = &globalCfg.Resources
		payload.ErrorPageI18n = globalCfg.ErrorPageI18n
		payload.ErrorPages = globalCfg.ErrorPages
		payload.GuardPages = globalCfg.GuardPages
		payload.DefaultConfig = &globalCfg.DefaultConfig
	}
	if nginxCfg := loadNginxConfig(); nginxCfg != nil {
		payload.Nginx = nginxCfg
	}

	expireCloseEnabled := true
	if systemCfg, err := LoadSystemConfig(); err == nil {
		if val, ok := systemCfg["package_expire_close_site"]; ok {
			expireCloseEnabled = ParseBoolFlag(val)
		}
		payload.FallbackCertData = strings.TrimSpace(systemCfg["https_cert"])
		payload.FallbackKeyData = strings.TrimSpace(systemCfg["https_key"])
	}

	// 1. Find Node Groups this node belongs to
	var lines []models.Line
	if err := db.DB.Where("node_id = ?", node.ID).Find(&lines).Error; err != nil {
		return nil, err
	}

	var groupIDs []int64
	for _, l := range lines {
		groupIDs = append(groupIDs, l.NodeGroupID)
	}
	pendingGroups := LoadPendingGroupIDs(node.ID)
	if len(pendingGroups) > 0 {
		groupIDs = append(groupIDs, pendingGroups...)
	}
	groupIDs = uniqueInt64(groupIDs)
	if len(groupIDs) == 0 {
		payload.Version = hashConfigVersion(payload)
		return payload, nil
	}

	groupL2Config := loadNodeGroupL2Config(groupIDs)
	l2TargetsByGroup := map[int64][]l2Target{}
	l2UpstreamKeyByGroup := map[int64]string{}
	if node.Level == 1 {
		l2TargetsByGroup = loadL2TargetsByGroup(groupIDs)
		for groupID, targets := range l2TargetsByGroup {
			if len(targets) == 0 {
				continue
			}
			upstreamKey := fmt.Sprintf("l2_upstream_%d", groupID)
			l2UpstreamKeyByGroup[groupID] = upstreamKey
			upstreamTargets := make([]models.EdgeUpstreamTarget, 0, len(targets))
			for _, target := range targets {
				if target.IP == "" {
					continue
				}
				upstreamTargets = append(upstreamTargets, models.EdgeUpstreamTarget{
					Addr:   target.IP,
					Weight: 1,
					NodeID: target.NodeID,
				})
			}
			if len(upstreamTargets) > 0 {
				payload.Upstreams = append(payload.Upstreams, models.EdgeUpstream{
					ID:      upstreamKey,
					Targets: upstreamTargets,
				})
			}
		}
	}

	// 2. Find Sites assigned to these Node Groups
	siteDB := db.DB
	if omitColumns := siteMissingColumnsForTask(siteDB); len(omitColumns) > 0 {
		siteDB = siteDB.Omit(omitColumns...)
	}
	sites, err := loadSitesForConfigGroups(siteDB, groupIDs)
	if err != nil {
		return nil, err
	}
	nodeGroupCounts := loadNodeGroupCounts(groupIDs)
	userPackageMap, err := loadUserPackageMap(sites)
	if err != nil {
		return nil, err
	}
	domainCountByUserGroup := buildDomainCountByUserGroup(sites)

	// Preload certs for HTTPS mapping
	var certs []models.Cert
	_ = db.DB.Where("enable = ?", true).Find(&certs).Error

	siteIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		if site.ID != 0 {
			siteIDs = append(siteIDs, site.ID)
		}
	}
	siteDefaultGroupMap := LoadSiteDefaultGroupMap(siteIDs)

	usedRuleIDs := make([]int64, 0)
	now := time.Now()
	for _, site := range sites {
		status := "active"
		state := strings.ToLower(strings.TrimSpace(site.State))
		switch state {
		case "stop", "locked", "site_locked":
			status = "locked"
		case "traffic_limit":
			status = "traffic_limit"
		case "conn_limit":
			status = "conn_limit"
		case "expired", "timeout":
			status = "expired"
		}
		if !site.Enable {
			status = "locked"
		}

		effectiveSite := cloneSiteForConfig(site)
		defaultGroupID := siteDefaultGroupMap[site.ID]
		if defaults, err := GetSiteDefaultMapWithGroup(site.UserID, defaultGroupID); err == nil {
			ApplySiteDefaults(effectiveSite, defaults)
		}

		cacheCfg := extractCacheConfig(effectiveSite.Settings)
		originProtocol, originHTTPPort, originHTTPSPort := extractOriginConfig(*effectiveSite)
		originHostHeader, originSNI, originVerifyTLS := extractOriginTLSConfig(*effectiveSite)
		httpsCfg := extractHTTPSConfig(effectiveSite.Settings)
		advCfg := extractAdvancedConfig(effectiveSite.Settings)
		proxyTimeouts := extractProxyTimeouts(effectiveSite.Settings)
		l2Mode := resolveL2Mode(effectiveSite.Settings)
		packageL2Enabled := false
		if pkg, ok := userPackageMap[effectiveSite.UserPackageID]; ok {
			packageL2Enabled = pkg.L2Origin
		}
		l2Enabled := node.Level == 1 && resolveL2Enabled(l2Mode, groupL2Config[effectiveSite.NodeGroupID], packageL2Enabled)
		l2UpstreamKey := ""
		if l2Enabled {
			l2UpstreamKey = l2UpstreamKeyByGroup[effectiveSite.NodeGroupID]
			if l2UpstreamKey == "" {
				l2Enabled = false
			}
		}
		l2HTTPPort := ""
		l2HTTPSPort := ""
		if l2Enabled {
			l2HTTPPort = resolveListenPort(effectiveSite.HttpListen, "80")
			l2HTTPSPort = resolveListenPort(effectiveSite.HttpsListen, "")
		}

		upstreamKey := fmt.Sprintf("upstream_%d", effectiveSite.ID)
		targets := buildUpstreamTargets(*effectiveSite, originProtocol, originHTTPPort, originHTTPSPort)
		if len(targets) > 0 {
			payload.Upstreams = append(payload.Upstreams, models.EdgeUpstream{
				ID:      upstreamKey,
				Targets: targets,
			})
		}

		policy := mapBalancePolicy(effectiveSite.BalanceWay)
		headers := buildHeaderMap(*effectiveSite)
		responseHeaders := buildResponseHeaderMap(effectiveSite.Settings)
		hasHTTPS := len(effectiveSite.HttpsListen) > 0 || strings.TrimSpace(effectiveSite.HttpsListenRaw) != ""
		selectedCertID := effectiveSite.CertID
		if effectiveSite.Settings != nil {
			if httpsMap := getMap(effectiveSite.Settings, "https"); httpsMap != nil {
				hasHTTPS = parseBoolValue(httpsMap["enable"], hasHTTPS)
				if selectedCertID == 0 {
					selectedCertID = int64(parseIntValue(httpsMap["certificate_id"], 0))
				}
			}
		}
		httpsListen := effectiveSite.HttpsListen
		if hasHTTPS && len(httpsListen) == 0 {
			httpsListen = []string{"443"}
		}
		if !hasHTTPS {
			httpsListen = nil
			selectedCertID = 0
		}
		if pkg, ok := userPackageMap[effectiveSite.UserPackageID]; ok {
			if expireCloseEnabled && !pkg.EndAt.IsZero() && pkg.EndAt.Before(now) {
				status = "expired"
			}
		}

		aclDefault, aclDenyStatus, aclRedirectURL, aclRules := buildACLForSite(*effectiveSite)
		ccAutoSwitch := extractCCAutoSwitch(effectiveSite.Settings)
		regionBlock := extractRegionBlock(*effectiveSite)
		hotlinkCfg := extractHotlinkConfig(effectiveSite.Settings)
		corsCfg := extractCorsConfig(effectiveSite.Settings)
		cookieCfg := extractCookieConfig(effectiveSite.Settings)
		wafEnable := extractWAFEnable(effectiveSite.Settings)
		blockTransparentProxy := extractBlockTransparentProxy(effectiveSite.Settings)
		crawlerAction := extractCrawlerAction(effectiveSite.Settings)
		guardPassTTL, guardBlockTTL := extractGuardTTLs(effectiveSite.Settings)
		urlRedirects := extractURLRedirects(effectiveSite.Settings)
		urlRewrites := extractURLRewrites(effectiveSite.Settings)
		originConditions := extractOriginConditions(effectiveSite.Settings)
		customCCRules := extractCustomCCRules(effectiveSite.Settings)
		siteType := strings.ToLower(strings.TrimSpace(parseString(effectiveSite.Settings["site_type"])))
		if siteType == "" {
			siteType = "website"
		}
		if effectiveSite.CcDefaultRule > 0 {
			usedRuleIDs = append(usedRuleIDs, effectiveSite.CcDefaultRule)
		}
		if ccAutoSwitch != nil && ccAutoSwitch.Enable && ccAutoSwitch.RuleID > 0 {
			usedRuleIDs = append(usedRuleIDs, ccAutoSwitch.RuleID)
		}
		for _, domain := range effectiveSite.Domains {
			normalizedDomain := normalizeDomainHostForEdge(domain)
			if normalizedDomain == "" {
				continue
			}
			limitRate := calcDomainLimitRate(userPackageMap[effectiveSite.UserPackageID].Bandwidth)
			connLimit := calcDomainConnLimit(
				effectiveSite.UserID,
				effectiveSite.NodeGroupID,
				userPackageMap[effectiveSite.UserPackageID].Connection,
				domainCountByUserGroup,
				nodeGroupCounts,
			)
			domainConf := models.EdgeDomain{
				Name:                           normalizedDomain,
				SiteType:                       siteType,
				UpstreamKey:                    upstreamKey,
				L2UpstreamKey:                  l2UpstreamKey,
				UseL2:                          l2Enabled,
				L2HTTPPort:                     l2HTTPPort,
				L2HTTPSPort:                    l2HTTPSPort,
				LoadBalancePolicy:              policy,
				Headers:                        headers,
				ResponseHeaders:                responseHeaders,
				Hotlink:                        hotlinkCfg,
				CORS:                           corsCfg,
				Cookie:                         cookieCfg,
				BlockTransparentProxy:          blockTransparentProxy,
				CrawlerAction:                  crawlerAction,
				GuardPassTTL:                   guardPassTTL,
				GuardBlockTTL:                  guardBlockTTL,
				URLRedirects:                   urlRedirects,
				URLRewrites:                    urlRewrites,
				OriginConditions:               originConditions,
				Status:                         status,
				WAFEnable:                      wafEnable,
				ACLDefaultAction:               aclDefault,
				ACLDefaultDenyStatus:           aclDenyStatus,
				ACLDefaultRedirectURL:          aclRedirectURL,
				ACLRules:                       aclRules,
				BlackIPs:                       parseIPList(effectiveSite.BlackIPRaw),
				WhiteIPs:                       parseIPList(effectiveSite.WhiteIPRaw),
				RegionBlock:                    regionBlock,
				CCRuleID:                       effectiveSite.CcDefaultRule,
				CCAutoSwitch:                   ccAutoSwitch,
				CustomCCRules:                  customCCRules,
				OriginProtocol:                 originProtocol,
				OriginHTTPPort:                 originHTTPPort,
				OriginHTTPSPort:                originHTTPSPort,
				OriginHostHeader:               originHostHeader,
				OriginSNI:                      originSNI,
				OriginVerifyTLS:                originVerifyTLS || advCfg.originCert,
				Cache:                          cacheCfg,
				HttpListen:                     effectiveSite.HttpListen,
				HttpsListen:                    httpsListen,
				HTTPSForce:                     httpsCfg.force,
				HTTPSRedirectPort:              httpsCfg.redirectPort,
				HTTPSHSTS:                      httpsCfg.hsts,
				HTTPSHTTP2:                     httpsCfg.http2,
				HTTPSOCSP:                      httpsCfg.ocsp,
				HTTPSHTTP3:                     httpsCfg.http3,
				HTTPSSSLProtocols:              httpsCfg.sslProtocols,
				HTTPSSSLCiphers:                httpsCfg.sslCiphers,
				HTTPSSSLPreferServerCiphers:    httpsCfg.sslPreferServerCiphers,
				ProxyConnectTimeout:            proxyTimeouts.connectTimeout,
				ProxyReadTimeout:               proxyTimeouts.readTimeout,
				ProxySendTimeout:               proxyTimeouts.sendTimeout,
				ProxyHTTPVersion:               advCfg.proxyHTTPVersion,
				OriginHTTPVersionPolicy:        advCfg.originHTTPVersionPolicy,
				OriginAutoDowngrade:            advCfg.originAutoDowngrade,
				OriginDowngradeThreshold:       advCfg.originDowngradeThreshold,
				OriginDowngradeWindowSeconds:   advCfg.originDowngradeWindowSeconds,
				OriginDowngradeCooldownSeconds: advCfg.originDowngradeCooldownSeconds,
				ProxySSLProtocols:              advCfg.proxySSLProtocols,
				EnableGzip:                     advCfg.gzip,
				GzipTypes:                      advCfg.gzipTypes,
				EnableWebsocket:                advCfg.websocket,
				EnableRange:                    advCfg.rangeEnabled,
				BodyLimit:                      advCfg.bodyLimit,
				LogRequestHeader:               advCfg.logRequestHeader,
				LogResponseHeader:              advCfg.logResponseHeader,
				LogRequestBody:                 advCfg.logRequestBody,
				LogRequestBodySizeLimit:        advCfg.logRequestBodySizeLimit,
				OriginCert:                     advCfg.originCert,
				RealtimeIdentify:               advCfg.realtimeIdentify,
				RealtimeSend:                   advCfg.realtimeSend,
				RealtimeReturn:                 advCfg.realtimeReturn,
				DefaultSite:                    advCfg.defaultSite,
				IPv6Enable:                     advCfg.ipv6Enable,
				LimitRate:                      limitRate,
				ConnLimit:                      connLimit,
				UpstreamKeepalive:              advCfg.keepalive,
				UpstreamKeepaliveConn:          advCfg.keepaliveConn,
				UpstreamKeepaliveTimeout:       advCfg.keepaliveTimeout,
				ErrorPageLang:                  extractErrorPageLang(effectiveSite.Settings),
			}
			if hasHTTPS {
				cert := findCertForSiteDomain(selectedCertID, normalizedDomain, certs)
				if cert != nil {
					domainConf.SSLCertData = cert.Cert
					// Decrypt key
					plainKey := cert.Key
					if dec, err := Crypto.Decrypt(cert.Key); err == nil {
						plainKey = dec
					}
					domainConf.SSLKeyData = plainKey
				} else if selectedCertID > 0 {
					// A selected certificate that does not cover this domain is
					// unsafe to serve. Keep the site out of HTTPS instead of
					// exposing the fallback certificate as if HTTPS succeeded.
					domainConf.HttpsListen = nil
					domainConf.HTTPSForce = false
					domainConf.HTTPSHSTS = false
					domainConf.HTTPSHTTP2 = false
					domainConf.HTTPSOCSP = false
					domainConf.HTTPSHTTP3 = false
				} else {
					// Keep cert path empty so agent can always fall back to its local fallback cert.
					// Avoid machine-specific absolute placeholder paths in payload.
					domainConf.SSLCertPath = ""
					domainConf.SSLKeyPath = ""
				}
			}
			payload.Domains = append(payload.Domains, domainConf)
		}
	}

	payload.Streams = buildStreamsForNode(node, groupIDs, l2TargetsByGroup, groupL2Config)

	ccRules, ccMatchers, ccFilters, err := loadAllCCData()
	if err == nil {
		payload.CCRules = ccRules
		payload.CCMatchers = ccMatchers
		payload.CCFilters = ccFilters
	}

	payload.IPUnblock = SnapshotIPUnblock()

	payload.Version = hashConfigVersion(payload)
	return payload, nil
}

func loadSitesForConfigGroups(siteDB *gorm.DB, groupIDs []int64) ([]models.Site, error) {
	groupIDs = uniqueInt64(groupIDs)
	if len(groupIDs) == 0 {
		return nil, nil
	}
	groupSet := int64Set(groupIDs)
	siteByID := map[int64]models.Site{}
	appendSites := func(rows []models.Site) {
		for _, site := range rows {
			if site.ID == 0 {
				continue
			}
			if _, ok := siteByID[site.ID]; ok {
				continue
			}
			siteByID[site.ID] = site
		}
	}

	var direct []models.Site
	if err := siteDB.Where("node_group_id IN ? OR (enable_backup_group = ? AND backup_node_group IN ?)", groupIDs, true, groupIDs).Find(&direct).Error; err != nil {
		return nil, err
	}
	appendSites(direct)

	packageIDs := loadConfigPackageIDsByGroups(groupIDs)
	if len(packageIDs) > 0 {
		var byPackage []models.Site
		if err := siteDB.Where("user_package IN ?", packageIDs).Find(&byPackage).Error; err != nil {
			return nil, err
		}
		appendSites(byPackage)
	}

	if len(siteByID) == 0 {
		return nil, nil
	}
	candidates := make([]models.Site, 0, len(siteByID))
	for _, site := range siteByID {
		candidates = append(candidates, site)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })

	packMap, err := loadUserPackageMap(candidates)
	if err != nil {
		return nil, err
	}
	planIDSet := map[int64]struct{}{}
	for _, pkg := range packMap {
		if pkg.PackageID != 0 {
			planIDSet[int64(pkg.PackageID)] = struct{}{}
		}
	}
	planIDs := make([]int64, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}
	planMap := loadPlanGroupMap(planIDs)

	sites := make([]models.Site, 0, len(candidates))
	for _, site := range candidates {
		pkg := packMap[site.UserPackageID]
		primary, backup, enableBackup := resolveSiteGroups(site, pkg, planMap[int64(pkg.PackageID)])
		if siteConfigGroupMatches(primary, backup, enableBackup, groupSet) {
			sites = append(sites, site)
		}
	}
	return sites, nil
}

func loadConfigPackageIDsByGroups(groupIDs []int64) []int64 {
	groupIDs = uniqueInt64(groupIDs)
	if len(groupIDs) == 0 {
		return nil
	}
	ids := make([]int64, 0)
	var direct []int64
	_ = db.DB.Model(&models.UserPackage{}).
		Where("node_group_id IN ? OR (enable_backup_group = ? AND backup_node_group IN ?)", groupIDs, true, groupIDs).
		Pluck("id", &direct).Error
	ids = append(ids, direct...)

	planIDs := loadPlanIDsByGroups(groupIDs)
	if len(planIDs) > 0 {
		var byPlan []int64
		_ = db.DB.Model(&models.UserPackage{}).
			Where("package IN ?", planIDs).
			Pluck("id", &byPlan).Error
		ids = append(ids, byPlan...)
	}
	return uniqueInt64(ids)
}

func siteConfigGroupMatches(primary int64, backup int64, enableBackup bool, groupSet map[int64]struct{}) bool {
	if len(groupSet) == 0 {
		return false
	}
	if primary != 0 {
		if _, ok := groupSet[primary]; ok {
			return true
		}
	}
	if enableBackup && backup != 0 {
		if _, ok := groupSet[backup]; ok {
			return true
		}
	}
	return false
}

func int64Set(items []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item != 0 {
			out[item] = struct{}{}
		}
	}
	return out
}

func hashConfigVersion(cfg *models.EdgeConfig) int64 {
	clone := *cfg
	clone.Version = 0
	b, err := json.Marshal(clone)
	if err != nil {
		return time.Now().Unix()
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return int64(h.Sum64())
}

func findNode(nodeID string) (*models.Node, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, ErrNodeIDMissing
	}

	var node models.Node
	if id, err := strconv.ParseInt(nodeID, 10, 64); err == nil {
		if err := db.DB.Where("id = ?", id).First(&node).Error; err == nil {
			return &node, nil
		}
	}

	if err := db.DB.Where("name = ? OR host = ? OR ip = ?", nodeID, nodeID, nodeID).First(&node).Error; err != nil {
		return nil, ErrNodeNotFound
	}
	return &node, nil
}

func loadGlobalConfig() *models.GlobalConfig {
	cfg := LoadGlobalConfigNormalized()
	if cfg == nil {
		cfg = &models.GlobalConfig{}
	}
	if len(cfg.ErrorPages) == 0 {
		cfg.ErrorPages = DefaultErrorPageDefinitions()
	}
	if len(cfg.GuardPages) == 0 {
		cfg.GuardPages = DefaultGuardPageDefinitions()
	}
	NormalizeGlobalConfigErrorPages(cfg)
	NormalizeGlobalConfigGuardPages(cfg)
	return cfg
}

func extractErrorPageLang(settings map[string]interface{}) string {
	if settings == nil {
		return ""
	}
	raw := strings.TrimSpace(parseString(settings["error_page_lang"]))
	if raw == "" {
		if adv := getMap(settings, "advanced"); adv != nil {
			raw = strings.TrimSpace(parseString(adv["error_page_lang"]))
		}
	}
	raw = strings.ToLower(raw)
	switch raw {
	case "", "inherit", "default":
		return ""
	case "browser":
		return "browser"
	default:
		return normalizeLocaleTag(raw)
	}
}

func GetGlobalDefaultConfig() *models.DefaultSiteConfig {
	if cfg := loadGlobalConfig(); cfg != nil {
		return &cfg.DefaultConfig
	}
	return nil
}

func buildACLForSite(site models.Site) (string, int, string, []models.EdgeACLRule) {
	if site.Settings == nil {
		return "", 0, "", nil
	}
	access, ok := site.Settings["access"].(map[string]interface{})
	if !ok {
		return "", 0, "", nil
	}
	aclID := parseACLID(access["acl"])
	if aclID == 0 {
		return "", 0, "", nil
	}
	var acl models.ACL
	if err := db.DB.Where("id = ? AND enable = ?", aclID, true).First(&acl).Error; err != nil {
		return "", 0, "", nil
	}
	defaultAction := normalizeACLAction(acl.DefaultAction)
	if defaultAction == "" {
		defaultAction = "allow"
	}
	denyStatus, redirectURL := parseACLDefaultDenyMeta(acl.Data)
	rules := parseACLRules(acl.Data)
	return defaultAction, denyStatus, redirectURL, rules
}

type aclCondition struct {
	Item     string `json:"item"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type aclRule struct {
	Conditions  []aclCondition `json:"conditions"`
	Action      string         `json:"action"`
	DenyStatus  int            `json:"deny_status"`
	RedirectURL string         `json:"redirect_url"`
}

type aclDataEnvelope struct {
	Rules              []aclRule `json:"rules"`
	DefaultDenyStatus  int       `json:"default_deny_status"`
	DefaultRedirectURL string    `json:"default_redirect_url"`
}

type legacyACLAPIRule struct {
	ACLAction  string                            `json:"acl_action"`
	ACLMatcher map[string]map[string]interface{} `json:"acl_matcher"`
	Action     string                            `json:"action"`
}

func normalizeACLAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "reject", "deny":
		return "deny"
	case "allow", "permit":
		return "allow"
	default:
		return action
	}
}

func normalizeACLConditionItem(item string) string {
	item = strings.ToLower(strings.TrimSpace(item))
	switch item {
	case "host", "req_host":
		return "domain"
	case "user_agent", "ua":
		return "user_agent"
	case "req_uri", "request_uri":
		return "uri"
	case "uri_path", "request_path":
		return "uri_path"
	case "req_method", "method":
		return "method"
	case "country_iso_code", "country_code":
		return "country"
	case "as_number", "asn":
		return "as_number"
	case "accept_language", "header_accept_language":
		return "accept_language"
	default:
		return item
	}
}

func normalizeACLOperator(op string) string {
	op = strings.TrimSpace(op)
	switch op {
	case "=":
		return "eq"
	case "!=":
		return "neq"
	case "contain":
		return "contains"
	case "!contain":
		return "not_contains"
	case "AC":
		return "ip_range"
	case "!AC":
		return "not_ip_range"
	default:
		return op
	}
}

func normalizeACLCondition(cond aclCondition) models.EdgeACLCondition {
	return models.EdgeACLCondition{
		Item:     normalizeACLConditionItem(cond.Item),
		Operator: normalizeACLOperator(cond.Operator),
		Value:    strings.TrimSpace(cond.Value),
	}
}

func parseACLDefaultDenyMeta(raw string) (int, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ""
	}
	var envelope aclDataEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil {
		return envelope.DefaultDenyStatus, strings.TrimSpace(envelope.DefaultRedirectURL)
	}
	return 0, ""
}

func parseACLRules(raw string) []models.EdgeACLRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var envelope aclDataEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && len(envelope.Rules) > 0 {
		return convertACLRules(envelope.Rules)
	}

	var legacyAPI []legacyACLAPIRule
	if err := json.Unmarshal([]byte(raw), &legacyAPI); err == nil && len(legacyAPI) > 0 && legacyAPI[0].ACLMatcher != nil {
		return convertLegacyACLAPIRules(legacyAPI)
	}

	var items []models.EdgeACLRule
	if err := json.Unmarshal([]byte(raw), &items); err == nil && len(items) > 0 {
		normalized := normalizeEdgeACLRules(items)
		if len(normalized) > 0 {
			return normalized
		}
	}

	var rules []aclRule
	if err := json.Unmarshal([]byte(raw), &rules); err == nil && len(rules) > 0 {
		converted := convertACLRules(rules)
		if len(converted) > 0 {
			return converted
		}
	}

	var generic []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil
	}
	out := make([]models.EdgeACLRule, 0, len(generic))
	for _, item := range generic {
		if entry := parseACLRuleMap(item); entry != nil {
			out = append(out, *entry)
		}
	}
	return out
}

func convertACLRules(rules []aclRule) []models.EdgeACLRule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]models.EdgeACLRule, 0, len(rules))
	for _, rule := range rules {
		entry := models.EdgeACLRule{
			Action:      normalizeACLAction(rule.Action),
			DenyStatus:  rule.DenyStatus,
			RedirectURL: strings.TrimSpace(rule.RedirectURL),
		}
		if entry.Action == "" {
			entry.Action = "allow"
		}
		for _, cond := range rule.Conditions {
			normalized := normalizeACLCondition(cond)
			if normalized.Item == "" {
				continue
			}
			entry.Conditions = append(entry.Conditions, normalized)
		}
		if len(entry.Conditions) == 0 {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func convertLegacyACLAPIRules(rules []legacyACLAPIRule) []models.EdgeACLRule {
	out := make([]models.EdgeACLRule, 0, len(rules))
	for _, rule := range rules {
		action := normalizeACLAction(rule.ACLAction)
		if action == "" {
			action = normalizeACLAction(rule.Action)
		}
		if action == "" {
			action = "allow"
		}
		entry := models.EdgeACLRule{Action: action}
		for item, matcher := range rule.ACLMatcher {
			if matcher == nil {
				continue
			}
			op := ""
			if v, ok := matcher["operator"].(string); ok {
				op = v
			}
			value := ""
			switch v := matcher["value"].(type) {
			case string:
				value = v
			case []interface{}:
				parts := make([]string, 0, len(v))
				for _, part := range v {
					if s, ok := part.(string); ok {
						parts = append(parts, s)
					}
				}
				value = strings.Join(parts, "\n")
			}
			entry.Conditions = append(entry.Conditions, models.EdgeACLCondition{
				Item:     normalizeACLConditionItem(item),
				Operator: normalizeACLOperator(op),
				Value:    value,
			})
		}
		sort.SliceStable(entry.Conditions, func(i, j int) bool {
			if entry.Conditions[i].Item == entry.Conditions[j].Item {
				return entry.Conditions[i].Operator < entry.Conditions[j].Operator
			}
			return entry.Conditions[i].Item < entry.Conditions[j].Item
		})
		if len(entry.Conditions) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func parseACLRuleMap(item map[string]interface{}) *models.EdgeACLRule {
	if item == nil {
		return nil
	}
	if _, ok := item["acl_matcher"]; ok {
		b, _ := json.Marshal(item)
		var legacy legacyACLAPIRule
		if json.Unmarshal(b, &legacy) == nil {
			converted := convertLegacyACLAPIRules([]legacyACLAPIRule{legacy})
			if len(converted) > 0 {
				return &converted[0]
			}
		}
	}
	entry := models.EdgeACLRule{
		Action: normalizeACLAction(parseString(item["action"])),
	}
	if entry.Action == "" {
		entry.Action = "allow"
	}
	if v, ok := item["deny_status"].(float64); ok {
		entry.DenyStatus = int(v)
	}
	entry.RedirectURL = parseString(item["redirect_url"])
	if ip := parseString(item["ip"]); ip != "" {
		entry.IP = ip
		entry.Conditions = []models.EdgeACLCondition{{
			Item: "ip", Operator: "eq", Value: ip,
		}}
		return &entry
	}
	if rawConds, ok := item["conditions"].([]interface{}); ok {
		for _, raw := range rawConds {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			entry.Conditions = append(entry.Conditions, models.EdgeACLCondition{
				Item:     normalizeACLConditionItem(parseString(m["item"])),
				Operator: normalizeACLOperator(parseString(m["operator"])),
				Value:    parseString(m["value"]),
			})
		}
	}
	if len(entry.Conditions) == 0 {
		return nil
	}
	return &entry
}

func normalizeEdgeACLRules(items []models.EdgeACLRule) []models.EdgeACLRule {
	out := make([]models.EdgeACLRule, 0, len(items))
	for _, item := range items {
		entry := item
		entry.Action = normalizeACLAction(entry.Action)
		if entry.Action == "" {
			entry.Action = "allow"
		}
		if entry.IP != "" && len(entry.Conditions) == 0 {
			entry.Conditions = []models.EdgeACLCondition{{
				Item: "ip", Operator: "eq", Value: entry.IP,
			}}
		}
		for i := range entry.Conditions {
			entry.Conditions[i] = models.EdgeACLCondition{
				Item:     normalizeACLConditionItem(entry.Conditions[i].Item),
				Operator: normalizeACLOperator(entry.Conditions[i].Operator),
				Value:    strings.TrimSpace(entry.Conditions[i].Value),
			}
		}
		if len(entry.Conditions) == 0 {
			continue
		}
		out = append(out, entry)
	}
	return out
}

var ccAutoSwitchRuleAliases = map[string]int64{
	"close":   10002,
	"lenient": 6,
	"normal":  5,
	"strict":  7,
	"js":      1,
	"captcha": 4,
}

func extractCCAutoSwitch(settings map[string]interface{}) *models.EdgeCCAutoSwitch {
	if settings == nil {
		return nil
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw := security["auto_switch"]
	if raw == nil {
		return nil
	}
	var payload map[string]interface{}
	switch v := raw.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if err := json.Unmarshal([]byte(v), &payload); err != nil {
			return nil
		}
	case map[string]interface{}:
		payload = v
	default:
		return nil
	}
	enable := parseBool(payload["enable"], false)
	if !enable {
		return nil
	}
	qps := int(parseIntValue(payload["qps"], 0))
	if qps <= 0 {
		qps = int(parseIntValue(payload["QPS"], 200))
	}
	ruleID := resolveCCAutoSwitchRuleID(payload["rule"])
	if ruleID <= 0 {
		return nil
	}
	return &models.EdgeCCAutoSwitch{
		Enable: true,
		QPS:    qps,
		RuleID: ruleID,
	}
}

func resolveCCAutoSwitchRuleID(raw interface{}) int64 {
	if raw == nil {
		return 0
	}
	if id := parseACLID(raw); id > 0 {
		return id
	}
	name := strings.ToLower(strings.TrimSpace(parseString(raw)))
	if name == "" {
		return 0
	}
	if id, ok := ccAutoSwitchRuleAliases[name]; ok {
		return id
	}
	var rule models.CCRule
	if err := db.DB.Where("enable = ? AND name LIKE ?", true, "%"+name+"%").Order("id asc").First(&rule).Error; err == nil {
		return rule.ID
	}
	return 0
}

func parseACLID(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return id
		}
	}
	return 0
}

func extractRegionBlock(site models.Site) []string {
	if site.Settings != nil {
		if access, ok := site.Settings["access"].(map[string]interface{}); ok {
			if rb, ok := access["region_block"]; ok {
				if isRegionBlockDisabled(rb) {
					return nil
				}
				if list := parseRegionBlockConfig(rb); len(list) > 0 {
					return list
				}
			}
		}
		if security, ok := site.Settings["security"].(map[string]interface{}); ok {
			if v, ok := security["region_block"]; ok {
				if list := parseRegionList(v); len(list) > 0 {
					return list
				}
			}
			if v, ok := security["region_custom"]; ok {
				if list := parseRegionList(v); len(list) > 0 {
					return list
				}
			}
		}
	}
	if list := parseRegionList(site.BlockRegionRaw); len(list) > 0 {
		return list
	}
	return nil
}

func extractHotlinkConfig(settings map[string]interface{}) *models.EdgeHotlinkConfig {
	if settings == nil {
		return nil
	}
	access, ok := settings["access"].(map[string]interface{})
	if !ok {
		return nil
	}
	hotlink, ok := access["hotlink"].(map[string]interface{})
	if !ok {
		return nil
	}
	enable := parseBoolValue(hotlink["enable"], false)
	if !enable {
		return nil
	}
	cfg := &models.EdgeHotlinkConfig{
		Enable:     enable,
		Scope:      parseString(hotlink["scope"]),
		Value:      parseString(hotlink["value"]),
		AllowEmpty: parseBoolValue(hotlink["allowEmpty"], true),
		Domains:    parseStringList(hotlink["domains"]),
	}
	return cfg
}

func extractCorsConfig(settings map[string]interface{}) *models.EdgeCorsConfig {
	if settings == nil {
		return nil
	}
	access, ok := settings["access"].(map[string]interface{})
	if !ok {
		return nil
	}
	cors, ok := access["cors"].(map[string]interface{})
	if !ok {
		return nil
	}
	enable := parseBoolValue(cors["enable"], false)
	if !enable {
		return nil
	}
	return &models.EdgeCorsConfig{
		Enable:           enable,
		AllowOrigin:      parseString(cors["allowOrigin"]),
		AllowMethods:     parseString(cors["allowMethods"]),
		AllowHeaders:     parseString(cors["allowHeaders"]),
		ExposeHeaders:    parseString(cors["exposeHeaders"]),
		AllowCredentials: parseBoolValue(cors["allowCredentials"], false),
		MaxAge:           parseString(cors["maxAge"]),
	}
}

func extractCookieConfig(settings map[string]interface{}) *models.EdgeCookieConfig {
	if settings == nil {
		return nil
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return nil
	}
	cookie, ok := security["cookie"].(map[string]interface{})
	if !ok {
		return nil
	}
	enable := parseBoolValue(cookie["enable"], false)
	domain := parseString(cookie["domain"])
	if !enable || domain == "" {
		return nil
	}
	return &models.EdgeCookieConfig{Enable: enable, Domain: domain}
}

func extractWAFEnable(settings map[string]interface{}) *bool {
	if settings == nil {
		return nil
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		return nil
	}
	raw, ok := security["waf_enable"]
	if !ok || raw == nil {
		return nil
	}
	val := parseBoolValue(raw, false)
	return &val
}

func extractBlockTransparentProxy(settings map[string]interface{}) bool {
	if settings == nil {
		return false
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return false
	}
	return parseBoolValue(security["block_transparent_proxy"], false)
}

func extractCrawlerAction(settings map[string]interface{}) string {
	if settings == nil {
		return ""
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return ""
	}
	return parseString(security["crawlers_action"])
}

func extractGuardTTLs(settings map[string]interface{}) (int, int) {
	if settings == nil {
		return 0, 0
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	passTTL := parseIntValue(security["ip_white_timeout"], 0)
	blockTTL := parseIntValue(security["ip_black_timeout"], 0)
	if passTTL <= 0 {
		passTTL = 21600
	}
	if blockTTL <= 0 {
		blockTTL = 3600
	}
	return passTTL, blockTTL
}

func extractURLRedirects(settings map[string]interface{}) []map[string]interface{} {
	if settings == nil {
		return nil
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw := advanced["url_redirects"]
	if raw == nil {
		return nil
	}
	switch list := raw.(type) {
	case []map[string]interface{}:
		if len(list) == 0 {
			return nil
		}
		return list
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		if b, err := json.Marshal(raw); err == nil {
			var parsed []map[string]interface{}
			if json.Unmarshal(b, &parsed) == nil && len(parsed) > 0 {
				return parsed
			}
		}
	}
	return nil
}

func extractURLRewrites(settings map[string]interface{}) []map[string]interface{} {
	if settings == nil {
		return nil
	}
	if raw, ok := settings["url_rewrites"]; ok {
		if list := normalizeURLRewritesRaw(raw); len(list) > 0 {
			return list
		}
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if !ok {
		return nil
	}
	if raw, ok := advanced["url_rewrites"]; ok {
		if list := normalizeURLRewritesRaw(raw); len(list) > 0 {
			return list
		}
	}
	return nil
}

func extractOriginConditions(settings map[string]interface{}) []map[string]interface{} {
	if settings == nil {
		return nil
	}
	origin, ok := settings["origin"].(map[string]interface{})
	if !ok || origin == nil {
		return withSearchEngineOriginCondition(settings, nil)
	}
	raw := origin["conditions"]
	if raw == nil {
		return withSearchEngineOriginCondition(settings, nil)
	}
	switch list := raw.(type) {
	case []map[string]interface{}:
		if len(list) == 0 {
			return withSearchEngineOriginCondition(settings, nil)
		}
		return withSearchEngineOriginCondition(settings, list)
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		if len(out) == 0 {
			return withSearchEngineOriginCondition(settings, nil)
		}
		return withSearchEngineOriginCondition(settings, out)
	default:
		if b, err := json.Marshal(raw); err == nil {
			var parsed []map[string]interface{}
			if json.Unmarshal(b, &parsed) == nil && len(parsed) > 0 {
				return withSearchEngineOriginCondition(settings, parsed)
			}
		}
	}
	return withSearchEngineOriginCondition(settings, nil)
}

var searchEngineCrawlerTokens = []string{
	"baiduspider",
	"googlebot",
	"bingbot",
	"yandex",
	"sogou",
	"360spider",
	"bytespider",
	"duckduckbot",
	"slurp",
	"facebot",
	"ia_archiver",
	"semrushbot",
}

func withSearchEngineOriginCondition(settings map[string]interface{}, conditions []map[string]interface{}) []map[string]interface{} {
	cond := buildSearchEngineOriginCondition(settings)
	if cond == nil {
		return conditions
	}
	if hasOriginCondition(conditions, cond) {
		return conditions
	}
	if conditions == nil {
		return []map[string]interface{}{cond}
	}
	return append([]map[string]interface{}{cond}, conditions...)
}

func buildSearchEngineOriginCondition(settings map[string]interface{}) map[string]interface{} {
	return buildSearchEngineOriginConditionWithAllowlist(settings, buildSpiderIPRangeValue())
}

func buildSearchEngineOriginConditionWithAllowlist(settings map[string]interface{}, allowlistValue string) map[string]interface{} {
	if settings == nil {
		return nil
	}
	if !parseBoolValue(settings["search_engine_origin"], false) {
		return nil
	}
	originIP := strings.TrimSpace(parseString(settings["search_engine_origin_ip"]))
	if originIP == "" {
		return nil
	}
	if strings.TrimSpace(allowlistValue) != "" {
		return map[string]interface{}{
			"item":     "client_ip",
			"operator": "ip_range",
			"value":    allowlistValue,
			"origin":   originIP,
		}
	}
	return nil
}

func buildSpiderIPRangeValue() string {
	allowlist := loadSpiderAllowlist()
	if allowlist == nil {
		return ""
	}
	entries := make(map[string]struct{})
	for ip := range allowlist.exact {
		entries[ip] = struct{}{}
	}
	for _, cidr := range allowlist.cidrs {
		if cidr == nil {
			continue
		}
		entries[cidr.String()] = struct{}{}
	}
	for _, prefix := range allowlist.prefixes {
		normalized := strings.TrimSpace(prefix)
		if normalized == "" {
			continue
		}
		normalized = strings.TrimSuffix(normalized, ".")
		if strings.Count(normalized, ".") != 2 {
			continue
		}
		entries[normalized+".0/24"] = struct{}{}
	}
	if len(entries) == 0 {
		return ""
	}
	values := make([]string, 0, len(entries))
	for entry := range entries {
		values = append(values, entry)
	}
	sort.Strings(values)
	return strings.Join(values, "|")
}

func hasOriginCondition(conditions []map[string]interface{}, cond map[string]interface{}) bool {
	if len(conditions) == 0 || cond == nil {
		return false
	}
	condItem := strings.ToLower(parseString(cond["item"]))
	condHeader := strings.ToLower(parseString(cond["header"]))
	condOrigin := strings.TrimSpace(parseString(cond["origin"]))
	condOperator := strings.ToLower(parseString(cond["operator"]))
	for _, item := range conditions {
		if strings.ToLower(parseString(item["item"])) != condItem {
			continue
		}
		if strings.ToLower(parseString(item["header"])) != condHeader {
			continue
		}
		if strings.TrimSpace(parseString(item["origin"])) != condOrigin {
			continue
		}
		if condOperator != "" && strings.ToLower(parseString(item["operator"])) != condOperator {
			continue
		}
		return true
	}
	return false
}

func isRegionBlockDisabled(value interface{}) bool {
	if value == nil {
		return false
	}
	if m, ok := value.(map[string]interface{}); ok {
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(m["mode"])))
		return mode == "disabled" || mode == "off" || mode == "none"
	}
	return false
}

func parseRegionBlockConfig(value interface{}) []string {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]interface{}); ok {
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(m["mode"])))
		if mode == "disabled" || mode == "off" || mode == "none" || mode == "" {
			return nil
		}
		if countries, ok := m["countries"]; ok {
			return parseRegionList(countries)
		}
	}
	return parseRegionList(value)
}

func parseRegionList(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return normalizeRegionList(v)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return normalizeRegionList(out)
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return nil
		}
		var parsed []string
		if json.Unmarshal([]byte(raw), &parsed) == nil && len(parsed) > 0 {
			return normalizeRegionList(parsed)
		}
		fields := strings.FieldsFunc(raw, func(r rune) bool {
			return r == '\n' || r == '\r' || r == '\t' || r == ',' || r == ';' || r == ' '
		})
		return normalizeRegionList(fields)
	default:
		return nil
	}
}

func parseStringList(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return splitAndNormalize(v)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return splitAndNormalize(out)
	case string:
		fields := splitListFields(v)
		return splitAndNormalize(fields)
	default:
		if value == nil {
			return nil
		}
		fields := splitListFields(fmt.Sprint(value))
		return splitAndNormalize(fields)
	}
}

func splitListFields(raw string) []string {
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.ReplaceAll(raw, "\r", " ")
	raw = strings.ReplaceAll(raw, "\t", " ")
	raw = strings.ReplaceAll(raw, ",", " ")
	raw = strings.ReplaceAll(raw, ";", " ")
	return strings.Fields(raw)
}

func splitAndNormalize(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeRegionList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		code := strings.ToUpper(strings.TrimSpace(item))
		if code == "" {
			continue
		}
		if idx := strings.Index(code, "-"); idx > 0 {
			code = code[:idx]
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildUpstreamTargets(site models.Site, originProtocol, originHTTPPort, originHTTPSPort string) []models.EdgeUpstreamTarget {
	targets := buildUpstreamTargetsFromSettings(site.Settings, originProtocol, originHTTPPort, originHTTPSPort)
	if len(targets) > 0 {
		return targets
	}
	out := make([]models.EdgeUpstreamTarget, 0, len(site.Backends))
	for _, backend := range site.Backends {
		if strings.TrimSpace(backend) == "" {
			continue
		}
		addr := normalizeOriginAddr(backend, originProtocol, originHTTPPort, originHTTPSPort)
		out = append(out, models.EdgeUpstreamTarget{
			Addr:   addr,
			Weight: 10,
		})
	}
	return out
}

func buildUpstreamTargetsFromSettings(settings map[string]interface{}, originProtocol, originHTTPPort, originHTTPSPort string) []models.EdgeUpstreamTarget {
	if settings == nil {
		return nil
	}
	origin, ok := settings["origin"].(map[string]interface{})
	if !ok || origin == nil {
		return nil
	}
	rawList := origin["list"]
	if rawList == nil {
		return nil
	}
	list, ok := rawList.([]interface{})
	if !ok {
		if cast, ok := rawList.([]map[string]interface{}); ok {
			list = make([]interface{}, 0, len(cast))
			for _, item := range cast {
				list = append(list, item)
			}
		}
	}
	if len(list) == 0 {
		return nil
	}
	targets := make([]models.EdgeUpstreamTarget, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		addr := parseString(entry["address"])
		if addr == "" {
			continue
		}
		addr = normalizeOriginAddr(addr, originProtocol, originHTTPPort, originHTTPSPort)
		if !parseBoolValue(entry["enable"], true) {
			continue
		}
		weight := parseIntValue(entry["weight"], 10)
		if weight <= 0 {
			weight = 10
		}
		targets = append(targets, models.EdgeUpstreamTarget{
			Addr:   addr,
			Weight: weight,
		})
	}
	if len(targets) == 0 {
		return nil
	}
	return targets
}

func normalizeOriginAddr(addr, originProtocol, originHTTPPort, originHTTPSPort string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return addr
	}
	if strings.Contains(addr, ":") {
		return addr
	}
	protocol := strings.ToLower(strings.TrimSpace(originProtocol))
	switch protocol {
	case "http":
		if originHTTPPort != "" {
			return addr + ":" + originHTTPPort
		}
	case "https":
		if originHTTPSPort != "" {
			return addr + ":" + originHTTPSPort
		}
	}
	return addr
}

func mapBalancePolicy(way string) string {
	switch way {
	case "ip_hash":
		return "ip_hash"
	case "random":
		return "random"
	default:
		return "round_robin"
	}
}

type nodeGroupCount struct {
	NodeGroupID int64 `gorm:"column:node_group_id"`
	Count       int64 `gorm:"column:cnt"`
}

func loadNodeGroupCounts(groupIDs []int64) map[int64]int64 {
	counts := map[int64]int64{}
	if len(groupIDs) == 0 {
		return counts
	}
	var rows []nodeGroupCount
	_ = db.DB.Model(&models.Line{}).
		Select("node_group_id, count(distinct node_id) as cnt").
		Where("node_group_id IN ?", groupIDs).
		Group("node_group_id").
		Scan(&rows).Error
	for _, row := range rows {
		counts[row.NodeGroupID] = row.Count
	}
	return counts
}

func loadUserPackageMap(sites []models.Site) (map[int64]models.UserPackage, error) {
	ids := make([]int64, 0, len(sites))
	seen := map[int64]struct{}{}
	for _, site := range sites {
		if site.UserPackageID == 0 {
			continue
		}
		if _, ok := seen[site.UserPackageID]; ok {
			continue
		}
		seen[site.UserPackageID] = struct{}{}
		ids = append(ids, site.UserPackageID)
	}
	result := map[int64]models.UserPackage{}
	if len(ids) == 0 {
		return result, nil
	}
	var packages []models.UserPackage
	if err := db.DB.Where("id IN ?", ids).Find(&packages).Error; err != nil {
		return nil, err
	}
	for _, pkg := range packages {
		result[pkg.ID] = pkg
	}
	return result, nil
}

func loadUserPackageMapForForwards(forwards []models.Forward) (map[int64]models.UserPackage, error) {
	ids := make([]int64, 0, len(forwards))
	seen := map[int64]struct{}{}
	for _, forward := range forwards {
		if forward.UserPackageID == 0 {
			continue
		}
		if _, ok := seen[forward.UserPackageID]; ok {
			continue
		}
		seen[forward.UserPackageID] = struct{}{}
		ids = append(ids, forward.UserPackageID)
	}
	result := map[int64]models.UserPackage{}
	if len(ids) == 0 {
		return result, nil
	}
	var packages []models.UserPackage
	if err := db.DB.Where("id IN ?", ids).Find(&packages).Error; err != nil {
		return nil, err
	}
	for _, pkg := range packages {
		result[pkg.ID] = pkg
	}
	return result, nil
}

func buildDomainCountByUserGroup(sites []models.Site) map[int64]map[int64]int {
	result := map[int64]map[int64]int{}
	for _, site := range sites {
		if !site.Enable {
			continue
		}
		if site.NodeGroupID == 0 || site.UserID == 0 {
			continue
		}
		groupMap, ok := result[site.NodeGroupID]
		if !ok {
			groupMap = map[int64]int{}
			result[site.NodeGroupID] = groupMap
		}
		count := len(site.Domains)
		if count == 0 && site.DomainRaw != "" {
			count = 1
		}
		groupMap[site.UserID] += count
	}
	return result
}

func calcDomainConnLimit(userID int64, nodeGroupID int64, connLimit int32, domainCountByUserGroup map[int64]map[int64]int, nodeGroupCounts map[int64]int64) int {
	if connLimit <= 0 {
		return 0
	}
	nodeCount := nodeGroupCounts[nodeGroupID]
	if nodeCount <= 0 {
		return 0
	}
	groupMap := domainCountByUserGroup[nodeGroupID]
	if groupMap == nil {
		return 0
	}
	domainCount := groupMap[userID]
	if domainCount <= 0 {
		return 0
	}
	perNode := float64(connLimit) / float64(nodeCount)
	perDomain := perNode / float64(domainCount)
	if perDomain < 1 {
		return 1
	}
	return int(perDomain)
}

func parseBandwidthMbps(raw string) float64 {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "0" || value == "unlimited" || value == "unlimit" || value == i18n.T("value.unlimited") {
		return 0
	}
	multiplier := 1.0
	switch {
	case strings.HasSuffix(value, "g"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "g")
	case strings.HasSuffix(value, "m"):
		value = strings.TrimSuffix(value, "m")
	case strings.HasSuffix(value, "k"):
		multiplier = 1.0 / 1024
		value = strings.TrimSuffix(value, "k")
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed * multiplier
}

func calcDomainLimitRate(bandwidth string) int64 {
	return parsePackageBandwidthToLimitRate(bandwidth)
}

func parsePackageBandwidthToLimitRate(raw string) int64 {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "0" || value == "unlimited" || value == "unlimit" || value == i18n.T("value.unlimited") {
		return 0
	}
	re := regexp.MustCompile(`^([+-]?[0-9]+(?:\.[0-9]+)?)\s*([kmg]?)$`)
	m := re.FindStringSubmatch(value)
	if len(m) < 2 {
		return 0
	}
	num, err := strconv.ParseFloat(m[1], 64)
	if err != nil || num <= 0 {
		return 0
	}
	unit := ""
	if len(m) >= 3 {
		unit = strings.TrimSpace(m[2])
	}
	mbps := num
	switch unit {
	case "g":
		mbps = num * 1024
	case "k":
		mbps = num / 1024
	case "m", "":
		// keep as Mbps
	default:
		return 0
	}
	if mbps <= 0 {
		return 0
	}
	return mbpsToLimitRate(mbps)
}

func mbpsToLimitRate(mbps float64) int64 {
	if mbps <= 0 {
		return 0
	}
	return int64(mbps * 1024 * 1024 / 8)
}

func buildHeaderMap(site models.Site) map[string]string {
	settings := site.Settings
	if settings == nil {
		return nil
	}
	result := make(map[string]string)
	headers, ok := settings["headers"].(map[string]interface{})
	if ok {
		for k, v := range headers {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				name := sanitizeHeaderName(k)
				value := sanitizeHeaderValue(s)
				if name != "" && value != "" {
					result[name] = value
				}
			}
		}
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if ok {
		if list, ok := advanced["origin_headers"].([]interface{}); ok {
			for _, item := range list {
				if m, ok := item.(map[string]interface{}); ok {
					name := parseString(m["name"])
					value := parseString(m["value"])
					name = sanitizeHeaderName(name)
					value = sanitizeHeaderValue(value)
					if name != "" && value != "" {
						result[name] = value
					}
				}
			}
		}
	}
	if _, ok := result["Host"]; !ok {
		if originHost := resolveOriginHost(site); originHost != "" {
			if value := sanitizeHeaderValue(originHost); value != "" {
				result["Host"] = value
			}
		}
	}
	if host, ok := result["Host"]; ok {
		normalized := normalizeHostHeaderMapValue(host, site)
		switch {
		case normalized == "":
			delete(result, "Host")
		case strings.HasPrefix(normalized, "$"):
			result["Host"] = normalized
		default:
			if value := sanitizeHeaderValue(normalized); value != "" {
				result["Host"] = value
			} else {
				delete(result, "Host")
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolveOriginHost(site models.Site) string {
	settings := site.Settings
	if settings == nil {
		return ""
	}
	if backsource, ok := settings["backsource"].(map[string]interface{}); ok {
		mode := strings.ToLower(parseString(backsource["host_mode"]))
		switch mode {
		case "follow", "":
			return ""
		case "domain":
			return firstDomain(site.Domains)
		case "custom":
			return parseString(backsource["host_custom"])
		default:
			if mode != "" {
				return mode
			}
		}
	}
	rawHost := parseString(settings["origin_host"])
	if rawHost == "" {
		if originCfg, ok := settings["origin"].(map[string]interface{}); ok {
			rawHost = parseString(originCfg["host"])
		}
	}
	rawHost = strings.ToLower(strings.TrimSpace(rawHost))
	switch rawHost {
	case "", "follow":
		return ""
	case "domain":
		return firstDomain(site.Domains)
	default:
		return rawHost
	}
}

func extractOriginTLSConfig(site models.Site) (string, string, bool) {
	hostHeader := resolveOriginHost(site)
	sni := ""
	verifyTLS := false
	if site.Settings != nil {
		if originCfg := getMap(site.Settings, "origin"); originCfg != nil {
			if v := normalizeOriginHostHeaderValue(parseString(originCfg["host_header"]), site); v != "" {
				hostHeader = v
			}
			sni = firstNonEmptyString(originCfg["sni"], originCfg["tls_server_name"])
			verifyTLS = parseBoolValue(originCfg["verify_tls"], false)
		}
	}
	hostHeader = sanitizeHeaderValue(hostHeader)
	sni = sanitizeNginxToken(sni)
	if sni == "" {
		sni = sanitizeNginxToken(hostHeader)
	}
	return hostHeader, sni, verifyTLS
}

func normalizeOriginHostHeaderValue(raw string, site models.Site) string {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "", "follow":
		return ""
	case "domain":
		return firstDomain(site.Domains)
	default:
		return raw
	}
}

func firstNonEmptyString(values ...interface{}) string {
	for _, value := range values {
		if s := parseString(value); s != "" {
			return s
		}
	}
	return ""
}

func firstDomain(domains []string) string {
	for _, domain := range domains {
		trimmed := strings.TrimSpace(domain)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildResponseHeaderMap(settings map[string]interface{}) map[string]string {
	if settings == nil {
		return nil
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if !ok {
		return nil
	}
	list, ok := advanced["cdn_headers"].([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			name := parseString(m["name"])
			value := parseString(m["value"])
			name = sanitizeHeaderName(name)
			value = sanitizeHeaderValue(value)
			if name != "" && value != "" {
				if isOriginLeakResponseHeaderName(name) {
					continue
				}
				result[name] = value
			}
		}
	}
	return result
}

func isOriginLeakResponseHeaderName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "x-origin-ip",
		"x-origin-addr",
		"x-origin-server",
		"x-backend-ip",
		"x-backend-addr",
		"x-backend-server",
		"x-server-ip",
		"x-upstream-addr",
		"x-upstream-server",
		"x-real-ip",
		"x-forwarded-for",
		"via":
		return true
	default:
		return false
	}
}

func normalizeHostHeaderMapValue(value string, site models.Site) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "":
		return ""
	case "follow":
		return "$host"
	case "domain":
		return firstDomain(site.Domains)
	default:
		return value
	}
}

func sanitizeHeaderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return ""
	}
	return name
}

func sanitizeHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, "\r\n;") {
		return ""
	}
	return value
}

func sanitizeNginxToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n;") {
		return ""
	}
	if strings.IndexFunc(value, unicode.IsSpace) != -1 {
		return ""
	}
	return value
}

func sanitizeNginxValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n;") {
		return ""
	}
	return value
}

func sanitizeProxyHTTPVersion(value string) string {
	value = sanitizeNginxToken(value)
	switch value {
	case "1.0", "1.1":
		return value
	default:
		return ""
	}
}

func sanitizeOriginHTTPVersionPolicy(value string) string {
	value = strings.ToLower(sanitizeNginxToken(value))
	switch value {
	case "auto", "http11", "compat":
		return value
	case "1.1", "http/1.1", "http11_keepalive":
		return "http11"
	case "1.0", "http/1.0", "http10":
		return "compat"
	default:
		return ""
	}
}

func findCertForSiteDomain(certID int64, domain string, certs []models.Cert) *models.Cert {
	if certID > 0 {
		for _, cert := range certs {
			if int64(cert.ID) == certID && CertificateCoversDomain(cert.Cert, domain).OK {
				return &cert
			}
		}
		// A selected certificate is authoritative. If it does not cover the
		// domain, do not silently fall back to another certificate.
		return nil
	}
	return findCertForDomain(domain, certs)
}

func findCertForDomain(domain string, certs []models.Cert) *models.Cert {
	domain = normalizeDomainHostForEdge(domain)
	if domain == "" {
		return nil
	}
	for _, cert := range certs {
		if strings.TrimSpace(cert.Cert) != "" {
			if CertificateCoversDomain(cert.Cert, domain).OK {
				return &cert
			}
			continue
		}
		// Handle multi-domain certs (comma separated)
		domains := strings.Split(cert.Domain, ",")
		for _, d := range domains {
			candidate := normalizeDomainHostForEdge(d)
			if candidate == "" {
				continue
			}
			if candidate == domain {
				return &cert
			}
			if strings.HasPrefix(candidate, "*.") {
				suffix := strings.TrimPrefix(candidate, "*")
				if suffix != "" && strings.HasSuffix(domain, suffix) {
					return &cert
				}
			}
			if strings.HasPrefix(candidate, ".") && strings.HasSuffix(domain, candidate) {
				return &cert
			}
		}
	}
	return nil
}

func normalizeDomainHostForEdge(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return strings.TrimSpace(host)
}

func parseIPList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed []string
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return normalizeIPList(parsed)
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r == ',' || r == ';' || r == ' '
	})
	return normalizeIPList(fields)
}

func normalizeIPList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		ip := strings.TrimSpace(item)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type httpsConfig struct {
	force                  bool
	redirectPort           string
	hsts                   bool
	http2                  bool
	http3                  bool
	ocsp                   bool
	sslProtocols           string
	sslCiphers             string
	sslPreferServerCiphers string
}

type advancedConfig struct {
	gzip                           bool
	gzipTypes                      string
	websocket                      bool
	rangeEnabled                   bool
	proxyHTTPVersion               string
	originHTTPVersionPolicy        string
	originAutoDowngrade            bool
	originDowngradeThreshold       int
	originDowngradeWindowSeconds   int
	originDowngradeCooldownSeconds int
	proxySSLProtocols              string
	bodyLimit                      int64
	logRequestHeader               bool
	logResponseHeader              bool
	logRequestBody                 bool
	logRequestBodySizeLimit        int
	originCert                     bool
	realtimeIdentify               bool
	realtimeSend                   bool
	realtimeReturn                 bool
	defaultSite                    bool
	ipv6Enable                     bool
	keepalive                      bool
	keepaliveConn                  int
	keepaliveTimeout               int
}

type proxyTimeoutConfig struct {
	connectTimeout string
	readTimeout    string
	sendTimeout    string
}

func cloneSiteForConfig(site models.Site) *models.Site {
	cloned := site
	if site.Settings != nil {
		var copyMap map[string]interface{}
		if b, err := json.Marshal(site.Settings); err == nil {
			_ = json.Unmarshal(b, &copyMap)
		}
		cloned.Settings = copyMap
	}
	return &cloned
}

func extractHTTPSConfig(settings map[string]interface{}) httpsConfig {
	cfg := httpsConfig{}
	httpsCfg := getMap(settings, "https")
	if httpsCfg == nil {
		return cfg
	}
	cfg.force = parseBoolValue(httpsCfg["force"], false)
	cfg.redirectPort = parseString(httpsCfg["redirect_port"])
	cfg.hsts = parseBoolValue(httpsCfg["hsts"], false)
	cfg.http2 = parseBoolValue(httpsCfg["http2"], false)
	cfg.http3 = parseBoolValue(httpsCfg["http3"], false)
	cfg.ocsp = parseBoolValue(httpsCfg["ocsp_stapling"], false)
	cfg.sslProtocols = sanitizeNginxValue(parseString(httpsCfg["ssl_protocols"]))
	cfg.sslCiphers = sanitizeNginxValue(parseString(httpsCfg["ssl_ciphers"]))
	profile := strings.ToLower(parseString(httpsCfg["ssl_profile"]))
	if profile == "modern" {
		if cfg.sslProtocols == "" {
			cfg.sslProtocols = "TLSv1.2 TLSv1.3"
		}
		if cfg.sslCiphers == "" {
			cfg.sslCiphers = "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256"
		}
	} else if profile == "compat" {
		if cfg.sslProtocols == "" {
			cfg.sslProtocols = "TLSv1 TLSv1.1 TLSv1.2 TLSv1.3"
		}
	}
	cfg.sslPreferServerCiphers = normalizeOnOff(httpsCfg["ssl_prefer_server_ciphers"])
	return cfg
}

func extractAdvancedConfig(settings map[string]interface{}) advancedConfig {
	cfg := advancedConfig{
		originHTTPVersionPolicy:        "auto",
		originAutoDowngrade:            true,
		originDowngradeThreshold:       3,
		originDowngradeWindowSeconds:   60,
		originDowngradeCooldownSeconds: 600,
		keepaliveConn:                  64,
		keepaliveTimeout:               60,
	}
	bodyLimitSet := false
	adv := getMap(settings, "advanced")
	if adv != nil {
		cfg.gzip = parseBoolValue(adv["gzip"], false)
		cfg.gzipTypes = sanitizeNginxValue(parseString(adv["gzip_types"]))
		cfg.websocket = parseBoolValue(adv["websocket"], false)
		cfg.rangeEnabled = parseBoolValue(adv["range"], false)
		cfg.proxyHTTPVersion = sanitizeProxyHTTPVersion(parseString(adv["proxy_http_version"]))
		cfg.originHTTPVersionPolicy = sanitizeOriginHTTPVersionPolicy(parseString(adv["origin_http_version_policy"]))
		if cfg.originHTTPVersionPolicy == "" {
			if cfg.proxyHTTPVersion == "1.1" {
				cfg.originHTTPVersionPolicy = "http11"
			} else if parseBoolValue(adv["ups_keepalive"], false) {
				cfg.originHTTPVersionPolicy = "auto"
			} else {
				cfg.originHTTPVersionPolicy = "auto"
			}
		}
		cfg.originAutoDowngrade = parseBoolValue(adv["origin_auto_downgrade"], true)
		cfg.originDowngradeThreshold = parseIntValue(adv["origin_downgrade_threshold"], 3)
		cfg.originDowngradeWindowSeconds = parseIntValue(adv["origin_downgrade_window_seconds"], 60)
		cfg.originDowngradeCooldownSeconds = parseIntValue(adv["origin_downgrade_cooldown_seconds"], 600)
		cfg.proxySSLProtocols = sanitizeNginxValue(parseString(adv["proxy_ssl_protocols"]))
		if raw, ok := adv["body_limit"]; ok {
			cfg.bodyLimit = normalizeBodyLimitToKB(raw, adv["body_limit_unit"])
			bodyLimitSet = true
		}
		cfg.logRequestHeader = parseBoolValue(adv["log_request_header"], false)
		cfg.logResponseHeader = parseBoolValue(adv["log_response_header"], false)
		cfg.logRequestBody = parseBoolValue(adv["log_request_body"], false)
		cfg.logRequestBodySizeLimit = parseIntValue(adv["log_request_body_size_limit"], 0)
		cfg.originCert = parseBoolValue(adv["origin_cert"], false)
		cfg.realtimeIdentify = parseBoolValue(adv["realtime_identify"], false)
		cfg.realtimeSend = parseBoolValue(adv["realtime_send"], false)
		cfg.realtimeReturn = parseBoolValue(adv["realtime_return"], false)
		cfg.defaultSite = parseBoolValue(adv["default_site"], false)
		cfg.ipv6Enable = parseBoolValue(adv["ipv6"], false)
		cfg.keepalive = parseBoolValue(adv["ups_keepalive"], false)
		cfg.keepaliveConn = parseIntValue(adv["ups_keepalive_conn"], cfg.keepaliveConn)
		cfg.keepaliveTimeout = parseIntValue(adv["ups_keepalive_timeout"], cfg.keepaliveTimeout)
	}
	if cfg.originHTTPVersionPolicy == "auto" || cfg.originHTTPVersionPolicy == "http11" {
		cfg.keepalive = true
	}
	if cfg.originHTTPVersionPolicy == "compat" {
		cfg.keepalive = false
	}
	if cfg.originDowngradeThreshold <= 0 {
		cfg.originDowngradeThreshold = 3
	}
	if cfg.originDowngradeWindowSeconds <= 0 {
		cfg.originDowngradeWindowSeconds = 60
	}
	if cfg.originDowngradeCooldownSeconds <= 0 {
		cfg.originDowngradeCooldownSeconds = 600
	}
	if cfg.keepaliveConn <= 0 {
		cfg.keepaliveConn = 64
	}
	if cfg.keepaliveTimeout <= 0 {
		cfg.keepaliveTimeout = 60
	}
	if !bodyLimitSet && settings != nil {
		if raw, ok := settings["upload_limit"]; ok {
			cfg.bodyLimit = normalizeBodyLimitToKB(raw, "mb")
			bodyLimitSet = true
		}
	}
	if settings != nil {
		if cfg.logRequestHeader == false {
			cfg.logRequestHeader = parseBoolValue(settings["log_request_header"], cfg.logRequestHeader)
		}
		if cfg.logResponseHeader == false {
			cfg.logResponseHeader = parseBoolValue(settings["log_response_header"], cfg.logResponseHeader)
		}
		if cfg.logRequestBody == false {
			cfg.logRequestBody = parseBoolValue(settings["log_request_body"], cfg.logRequestBody)
		}
		if cfg.logRequestBodySizeLimit == 0 {
			cfg.logRequestBodySizeLimit = parseIntValue(settings["log_request_body_size_limit"], cfg.logRequestBodySizeLimit)
		}
		if cfg.originCert == false {
			cfg.originCert = parseBoolValue(settings["origin_cert"], cfg.originCert)
		}
		if cfg.realtimeIdentify == false {
			cfg.realtimeIdentify = parseBoolValue(settings["realtime_identify"], cfg.realtimeIdentify)
		}
		if cfg.realtimeSend == false {
			cfg.realtimeSend = parseBoolValue(settings["realtime_send"], cfg.realtimeSend)
		}
		if cfg.realtimeReturn == false {
			cfg.realtimeReturn = parseBoolValue(settings["realtime_return"], cfg.realtimeReturn)
		}
		if cfg.defaultSite == false {
			cfg.defaultSite = parseBoolValue(settings["default_site"], cfg.defaultSite)
		}
		if cfg.ipv6Enable == false {
			cfg.ipv6Enable = parseBoolValue(settings["ipv6"], cfg.ipv6Enable)
			if cfg.ipv6Enable == false {
				cfg.ipv6Enable = parseBoolValue(settings["ipv6_enable"], cfg.ipv6Enable)
			}
		}
	}
	return cfg
}

func normalizeBodyLimitToKB(raw interface{}, unitRaw interface{}) int64 {
	value := int64(parseIntValue(raw, 0))
	if value <= 0 {
		return 0
	}
	unit := strings.ToLower(strings.TrimSpace(parseString(unitRaw)))
	switch unit {
	case "kb", "k":
		return value
	case "mb", "m":
		return value * 1024
	default:
		return value * 1024
	}
}

func extractProxyTimeouts(settings map[string]interface{}) proxyTimeoutConfig {
	cfg := proxyTimeoutConfig{}
	backsource := getMap(settings, "backsource")
	if backsource == nil {
		return cfg
	}
	cfg.connectTimeout = parseString(backsource["connect_timeout"])
	if cfg.connectTimeout == "" {
		cfg.connectTimeout = parseString(backsource["timeout"])
	}
	cfg.connectTimeout = sanitizeNginxToken(cfg.connectTimeout)
	cfg.readTimeout = sanitizeNginxToken(parseString(backsource["timeout"]))
	cfg.sendTimeout = sanitizeNginxToken(parseString(backsource["timeout"]))
	return cfg
}

func loadNginxConfig() *models.EdgeNginxConfig {
	values, err := LoadConfigMap("nginx_config", "global", 0)
	if err != nil {
		return nil
	}
	raw := strings.TrimSpace(values["nginx-config-file"])
	if raw == "" {
		return nil
	}
	var cfg models.EdgeNginxConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	return &cfg
}

func extractCacheConfig(settings map[string]interface{}) *models.EdgeCacheConfig {
	if settings == nil {
		return nil
	}
	cacheCfg := getMap(settings, "cache")
	if cacheCfg == nil {
		return nil
	}
	enable := parseBoolValue(cacheCfg["enable"], true)
	defaultTTL := parseIntValue(cacheCfg["ttl"], 0)
	rules := parseCacheRulesFromSettings(cacheCfg["rules"])
	return &models.EdgeCacheConfig{
		Enable:     enable,
		DefaultTTL: defaultTTL,
		Rules:      rules,
	}
}

func parseCacheRulesFromSettings(raw interface{}) []models.EdgeCacheRule {
	if raw == nil {
		return nil
	}
	rules := make([]models.EdgeCacheRule, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				rules = append(rules, mapToCacheRules(m)...)
			}
		}
	case []map[string]interface{}:
		for _, item := range v {
			rules = append(rules, mapToCacheRules(item)...)
		}
	default:
		if b, err := json.Marshal(raw); err == nil {
			var list []map[string]interface{}
			if err := json.Unmarshal(b, &list); err == nil {
				for _, item := range list {
					rules = append(rules, mapToCacheRules(item)...)
				}
			}
		}
	}
	if len(rules) == 0 {
		return nil
	}
	rules = dedupeEdgeCacheRules(rules)
	if len(rules) == 0 {
		return nil
	}
	return rules
}

func mapToCacheRules(raw map[string]interface{}) []models.EdgeCacheRule {
	base := mapToCacheRule(raw)
	if base.Rule != "" || base.Ext != "" || base.URI != "" || base.Prefix != "" {
		return []models.EdgeCacheRule{base}
	}

	ruleType := strings.ToLower(parseString(raw["type"]))
	rawValue := parseString(raw["value"])
	values := splitCacheRuleValues(rawValue)
	if len(values) == 0 {
		values = []string{""}
	}

	switch ruleType {
	case "suffix":
		return buildCacheRulesWithExt(base, values)
	case "dir":
		return buildCacheRulesWithPrefix(base, values)
	case "path":
		return buildCacheRulesWithURI(base, values)
	case "all":
		base.Prefix = "/"
		return []models.EdgeCacheRule{base}
	case "index":
		base.URI = "/"
		return []models.EdgeCacheRule{base}
	default:
		return []models.EdgeCacheRule{base}
	}
}

func mapToCacheRule(raw map[string]interface{}) models.EdgeCacheRule {
	ignoreArgs := parseBoolValue(raw["ignore_args"], false)
	if !ignoreArgs {
		ignoreArgs = parseBoolValue(raw["ignore_query"], false)
	}
	enableRange := parseBoolValue(raw["enable_range"], false)
	if !enableRange {
		enableRange = parseBoolValue(raw["enable_slice"], false)
	}
	return models.EdgeCacheRule{
		Rule:           parseString(raw["rule"]),
		Ext:            parseString(raw["ext"]),
		URI:            parseString(raw["uri"]),
		Prefix:         parseString(raw["prefix"]),
		TTL:            parseIntValue(raw["ttl"], 0),
		Enable:         parseBoolPtr(raw["enable"]),
		NoCache:        parseBoolValue(raw["no_cache"], false),
		ForceCache:     parseBoolValue(raw["force_cache"], false),
		EnableRange:    enableRange,
		IgnoreVary:     parseBoolValue(raw["ignore_vary"], false),
		SkipConditions: parseCacheSkipConditions(raw["skip_conditions"]),
		Priority:       parseIntValue(raw["priority"], 0),
		IgnoreArgs:     ignoreArgs,
		CacheKey:       parseString(raw["cache_key"]),
	}
}

func parseCacheSkipConditions(raw interface{}) []models.EdgeCacheSkipCondition {
	if raw == nil {
		return nil
	}
	out := make([]models.EdgeCacheSkipCondition, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				condType := parseString(m["type"])
				condValue := parseString(m["value"])
				if condType != "" && condValue != "" {
					out = append(out, models.EdgeCacheSkipCondition{Type: condType, Value: condValue})
				}
			}
		}
	case []map[string]interface{}:
		for _, item := range v {
			condType := parseString(item["type"])
			condValue := parseString(item["value"])
			if condType != "" && condValue != "" {
				out = append(out, models.EdgeCacheSkipCondition{Type: condType, Value: condValue})
			}
		}
	default:
		if b, err := json.Marshal(raw); err == nil {
			var list []map[string]interface{}
			if err := json.Unmarshal(b, &list); err == nil {
				for _, item := range list {
					condType := parseString(item["type"])
					condValue := parseString(item["value"])
					if condType != "" && condValue != "" {
						out = append(out, models.EdgeCacheSkipCondition{Type: condType, Value: condValue})
					}
				}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func splitCacheRuleValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var parts []string
	if strings.Contains(value, "|") {
		parts = strings.Split(value, "|")
	} else {
		parts = strings.Fields(value)
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func buildCacheRulesWithExt(base models.EdgeCacheRule, values []string) []models.EdgeCacheRule {
	rules := make([]models.EdgeCacheRule, 0, len(values))
	for _, val := range values {
		item := strings.TrimSpace(val)
		if item == "" {
			continue
		}
		item = strings.TrimPrefix(item, "*")
		item = strings.TrimPrefix(item, ".")
		if item == "" {
			continue
		}
		rule := base
		rule.Ext = item
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		return []models.EdgeCacheRule{base}
	}
	return rules
}

func buildCacheRulesWithPrefix(base models.EdgeCacheRule, values []string) []models.EdgeCacheRule {
	rules := make([]models.EdgeCacheRule, 0, len(values))
	for _, val := range values {
		item := normalizeCachePath(val)
		if item == "" {
			continue
		}
		rule := base
		rule.Prefix = item
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		return []models.EdgeCacheRule{base}
	}
	return rules
}

func buildCacheRulesWithURI(base models.EdgeCacheRule, values []string) []models.EdgeCacheRule {
	rules := make([]models.EdgeCacheRule, 0, len(values))
	for _, val := range values {
		item := normalizeCachePath(val)
		if item == "" {
			continue
		}
		rule := base
		rule.URI = item
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		return []models.EdgeCacheRule{base}
	}
	return rules
}

func normalizeCachePath(value string) string {
	item := strings.TrimSpace(value)
	if item == "" {
		return ""
	}
	if strings.HasPrefix(item, "/") {
		return item
	}
	return "/" + item
}

func extractOriginConfig(site models.Site) (string, string, string) {
	protocol := strings.TrimSpace(site.BackendProtocol)
	httpPort := ""
	httpsPort := ""
	if site.Settings != nil {
		backsourceCfg := getMap(site.Settings, "backsource")
		if backsourceCfg != nil {
			if protocol == "" {
				protocol = parseString(backsourceCfg["protocol"])
			}
			httpPort = parseString(backsourceCfg["http_port"])
			httpsPort = parseString(backsourceCfg["https_port"])
		}
	}
	return protocol, httpPort, httpsPort
}

func buildStreamsForNode(node *models.Node, groupIDs []int64, l2TargetsByGroup map[int64][]l2Target, groupL2Config map[int64]string) []models.EdgeStream {
	if len(groupIDs) == 0 {
		return nil
	}
	var forwards []models.Forward
	if err := db.DB.Where("node_group_id IN ? AND enable = ?", groupIDs, true).Find(&forwards).Error; err != nil {
		return nil
	}
	if len(forwards) == 0 {
		return nil
	}
	userPackageMap, err := loadUserPackageMapForForwards(forwards)
	if err != nil {
		return nil
	}
	streams := make([]models.EdgeStream, 0, len(forwards))
	for _, forward := range forwards {
		effectiveForward := forward
		if defaults, err := GetStreamDefaultMap(forward.UserID); err == nil {
			ApplyForwardDefaults(&effectiveForward, defaults)
		}
		streamSettings := effectiveForward.Settings
		originCfg := getMap(streamSettings, "origin")

		connectTimeout := ""
		proxyTimeout := ""
		connLimit := 0
		listenProtocol := "tcp"
		if streamSettings != nil {
			if raw := strings.ToLower(strings.TrimSpace(parseString(streamSettings["listen_protocol"]))); raw == "udp" || raw == "tcp" {
				listenProtocol = raw
			}
		}
		if originCfg != nil {
			connectTimeout = parseString(originCfg["connect_timeout"])
			proxyTimeout = parseString(originCfg["proxy_timeout"])
			connLimit = parseIntValue(originCfg["conn_limit"], 0)
		}
		if connectTimeout == "" {
			connectTimeout = "10s"
		}
		if proxyTimeout == "" {
			proxyTimeout = "60s"
		}

		useL2 := false
		if node != nil && node.Level == 1 {
			packageL2Enabled := false
			if pkg, ok := userPackageMap[forward.UserPackageID]; ok {
				packageL2Enabled = pkg.L2Origin
			}
			if resolveL2Enabled("current", groupL2Config[forward.NodeGroupID], packageL2Enabled) && len(l2TargetsByGroup[forward.NodeGroupID]) > 0 {
				useL2 = true
			}
		}
		targets := make([]models.EdgeStreamTarget, 0, len(forward.Origins))
		if useL2 {
			for _, l2Node := range l2TargetsByGroup[forward.NodeGroupID] {
				if l2Node.IP == "" {
					continue
				}
				targets = append(targets, models.EdgeStreamTarget{
					Addr:   l2Node.IP,
					Weight: 1,
					Enable: true,
					NodeID: l2Node.NodeID,
				})
			}
			for _, origin := range effectiveForward.Origins {
				if !origin.Enable {
					continue
				}
				targets = append(targets, models.EdgeStreamTarget{
					Addr:   origin.Address,
					Weight: origin.Weight,
					Enable: origin.Enable,
					Backup: true,
				})
			}
		} else {
			for _, origin := range effectiveForward.Origins {
				if !origin.Enable {
					continue
				}
				targets = append(targets, models.EdgeStreamTarget{
					Addr:   origin.Address,
					Weight: origin.Weight,
					Enable: origin.Enable,
				})
			}
		}
		streams = append(streams, models.EdgeStream{
			ID:                  effectiveForward.ID,
			ListenPorts:         effectiveForward.ListenPorts,
			ListenProtocol:      listenProtocol,
			Targets:             targets,
			UseListenPort:       useL2,
			BalanceWay:          strings.TrimSpace(effectiveForward.BalanceWay),
			ProxyProtocol:       effectiveForward.ProxyProtocol,
			ProxyConnectTimeout: connectTimeout,
			ProxyTimeout:        proxyTimeout,
			ConnLimit:           connLimit,
		})
	}
	return streams
}

func getMap(root map[string]interface{}, key string) map[string]interface{} {
	if root == nil {
		return nil
	}
	if val, ok := root[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func parseIntValue(value interface{}, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return fallback
}

func parseBoolValue(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "true" || v == "1" || v == "yes" || v == "on" {
			return true
		}
		if v == "false" || v == "0" || v == "no" || v == "off" {
			return false
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return fallback
}

func parseBoolPtr(value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case string:
		if v == "" {
			return nil
		}
		parsed := parseBoolValue(v, false)
		return &parsed
	case float64:
		parsed := v != 0
		return &parsed
	case int:
		parsed := v != 0
		return &parsed
	case int64:
		parsed := v != 0
		return &parsed
	}
	return nil
}

type l2Target struct {
	NodeID int64
	IP     string
}

func resolveL2Mode(settings map[string]interface{}) string {
	if settings == nil {
		return ""
	}
	if val := parseString(settings["l2_config"]); val != "" {
		return strings.ToLower(strings.TrimSpace(val))
	}
	if adv := getMap(settings, "advanced"); adv != nil {
		if val := parseString(adv["l2_config"]); val != "" {
			return strings.ToLower(strings.TrimSpace(val))
		}
	}
	return ""
}

func resolveL2Enabled(mode string, groupConfig string, packageEnabled bool) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "current"
	}
	if mode == "none" {
		return false
	}
	groupConfig = strings.ToLower(strings.TrimSpace(groupConfig))
	if groupConfig == "none" {
		return false
	}
	if mode == "current" {
		return packageEnabled
	}
	return true
}

func resolveListenPort(ports []string, fallback string) string {
	for _, port := range ports {
		port = strings.TrimSpace(port)
		if port != "" {
			return port
		}
	}
	return fallback
}

func loadNodeGroupL2Config(groupIDs []int64) map[int64]string {
	result := map[int64]string{}
	if db.DB == nil || len(groupIDs) == 0 {
		return result
	}
	var groups []models.NodeGroup
	if err := db.DB.Select("id", "backup_switch_policy").
		Where("id IN ?", groupIDs).
		Find(&groups).Error; err != nil {
		return result
	}
	type policy struct {
		L2Config string `json:"l2_config"`
	}
	for _, group := range groups {
		cfg := ""
		if strings.TrimSpace(group.BackupSwitchPolicy) != "" {
			var parsed policy
			if json.Unmarshal([]byte(group.BackupSwitchPolicy), &parsed) == nil {
				cfg = strings.TrimSpace(parsed.L2Config)
			}
		}
		result[group.ID] = cfg
	}
	return result
}

func loadL2TargetsByGroup(groupIDs []int64) map[int64][]l2Target {
	result := map[int64][]l2Target{}
	if db.DB == nil || len(groupIDs) == 0 {
		return result
	}
	var lines []models.Line
	if err := db.DB.Select("node_group_id", "node_id", "enable").
		Where("node_group_id IN ? AND enable = ?", groupIDs, true).
		Find(&lines).Error; err != nil {
		return result
	}
	nodeSet := map[int64]struct{}{}
	for _, line := range lines {
		if line.NodeID != 0 {
			nodeSet[line.NodeID] = struct{}{}
		}
	}
	if len(nodeSet) == 0 {
		return result
	}
	nodeIDs := make([]int64, 0, len(nodeSet))
	for id := range nodeSet {
		nodeIDs = append(nodeIDs, id)
	}
	var nodes []models.Node
	if err := db.DB.Select("id", "ip", "level", "enable").
		Where("id IN ? AND level = ? AND enable = ?", nodeIDs, 2, true).
		Find(&nodes).Error; err != nil {
		return result
	}
	nodeMap := map[int64]models.Node{}
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}
	added := map[int64]map[int64]struct{}{}
	for _, line := range lines {
		node, ok := nodeMap[line.NodeID]
		if !ok || node.IP == "" {
			continue
		}
		if _, ok := added[line.NodeGroupID]; !ok {
			added[line.NodeGroupID] = map[int64]struct{}{}
		}
		if _, exists := added[line.NodeGroupID][node.ID]; exists {
			continue
		}
		added[line.NodeGroupID][node.ID] = struct{}{}
		result[line.NodeGroupID] = append(result[line.NodeGroupID], l2Target{
			NodeID: node.ID,
			IP:     node.IP,
		})
	}
	return result
}

func loadCCData(ruleIDs []int64) (map[int64][]models.EdgeCCRuleItem, map[int64]models.EdgeCCMatcher, map[int64]models.EdgeCCFilter, error) {
	uniqueRuleIDs := uniqueInt64(ruleIDs)
	if len(uniqueRuleIDs) == 0 {
		return nil, nil, nil, nil
	}

	var rules []models.CCRule
	if err := db.DB.Where("id IN ? AND enable = ?", uniqueRuleIDs, true).Find(&rules).Error; err != nil {
		return nil, nil, nil, err
	}

	ccRules := make(map[int64][]models.EdgeCCRuleItem)
	matcherIDs := make([]int64, 0)
	filterIDs := make([]int64, 0)

	for _, rule := range rules {
		items := parseCCRuleData(rule.Data)
		if len(items) == 0 {
			ccRules[rule.ID] = []models.EdgeCCRuleItem{}
			continue
		}
		for _, item := range items {
			entry := parseEdgeCCRuleItem(item)
			ccRules[rule.ID] = append(ccRules[rule.ID], entry)
			if entry.MatcherID > 0 {
				matcherIDs = append(matcherIDs, entry.MatcherID)
			}
			if entry.FilterID > 0 {
				filterIDs = append(filterIDs, entry.FilterID)
			}
			if entry.Filter2ID > 0 {
				filterIDs = append(filterIDs, entry.Filter2ID)
			}
		}
	}

	ccMatchers := make(map[int64]models.EdgeCCMatcher)
	if len(matcherIDs) > 0 {
		var matchers []models.CCMatch
		if err := db.DB.Where("id IN ? AND enable = ?", uniqueInt64(matcherIDs), true).Find(&matchers).Error; err != nil {
			return ccRules, ccMatchers, nil, err
		}
		for _, matcher := range matchers {
			ccMatchers[matcher.ID] = models.EdgeCCMatcher{
				ID:   matcher.ID,
				Data: matcher.Data,
			}
		}
	}

	ccFilters := make(map[int64]models.EdgeCCFilter)
	if len(filterIDs) > 0 {
		var filters []models.CCFilter
		if err := db.DB.Where("id IN ? AND enable = ?", uniqueInt64(filterIDs), true).Find(&filters).Error; err != nil {
			return ccRules, ccMatchers, nil, err
		}
		for _, filter := range filters {
			ccFilters[filter.ID] = models.EdgeCCFilter{
				ID:           filter.ID,
				Type:         filter.Type,
				WithinSecond: filter.WithinSecond,
				MaxReq:       filter.MaxReq,
				MaxReqPerURI: filter.MaxReqPerUri,
				Extra:        filter.Extra,
			}
		}
	}

	return ccRules, ccMatchers, ccFilters, nil
}

func parseCCRuleData(raw string) []map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return list
	}
	var wrapper struct {
		Rules []map[string]interface{} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil {
		return wrapper.Rules
	}
	return nil
}

func parseEdgeCCRuleItem(item map[string]interface{}) models.EdgeCCRuleItem {
	entry := models.EdgeCCRuleItem{
		MatcherID: parseACLID(item["matcher"]),
		FilterID:  parseACLID(item["filter1"]),
		Filter2ID: parseACLID(item["filter2"]),
		Action:    parseString(item["action"]),
		Mode:      parseString(item["mode"]),
		Enabled:   parseBool(item["state"], true),
	}
	if entry.MatcherID == 0 {
		entry.MatcherID = parseACLID(item["matcher_id"])
	}
	if entry.FilterID == 0 {
		entry.FilterID = parseACLID(item["filter1_id"])
	}
	if entry.FilterID == 0 {
		entry.FilterID = parseACLID(item["filter_id"])
	}
	if entry.Filter2ID == 0 {
		entry.Filter2ID = parseACLID(item["filter2_id"])
	}
	if v, ok := item["is_on"]; ok {
		entry.Enabled = parseBool(v, entry.Enabled)
	}
	if v, ok := item["on"]; ok {
		entry.Enabled = parseBool(v, entry.Enabled)
	}
	if v, ok := item["enabled"]; ok {
		entry.Enabled = parseBool(v, entry.Enabled)
	}
	return entry
}

func normalizeCustomCCRulesRaw(raw interface{}) []map[string]interface{} {
	if raw == nil {
		return nil
	}
	switch list := raw.(type) {
	case []map[string]interface{}:
		if len(list) == 0 {
			return nil
		}
		return list
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		if b, err := json.Marshal(raw); err == nil {
			var parsed []map[string]interface{}
			if json.Unmarshal(b, &parsed) == nil && len(parsed) > 0 {
				return parsed
			}
		}
	}
	return nil
}

func extractCustomCCRules(settings map[string]interface{}) []map[string]interface{} {
	if settings == nil {
		return nil
	}
	var rules []map[string]interface{}
	if security, ok := settings["security"].(map[string]interface{}); ok {
		if normalized := normalizeCustomCCRulesRaw(security["custom_rules"]); len(normalized) > 0 {
			rules = normalized
		} else if cc, ok := security["cc"].(map[string]interface{}); ok {
			if normalized := normalizeCustomCCRulesRaw(cc["customRules"]); len(normalized) > 0 {
				rules = normalized
			}
		}
	}
	if len(rules) == 0 {
		return nil
	}
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if _, ok := rule["on"]; !ok {
			if _, ok := rule["enabled"]; !ok {
				if _, ok := rule["is_on"]; !ok {
					if _, ok := rule["state"]; !ok {
						rule["on"] = true
					}
				}
			}
		}
	}
	return rules
}

func uniqueInt64(input []int64) []int64 {
	if len(input) == 0 {
		return nil
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(input))
	for _, id := range input {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func parseString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func loadAllCCData() (map[int64][]models.EdgeCCRuleItem, map[int64]models.EdgeCCMatcher, map[int64]models.EdgeCCFilter, error) {
	ccRules := make(map[int64][]models.EdgeCCRuleItem)
	var rules []models.CCRule
	if err := db.DB.Where("enable = ?", true).Find(&rules).Error; err != nil {
		return ccRules, nil, nil, err
	}
	for _, rule := range rules {
		entries := parseCCRuleData(rule.Data)
		if len(entries) == 0 {
			ccRules[rule.ID] = []models.EdgeCCRuleItem{}
			continue
		}
		items := make([]models.EdgeCCRuleItem, 0, len(entries))
		for _, entry := range entries {
			item := parseEdgeCCRuleItem(entry)
			items = append(items, item)
		}
		ccRules[rule.ID] = items
	}

	ccMatchers := make(map[int64]models.EdgeCCMatcher)
	var matchers []models.CCMatch
	if err := db.DB.Where("enable = ?", true).Find(&matchers).Error; err != nil {
		return ccRules, ccMatchers, nil, err
	}
	for _, matcher := range matchers {
		ccMatchers[matcher.ID] = models.EdgeCCMatcher{
			ID:   matcher.ID,
			Data: matcher.Data,
		}
	}

	ccFilters := make(map[int64]models.EdgeCCFilter)
	var filters []models.CCFilter
	if err := db.DB.Where("enable = ?", true).Find(&filters).Error; err != nil {
		return ccRules, ccMatchers, nil, err
	}
	for _, filter := range filters {
		ccFilters[filter.ID] = models.EdgeCCFilter{
			ID:           filter.ID,
			Type:         filter.Type,
			WithinSecond: filter.WithinSecond,
			MaxReq:       filter.MaxReq,
			MaxReqPerURI: filter.MaxReqPerUri,
			Extra:        filter.Extra,
		}
	}

	return ccRules, ccMatchers, ccFilters, nil
}

func normalizeOnOff(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case bool:
		if v {
			return "on"
		}
		return "off"
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		switch s {
		case "true", "1", "yes", "on":
			return "on"
		case "false", "0", "no", "off":
			return "off"
		default:
			return strings.TrimSpace(v)
		}
	default:
		s := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", v)))
		switch s {
		case "true", "1", "yes", "on":
			return "on"
		case "false", "0", "no", "off":
			return "off"
		default:
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
}

func parseBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "true" || v == "1" || v == "yes" {
			return true
		}
		if v == "false" || v == "0" || v == "no" {
			return false
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return fallback
}
