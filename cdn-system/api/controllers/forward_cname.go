package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"strings"
)

const forwardDefaultCnameDomain = "cdn.node.com"

func resolveForwardCnameMode(forward *models.Forward, pkg *models.UserPackage) string {
	mode := ""
	if forward != nil {
		mode = strings.TrimSpace(strings.ToLower(forward.CnameMode))
	}
	if mode == "" && pkg != nil {
		mode = strings.TrimSpace(strings.ToLower(pkg.CnameMode))
	}
	return mode
}

func resolveForwardCnameDomain(forward *models.Forward, pkg *models.UserPackage) string {
	domain := ""
	if forward != nil {
		domain = strings.TrimSpace(forward.CnameDomain)
	}
	if domain == "" && pkg != nil {
		domain = strings.TrimSpace(pkg.CnameDomain)
	}
	if domain == "" {
		domain = forwardDefaultCnameDomain
	}
	return normalizeDomainInput(domain)
}

func resolvePackageCnameHost(pkg *models.UserPackage) string {
	if pkg == nil {
		return ""
	}
	if host := strings.TrimSpace(pkg.CnameHostname); host != "" {
		return host
	}
	if host := strings.TrimSpace(pkg.CnameHostname2); host != "" {
		return host
	}
	if host := strings.TrimSpace(pkg.RecordID); host != "" {
		return host
	}
	return ""
}

func buildCnameHostname(host, domain string) string {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	domain = strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if host == "" {
		if domain == "" {
			return ""
		}
		return domain
	}
	if host == "@" {
		return domain
	}
	if domain == "" {
		return host
	}
	if strings.EqualFold(host, domain) {
		return domain
	}
	suffix := "." + domain
	if strings.HasSuffix(host, suffix) {
		return host
	}
	return host + suffix
}

func generateUniqueForwardHostname(domain string) (string, error) {
	domain = normalizeDomainInput(domain)
	for i := 0; i < 5; i++ {
		token, err := randomToken(8)
		if err != nil {
			return "", err
		}
		full := buildCnameHostname(token, domain)
		if full == "" {
			continue
		}
		var count int64
		if err := db.DB.Model(&models.Site{}).Where("cname_hostname = ?", full).Count(&count).Error; err != nil {
			return "", err
		}
		if count != 0 {
			continue
		}
		if err := db.DB.Model(&models.Forward{}).Where("cname_hostname = ?", full).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return token, nil
		}
	}
	return "", errors.New("failed to generate unique cname hostname")
}

func applyForwardCname(forward *models.Forward, pkg *models.UserPackage) (bool, error) {
	if forward == nil {
		return false, nil
	}
	updated := false
	mode := resolveForwardCnameMode(forward, pkg)
	domain := resolveForwardCnameDomain(forward, pkg)

	if mode == "package" {
		host := resolvePackageCnameHost(pkg)
		if host == "" {
			host = strings.TrimSpace(forward.Cname)
		}
		cname := buildCnameHostname(host, domain)
		if cname != "" && cname != forward.Cname {
			forward.Cname = cname
			updated = true
		}
		if domain != "" && domain != forward.CnameDomain {
			forward.CnameDomain = domain
			updated = true
		}
		return updated, nil
	}

	cname := strings.TrimSpace(forward.Cname)
	if cname == "" {
		host, err := generateUniqueForwardHostname(domain)
		if err != nil {
			return false, err
		}
		cname = buildCnameHostname(host, domain)
		if cname != "" {
			forward.Cname = cname
			updated = true
		}
	}
	if domain != "" && domain != forward.CnameDomain {
		forward.CnameDomain = domain
		updated = true
	}
	return updated, nil
}
