package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// siteListItem and siteQueryResult are defined in site_controller.go.

func parseListenPorts(ports []string, raw string, fallback string) []string {
	if len(ports) > 0 {
		return ports
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}
	raw = strings.Trim(raw, "[]{}\"")
	raw = strings.ReplaceAll(raw, "port", "")
	raw = strings.ReplaceAll(raw, ":", " ")
	fields := strings.Fields(raw)
	if len(fields) == 0 && fallback != "" {
		return []string{fallback}
	}
	return fields
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

func buildListenDisplay(httpPorts, httpsPorts []string) string {
	parts := make([]string, 0, len(httpPorts)+len(httpsPorts))
	for _, p := range httpPorts {
		if p != "" {
			parts = append(parts, p)
		}
	}
	for _, p := range httpsPorts {
		if p != "" {
			parts = append(parts, p+"s")
		}
	}
	return strings.Join(parts, " ")
}

func encodeList(items []string) string {
	return encodeStringList(items)
}

func extractSecurityIPList(settings map[string]interface{}, key string) ([]string, bool) {
	if settings == nil {
		return nil, false
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		return nil, false
	}
	raw, ok := security[key]
	if !ok {
		return nil, false
	}
	return parseStringListValue(raw), true
}

func setSecurityIPList(settings map[string]interface{}, key string, list []string) {
	if settings == nil {
		return
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		security = map[string]interface{}{}
		settings["security"] = security
	}
	if list == nil {
		list = []string{}
	}
	security[key] = list
}

func mergeSecurityIPList(settings map[string]interface{}, key string, raw string) {
	list := parseStringListValue(raw)
	if len(list) == 0 {
		return
	}
	if settings == nil {
		return
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		security = map[string]interface{}{}
		settings["security"] = security
	}
	if existing, ok := security[key]; ok {
		if len(parseStringListValue(existing)) > 0 {
			return
		}
	}
	security[key] = list
}

func parseStringListValue(value interface{}) []string {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []string:
		return normalizeStringList(v)
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, fmt.Sprint(item))
		}
		return normalizeStringList(items)
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return nil
		}
		var list []string
		if err := json.Unmarshal([]byte(raw), &list); err == nil {
			return normalizeStringList(list)
		}
		fields := strings.FieldsFunc(raw, func(r rune) bool {
			return r == '\n' || r == '\r' || r == '\t' || r == ',' || r == ';' || r == ' '
		})
		return normalizeStringList(fields)
	default:
		return normalizeStringList([]string{fmt.Sprint(v)})
	}
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(item)
		if val == "" {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeSettingsMaps(dst map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	if src == nil {
		return dst
	}
	for key, value := range src {
		if value == nil {
			dst[key] = nil
			continue
		}
		if srcMap, ok := value.(map[string]interface{}); ok {
			if dstMap, ok := dst[key].(map[string]interface{}); ok {
				dst[key] = mergeSettingsMaps(dstMap, srcMap)
			} else {
				dst[key] = mergeSettingsMaps(nil, srcMap)
			}
			continue
		}
		dst[key] = value
	}
	return dst
}

func copyUpdateMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func loadUsers(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.UserID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var users []models.User
	if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		result[u.ID] = u.Name
	}
	return result, nil
}

func loadUserPackages(sites []models.Site) (map[int64]models.UserPackage, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.UserPackageID })
	result := map[int64]models.UserPackage{}
	if len(ids) == 0 {
		return result, nil
	}
	var pkgs []models.UserPackage
	if err := db.DB.Where("id IN ?", ids).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		result[p.ID] = p
	}
	return result, nil
}

func loadSiteGroups(sites []models.Site) (map[int64]string, map[int64][]int64, error) {
	siteIDs := uniqueIDs(sites, func(s models.Site) int64 { return s.ID })
	groupMap := map[int64]string{}
	relMap := map[int64][]int64{}
	if len(siteIDs) == 0 {
		return groupMap, relMap, nil
	}

	var relations []models.SiteGroupRelation
	if err := db.DB.Where("site_id IN ?", siteIDs).Find(&relations).Error; err != nil {
		return nil, nil, err
	}
	groupIDs := make([]int64, 0, len(relations))
	for _, rel := range relations {
		relMap[rel.SiteID] = append(relMap[rel.SiteID], rel.GroupID)
		groupIDs = append(groupIDs, rel.GroupID)
	}
	if len(groupIDs) == 0 {
		return groupMap, relMap, nil
	}
	var groups []models.SiteGroup
	if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}
	return groupMap, relMap, nil
}

func loadNodeGroups(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.NodeGroupID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var groups []models.NodeGroup
	if err := db.DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, g := range groups {
		result[g.ID] = g.Name
	}
	return result, nil
}

func findUserIDsByKeyword(keyword string) ([]int64, error) {
	var users []models.User
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", like, like, like).Find(&users).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	return ids, nil
}

func findUserPackageIDsByName(keyword string) ([]int64, error) {
	var pkgs []models.UserPackage
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ?", like).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(pkgs))
	for _, p := range pkgs {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

func findSiteIDsByGroupName(keyword string) ([]int64, error) {
	var groups []models.SiteGroup
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ?", like).Find(&groups).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	return findSiteIDsByGroupIDs(ids)
}

func findSiteIDsByGroupID(groupID int64) ([]int64, error) {
	return findSiteIDsByGroupIDs([]int64{groupID})
}

func findSiteIDsByGroupIDs(groupIDs []int64) ([]int64, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var rels []models.SiteGroupRelation
	if err := db.DB.Where("group_id IN ?", groupIDs).Find(&rels).Error; err != nil {
		return nil, err
	}
	siteIDs := make([]int64, 0, len(rels))
	for _, rel := range rels {
		siteIDs = append(siteIDs, rel.SiteID)
	}
	return siteIDs, nil
}

func loadRegions(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.RegionID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var regions []models.Region
	if err := db.DB.Where("id IN ?", ids).Find(&regions).Error; err != nil {
		return nil, err
	}
	for _, r := range regions {
		result[r.ID] = r.Name
	}
	return result, nil
}

func uniqueIDs(sites []models.Site, getter func(s models.Site) int64) []int64 {
	seen := map[int64]struct{}{}
	for _, s := range sites {
		id := getter(s)
		if id == 0 {
			continue
		}
		seen[id] = struct{}{}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

type batchSiteItem struct {
	Domains  []string
	Backends []string
}

func parseBatchLine(line string) (*batchSiteItem, error) {
	item := &batchSiteItem{}
	segments := strings.Split(line, "|")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		kv := strings.SplitN(seg, "=", 2)
		if len(kv) != 2 {
			return nil, errors.New("invalid line format")
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "domain":
			item.Domains = splitByComma(val)
		case "ip":
			item.Backends = splitByComma(val)
		}
	}
	if len(item.Domains) == 0 {
		return nil, errors.New("domain is required")
	}
	item.Domains = normalizeSiteDomains(item.Domains)
	if len(item.Domains) == 0 {
		return nil, errors.New("domain is required")
	}
	return item, nil
}

func splitByComma(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildSiteCname(domain string, cnameDomain string) string {
	return services.ComposeSiteCname(domain, cnameDomain)
}

func splitRootDomain(domain string) (string, string) {
	host := normalizeDomainHost(domain)
	if host == "" || net.ParseIP(host) != nil {
		return "", ""
	}
	if strings.HasPrefix(host, "*.") {
		host = strings.TrimPrefix(host, "*.")
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", ""
	}
	root := strings.Join(parts[len(parts)-2:], ".")
	name := "@"
	if len(parts) > 2 {
		name = strings.Join(parts[:len(parts)-2], ".")
	}
	return root, name
}

func normalizeDomainHost(input string) string {
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
	return strings.TrimRight(host, ".")
}

func splitLines(raw string) []string {
	parts := strings.Split(raw, "\n")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitFields(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", " ")
	raw = strings.ReplaceAll(raw, ";", " ")
	raw = strings.ReplaceAll(raw, "\n", " ")
	fields := strings.Fields(raw)
	result := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

func resolveNodeGroupFromPackage(userPackageID int64, requestedID int64) (int64, error) {
	if requestedID != 0 {
		return requestedID, nil
	}
	if userPackageID == 0 {
		return 0, nil
	}
	var pkg models.UserPackage
	if err := db.DB.Select("node_group_id", "package").Where("id = ?", userPackageID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if pkg.NodeGroupID != 0 {
		return pkg.NodeGroupID, nil
	}
	if pkg.PackageID != 0 {
		var plan models.Package
		if err := db.DB.Select("node_group_id").Where("id = ?", pkg.PackageID).First(&plan).Error; err == nil {
			return plan.NodeGroupID, nil
		}
	}
	return 0, nil
}

func resolveRegionFromPackage(userPackageID int64, nodeGroupID int64) int64 {
	if userPackageID != 0 {
		var pkg models.UserPackage
		if err := db.DB.Select("region_id", "package").Where("id = ?", userPackageID).First(&pkg).Error; err == nil {
			if pkg.RegionID != 0 {
				return pkg.RegionID
			}
			if pkg.PackageID != 0 {
				var plan models.Package
				if err := db.DB.Select("region_id").Where("id = ?", pkg.PackageID).First(&plan).Error; err == nil {
					if plan.RegionID != 0 {
						return plan.RegionID
					}
				}
			}
		}
	}
	if nodeGroupID != 0 {
		var group models.NodeGroup
		if err := db.DB.Select("region_id").Where("id = ?", nodeGroupID).First(&group).Error; err == nil {
			if group.RegionID != nil && *group.RegionID > 0 {
				return *group.RegionID
			}
		}
	}
	return 0
}

func isUserRequest(c *gin.Context) bool {
	if val, ok := c.Get("role"); ok && val == "user" {
		return true
	}
	return strings.Contains(c.Request.URL.Path, "/api/v1/user")
}

func parseUserID(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			return id
		}
	}
	return 0
}
