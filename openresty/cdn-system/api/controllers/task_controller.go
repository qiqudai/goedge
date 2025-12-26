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

type TaskController struct{}

type purgeTaskMeta struct {
	UserID int64 `json:"user_id"`
}

type purgeUsage struct {
	Date       string `json:"date"`
	RefreshURL int    `json:"refresh_url"`
	RefreshDir int    `json:"refresh_dir"`
	Preheat    int    `json:"preheat"`
}

type purgeLimit struct {
	RefreshURL int `json:"refresh_url"`
	RefreshDir int `json:"refresh_dir"`
	Preheat    int `json:"preheat"`
}

// Create handles purge/preheat task creation
func (c *TaskController) Create(ctx *gin.Context) {
	var input struct {
		Type   string `json:"type"` // refresh_url, refresh_dir, preheat
		URLs   string `json:"urls"` // newline separated
		UserID int64  `json:"user_id"`
	}
	if err := ctx.BindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid params"})
		return
	}

	input.Type = strings.TrimSpace(strings.ToLower(input.Type))
	if !isPurgeType(input.Type) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid type"})
		return
	}
	urls := splitTaskLines(input.URLs)
	if len(urls) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "URLs cannot be empty"})
		return
	}
	if err := validateURLs(urls); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	userID := input.UserID
	if userID == 0 || isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "user_id is required"})
		return
	}

	if err := consumePurgeQuota(userID, input.Type, len(urls)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	meta := purgeTaskMeta{UserID: userID}
	metaRaw, _ := json.Marshal(meta)

	task := models.Task{
		Type:     input.Type,
		Data:     strings.Join(urls, "\n"),
		Res:      string(metaRaw),
		State:    "waiting",
		CreateAt: time.Now(),
		Enable:   true,
	}

	if err := db.DB.Create(&task).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to create task"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

// Usage returns daily limits and remaining counts
func (c *TaskController) Usage(ctx *gin.Context) {
	userID := int64(0)
	if uidStr := strings.TrimSpace(ctx.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			userID = uid
		}
	}
	if userID == 0 || isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "user_id is required"})
		return
	}
	limits, err := loadPurgeLimits()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to load limits"})
		return
	}
	usage, err := loadUserPurgeUsage(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to load usage"})
		return
	}
	remaining := purgeLimit{
		RefreshURL: calcRemaining(limits.RefreshURL, usage.RefreshURL),
		RefreshDir: calcRemaining(limits.RefreshDir, usage.RefreshDir),
		Preheat:    calcRemaining(limits.Preheat, usage.Preheat),
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"limits":    limits,
			"used":      usage,
			"remaining": remaining,
		},
	})
}

// Resubmit creates a new task from an existing one
func (c *TaskController) Resubmit(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid id"})
		return
	}

	var task models.Task
	if err := db.DB.Where("id = ?", id).First(&task).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "Task not found"})
		return
	}

	meta := parseTaskMeta(task.Res)
	userID := meta.UserID
	if isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "user_id is required"})
		return
	}

	urls := splitTaskLines(task.Data)
	if len(urls) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "URLs cannot be empty"})
		return
	}
	if err := validateURLs(urls); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if err := consumePurgeQuota(userID, task.Type, len(urls)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	meta = purgeTaskMeta{UserID: userID}
	metaRaw, _ := json.Marshal(meta)

	newTask := models.Task{
		Type:     task.Type,
		Data:     task.Data,
		Res:      string(metaRaw),
		State:    "waiting",
		CreateAt: time.Now(),
		Enable:   true,
	}
	if err := db.DB.Create(&newTask).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to create task"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

// List returns tasks list
func (c *TaskController) List(ctx *gin.Context) {
	var list []models.Task
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	keyword := ctx.Query("keyword")
	taskType := ctx.Query("type")
	userID := int64(0)
	if uidStr := strings.TrimSpace(ctx.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			userID = uid
		}
	}
	if isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}

	query := db.DB.Model(&models.Task{})

	if keyword != "" {
		query = query.Where("data LIKE ?", "%"+keyword+"%")
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if userID != 0 {
		query = query.Where("res LIKE ?", "%\"user_id\":"+strconv.FormatInt(userID, 10)+"%")
	}

	var total int64
	query.Count(&total)

	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to fetch list"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
			"page":  page,
		},
		"list":  list,
		"total": total,
		"page":  page,
	})
}

func isTaskUserRequest(ctx *gin.Context) bool {
	path := ctx.FullPath()
	if strings.HasPrefix(path, "/api/v1/user/") {
		return true
	}
	return strings.HasPrefix(ctx.Request.URL.Path, "/api/v1/user/")
}

func parseTaskMeta(raw string) purgeTaskMeta {
	if raw == "" {
		return purgeTaskMeta{}
	}
	var meta purgeTaskMeta
	_ = json.Unmarshal([]byte(raw), &meta)
	return meta
}

func splitTaskLines(input string) []string {
	parts := strings.Split(strings.ReplaceAll(strings.ReplaceAll(input, "\r\n", "\n"), "\r", "\n"), "\n")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func validateURLs(urls []string) error {
	for _, raw := range urls {
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			return errors.New("url must start with http:// or https://")
		}
	}
	return nil
}
func isPurgeType(val string) bool {
	switch val {
	case "refresh_url", "refresh_dir", "preheat":
		return true
	default:
		return false
	}
}

func loadPurgeLimits() (purgeLimit, error) {
	limits := purgeLimit{
		RefreshURL: 2000,
		RefreshDir: 500,
		Preheat:    2000,
	}
	var configs []models.SysConfig
	if err := db.DB.Where("type = ? AND name IN ?", "site", []string{"clean_url", "clean_dir", "pre_cache_url"}).Find(&configs).Error; err != nil {
		return limits, err
	}
	for _, cfg := range configs {
		val, _ := strconv.Atoi(strings.TrimSpace(cfg.Value))
		switch cfg.Name {
		case "clean_url":
			if val > 0 {
				limits.RefreshURL = val
			}
		case "clean_dir":
			if val > 0 {
				limits.RefreshDir = val
			}
		case "pre_cache_url":
			if val > 0 {
				limits.Preheat = val
			}
		}
	}
	return limits, nil
}

func loadUserPurgeUsage(userID int64) (purgeUsage, error) {
	usage := purgeUsage{Date: today()}
	var cfg models.SysConfig
	err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "purge_usage", "user", "user", userID).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usage, nil
		}
		return usage, err
	}
	if err := json.Unmarshal([]byte(cfg.Value), &usage); err != nil {
		return purgeUsage{Date: today()}, nil
	}
	if usage.Date != today() {
		return purgeUsage{Date: today()}, nil
	}
	return usage, nil
}

func saveUserPurgeUsage(userID int64, usage purgeUsage) error {
	usage.Date = today()
	raw, _ := json.Marshal(usage)
	var cfg models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "purge_usage", "user", "user", userID)
	if err := query.First(&cfg).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		cfg = models.SysConfig{
			Name:      "purge_usage",
			Value:     string(raw),
			Type:      "user",
			ScopeID:   int(userID),
			ScopeName: "user",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return db.DB.Create(&cfg).Error
	}
	cfg.Value = string(raw)
	cfg.UpdatedAt = time.Now()
	return db.DB.Save(&cfg).Error
}

func consumePurgeQuota(userID int64, taskType string, count int) error {
	if count <= 0 {
		return nil
	}
	limits, err := loadPurgeLimits()
	if err != nil {
		return errors.New("Failed to load limits")
	}
	usage, err := loadUserPurgeUsage(userID)
	if err != nil {
		return errors.New("Failed to load usage")
	}
	switch taskType {
	case "refresh_url":
		if exceedsLimit(limits.RefreshURL, usage.RefreshURL, count) {
			return errors.New("今日刷新URL次数不足")
		}
		usage.RefreshURL += count
	case "refresh_dir":
		if exceedsLimit(limits.RefreshDir, usage.RefreshDir, count) {
			return errors.New("今日刷新目录次数不足")
		}
		usage.RefreshDir += count
	case "preheat":
		if exceedsLimit(limits.Preheat, usage.Preheat, count) {
			return errors.New("今日预热次数不足")
		}
		usage.Preheat += count
	}
	if err := saveUserPurgeUsage(userID, usage); err != nil {
		return errors.New("Failed to save usage")
	}
	return nil
}

func exceedsLimit(limit int, used int, add int) bool {
	if limit <= 0 {
		return false
	}
	return used+add > limit
}

func calcRemaining(limit int, used int) int {
	if limit <= 0 {
		return 0
	}
	remain := limit - used
	if remain < 0 {
		return 0
	}
	return remain
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func parseTaskUserIDAny(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			return id
		}
	}
	return 0
}

func taskMustGet(ctx *gin.Context, key string) interface{} {
	if val, ok := ctx.Get(key); ok {
		return val
	}
	return nil
}
