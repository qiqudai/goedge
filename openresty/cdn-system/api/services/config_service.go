package services

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)

type ConfigService struct{}

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
		Version:   0,
		NodeID:    strconv.FormatInt(node.ID, 10),
		Domains:   make([]models.EdgeDomain, 0),
		Upstreams: make([]models.EdgeUpstream, 0),
	}

	redisCfg := buildRedisConfig()
	if redisCfg != nil {
		payload.Redis = redisCfg
	}
	if globalCfg := loadGlobalConfig(); globalCfg != nil {
		payload.WAF = &globalCfg.WAF
	}

	// 1. Find Node Groups this node belongs to
	var lines []models.Line
	if err := db.DB.Where("node_id = ?", node.ID).Find(&lines).Error; err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		payload.Version = hashConfigVersion(payload)
		return payload, nil
	}

	var groupIDs []int64
	for _, l := range lines {
		groupIDs = append(groupIDs, l.NodeGroupID)
	}

	// 2. Find Sites assigned to these Node Groups
	var sites []models.Site
	if err := db.DB.Where("node_group_id IN ?", groupIDs).Find(&sites).Error; err != nil {
		return nil, err
	}

	// Preload certs for HTTPS mapping
	var certs []models.Cert
	_ = db.DB.Where("enable = ?", true).Find(&certs).Error

	for _, site := range sites {
		if !site.Enable {
			continue
		}

		upstreamKey := fmt.Sprintf("upstream_%d", site.ID)
		targets := buildUpstreamTargets(site.Backends, site.BackendProtocol)
		if len(targets) > 0 {
			payload.Upstreams = append(payload.Upstreams, models.EdgeUpstream{
				ID:      upstreamKey,
				Targets: targets,
			})
		}

		policy := mapBalancePolicy(site.BalanceWay)
		headers := buildHeaderMap(site.Settings)
		hasHTTPS := len(site.HttpsListen) > 0 || strings.TrimSpace(site.HttpsListenRaw) != ""

		aclDefault, aclRules := buildACLForSite(site)
		for _, domain := range site.Domains {
			domainConf := models.EdgeDomain{
				Name:              domain,
				UpstreamKey:       upstreamKey,
				LoadBalancePolicy: policy,
				Headers:           headers,
				Status:            "active",
				ACLDefaultAction:  aclDefault,
				ACLRules:          aclRules,
			}
			if hasHTTPS {
				cert := findCertForDomain(domain, certs)
				if cert != nil {
					domainConf.SSLCertData = cert.Cert
					domainConf.SSLKeyData = cert.Key
				}
			}
			payload.Domains = append(payload.Domains, domainConf)
		}
	}

	payload.Version = hashConfigVersion(payload)
	return payload, nil
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
	if nodeID == "" {
		return nil, errors.New("node_id is required")
	}

	var node models.Node
	if id, err := strconv.ParseInt(nodeID, 10, 64); err == nil {
		if err := db.DB.Where("id = ?", id).First(&node).Error; err == nil {
			return &node, nil
		}
	}

	if err := db.DB.Where("name = ?", nodeID).First(&node).Error; err != nil {
		return nil, errors.New("node not found")
	}
	return &node, nil
}

func loadGlobalConfig() *models.GlobalConfig {
	var sys models.SysConfig
	if err := db.DB.First(&sys, "`key` = ?", "global_config").Error; err != nil {
		return nil
	}
	if sys.Value == "" {
		return nil
	}
	var cfg models.GlobalConfig
	if err := json.Unmarshal([]byte(sys.Value), &cfg); err != nil {
		return nil
	}
	return &cfg
}

func buildACLForSite(site models.Site) (string, []models.EdgeACLRule) {
	if site.Settings == nil {
		return "", nil
	}
	access, ok := site.Settings["access"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	aclID := parseACLID(access["acl"])
	if aclID == 0 {
		return "", nil
	}
	var acl models.ACL
	if err := db.DB.Where("id = ?", aclID).First(&acl).Error; err != nil {
		return "", nil
	}
	defaultAction := strings.TrimSpace(acl.DefaultAction)
	if defaultAction == "" {
		defaultAction = "allow"
	}
	rules := parseACLRules(acl.Data)
	return defaultAction, rules
}

func parseACLRules(raw string) []models.EdgeACLRule {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []models.EdgeACLRule
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	// Try list of objects
	var generic []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil
	}
	for _, item := range generic {
		entry := models.EdgeACLRule{}
		if v, ok := item["ip"].(string); ok {
			entry.IP = v
		}
		if v, ok := item["action"].(string); ok {
			entry.Action = v
		}
		if entry.IP != "" {
			if entry.Action == "" {
				entry.Action = "allow"
			}
			items = append(items, entry)
		}
	}
	return items
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

func buildRedisConfig() *models.EdgeRedisConfig {
	// Parse RedisAddr "host:port"
	addr := config.App.RedisAddr
	if addr == "" {
		return nil
	}
	host := "127.0.0.1"
	port := 6379

	parts := strings.Split(addr, ":")
	if len(parts) > 0 {
		host = parts[0]
	}
	if len(parts) > 1 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	return &models.EdgeRedisConfig{
		Host:     host,
		Port:     port,
	}
}

func buildUpstreamTargets(backends []string, protocol string) []models.EdgeUpstreamTarget {
	var targets []models.EdgeUpstreamTarget
	for _, backend := range backends {
		if strings.TrimSpace(backend) == "" {
			continue
		}
		// Basic implementation
		targets = append(targets, models.EdgeUpstreamTarget{
			Addr:   backend,
			Weight: 10,
		})
	}
	return targets
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

func buildHeaderMap(settings map[string]interface{}) map[string]string {
	if settings == nil {
		return nil
	}
	headers, ok := settings["headers"].(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]string)
	for k, v := range headers {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

func findCertForDomain(domain string, certs []models.Cert) *models.Cert {
	for _, cert := range certs {
		if cert.Domain == domain {
			return &cert
		}
	}
	return nil
}
