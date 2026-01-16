package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"sort"
	"strings"
)

type HostFilter struct {
	Exact     []string
	Wildcards []string
}

type HostMatch struct {
	SiteID int64
	Domain string
}

type wildcardMatch struct {
	suffix string
	match  HostMatch
}

type HostMatcher struct {
	Exact     map[string]HostMatch
	Wildcards []wildcardMatch
}

type SiteHostIndex struct {
	Filter      HostFilter
	Matcher     HostMatcher
	SiteFilters map[int64]HostFilter
}

func LoadSiteHostIndex(userID int64) (*SiteHostIndex, error) {
	var sites []models.Site
	query := db.DB.Model(&models.Site{}).Select("id", "uid", "domain")
	if userID > 0 {
		query = query.Where("uid = ?", userID)
	}
	if err := query.Find(&sites).Error; err != nil {
		return nil, err
	}
	idx := &SiteHostIndex{
		Filter:      HostFilter{},
		Matcher:     HostMatcher{Exact: map[string]HostMatch{}},
		SiteFilters: map[int64]HostFilter{},
	}
	seenExact := map[string]struct{}{}
	seenWildcard := map[string]struct{}{}
	for _, site := range sites {
		for _, raw := range site.Domains {
			exact, wildcard := splitHostPattern(raw)
			if exact == "" && wildcard == "" {
				continue
			}
			if exact != "" {
				if _, ok := seenExact[exact]; !ok {
					seenExact[exact] = struct{}{}
					idx.Filter.Exact = append(idx.Filter.Exact, exact)
				}
				if _, ok := idx.Matcher.Exact[exact]; !ok {
					idx.Matcher.Exact[exact] = HostMatch{SiteID: site.ID, Domain: exact}
				}
				addSiteFilter(idx.SiteFilters, site.ID, exact, "")
				continue
			}
			if _, ok := seenWildcard[wildcard]; !ok {
				seenWildcard[wildcard] = struct{}{}
				idx.Filter.Wildcards = append(idx.Filter.Wildcards, wildcard)
			}
			idx.Matcher.Wildcards = append(idx.Matcher.Wildcards, wildcardMatch{
				suffix: wildcard,
				match:  HostMatch{SiteID: site.ID, Domain: wildcard},
			})
			addSiteFilter(idx.SiteFilters, site.ID, "", wildcard)
		}
	}
	sort.Strings(idx.Filter.Exact)
	sort.Strings(idx.Filter.Wildcards)
	sort.Slice(idx.Matcher.Wildcards, func(i, j int) bool {
		return len(idx.Matcher.Wildcards[i].suffix) > len(idx.Matcher.Wildcards[j].suffix)
	})
	return idx, nil
}

func LoadHostFilter(userID int64) (HostFilter, error) {
	idx, err := LoadSiteHostIndex(userID)
	if err != nil {
		return HostFilter{}, err
	}
	return idx.Filter, nil
}

func (f HostFilter) Empty() bool {
	return len(f.Exact) == 0 && len(f.Wildcards) == 0
}

func (f HostFilter) SQLCondition() (string, []interface{}) {
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	if len(f.Exact) > 0 {
		placeholders := make([]string, 0, len(f.Exact))
		for _, host := range f.Exact {
			placeholders = append(placeholders, "?")
			args = append(args, host)
		}
		conditions = append(conditions, "host IN ("+strings.Join(placeholders, ",")+")")
	}
	for _, suffix := range f.Wildcards {
		conditions = append(conditions, "host LIKE ?")
		args = append(args, "%"+suffix)
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return "(" + strings.Join(conditions, " OR ") + ")", args
}

func (f HostFilter) HTTPCondition() string {
	conditions := make([]string, 0)
	if len(f.Exact) > 0 {
		quoted := make([]string, 0, len(f.Exact))
		for _, host := range f.Exact {
			quoted = append(quoted, quoteClickHouseString(host))
		}
		conditions = append(conditions, "host IN ("+strings.Join(quoted, ",")+")")
	}
	for _, suffix := range f.Wildcards {
		conditions = append(conditions, "host LIKE "+quoteClickHouseString("%"+suffix))
	}
	if len(conditions) == 0 {
		return ""
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func (m HostMatcher) Match(host string) (HostMatch, bool) {
	normalized := normalizeHostValue(host)
	if normalized == "" {
		return HostMatch{}, false
	}
	if match, ok := m.Exact[normalized]; ok {
		return match, true
	}
	for _, entry := range m.Wildcards {
		if strings.HasSuffix(normalized, entry.suffix) {
			return entry.match, true
		}
	}
	return HostMatch{}, false
}

func addSiteFilter(siteFilters map[int64]HostFilter, siteID int64, exact, wildcard string) {
	if siteID == 0 {
		return
	}
	filter := siteFilters[siteID]
	if exact != "" {
		filter.Exact = append(filter.Exact, exact)
	}
	if wildcard != "" {
		filter.Wildcards = append(filter.Wildcards, wildcard)
	}
	siteFilters[siteID] = filter
}

func splitHostPattern(raw string) (string, string) {
	host := strings.TrimSpace(strings.ToLower(raw))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return "", ""
	}
	if strings.Contains(host, "*") {
		host = strings.TrimPrefix(host, "*.")
		host = strings.TrimPrefix(host, "*")
		host = strings.TrimPrefix(host, ".")
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		host = strings.TrimSuffix(host, ".")
		return "", strings.TrimSpace(host)
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return strings.TrimSpace(host), ""
}

func normalizeHostValue(raw string) string {
	host := strings.TrimSpace(strings.ToLower(raw))
	if host == "" {
		return ""
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return host
}
