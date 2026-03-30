package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"strconv"
	"strings"
	"time"
)

func WriteNodeMonitorLog(nodeID int64, logType string, success bool, ip string) {
	if nodeID == 0 {
		return
	}
	WriteNodeMonitorLogs([]int64{nodeID}, logType, success, map[int64]string{nodeID: ip})
}

func WriteNodeMonitorLogs(nodeIDs []int64, logType string, success bool, ipMap map[int64]string) {
	if db.DB == nil || len(nodeIDs) == 0 {
		return
	}
	logType = strings.TrimSpace(logType)
	if logType == "" {
		logType = "heartbeat"
	}
	now := time.Now()
	eventID := strconv.FormatInt(now.Unix()/30, 10)
	successValue := "0"
	if success {
		successValue = "1"
	}
	logs := make([]models.NodeMonitorLog, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		if nodeID == 0 {
			continue
		}
		logs = append(logs, models.NodeMonitorLog{
			CreateAt: now,
			Type:     logType,
			EventID:  eventID,
			IP:       strings.TrimSpace(ipMap[nodeID]),
			Success:  successValue,
			NodeID:   nodeID,
		})
	}
	if len(logs) == 0 {
		return
	}
	_ = db.DB.Create(&logs).Error
}
