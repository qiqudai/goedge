package controllers

import (
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SystemInfoController struct{}

type SystemInfoPayload struct {
	SysName           string `json:"sys_name"`
	UserConsoleTitle  string `json:"user_console_title"`
	AdminConsoleTitle string `json:"admin_console_title"`
	FooterLink        string `json:"footer_link"`
	FooterCopyright   string `json:"footer_copyright"`
	FaviconFile       string `json:"favicon_file"`
	LogoFile          string `json:"logo_file"`
	LoginAdFile       string `json:"login_ad_file"`
	EnableEmailLogin  bool   `json:"enable_email_login"`
	EnableSMSLogin    bool   `json:"enable_sms_login"`
	AllowRegister     bool   `json:"allow_register"`
}

func (ctr *SystemInfoController) Get(c *gin.Context) {
	var payload SystemInfoPayload
	cfg, err := services.LoadSystemConfig()
	if err == nil {
		raw := strings.TrimSpace(cfg["system_info"])
		if raw != "" {
			_ = json.Unmarshal([]byte(raw), &payload)
		}
		payload.EnableEmailLogin = services.ParseBoolFlag(cfg["allow-enable-email-captcha-login"])
		payload.EnableSMSLogin = services.ParseBoolFlag(cfg["allow-enable-sms-captcha-login"])
		payload.AllowRegister = services.ParseBoolFlag(cfg["allow_register"])
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": payload,
	})
}
