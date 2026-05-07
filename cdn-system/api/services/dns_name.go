package services

import (
	"net"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func SplitDNSZoneAndRecord(host string) (zone string, record string) {
	host = normalizeDomainHost(host)
	if host == "" || net.ParseIP(host) != nil {
		return "", ""
	}
	wildcard := false
	if strings.HasPrefix(host, "*.") {
		wildcard = true
		host = strings.TrimPrefix(host, "*.")
	}
	zone, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || zone == "" {
		return "", ""
	}
	prefix := strings.TrimSuffix(host, "."+zone)
	if prefix == host {
		prefix = ""
	}
	if wildcard {
		if prefix == "" {
			prefix = "*"
		} else {
			prefix = "*." + prefix
		}
	}
	if prefix == "" {
		return zone, "@"
	}
	return zone, prefix
}
