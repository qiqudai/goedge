package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"strings"
	"time"
)

func WriteIPSwitchLogsForLines(lines []models.Line, action, logType string) {
	if db.DB == nil || len(lines) == 0 {
		return
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return
	}
	logType = strings.TrimSpace(logType)
	if logType == "" {
		logType = "line"
	}
	nodeIDs := make([]int64, 0, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID != 0 {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	nodeIPs := map[int64]string{}
	if len(nodeIDs) > 0 {
		var nodes []models.Node
		if err := db.DB.Select("id", "ip").Where("id IN ?", uniqueInt64List(nodeIDs)).Find(&nodes).Error; err == nil {
			for _, node := range nodes {
				if strings.TrimSpace(node.IP) == "" {
					continue
				}
				nodeIPs[node.ID] = strings.TrimSpace(node.IP)
			}
		}
	}
	now := time.Now()
	logs := make([]models.IPSwitchLog, 0, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		logs = append(logs, models.IPSwitchLog{
			CreatedAt:   now,
			Type:        logType,
			Action:      action,
			NodeGroupID: line.NodeGroupID,
			NodeID:      nodeID,
			LineID:      line.ID,
			IP:          nodeIPs[nodeID],
			Content:     buildLineSwitchContent(line),
		})
	}
	if len(logs) == 0 {
		return
	}
	_ = db.DB.Create(&logs).Error
}

func WriteIPSwitchLogForNode(node models.Node, action, logType, content string) {
	if db.DB == nil || node.ID == 0 {
		return
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return
	}
	logType = strings.TrimSpace(logType)
	if logType == "" {
		logType = "node"
	}
	log := models.IPSwitchLog{
		CreatedAt: time.Now(),
		Type:      logType,
		Action:    action,
		NodeID:    node.ID,
		IP:        strings.TrimSpace(node.IP),
		Content:   content,
	}
	_ = db.DB.Create(&log).Error
}

func buildLineSwitchContent(line models.Line) string {
	lineID := strings.TrimSpace(line.LineID)
	lineName := strings.TrimSpace(line.LineName)
	if lineID == "" && lineName == "" {
		return ""
	}
	if lineName == "" {
		return "line_id=" + lineID
	}
	if lineID == "" {
		return "line_name=" + lineName
	}
	return "line_id=" + lineID + " line_name=" + lineName
}
