package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"strconv"
	"strings"
)

func parseLineWeight(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	weight, err := strconv.Atoi(trimmed)
	if err != nil || weight < 0 {
		return 0
	}
	return weight
}

func loadLineWeightMap(groupID int64, lineID string, nodeIDs []int64) map[int64]int {
	if groupID == 0 || len(nodeIDs) == 0 || db.DB == nil {
		return map[int64]int{}
	}
	var lines []models.Line
	_ = db.DB.Select("node_id", "node_ip_id", "weight").
		Where("node_group_id = ? AND line_id = ? AND (node_id IN ? OR node_ip_id IN ?)", groupID, lineID, nodeIDs, nodeIDs).
		Find(&lines).Error

	weights := make(map[int64]int, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID == 0 {
			continue
		}
		weights[nodeID] = parseLineWeight(line.Weight)
	}
	return weights
}
