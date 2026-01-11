package controllers

import (
	"cdn-api/models"
	"time"
)

type ForwardController struct{}

type forwardListItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	UserName        string    `json:"user_name"`
	ListenPorts     string    `json:"listen_ports"`
	OriginDisplay   string    `json:"origin_display"`
	UserPackageID   int64     `json:"user_package_id"`
	UserPackageName string    `json:"user_package_name"`
	GroupID         int64     `json:"group_id"` // Deprecated: use group_ids
	GroupIDs        []int64   `json:"group_ids"`
	GroupName       string    `json:"group_name"`
	NodeGroupID     int64     `json:"node_group_id"`
	NodeGroupName   string    `json:"node_group_name"`
	CNAME           string    `json:"cname"`
	Status          bool      `json:"status"`
	Remark          string    `json:"remark"`
	CreatedAt       time.Time `json:"created_at"`
}

type forwardQueryResult struct {
	Forwards []models.Forward
	Total    int64
}
