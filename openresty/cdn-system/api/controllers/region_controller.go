package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RegionController struct{}

type regionMeta struct {
	L2CheckPort int `json:"l2_check_port"`
	SortOrder   int `json:"sort_order"`
}

type regionView struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Remark      string    `json:"remark"`
	L2CheckPort int       `json:"l2_check_port"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"create_at"`
	UpdatedAt   time.Time `json:"update_at"`
}

// ListRegions
// GET /api/v1/admin/regions
func (ctr *RegionController) ListRegions(c *gin.Context) {
	var regions []models.Region
	if err := db.DB.Order("id asc").Find(&regions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	metaMap := loadRegionMeta()
	views := make([]regionView, 0, len(regions))
	for _, region := range regions {
		meta := metaMap[strconv.FormatInt(region.ID, 10)]
		views = append(views, regionView{
			ID:          region.ID,
			Name:        region.Name,
			Remark:      region.Desc,
			L2CheckPort: meta.L2CheckPort,
			SortOrder:   meta.SortOrder,
			CreatedAt:   region.CreatedAt,
			UpdatedAt:   region.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  views,
			"total": len(views),
		},
	})
}

// CreateRegion
// POST /api/v1/admin/regions
func (ctr *RegionController) CreateRegion(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Remark      string `json:"remark"`
		L2CheckPort int    `json:"l2_check_port"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "name required"})
		return
	}
	if req.L2CheckPort == 0 {
		req.L2CheckPort = 80
	}
	if req.SortOrder == 0 {
		req.SortOrder = 100
	}

	region := models.Region{
		Name:      req.Name,
		Desc:      req.Remark,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.DB.Create(&region).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
	}
	metaMap := loadRegionMeta()
	metaMap[strconv.FormatInt(region.ID, 10)] = regionMeta{
		L2CheckPort: req.L2CheckPort,
		SortOrder:   req.SortOrder,
	}
	if err := saveRegionMeta(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Save Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateRegion
// PUT /api/v1/admin/regions/:id
func (ctr *RegionController) UpdateRegion(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid id"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Remark      string `json:"remark"`
		L2CheckPort int    `json:"l2_check_port"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "name required"})
		return
	}
	if req.L2CheckPort == 0 {
		req.L2CheckPort = 80
	}
	if req.SortOrder == 0 {
		req.SortOrder = 100
	}

	if err := db.DB.Model(&models.Region{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":      req.Name,
		"des":       req.Remark,
		"update_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}
	metaMap := loadRegionMeta()
	metaMap[strconv.FormatInt(id, 10)] = regionMeta{
		L2CheckPort: req.L2CheckPort,
		SortOrder:   req.SortOrder,
	}
	if err := saveRegionMeta(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Save Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// DeleteRegion
// DELETE /api/v1/admin/regions/:id
func (ctr *RegionController) DeleteRegion(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid id"})
		return
	}
	if err := db.DB.Delete(&models.Region{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Delete Failed"})
		return
	}
	metaMap := loadRegionMeta()
	delete(metaMap, strconv.FormatInt(id, 10))
	if err := saveRegionMeta(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Save Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func loadRegionMeta() map[string]regionMeta {
	var item models.ConfigItem
	err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		First(&item).Error
	if err != nil {
		return map[string]regionMeta{}
	}
	if item.Value == "" {
		return map[string]regionMeta{}
	}
	metaMap := map[string]regionMeta{}
	if jsonErr := json.Unmarshal([]byte(item.Value), &metaMap); jsonErr != nil {
		return map[string]regionMeta{}
	}
	return metaMap
}

func saveRegionMeta(metaMap map[string]regionMeta) error {
	b, err := json.Marshal(metaMap)
	if err != nil {
		return err
	}
	var item models.ConfigItem
	err = db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newItem := models.ConfigItem{
				Name:      "region_meta",
				Value:     string(b),
				Type:      "system",
				ScopeID:   0,
				ScopeName: "global",
				Enable:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return db.DB.Create(&newItem).Error
		}
		return err
	}
	return db.DB.Model(&models.ConfigItem{}).
		Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		Updates(map[string]interface{}{
			"value":     string(b),
			"update_at": time.Now(),
		}).Error
}
