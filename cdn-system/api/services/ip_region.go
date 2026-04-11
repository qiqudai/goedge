package services

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

const defaultIP2RegionPath = "/www/server/go_project/openresty/cdn-system/agent/edge-node/data/ip2region.xdb"

var ipSearcherOnce sync.Once
var ipSearcher *xdb.Searcher
var ipSearcherErr error
var ipSearcherErrLogged bool

func loadIPSearcher() (*xdb.Searcher, error) {
	ipSearcherOnce.Do(func() {
		for _, path := range resolveIP2RegionPaths() {
			searcher, err := xdb.NewWithFileOnly(xdb.IPv4, path)
			if err == nil && searcher != nil {
				ipSearcher = searcher
				ipSearcherErr = nil
				return
			}
			ipSearcherErr = err
		}
		if ipSearcherErr != nil && !ipSearcherErrLogged {
			ipSearcherErrLogged = true
			log.Printf("[ip_region] failed to load ip2region.xdb: %v", ipSearcherErr)
		}
	})
	return ipSearcher, ipSearcherErr
}

// LookupIPRegion returns country and province from ip2region data.
func LookupIPRegion(ip string) (string, string) {
	ip = normalizeIPForRegion(ip)
	if ip == "" {
		return "", ""
	}
	if net.ParseIP(ip) == nil {
		return "", ""
	}
	searcher, err := loadIPSearcher()
	if err != nil || searcher == nil {
		return "", ""
	}
	region, err := searcher.SearchByStr(ip)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(region, "|")
	country := ""
	province := ""
	if len(parts) > 0 {
		country = cleanRegion(parts[0])
	}
	if len(parts) > 1 {
		province = cleanRegion(parts[1])
	}
	return country, province
}

func resolveIP2RegionPaths() []string {
	paths := make([]string, 0, 6)
	if env := strings.TrimSpace(os.Getenv("IP2REGION_XDB_PATH")); env != "" {
		paths = append(paths, env)
	}
	paths = append(paths,
		defaultIP2RegionPath,
		"/www/server/go_project/openresty/agent/edge-node/data/ip2region.xdb",
		"./agent/edge-node/data/ip2region.xdb",
		"../agent/edge-node/data/ip2region.xdb",
	)
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		paths = append(paths, filepath.Join(cwd, "agent/edge-node/data/ip2region.xdb"))
	}
	return dedupeNonEmpty(paths)
}

func normalizeIPForRegion(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.Contains(s, ",") {
		s = strings.TrimSpace(strings.Split(s, ",")[0])
	}
	s = strings.Trim(s, "[]")
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(s); err == nil {
		host = strings.Trim(strings.TrimSpace(host), "[]")
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func dedupeNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func cleanRegion(value string) string {
	value = strings.TrimSpace(value)
	if value == "0" || value == "-" {
		return ""
	}
	return value
}
