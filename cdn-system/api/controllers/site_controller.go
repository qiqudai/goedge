package controllers

import (
	"cdn-api/models"
	"time"
)

type SiteController struct{}

type siteListItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	UserName        string    `json:"user_name"`
	Domains         []string  `json:"domains"`
	DomainDisplay   string    `json:"domain_display"`
	ListenPorts     string    `json:"listen_ports"`
	HttpListen      []string  `json:"http_listen"`
	HttpsListen     []string  `json:"https_listen"`
	OriginDisplay   string    `json:"origin_display"`
	CNAME           string    `json:"cname"`
	Backends        []string  `json:"backends"`
	HTTPS           bool      `json:"https"`
	UserPackageID   int64     `json:"user_package_id"`
	UserPackageName string    `json:"user_package_name"`
	DNSProviderID   int64     `json:"dns_provider_id"`
	GroupID         int64     `json:"group_id"` // Deprecated: use GroupIDs
	GroupIDs        []int64   `json:"group_ids"`
	GroupName       string    `json:"group_name"`
	NodeGroupID     int64     `json:"node_group_id"`
	NodeGroupName   string    `json:"node_group_name"`
	RegionID        int64     `json:"region_id"`
	RegionName      string    `json:"region_name"`
	Status          bool                   `json:"status"`
	State           string                 `json:"state"`
	Settings        map[string]interface{} `json:"settings"`
	ExpireTime      string                 `json:"expire_time"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type siteQueryResult struct {
	Sites []models.Site
	Total int64
}
