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

type ForwardController struct{}

type forwardListItem struct {
  ID              int64     `json:"id"`
  UserID          int64     `json:"user_id"`
  UserName        string    `json:"user_name"`
  ListenPorts     string    `json:"listen_ports"`
  OriginDisplay   string    `json:"origin_display"`
  UserPackageID   int64     `json:"user_package_id"`
  UserPackageName string    `json:"user_package_name"`
  GroupID         int64     `json:"group_id"`
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

func (ctrl *ForwardController) AdminList(c *gin.Context) {
  result, err := queryForwards(c)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forwards"})
    return
  }

  items, err := buildForwardListItems(result.Forwards)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build forwards"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"list": items, "total": result.Total})
}

func (ctrl *ForwardController) AdminCreate(c *gin.Context) {
  forward, groupID, err := parseForwardCreateRequest(c, true)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  if err := createForwardWithGroup(forward, groupID); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create forward"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"message": "Forward created", "data": forward})
}

func (ctrl *ForwardController) AdminBatchCreate(c *gin.Context) {
  var req struct {
    UserID        int64  `json:"user_id"`
    UserPackageID int64  `json:"user_package_id"`
    GroupID       int64  `json:"group_id"`
    Data          string `json:"data"`
    IgnoreError   bool   `json:"ignore_error"`
    Remark        string `json:"remark"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
    return
  }
  if req.UserID == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
    return
  }
  if strings.TrimSpace(req.Data) == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "data is required"})
    return
  }

  lines := splitLines(req.Data)
  created := 0
  for _, line := range lines {
    listenPorts, origins, err := parseForwardBatchLine(line)
    if err != nil {
      if req.IgnoreError {
        continue
      }
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }
    forward := &models.Forward{
      UserID:        req.UserID,
      UserPackageID: req.UserPackageID,
      ListenPorts:   listenPorts,
      Origins:       origins,
      Remark:        req.Remark,
      Enable:        true,
      State:         "running",
      CreatedAt:     time.Now(),
      UpdatedAt:     time.Now(),
    }
    if err := createForwardWithGroup(forward, req.GroupID); err != nil {
      if req.IgnoreError {
        continue
      }
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create forward"})
      return
    }
    created++
  }

  c.JSON(http.StatusOK, gin.H{"message": "Batch create completed", "created": created})
}

func (ctrl *ForwardController) AdminBatchUpdate(c *gin.Context) {
  var req struct {
    IDs           []int64               `json:"ids"`
    UserPackageID *int64                `json:"user_package_id"`
    GroupID       *int64                `json:"group_id"`
    ListenPorts   *[]string             `json:"listen_ports"`
    Origins       *[]models.ForwardOrigin `json:"origins"`
    Remark        *string               `json:"remark"`
    Settings      map[string]interface{} `json:"settings"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
    return
  }
  if len(req.IDs) == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
    return
  }

  err := db.DB.Transaction(func(tx *gorm.DB) error {
    updates := map[string]interface{}{}
    if req.UserPackageID != nil {
      updates["user_package"] = *req.UserPackageID
    }
    if req.ListenPorts != nil {
      updates["listen_ports"] = encodeStringList(*req.ListenPorts)
    }
    if req.Origins != nil {
      b, _ := json.Marshal(*req.Origins)
      updates["origins"] = string(b)
    }
    if req.Remark != nil {
      updates["remark"] = *req.Remark
    }
    if req.Settings != nil {
      b, _ := json.Marshal(req.Settings)
      updates["settings"] = string(b)
    }
    if len(updates) > 0 {
      if err := tx.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
        return err
      }
    }

    if req.GroupID != nil {
      if err := tx.Where("forward_id IN ?", req.IDs).Delete(&models.ForwardGroupRelation{}).Error; err != nil {
        return err
      }
      if *req.GroupID != 0 {
        relations := make([]models.ForwardGroupRelation, 0, len(req.IDs))
        for _, id := range req.IDs {
          relations = append(relations, models.ForwardGroupRelation{ForwardID: id, GroupID: *req.GroupID})
        }
        if err := tx.Create(&relations).Error; err != nil {
          return err
        }
      }
    }
    return nil
  })
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Batch update failed"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"message": "Batch update completed"})
}

func (ctrl *ForwardController) AdminBatchAction(c *gin.Context) {
  var req struct {
    Action string  `json:"action"`
    IDs    []int64 `json:"ids"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
    return
  }
  if len(req.IDs) == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
    return
  }

  switch strings.ToLower(req.Action) {
  case "enable":
    if err := db.DB.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
      "enable": true,
      "state":  "running",
    }).Error; err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
      return
    }
  case "disable":
    if err := db.DB.Model(&models.Forward{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
      "enable": false,
      "state":  "stop",
    }).Error; err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
      return
    }
  case "delete":
    err := db.DB.Transaction(func(tx *gorm.DB) error {
      if err := tx.Where("forward_id IN ?", req.IDs).Delete(&models.ForwardGroupRelation{}).Error; err != nil {
        return err
      }
      return tx.Where("id IN ?", req.IDs).Delete(&models.Forward{}).Error
    })
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
      return
    }
  default:
    c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown action"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"message": "Action completed"})
}

func parseForwardCreateRequest(c *gin.Context, admin bool) (*models.Forward, int64, error) {
  var req struct {
    UserID          int64  `json:"user_id"`
    UserPackageID   int64  `json:"user_package_id"`
    GroupID         int64  `json:"group_id"`
    ListenPorts     []string `json:"listen_ports"`
    ListenPortsInput string `json:"listen_ports_input"`
    OriginInput     string `json:"origin_input"`
    Remark          string `json:"remark"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    return nil, 0, errors.New("invalid request")
  }

  userID := req.UserID
  if !admin {
    userID = parseInt64(mustGet(c, "userID"))
  }
  if userID == 0 {
    return nil, 0, errors.New("user_id is required")
  }

  listenPorts := req.ListenPorts
  if len(listenPorts) == 0 && strings.TrimSpace(req.ListenPortsInput) != "" {
    listenPorts = splitFields(req.ListenPortsInput)
  }
  if len(listenPorts) == 0 {
    return nil, 0, errors.New("listen_ports is required")
  }

  origins := parseOrigins(req.OriginInput)
  if len(origins) == 0 {
    return nil, 0, errors.New("origin is required")
  }

  forward := &models.Forward{
    UserID:        userID,
    UserPackageID: req.UserPackageID,
    ListenPorts:   listenPorts,
    Origins:       origins,
    Remark:        req.Remark,
    Enable:        true,
    State:         "running",
    CreatedAt:     time.Now(),
    UpdatedAt:     time.Now(),
  }

  if forward.Cname == "" && len(listenPorts) > 0 {
    forward.Cname = listenPorts[0] + ".cdn.node.com"
  }

  return forward, req.GroupID, nil
}

func createForwardWithGroup(forward *models.Forward, groupID int64) error {
  return db.DB.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(forward).Error; err != nil {
      return err
    }
    if groupID != 0 {
      rel := models.ForwardGroupRelation{ForwardID: forward.ID, GroupID: groupID}
      if err := tx.Create(&rel).Error; err != nil {
        return err
      }
    }
    return nil
  })
}

func queryForwards(c *gin.Context) (*forwardQueryResult, error) {
  query := db.DB.Model(&models.Forward{})

  keyword := strings.TrimSpace(c.Query("keyword"))
  searchField := strings.TrimSpace(c.DefaultQuery("search_field", "all"))
  like := "%" + keyword + "%"

  if keyword != "" {
    switch searchField {
    case "forward_id":
      if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
        query = query.Where("id = ?", id)
      } else {
        return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
      }
    case "listen":
      query = query.Where("listen_ports LIKE ?", like)
    case "origin":
      query = query.Where("origins LIKE ?", like)
    case "cname":
      query = query.Where("cname LIKE ?", like)
    case "package":
      ids, err := findUserPackageIDsByName(keyword)
      if err != nil {
        return nil, err
      }
      if len(ids) == 0 {
        return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
      }
      query = query.Where("user_package IN ?", ids)
    case "group":
      ids, err := findForwardIDsByGroupName(keyword)
      if err != nil {
        return nil, err
      }
      if len(ids) == 0 {
        return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
      }
      query = query.Where("id IN ?", ids)
    case "user":
      userIDs, err := findUserIDsByKeyword(keyword)
      if err != nil {
        return nil, err
      }
      if len(userIDs) == 0 {
        return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
      }
      query = query.Where("uid IN ?", userIDs)
    default:
      cond := db.DB.Where("listen_ports LIKE ? OR origins LIKE ? OR cname LIKE ?", like, like, like)
      if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
        cond = cond.Or("id = ?", id)
      }
      if userIDs, err := findUserIDsByKeyword(keyword); err == nil && len(userIDs) > 0 {
        cond = cond.Or("uid IN ?", userIDs)
      }
      if pkgIDs, err := findUserPackageIDsByName(keyword); err == nil && len(pkgIDs) > 0 {
        cond = cond.Or("user_package IN ?", pkgIDs)
      }
      if forwardIDs, err := findForwardIDsByGroupName(keyword); err == nil && len(forwardIDs) > 0 {
        cond = cond.Or("id IN ?", forwardIDs)
      }
      query = query.Where(cond)
    }
  }

  if uidStr := c.Query("user_id"); uidStr != "" {
    if uid, err := strconv.Atoi(uidStr); err == nil {
      query = query.Where("uid = ?", uid)
    }
  }
  if pkgStr := c.Query("user_package_id"); pkgStr != "" {
    if id, err := strconv.Atoi(pkgStr); err == nil {
      query = query.Where("user_package = ?", id)
    }
  }
  if groupStr := c.Query("group_id"); groupStr != "" {
    if id, err := strconv.Atoi(groupStr); err == nil {
      ids, err := findForwardIDsByGroupID(int64(id))
      if err != nil {
        return nil, err
      }
      if len(ids) == 0 {
        return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
      }
      query = query.Where("id IN ?", ids)
    }
  }

  page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
  pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
  if page < 1 {
    page = 1
  }
  if pageSize < 1 {
    pageSize = 10
  }

  var total int64
  if err := query.Count(&total).Error; err != nil {
    return nil, err
  }

  var forwards []models.Forward
  if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&forwards).Error; err != nil {
    return nil, err
  }

  return &forwardQueryResult{Forwards: forwards, Total: total}, nil
}

func buildForwardListItems(forwards []models.Forward) ([]forwardListItem, error) {
  userMap, err := loadUsersForForward(forwards)
  if err != nil {
    return nil, err
  }
  pkgMap, err := loadUserPackagesForForward(forwards)
  if err != nil {
    return nil, err
  }
  groupMap, relMap, err := loadForwardGroups(forwards)
  if err != nil {
    return nil, err
  }
  nodeGroupMap, err := loadNodeGroupsForForward(forwards)
  if err != nil {
    return nil, err
  }

  items := make([]forwardListItem, 0, len(forwards))
  for _, forward := range forwards {
    originDisplay := ""
    if len(forward.Origins) > 0 {
      parts := make([]string, 0, len(forward.Origins))
      for _, o := range forward.Origins {
        parts = append(parts, o.Address)
      }
      originDisplay = strings.Join(parts, ",")
    } else if forward.OriginsRaw != "" {
      originDisplay = forward.OriginsRaw
    }

    cname := strings.TrimSpace(forward.Cname)
    if cname == "" {
      cname = "-"
    }

    item := forwardListItem{
      ID:              forward.ID,
      UserID:          forward.UserID,
      UserName:        userMap[forward.UserID],
      ListenPorts:     strings.Join(forward.ListenPorts, " "),
      OriginDisplay:   originDisplay,
      UserPackageID:   forward.UserPackageID,
      UserPackageName: pkgMap[forward.UserPackageID],
      GroupID:         relMap[forward.ID],
      GroupName:       groupMap[relMap[forward.ID]],
      NodeGroupID:     forward.NodeGroupID,
      NodeGroupName:   nodeGroupMap[forward.NodeGroupID],
      CNAME:           cname,
      Status:          forward.Enable,
      Remark:          forward.Remark,
      CreatedAt:       forward.CreatedAt,
    }
    items = append(items, item)
  }
  return items, nil
}

func parseForwardBatchLine(line string) ([]string, []models.ForwardOrigin, error) {
  parts := strings.Split(line, "|")
  if len(parts) < 2 {
    return nil, nil, errors.New("invalid batch data format")
  }
  listenPorts := splitFields(parts[0])
  if len(listenPorts) == 0 {
    return nil, nil, errors.New("listen ports required")
  }
  origins := parseOrigins(parts[1])
  if len(origins) == 0 {
    return nil, nil, errors.New("origin required")
  }
  return listenPorts, origins, nil
}

func parseOrigins(input string) []models.ForwardOrigin {
  origins := []models.ForwardOrigin{}
  for _, item := range splitFields(input) {
    origins = append(origins, models.ForwardOrigin{Address: item, Weight: 1, Enable: true})
  }
  return origins
}

func encodeStringList(items []string) string {
  if len(items) == 0 {
    return ""
  }
  b, _ := json.Marshal(items)
  return string(b)
}

func loadUsersForForward(items []models.Forward) (map[int64]string, error) {
  ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.UserID })
  result := map[int64]string{}
  if len(ids) == 0 {
    return result, nil
  }
  var users []models.User
  if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
    return nil, err
  }
  for _, u := range users {
    result[u.ID] = u.Name
  }
  return result, nil
}

func loadUserPackagesForForward(items []models.Forward) (map[int64]string, error) {
  ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.UserPackageID })
  result := map[int64]string{}
  if len(ids) == 0 {
    return result, nil
  }
  var pkgs []models.UserPackage
  if err := db.DB.Where("id IN ?", ids).Find(&pkgs).Error; err != nil {
    return nil, err
  }
  for _, p := range pkgs {
    result[p.ID] = p.Name
  }
  return result, nil
}

func loadForwardGroups(items []models.Forward) (map[int64]string, map[int64]int64, error) {
  ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.ID })
  groupMap := map[int64]string{}
  relMap := map[int64]int64{}
  if len(ids) == 0 {
    return groupMap, relMap, nil
  }
  var relations []models.ForwardGroupRelation
  if err := db.DB.Where("forward_id IN ?", ids).Find(&relations).Error; err != nil {
    return nil, nil, err
  }
  groupIDs := make([]int64, 0, len(relations))
  for _, rel := range relations {
    relMap[rel.ForwardID] = rel.GroupID
    groupIDs = append(groupIDs, rel.GroupID)
  }
  if len(groupIDs) == 0 {
    return groupMap, relMap, nil
  }
  var groups []models.ForwardGroup
  if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
    return nil, nil, err
  }
  for _, g := range groups {
    groupMap[g.ID] = g.Name
  }
  return groupMap, relMap, nil
}

func loadNodeGroupsForForward(items []models.Forward) (map[int64]string, error) {
  ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.NodeGroupID })
  result := map[int64]string{}
  if len(ids) == 0 {
    return result, nil
  }
  var groups []models.NodeGroup
  if err := db.DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
    return nil, err
  }
  for _, g := range groups {
    result[g.ID] = g.Name
  }
  return result, nil
}

func uniqueIDsForward(items []models.Forward, fn func(models.Forward) int64) []int64 {
  seen := map[int64]struct{}{}
  for _, item := range items {
    id := fn(item)
    if id != 0 {
      seen[id] = struct{}{}
    }
  }
  result := make([]int64, 0, len(seen))
  for id := range seen {
    result = append(result, id)
  }
  return result
}

func findForwardIDsByGroupName(keyword string) ([]int64, error) {
  var groups []models.ForwardGroup
  if err := db.DB.Where("name LIKE ?", "%"+keyword+"%").Find(&groups).Error; err != nil {
    return nil, err
  }
  if len(groups) == 0 {
    return []int64{}, nil
  }
  ids := make([]int64, 0, len(groups))
  for _, g := range groups {
    ids = append(ids, g.ID)
  }
  return findForwardIDsByGroupIDs(ids)
}

func findForwardIDsByGroupID(groupID int64) ([]int64, error) {
  return findForwardIDsByGroupIDs([]int64{groupID})
}

func findForwardIDsByGroupIDs(groupIDs []int64) ([]int64, error) {
  if len(groupIDs) == 0 {
    return []int64{}, nil
  }
  var relations []models.ForwardGroupRelation
  if err := db.DB.Where("group_id IN ?", groupIDs).Find(&relations).Error; err != nil {
    return nil, err
  }
  ids := make([]int64, 0, len(relations))
  for _, rel := range relations {
    ids = append(ids, rel.ForwardID)
  }
  return ids, nil
}
