package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PackageController struct{}

type uploadVersionReq struct {
	Version string `form:"version" json:"version"`
}

func (ctr *PackageController) ListVersions(c *gin.Context) {
	list, err := services.ListAgentPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	resp := make([]gin.H, 0, len(list))
	for _, item := range list {
		resp = append(resp, gin.H{
			"version":      item.Version,
			"status":       item.Status,
			"gray_percent": item.GrayPercent,
			"upload_time":  item.UploadTime,
			"filename":     item.Filename,
			"size":         item.Size,
			"sha256":       item.Sha256,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": resp}})
}

func (ctr *PackageController) UploadVersion(c *gin.Context) {
	var req uploadVersionReq
	_ = c.ShouldBind(&req)
	version := strings.TrimSpace(req.Version)
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("version is required")})
		return
	}
	if !isValidVersionToken(version) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid version format")})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("file is required")})
		return
	}
	ext, ok := normalizePackageExt(file.Filename)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid file type")})
		return
	}
	dir, err := services.EnsureAgentPackageDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to save file")})
		return
	}
	filename := fmt.Sprintf("agent_%s%s", version, ext)
	targetPath := filepath.Join(dir, filename)
	tmpPath := targetPath + ".tmp"
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to save file")})
		return
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to save file")})
		return
	}
	size, sha256Sum := fileMeta(targetPath)
	pkg := services.AgentPackage{
		Version:     version,
		Filename:    filename,
		Status:      "normal",
		GrayPercent: 0,
		UploadTime:  time.Now(),
		Size:        size,
		Sha256:      sha256Sum,
	}
	if err := services.UpsertAgentPackage(pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Version Uploaded"), "data": pkg})
}

func (ctr *PackageController) UpdateGrayScale(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
		Percent int    `json:"percent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Version) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid request")})
		return
	}
	if err := services.UpdateAgentPackageGray(strings.TrimSpace(req.Version), req.Percent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Gray Scale Config Updated")})
}

func (ctr *PackageController) SetStable(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Version) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid request")})
		return
	}
	if err := services.SetAgentPackageStable(strings.TrimSpace(req.Version)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

func (ctr *PackageController) ListNodes(c *gin.Context) {
	preferredVersion := strings.TrimSpace(c.Query("version"))
	latest, _ := services.ResolveLatestAgentVersion(preferredVersion)

	var nodes []models.Node
	if err := db.DB.Model(&models.Node{}).
		Where("pid = 0").
		Select("id", "name", "ip", "region_id", "enable").
		Order("id asc").
		Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	nodeIDs := make([]int64, 0, len(nodes))
	regionIDs := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
		if node.RegionID != nil && *node.RegionID > 0 {
			regionIDs = append(regionIDs, *node.RegionID)
		}
	}

	regionMap := map[int64]string{}
	if len(regionIDs) > 0 {
		var regions []models.Region
		_ = db.DB.Select("id", "name").Find(&regions, regionIDs).Error
		for _, r := range regions {
			regionMap[r.ID] = r.Name
		}
	}

	groupNameMap := map[int64]string{}
	if len(nodeIDs) > 0 {
		var lines []struct {
			NodeID      int64 `gorm:"column:node_id"`
			NodeGroupID int64 `gorm:"column:node_group_id"`
		}
		_ = db.DB.Model(&models.Line{}).
			Select("node_id, node_group_id").
			Where("node_id IN ?", nodeIDs).
			Find(&lines).Error
		groupIDs := make([]int64, 0)
		nodeGroupMap := map[int64]map[int64]struct{}{}
		for _, line := range lines {
			if line.NodeGroupID == 0 {
				continue
			}
			if _, ok := nodeGroupMap[line.NodeID]; !ok {
				nodeGroupMap[line.NodeID] = map[int64]struct{}{}
			}
			if _, ok := nodeGroupMap[line.NodeID][line.NodeGroupID]; !ok {
				nodeGroupMap[line.NodeID][line.NodeGroupID] = struct{}{}
				groupIDs = append(groupIDs, line.NodeGroupID)
			}
		}
		groupNameLookup := map[int64]string{}
		if len(groupIDs) > 0 {
			var groups []models.NodeGroup
			_ = db.DB.Select("id", "name").Where("id IN ?", groupIDs).Find(&groups).Error
			for _, g := range groups {
				groupNameLookup[g.ID] = g.Name
			}
		}
		for nodeID, groupSet := range nodeGroupMap {
			names := make([]string, 0, len(groupSet))
			for gid := range groupSet {
				if name := groupNameLookup[gid]; name != "" {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			groupNameMap[nodeID] = strings.Join(names, ", ")
		}
	}

	versionMap, _ := services.GetNodeConfigMap("agent_version")
	result := make([]gin.H, 0, len(nodes))
	for _, node := range nodes {
		current := strings.TrimSpace(versionMap[node.ID])
		status := "idle"
		if latest != "" && current != "" {
			if services.CompareVersion(latest, current) > 0 {
				status = "upgrade_available"
			} else {
				status = "up_to_date"
			}
		}
		regionName := ""
		if node.RegionID != nil {
			regionName = regionMap[*node.RegionID]
		}
		result = append(result, gin.H{
			"id":              node.ID,
			"name":            node.Name,
			"ip":              node.IP,
			"region_id":       node.RegionID,
			"region_name":     regionName,
			"group_name":      groupNameMap[node.ID],
			"current_version": current,
			"latest_version":  latest,
			"status":          status,
			"online":          services.IsNodeOnline(node.ID, 30*time.Second),
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": result}})
}

func (ctr *PackageController) SyncVersion(c *gin.Context) {
	var req struct {
		Version   string  `json:"version"`
		NodeIDs   []int64 `json:"node_ids"`
		GroupIDs  []int64 `json:"group_ids"`
		RegionIDs []int64 `json:"region_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Version) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid request")})
		return
	}
	pkg, err := services.GetAgentPackage(strings.TrimSpace(req.Version))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("version not found")})
		return
	}
	nodeSet := map[int64]struct{}{}
	for _, id := range req.NodeIDs {
		if id > 0 {
			nodeSet[id] = struct{}{}
		}
	}
	if len(req.GroupIDs) > 0 {
		var groupNodeIDs []int64
		_ = db.DB.Model(&models.Line{}).
			Select("distinct node_id").
			Where("node_group_id IN ?", req.GroupIDs).
			Pluck("node_id", &groupNodeIDs).Error
		for _, id := range groupNodeIDs {
			if id > 0 {
				nodeSet[id] = struct{}{}
			}
		}
	}
	if len(req.RegionIDs) > 0 {
		var regionNodeIDs []int64
		_ = db.DB.Model(&models.Node{}).
			Select("id").
			Where("pid = 0 AND region_id IN ?", req.RegionIDs).
			Pluck("id", &regionNodeIDs).Error
		for _, id := range regionNodeIDs {
			if id > 0 {
				nodeSet[id] = struct{}{}
			}
		}
	}

	nodeIDs := make([]int64, 0, len(nodeSet))
	for id := range nodeSet {
		nodeIDs = append(nodeIDs, id)
	}
	if len(nodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("node_ids is required")})
		return
	}
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })

	apiBase := services.ResolveAPIBaseURL(c.Request)
	downloadURL := ""
	if apiBase != "" {
		downloadURL = fmt.Sprintf("%s/api/v1/agent/upgrade/package?version=%s", apiBase, urlEncode(pkg.Version))
	}

	payload := map[string]interface{}{
		"version":      pkg.Version,
		"file_name":    pkg.Filename,
		"sha256":       pkg.Sha256,
		"download_url": downloadURL,
	}
	payloadRaw, _ := json.Marshal(payload)
	targets := services.NewTaskTargets(nodeIDs)

	task := models.Task{
		Type:        "agent_upgrade",
		Name:        "Agent Upgrade " + pkg.Version,
		Data:        string(payloadRaw),
		TargetsJSON: targets.Marshal(),
		State:       "waiting",
		Enable:      true,
		CreateAt:    time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Create Failed")})
		return
	}
	services.TriggerDispatchPending()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"task_id": task.ID}})
}

func (ctr *PackageController) UpgradeStatus(c *gin.Context) {
	taskID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("task_id")), 10, 64)
	if taskID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("task_id is required")})
		return
	}
	var task models.Task
	if err := db.DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("Task not found")})
		return
	}
	targets := services.ParseTaskTargets(task.TargetsJSON)
	nodes := make([]gin.H, 0, len(targets.Nodes))
	for nodeID, target := range targets.Nodes {
		progress := target.Progress
		message := target.Message
		if progress == 0 && strings.TrimSpace(target.Ret) != "" {
			var payload struct {
				Progress int    `json:"progress"`
				Message  string `json:"message"`
			}
			if json.Unmarshal([]byte(target.Ret), &payload) == nil {
				if payload.Progress > 0 {
					progress = payload.Progress
				}
				if payload.Message != "" {
					message = payload.Message
				}
			}
		}
		nodes = append(nodes, gin.H{
			"node_id":  nodeID,
			"state":    target.State,
			"progress": progress,
			"message":  message,
			"ret":      target.Ret,
			"last_at":  target.LastAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"task_id": task.ID,
		"state":   task.State,
		"nodes":   nodes,
	}})
}

func (ctr *PackageController) DownloadPackage(c *gin.Context) {
	version := strings.TrimSpace(c.Query("version"))
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("version is required")})
		return
	}
	pkg, err := services.GetAgentPackage(version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("version not found")})
		return
	}
	dir, err := services.ResolveAgentPackageDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("file not found")})
		return
	}
	path := filepath.Join(dir, pkg.Filename)
	if !fileExists(path) {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("file not found")})
		return
	}
	c.FileAttachment(path, pkg.Filename)
}

func fileMeta(path string) (int64, string) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, ""
	}
	fp, err := os.Open(path)
	if err != nil {
		return info.Size(), ""
	}
	defer fp.Close()
	h := sha256.New()
	_, _ = io.Copy(h, fp)
	return info.Size(), hex.EncodeToString(h.Sum(nil))
}

func isValidVersionToken(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	for _, r := range version {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func normalizePackageExt(name string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(lower, ".tar.gz") {
		return ".tar.gz", true
	}
	if strings.HasSuffix(lower, ".zip") {
		return ".zip", true
	}
	return "", false
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func urlEncode(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, " ", "%20"), "+", "%2B")
}
