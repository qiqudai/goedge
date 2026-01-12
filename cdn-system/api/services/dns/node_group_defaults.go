package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"crypto/rand"
	"errors"
	"strings"
	"time"
)

const groupTokenLetters = "abcdefghijklmnopqrstuvwxyz0123456789"

func EnsureGroupDNSConfig(groupID int64) (models.NodeGroup, error) {
	var group models.NodeGroup
	if groupID == 0 {
		return group, errors.New("group id is empty")
	}
	if err := db.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return group, err
	}

	updates := map[string]interface{}{}
	rawDomain := strings.TrimSpace(group.CnameDomain)
	domain := normalizeDomainName(rawDomain)
	if domain == "" {
		fallback, err := loadFirstCnameDomain()
		if err != nil {
			return group, err
		}
		domain = fallback
		updates["cname_domain"] = domain
	} else if rawDomain != domain {
		updates["cname_domain"] = domain
	}
	group.CnameDomain = domain

	host := strings.TrimSpace(group.CnameHostname)
	if host == "" {
		hostname, err := generateUniqueGroupHostname()
		if err != nil {
			return group, err
		}
		host = hostname
		updates["cname_hostname"] = host
	}
	group.CnameHostname = host

	if len(updates) > 0 {
		updates["update_at"] = time.Now()
		if err := db.DB.Model(&models.NodeGroup{}).Where("id = ?", groupID).Updates(updates).Error; err != nil {
			return group, err
		}
	}

	return group, nil
}

func loadFirstCnameDomain() (string, error) {
	var row models.CnameDomain
	if err := db.DB.Order("id asc").First(&row).Error; err != nil {
		return "", errors.New("cname domains not configured")
	}
	domain := normalizeDomainName(row.Domain)
	if domain == "" {
		return "", errors.New("cname domains not configured")
	}
	return domain, nil
}

func generateUniqueGroupHostname() (string, error) {
	for i := 0; i < 5; i++ {
		token, err := randomToken(8)
		if err != nil {
			return "", err
		}
		var count int64
		if err := db.DB.Model(&models.NodeGroup{}).Where("cname_hostname = ?", token).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return token, nil
		}
	}
	return "", errors.New("failed to generate unique resolution")
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = groupTokenLetters[int(buf[i])%len(groupTokenLetters)]
	}
	return string(buf), nil
}
