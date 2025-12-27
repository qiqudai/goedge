package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"strings"
)

type dnsAuthMeta struct {
	TTL int `json:"ttl"`
}

func SyncLineRecords(groupID int64, lineID, lineName, action string, nodeIPIDs []int64) error {
	if groupID == 0 || len(nodeIPIDs) == 0 {
		return nil
	}
	var group models.NodeGroup
	if err := db.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return err
	}
	if strings.TrimSpace(group.CnameHostname) == "" {
		return nil
	}

	var domains []models.CnameDomain
	if err := db.DB.Find(&domains).Error; err != nil {
		return err
	}
	if len(domains) == 0 {
		return nil
	}

	var api models.DNSAPI
	if err := db.DB.Order("id desc").First(&api).Error; err != nil {
		return nil
	}
	provider, err := GetProvider(api.Type, api.Auth)
	if err != nil || provider == nil {
		return err
	}

	ttl := 600
	var meta dnsAuthMeta
	if err := json.Unmarshal([]byte(api.Auth), &meta); err == nil && meta.TTL > 0 {
		ttl = meta.TTL
	}

	var nodes []models.Node
	if err := db.DB.Select("id", "ip").Where("id IN ?", nodeIPIDs).Find(&nodes).Error; err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}

	lineValue := ResolveLineValue(api.Type, lineID, lineName)
	action = strings.ToLower(strings.TrimSpace(action))
	for _, domain := range domains {
		root := strings.TrimSpace(domain.Domain)
		if root == "" {
			continue
		}
		for _, node := range nodes {
			if strings.TrimSpace(node.IP) == "" {
				continue
			}
			record := DNSRecord{
				Type:  "A",
				Name:  strings.TrimSpace(group.CnameHostname),
				Value: strings.TrimSpace(node.IP),
				TTL:   ttl,
				Line:  lineValue,
			}
			switch action {
			case "add", "enable":
				_ = provider.AddRecord(root, record)
			case "delete", "disable":
				_ = provider.DeleteRecord(root, record)
			default:
				return errors.New("unsupported action")
			}
		}
	}

	return nil
}
