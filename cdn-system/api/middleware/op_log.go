package middleware

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type opLogContent struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Query       string `json:"query"`
	Status      int    `json:"status"`
	ContentSize int64  `json:"content_size"`
	URL         string `json:"url,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
}

// OperationLog writes op_log for mutating admin requests.
func OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodOptions {
			return
		}

		userIDAny, ok := c.Get("userID")
		if !ok {
			return
		}

		userID, ok := userIDAny.(int64)
		if !ok || userID <= 0 {
			return
		}

		role := "admin"
		if roleAny, ok := c.Get("role"); ok {
			if roleStr, ok := roleAny.(string); ok && roleStr != "" {
				role = roleStr
			}
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		content := opLogContent{
			Path:        path,
			Method:      method,
			Query:       c.Request.URL.RawQuery,
			Status:      c.Writer.Status(),
			ContentSize: c.Request.ContentLength,
		}
		if urlVal, ok := c.Get("op_log_url"); ok {
			if urlStr, ok := urlVal.(string); ok {
				content.URL = strings.TrimSpace(urlStr)
			}
		}
		if taskTypeVal, ok := c.Get("op_log_task_type"); ok {
			if taskType, ok := taskTypeVal.(string); ok {
				content.TaskType = strings.TrimSpace(taskType)
			}
		}
		payload, _ := json.Marshal(content)

		log := models.UserOperationLog{
			UserID:    userID,
			Type:      role,
			Action:    method + " " + path,
			Content:   string(payload),
			Diff:      "",
			IP:        resolveClientIP(c),
			Process:   "status=" + http.StatusText(c.Writer.Status()),
			CreatedAt: time.Now(),
		}

		_ = db.DB.Create(&log).Error
	}
}

func resolveClientIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	header := services.ResolveMasterClientIPHeader()
	if header != "" {
		raw := strings.TrimSpace(c.GetHeader(header))
		if raw != "" {
			if idx := strings.Index(raw, ","); idx != -1 {
				raw = strings.TrimSpace(raw[:idx])
			}
			if raw != "" {
				return raw
			}
		}
	}
	return c.ClientIP()
}
