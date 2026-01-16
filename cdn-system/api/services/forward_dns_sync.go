package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"log"
	"strings"
)

func SyncForwardCnameRecords(forward *models.Forward) error {
	if forward == nil || db.DB == nil {
		return nil
	}
	domainKey, host := resolveForwardCnameTarget(forward)
	if domainKey == "" || host == "" {
		return nil
	}

	var domain models.CnameDomain
	if err := db.DB.Where("domain = ?", domainKey).First(&domain).Error; err != nil {
		log.Printf("[DNS] forward cname sync skip: domain=%s err=%v", domainKey, err)
		return err
	}

	groupID := forward.NodeGroupID
	if groupID == 0 && forward.UserPackageID != 0 {
		var pkg models.UserPackage
		if err := db.DB.Select("node_group_id").Where("id = ?", forward.UserPackageID).First(&pkg).Error; err == nil {
			groupID = pkg.NodeGroupID
		}
	}
	if groupID == 0 {
		log.Printf("[DNS] forward cname sync skip: forward=%d no node group", forward.ID)
		return nil
	}

	return resyncForwardLineCnames(domain, host, groupID)
}

func resolveForwardCnameTarget(forward *models.Forward) (string, string) {
	domainKey := normalizePackageDomain(forward.CnameDomain)
	full := normalizePackageDomain(forward.Cname)
	if full == "" {
		return domainKey, ""
	}
	if domainKey == "" {
		root, name := splitRootDomain(full)
		if root == "" || name == "" {
			return "", ""
		}
		return normalizePackageDomain(root), name
	}

	host := full
	suffix := "." + domainKey
	if full == domainKey {
		host = "@"
	} else if strings.HasSuffix(full, suffix) {
		host = strings.TrimSuffix(full, suffix)
	} else {
		root, name := splitRootDomain(full)
		if root != "" && name != "" {
			return normalizePackageDomain(root), name
		}
	}
	host = strings.TrimSuffix(host, ".")
	return domainKey, host
}

func resyncForwardLineCnames(domain models.CnameDomain, host string, groupID int64) error {
	if groupID == 0 {
		return nil
	}
	var lines []models.Line
	if err := db.DB.Select("line_id", "line_name").
		Where("node_group_id = ?", groupID).
		Find(&lines).Error; err != nil {
		return err
	}
	lineMap := map[string]string{}
	for _, line := range lines {
		lineID := strings.TrimSpace(line.LineID)
		if lineID == "" {
			lineID = "default"
		}
		lineName := strings.TrimSpace(line.LineName)
		if lineName == "" {
			lineName = lineID
		}
		if _, ok := lineMap[lineID]; !ok {
			lineMap[lineID] = lineName
		}
	}
	if len(lineMap) == 0 {
		log.Printf("[DNS] forward cname sync skip: no lines for group=%d host=%s.%s", groupID, host, domain.Domain)
		return nil
	}
	for lineID, lineName := range lineMap {
		ids := loadLineNodeIDs(groupID, lineID)
		if err := dns.SyncPackageLineRecords(domain, host, groupID, lineID, lineName, "resync", ids); err != nil {
			log.Printf("[DNS] forward cname sync failed host=%s.%s group=%d line=%s err=%v", host, domain.Domain, groupID, lineID, err)
			return err
		}
	}
	log.Printf("[DNS] forward cname sync success host=%s.%s group=%d", host, domain.Domain, groupID)
	return nil
}
