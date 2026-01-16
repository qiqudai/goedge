package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RegionController struct{}

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
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	log.Printf("[Debug] ListRegions: found %d regions", len(regions))

	metaMap := services.LoadRegionMetaMap()
	views := make([]regionView, 0, len(regions))
	for _, region := range regions {
		meta := metaMap[strconv.FormatInt(region.ID, 10)]
		l2Port := meta.L2CheckPort
		if l2Port == 0 {
			l2Port = 80
		}
		sortOrder := meta.SortOrder
		if sortOrder == 0 {
			sortOrder = 100
		}
		views = append(views, regionView{
			ID:          region.ID,
			Name:        region.Name,
			Remark:      region.Desc,
			L2CheckPort: l2Port,
			SortOrder:   sortOrder,
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("name required")})
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
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Create Failed")})
		return
	}
	metaMap := services.LoadRegionMetaMap()
	metaMap[strconv.FormatInt(region.ID, 10)] = services.RegionMeta{
		L2CheckPort: req.L2CheckPort,
		SortOrder:   req.SortOrder,
	}
	if err := services.SaveRegionMetaMap(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Save Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateRegion
// PUT /api/v1/admin/regions/:id
func (ctr *RegionController) UpdateRegion(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid id")})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Remark      string `json:"remark"`
		L2CheckPort int    `json:"l2_check_port"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("name required")})
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
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}
	metaMap := services.LoadRegionMetaMap()
	metaMap[strconv.FormatInt(id, 10)] = services.RegionMeta{
		L2CheckPort: req.L2CheckPort,
		SortOrder:   req.SortOrder,
	}
	if err := services.SaveRegionMetaMap(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Save Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// DeleteRegion
// DELETE /api/v1/admin/regions/:id
func (ctr *RegionController) DeleteRegion(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid id")})
		return
	}
	var nodeCount int64
	if err := db.DB.Model(&models.Node{}).Where("region_id = ?", id).Count(&nodeCount).Error; err == nil {
		if nodeCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("region.has_nodes")})
			return
		}
	}
	if err := db.DB.Delete(&models.Region{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Delete Failed")})
		return
	}
	metaMap := services.LoadRegionMetaMap()
	delete(metaMap, strconv.FormatInt(id, 10))
	if err := services.SaveRegionMetaMap(metaMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Save Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}
