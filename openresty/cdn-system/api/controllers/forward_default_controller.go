package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ForwardDefaultController struct{}

type forwardDefaultItem struct {
  ID       int64       `json:"id"`
  Key      string      `json:"key"`
  Value    interface{} `json:"value"`
  Scope    string      `json:"scope"`
  GroupID  int64       `json:"group_id"`
  GroupName string     `json:"group_name"`
}

const forwardDefaultKey = "forward_default_settings"

func (ctrl *ForwardDefaultController) List(c *gin.Context) {
  items, err := loadForwardDefaultItems()
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load settings"})
    return
  }

  if len(items) > 0 {
    groupMap, _ := loadForwardGroupMap(items)
    for i := range items {
      if items[i].GroupID != 0 {
        items[i].GroupName = groupMap[items[i].GroupID]
      }
    }
  }

  c.JSON(http.StatusOK, gin.H{"list": items})
}

func (ctrl *ForwardDefaultController) Create(c *gin.Context) {
  var req struct {
    Key     string      `json:"key"`
    Value   interface{} `json:"value"`
    Scope   string      `json:"scope"`
    GroupID int64       `json:"group_id"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
    return
  }
  items, _ := loadForwardDefaultItems()
  items = append(items, forwardDefaultItem{
    ID:      time.Now().UnixNano(),
    Key:     req.Key,
    Value:   req.Value,
    Scope:   req.Scope,
    GroupID: req.GroupID,
  })
  if err := saveForwardDefaultItems(items); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (ctrl *ForwardDefaultController) Delete(c *gin.Context) {
  var req struct {
    ID int64 `json:"id"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
    return
  }
  items, _ := loadForwardDefaultItems()
  nextItems := make([]forwardDefaultItem, 0, len(items))
  for _, item := range items {
    if item.ID != req.ID {
      nextItems = append(nextItems, item)
    }
  }
  if err := saveForwardDefaultItems(nextItems); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func loadForwardDefaultItems() ([]forwardDefaultItem, error) {
  var cfg models.SysConfig
  if err := db.DB.Where("name = ? AND type = ?", forwardDefaultKey, "system").First(&cfg).Error; err != nil {
    return []forwardDefaultItem{}, nil
  }
  var items []forwardDefaultItem
  if cfg.Value != "" {
    _ = json.Unmarshal([]byte(cfg.Value), &items)
  }
  return items, nil
}

func saveForwardDefaultItems(items []forwardDefaultItem) error {
  data, _ := json.Marshal(items)
  var cfg models.SysConfig
  if err := db.DB.Where("name = ? AND type = ?", forwardDefaultKey, "system").First(&cfg).Error; err != nil {
    // create
    cfg = models.SysConfig{
        Name:      forwardDefaultKey,
        Value:     string(data),
        Type:      "system",
        ScopeID:   0,
        ScopeName: "global",
        Enable:    true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    return db.DB.Create(&cfg).Error
  }
  // update
  cfg.Value = string(data)
  cfg.UpdatedAt = time.Now()
  return db.DB.Save(&cfg).Error
}

func loadForwardGroupMap(items []forwardDefaultItem) (map[int64]string, error) {
  groupIDs := make([]int64, 0)
  seen := map[int64]struct{}{}
  for _, item := range items {
    if item.GroupID != 0 {
      if _, ok := seen[item.GroupID]; !ok {
        seen[item.GroupID] = struct{}{}
        groupIDs = append(groupIDs, item.GroupID)
      }
    }
  }
  result := map[int64]string{}
  if len(groupIDs) == 0 {
    return result, nil
  }
  var groups []models.ForwardGroup
  if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
    return nil, err
  }
  for _, g := range groups {
    result[g.ID] = g.Name
  }
  return result, nil
}
