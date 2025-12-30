package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// siteListItem 和 siteQueryResult 结构体已在 site_controller.go 中定义

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
	if domain == "" || cnameDomain == "" {
		return ""
	}
	if net.ParseIP(domain) != nil {
		return domain + "." + cnameDomain
	}
	return domain + "." + cnameDomain
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
	if err := db.DB.Select("node_group_id").Where("id = ?", userPackageID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return pkg.NodeGroupID, nil
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
