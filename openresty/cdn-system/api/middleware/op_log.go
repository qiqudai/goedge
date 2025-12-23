package middleware

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type opLogContent struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Query       string `json:"query"`
	Status      int    `json:"status"`
	ContentSize int64  `json:"content_size"`
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
		payload, _ := json.Marshal(content)

		log := models.UserOperationLog{
			UserID:    userID,
			Type:      role,
			Action:    method + " " + path,
			Content:   string(payload),
			Diff:      "",
			IP:        c.ClientIP(),
			Process:   "status=" + http.StatusText(c.Writer.Status()),
			CreatedAt: time.Now(),
		}

		_ = db.DB.Create(&log).Error
	}
}
