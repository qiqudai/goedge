package services

import (
	"net"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

const defaultIP2RegionPath = "/www/server/go_project/openresty/cdn-system/agent/edge-node/data/ip2region.xdb"

var ipSearcherOnce sync.Once
var ipSearcher *xdb.Searcher
var ipSearcherErr error

func loadIPSearcher() (*xdb.Searcher, error) {
	ipSearcherOnce.Do(func() {
		searcher, err := xdb.NewWithFileOnly(xdb.IPv4, defaultIP2RegionPath)
		if err != nil {
			ipSearcherErr = err
			return
		}
		ipSearcher = searcher
	})
	return ipSearcher, ipSearcherErr
}

// LookupIPRegion returns country and province from ip2region data.
func LookupIPRegion(ip string) (string, string) {
	ip = strings.TrimSpace(ip)
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
	if len(parts) > 2 {
		province = cleanRegion(parts[2])
	}
	return country, province
}

func cleanRegion(value string) string {
	value = strings.TrimSpace(value)
	if value == "0" || value == "-" {
		return ""
	}
	return value
}
