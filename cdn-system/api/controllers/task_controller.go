package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("Invalid params")})
		return
	}

	input.Type = strings.TrimSpace(strings.ToLower(input.Type))
	if !isPurgeType(input.Type) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("Invalid type")})
		return
	}
	urls := splitTaskLines(input.URLs)
	if len(urls) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("URLs cannot be empty")})
		return
	}

	userID := input.UserID
	if userID == 0 || isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("user_id is required")})
		return
	}

	adminMode := !isTaskUserRequest(ctx)
	urls, err := normalizePurgeURLs(urls, adminMode, userID)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 1, "msg": T(err.Error())})
		return
	}

	if err := consumePurgeQuota(userID, input.Type, len(urls)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T(err.Error())})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to create task")})
		return
	}
	services.TriggerDispatchPending()

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("user_id is required")})
		return
	}
	limits, err := loadPurgeLimits()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to load limits")})
		return
	}
	usage, err := loadUserPurgeUsage(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to load usage")})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("Invalid id")})
		return
	}

	var task models.Task
	if err := db.DB.Where("id = ?", id).First(&task).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": T("Task not found")})
		return
	}

	meta := parseTaskMeta(task.Res)
	userID := meta.UserID
	if isTaskUserRequest(ctx) {
		userID = parseTaskUserIDAny(taskMustGet(ctx, "userID"))
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("user_id is required")})
		return
	}

	urls := splitTaskLines(task.Data)
	if len(urls) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("URLs cannot be empty")})
		return
	}
	adminMode := !isTaskUserRequest(ctx)
	urls, err := normalizePurgeURLs(urls, adminMode, userID)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 1, "msg": T(err.Error())})
		return
	}
	if err := consumePurgeQuota(userID, task.Type, len(urls)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T(err.Error())})
		return
	}
	meta = purgeTaskMeta{UserID: userID}
	metaRaw, _ := json.Marshal(meta)

	newTask := models.Task{
		Type:     task.Type,
		Data:     strings.Join(urls, "\n"),
		Res:      string(metaRaw),
		State:    "waiting",
		CreateAt: time.Now(),
		Enable:   true,
	}
	if err := db.DB.Create(&newTask).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to create task")})
		return
	}
	services.TriggerDispatchPending()
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// List returns tasks list
func (c *TaskController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	// Hard cap to keep response size and UI rendering bounded.
	if pageSize > 500 {
		pageSize = 500
	}
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
	if err := query.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to count tasks")})
		return
	}

	// Important: avoid returning huge payloads (Task.Data/Task.Res can be very large).
	// The list page only needs summary fields.
	type taskListItem struct {
		ID       int64      `json:"id"`
		PID      int64      `json:"pid"`
		Pry      int        `json:"pry"`
		Name     string     `json:"name"`
		Type     string     `json:"type"`
		Depend   string     `json:"depend"`
		CreateAt time.Time  `json:"create_at"`
		StartAt  *time.Time `json:"start_at"`
		EndAt    *time.Time `json:"end_at"`
		State    string     `json:"state"`
		ErrTimes int        `json:"err_times"`
		Progress string     `json:"progress"`
	}

	var list []taskListItem
	if err := query.Select("id,pid,pry,name,type,depend,create_at,start_at,end_at,state,err_times,progress").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to fetch list")})
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

// Get returns a single task with progress/log details (admin or the task owner).
func (c *TaskController) Get(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": T("Invalid id")})
		return
	}

	var task models.Task
	if err := db.DB.Where("id = ?", id).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": T("Task not found")})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Failed to load task")})
		return
	}

	if isTaskUserRequest(ctx) {
		userID := parseTaskUserIDAny(taskMustGet(ctx, "userID"))
		meta := parseTaskMeta(task.Res)
		if userID == 0 || meta.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":        task.ID,
			"pid":       task.PID,
			"pry":       task.Pry,
			"name":      task.Name,
			"type":      task.Type,
			"depend":    task.Depend,
			"create_at": task.CreateAt,
			"start_at":  task.StartAt,
			"end_at":    task.EndAt,
			"state":     task.State,
			"err_times": task.ErrTimes,
			"progress":  task.Progress,
			"ret":       task.Ret,
		},
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

func normalizePurgeURLs(urls []string, adminMode bool, userID int64) ([]string, error) {
	knownDomains, err := loadSiteDomains(adminMode, userID)
	if err != nil {
		return nil, errors.New("Failed to load site domains")
	}
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			return nil, errors.New(i18n.T("task.url_invalid_prefix"))
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return nil, errors.New(i18n.T("task.url_invalid_format"))
		}
		host := parsed.Hostname()
		if host == "" {
			return nil, errors.New(i18n.T("task.url_invalid_format"))
		}
		port := parsed.Port()
		if strings.Contains(host, "*") {
			suffix := strings.TrimPrefix(host, "*.")
			if suffix == host || suffix == "" {
				return nil, errors.New(i18n.T("task.wildcard_invalid"))
			}
			matches := matchWildcardDomains(knownDomains, suffix)
			if len(matches) == 0 {
				return nil, fmt.Errorf(i18n.T("task.domain_not_found"), host)
			}
			for _, domain := range matches {
				copyURL := *parsed
				if port != "" {
					copyURL.Host = domain + ":" + port
				} else {
					copyURL.Host = domain
				}
				out = append(out, copyURL.String())
			}
			continue
		}
		if !isKnownDomain(knownDomains, host, port) {
			return nil, fmt.Errorf(i18n.T("task.domain_not_found"), host)
		}
		out = append(out, raw)
	}
	return out, nil
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
	return db.DB.Omit("CreatedAt").Save(&cfg).Error
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
			return errors.New(i18n.T("task.today_refresh_url_limit"))
		}
		usage.RefreshURL += count
	case "refresh_dir":
		if exceedsLimit(limits.RefreshDir, usage.RefreshDir, count) {
			return errors.New(i18n.T("task.today_refresh_dir_limit"))
		}
		usage.RefreshDir += count
	case "preheat":
		if exceedsLimit(limits.Preheat, usage.Preheat, count) {
			return errors.New(i18n.T("task.today_preheat_limit"))
		}
		usage.Preheat += count
	}
	if err := saveUserPurgeUsage(userID, usage); err != nil {
		return errors.New("Failed to save usage")
	}
	return nil
}

type knownDomainSet struct {
	Exact map[string]struct{}
	Host  map[string]struct{}
}

func loadSiteDomains(adminMode bool, userID int64) (*knownDomainSet, error) {
	query := db.DB.Model(&models.Site{})
	if !adminMode && userID != 0 {
		query = query.Where("uid = ?", userID)
	}
	var sites []models.Site
	if err := query.Find(&sites).Error; err != nil {
		return nil, err
	}
	exact := map[string]struct{}{}
	hostOnly := map[string]struct{}{}
	for _, site := range sites {
		for _, domain := range site.Domains {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			exact[domain] = struct{}{}
			if h := splitDomainHost(domain); h != "" {
				hostOnly[h] = struct{}{}
			}
		}
	}
	return &knownDomainSet{Exact: exact, Host: hostOnly}, nil
}

func splitDomainHost(domain string) string {
	if strings.Contains(domain, "://") {
		if parsed, err := url.Parse(domain); err == nil {
			return parsed.Hostname()
		}
	}
	if idx := strings.Index(domain, ":"); idx >= 0 {
		return domain[:idx]
	}
	return domain
}

func isKnownDomain(known *knownDomainSet, host, port string) bool {
	if known == nil {
		return false
	}
	if port != "" {
		if _, ok := known.Exact[host+":"+port]; ok {
			return true
		}
	}
	if _, ok := known.Exact[host]; ok {
		return true
	}
	if _, ok := known.Host[host]; ok {
		return true
	}
	return false
}

func matchWildcardDomains(known *knownDomainSet, suffix string) []string {
	if known == nil {
		return nil
	}
	result := make([]string, 0)
	dotted := "." + suffix
	for domain := range known.Exact {
		host := splitDomainHost(domain)
		if host == suffix || strings.HasSuffix(host, dotted) {
			result = append(result, host)
		}
	}
	return result
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
