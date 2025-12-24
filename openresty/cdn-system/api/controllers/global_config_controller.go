package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type GlobalConfigController struct{}

var (
	defaultConfigOnce sync.Once
)

const GlobalConfigKey = "global_config"

// Helper to load error page content
func loadErrorPage(filename string, fallback string) string {
	path := "e:\\cdn\\goedge\\openresty\\public\\error\\" + filename
	// In production, might want to make this path configurable or relative
	content, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(content)
}

func getDefaultConfig() models.GlobalConfig {
	return models.GlobalConfig{
		WAF: models.WAFConfig{
			Enable:             true,
			DefaultBlockAction: "disconnect",
			AutoIPSetEnable:    true,
			AutoIPSetThreshold: 200,

			BlockPageRateLimitEnable: true,
			BlockPageRateLimit:       200,

			BlacklistTimeout:     3600,
			TempWhitelistTimeout: 21600,

			TempWhitelistLimitTotal: 400,
			TempWhitelistLimitURL:   50,

			PreventTLSHandshake: true,
			BlockUnboundDomain:  true,
			DisablePing:         false,

			DefaultPageProtection:          "auto",
			DefaultPageProtectionThreshold: 100,

			SecretKey:            "KPS1CC6oGp",
			NodeLogCleanStrategy: "none",
			CCRuleAutoSwitch:     false,

			AntiCCImageSource: "system",
			AntiCCType:        "slide",

			WellKnownProtectionThreshold: 600,

			ResourceProtectionEnable:       false,
			ResourceProtectionThreshold:    50,
			ResourceProtectionBlockTimeout: 3600,
			ResourceProtectionRules: []models.ResourceRule{
				{Duration: 120, MaxRequests: 20},
			},
		},
		Nginx: models.NginxConfig{
			WorkerProcesses:       "auto",
			WorkerConnections:     51200,
			WorkerRlimitNofile:    51200,
			WorkerShutdownTimeout: "60s",
			LogDirectory:          "/usr/local/openresty/nginx/logs/",
			KeepaliveTimeout:      60,
			Gzip:                  true,
		},
		DefaultConfig: models.DefaultSiteConfig{
			Website:  models.SiteTemplate{CacheEnable: true, CacheTTL: 86400, Gzip: true, WAFEnable: true},
			API:      models.SiteTemplate{CacheEnable: false, CacheTTL: 0, Gzip: true, WAFEnable: true},
			Download: models.SiteTemplate{CacheEnable: false, CacheTTL: 0, Gzip: false, WAFEnable: true},
		},
		Resources: models.GlobalResourceConfig{
			Website: models.WebsiteResourceConfig{
				MinLimit:              1000,
				MaxLimitMultiplier:    200,
				MaxBlacklistIPs:       50,
				MaxWhitelistIPs:       50,
				DailyURLPurgeLimit:    2000,
				DailyDirPurgeLimit:    500,
				DailyPreloadLimit:     2000,
				DailyUnlockIPLimit:    1000,
				UnlockIPBatchLimit:    50,
				MaxCCRulesPerGroup:    5,
				MaxACLRules:           5,
				DailyLogDownloadLimit: 10,
				LogStorageDir:         "/data/download-temp/",
				LogStorageHours:       12,
				MaxDomainsPerSite:     100,
				DefaultListen80:       true,
			},
			Forward: models.ForwardResourceConfig{
				DisabledPorts:      "80 443",
				MinLimit:           1000,
				MaxLimitMultiplier: 200,
				MaxACLRules:        10,
			},
			Public: models.PublicResourceConfig{
				DisabledCustomPorts: "22",
				AllowedCustomPorts:  "1-65535",
			},
		},
		ErrorPages: map[string]string{
			"400":            loadErrorPage("400.txt", "<html><body><h1>400 Bad Request</h1><p>Our systems have detected unusual traffic.</p></body></html>"),
			"403":            loadErrorPage("403.txt", "<html><body><h1>403 Forbidden</h1><p>Access Denied.</p></body></html>"),
			"502":            loadErrorPage("502.txt", "<html><body><h1>502 Bad Gateway</h1><p>The server is busy.</p></body></html>"),
			"504":            loadErrorPage("504.txt", "<html><body><h1>504 Gateway Timeout</h1><p>The origin server did not respond.</p></body></html>"),
			"traffic_limit":  loadErrorPage("llout.txt", "<h1>Traffic Limit Exceeded</h1>"),
			"site_locked":    loadErrorPage("domainlock.txt", "<h1>Site Locked</h1>"),
			"domain_invalid": loadErrorPage("unkowndomain.txt", "<h1>Domain Not Configured</h1>"),
			"conn_limit":     loadErrorPage("connectout.txt", "<h1>Connection Limit Exceeded</h1>"),
		},
	}
}

// GetConfig
func (ctr *GlobalConfigController) GetConfig(c *gin.Context) {
	var sysConfig models.SysConfig
	// Match db.sql config structure: name="global_config", type="system", scope_id=0, scope_name="global"
	result := db.DB.Where("name = ? AND type = ?", GlobalConfigKey, "system").Take(&sysConfig)

	var config models.GlobalConfig

	if result.Error != nil {
		// Not found, use default and save
		config = getDefaultConfig()
		jsonBytes, _ := json.Marshal(config)
		sysConfig = models.SysConfig{
			Name:      GlobalConfigKey,
			Value:     string(jsonBytes),
			Type:      "system",
			ScopeID:   0,
			ScopeName: "global",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.DB.Create(&sysConfig)
	} else {
		if err := json.Unmarshal([]byte(sysConfig.Value), &config); err != nil {
			// If parse error, fallback to default
			config = getDefaultConfig()
		}
	}

	// Ensure maps are not nil
	if config.ErrorPages == nil {
		config.ErrorPages = make(map[string]string)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": config,
	})
}

// UpdateConfig
func (ctr *GlobalConfigController) UpdateConfig(c *gin.Context) {
	var req models.GlobalConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Request"})
		return
	}

	// 1. Fetch existing
	var sysConfig models.SysConfig
	err := db.DB.Where("name = ? AND type = ?", GlobalConfigKey, "system").Take(&sysConfig).Error
    
    isNew := false
    if err != nil {
        isNew = true
        sysConfig = models.SysConfig{
            Name: GlobalConfigKey,
            Type: "system",
            ScopeID: 0,
            ScopeName: "global",
            Enable: true,
            CreatedAt: time.Now(),
        }
    }

	// 2. Marshal new config
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Serialization Error"})
		return
	}

	// 3. Save to DB
	sysConfig.Value = string(jsonBytes)
	sysConfig.UpdatedAt = time.Now()

    if isNew {
        if db.DB.Create(&sysConfig).Error != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Create Error"})
            return
        }
    	} else {
		// Use Updates because model has no primary key (ID)
		if db.DB.Model(&models.SysConfig{}).Where("name = ? AND type = ?", GlobalConfigKey, "system").Updates(map[string]interface{}{
			"value":     sysConfig.Value,
			"update_at": sysConfig.UpdatedAt,
		}).Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Update Error"})
			return
		}
	}

	// 4. Trigger Node Sync (Use timestamp as version)
	version := sysConfig.UpdatedAt.Unix()
	go notifyNodes(version)
	services.BumpConfigVersion("global_config", []int64{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Global Config Updated & Nodes Notified"})
}

func notifyNodes(version int64) {
	// Logic to notify edge nodes
	// In a real system, this might push a message via gRPC, Redis PubSub, or update a generic 'version' key
	log.Printf("[Sync] Global Config updated to version %d. Notifying all connected nodes...", version)
	// mock notification delay
	time.Sleep(100 * time.Millisecond)
	log.Printf("[Sync] Notification sent.")
}
