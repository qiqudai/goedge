package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (ctrl *ForwardController) AdminList(c *gin.Context) {
	result, err := queryForwards(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to fetch forwards")})
		return
	}

	items, err := buildForwardListItems(result.Forwards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to build forwards")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": items, "total": result.Total})
}

func (ctrl *ForwardController) AdminCreate(c *gin.Context) {
	adminMode := true
	if isUserRequest(c) {
		adminMode = false
	}
	forward, groupIDs, err := parseForwardCreateRequest(c, adminMode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if err := createForwardWithGroup(forward, groupIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create forward")})
		return
	}

	services.BumpConfigVersion("forward", []int64{forward.ID})

	c.JSON(http.StatusOK, gin.H{"message": T("Forward created"), "data": forward})
}

func (ctrl *ForwardController) AdminBatchCreate(c *gin.Context) {
	var req struct {
		UserID        int64   `json:"user_id"`
		UserPackageID int64   `json:"user_package_id"`
		GroupID       int64   `json:"group_id"` // Desc: use group_ids
		GroupIDs      []int64 `json:"group_ids"`
		Data          string  `json:"data"`
		IgnoreError   bool    `json:"ignore_error"`
		Remark        string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if isUserRequest(c) {
		req.UserID = parseInt64(mustGet(c, "userID"))
	} else if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
		return
	}
	if strings.TrimSpace(req.Data) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("data is required")})
		return
	}

	defaults, err := services.GetStreamDefaultMap(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load defaults")})
		return
	}

	lines := splitLines(req.Data)
	created := 0
	createdIDs := make([]int64, 0)
	for _, line := range lines {
		listenPorts, origins, err := parseForwardBatchLine(line)
		if err != nil {
			if req.IgnoreError {
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}
		nodeGroupID, _ := resolveNodeGroupFromPackage(req.UserPackageID, 0)
		forward := &models.Forward{
			UserID:        req.UserID,
			UserPackageID: req.UserPackageID,
			NodeGroupID:   nodeGroupID,
			ListenPorts:   listenPorts,
			Origins:       origins,
			Remark:        req.Remark,
			BackendPort:   extractBackendPort(origins),
			Enable:        true,
			State:         "running",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		services.ApplyForwardDefaults(forward, defaults)
		groupIDs := req.GroupIDs
		if len(groupIDs) == 0 && req.GroupID != 0 {
			groupIDs = []int64{req.GroupID}
		}

		if err := createForwardWithGroup(forward, groupIDs); err != nil {
			if req.IgnoreError {
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create forward")})
			return
		}
		created++
		createdIDs = append(createdIDs, forward.ID)
	}

	if created > 0 {
		services.BumpConfigVersion("forward", createdIDs)
	}

	c.JSON(http.StatusOK, gin.H{"message": T("Batch create completed"), "created": created})
}

func (ctrl *ForwardController) AdminBatchUpdate(c *gin.Context) {
	var req struct {
		IDs           []int64                 `json:"ids"`
		UserPackageID *int64                  `json:"user_package_id"`
		GroupID       *int64                  `json:"group_id"` // Desc: use group_ids
		GroupIDs      *[]int64                `json:"group_ids"`
		ListenPorts   *[]string               `json:"listen_ports"`
		Origins       *[]models.ForwardOrigin `json:"origins"`
		Remark        *string                 `json:"remark"`
		Settings      map[string]interface{}  `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
	}
	if isUserRequest(c) {
		userID := parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		allowed, err := filterForwardIDsForUser(req.IDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load forwards")})
			return
		}
		if len(allowed) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("no forwards available")})
			return
		}
		req.IDs = allowed
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.UserPackageID != nil {
			updates["user_package"] = *req.UserPackageID
		}
		if req.ListenPorts != nil {
			updates["listen"] = encodeStringList(*req.ListenPorts)
		}
		if req.Origins != nil {
			updates["backend"] = encodeOrigins(*req.Origins)
		}
		if req.Settings != nil {
			b, _ := json.Marshal(req.Settings)
			updates["acl"] = string(b)
			applyOriginSettings(req.Settings, updates)
		}
		if req.Remark != nil {
			if _, ok := updates["acl"]; !ok {
				settings := map[string]interface{}{"remark": *req.Remark}
				b, _ := json.Marshal(settings)
				updates["acl"] = string(b)
			}
		}
		if len(updates) > 0 {
			if err := tx.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
				return err
			}
		}

		updateGroupIDs := req.GroupIDs
		if updateGroupIDs == nil && req.GroupID != nil {
			ids := []int64{*req.GroupID}
			updateGroupIDs = &ids
		}

		if updateGroupIDs != nil {
			if err := tx.Where("stream_id IN ?", req.IDs).Delete(&models.ForwardGroupRelation{}).Error; err != nil {
				return err
			}
			if len(*updateGroupIDs) > 0 {
				relations := make([]models.ForwardGroupRelation, 0, len(req.IDs)*len(*updateGroupIDs))
				for _, sid := range req.IDs {
					for _, gid := range *updateGroupIDs {
						if gid != 0 {
							relations = append(relations, models.ForwardGroupRelation{ForwardID: sid, GroupID: gid})
						}
					}
				}
				if len(relations) > 0 {
					if err := tx.Create(&relations).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Batch update failed")})
		return
	}

	services.BumpConfigVersion("forward", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": T("Batch update completed")})
}

func (ctrl *ForwardController) AdminBatchAction(c *gin.Context) {
	var req struct {
		Action string  `json:"action"`
		IDs    []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
	}
	if isUserRequest(c) {
		userID := parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		allowed, err := filterForwardIDsForUser(req.IDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load forwards")})
			return
		}
		if len(allowed) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("no forwards available")})
			return
		}
		req.IDs = allowed
	}

	switch strings.ToLower(req.Action) {
	case "enable":
		if err := db.DB.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": true,
			"state":  "running",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Update failed")})
			return
		}
	case "disable":
		if err := db.DB.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": false,
			"state":  "stop",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Update failed")})
			return
		}
	case "delete":
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("stream_id IN ?", req.IDs).Delete(&models.ForwardGroupRelation{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.IDs).Delete(&models.Forward{}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Delete failed")})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Unknown action")})
		return
	}

	services.BumpConfigVersion("forward", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": T("Action completed")})
}
