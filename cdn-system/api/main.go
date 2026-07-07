package main

// CDN Core API - Stateless & Scalable
// Developed by Antigravity

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/middleware"
	"cdn-api/models"
	"cdn-api/routers"
	"cdn-api/services"
	"cdn-common/i18n"

	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var Version = "1.0.12"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()
	if *versionFlag {
		fmt.Println(Version)
		return
	}

	// 1. Load Config (Env or File)
	config.Load()

	if err := i18n.Load(""); err != nil {
		log.Printf("i18n load failed: %v", err)
	}

	// 2. Connect Database (MySQL)
	db.Init()
	db.InitClickHouse()

	runMigrationsWithRetry()
	if db.DB.Migrator().HasTable(&models.Task{}) {
		if !db.DB.Migrator().HasColumn(&models.Task{}, "targets_json") {
			if err := db.DB.Migrator().AddColumn(&models.Task{}, "TargetsJSON"); err != nil {
				log.Printf("Failed to add task.targets_json: %v", err)
			}
		}
	}
	ensureNodeColumns()
	ensureSiteColumns()
	if db.DB.Migrator().HasTable("job") {
		if err := db.DB.Migrator().DropTable("job"); err != nil {
			log.Printf("Failed to drop job table: %v", err)
		}
	}

	// 3. Initialize Router (Gin)
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if allowedOrigin := resolveAllowedCORSOrigin(origin); allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Add("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, Pragma, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Auth-Token")

		if c.Request.Method == "OPTIONS" {
			if origin != "" && resolveAllowedCORSOrigin(origin) == "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Unified response wrapper (JSON only)
	r.Use(middleware.ResponseWrapper())

	routers.Setup(r)

	safeGo("node-health-loop", func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			enabled, maxFails := services.ResolveNodeHealthConfig()
			if !enabled {
				continue
			}
			offlineIDs := services.EvaluateNodeHealth(5*time.Second, maxFails)
			for _, nodeID := range offlineIDs {
				services.HandleNodeOffline(nodeID)
			}
		}
	})

	// Start DNS Worker
	go services.StartDNSWorker()
	// Start DNS record repair worker
	go services.StartDNSRecordRepairWorker()
	// Start Backup Line Group switch worker
	go services.StartBackupGroupSwitchWorker()
	// Start Node Auto Weight Worker
	go services.StartNodeAutoSwitchWorker()
	// Start Cert Auto Renew Worker
	go services.StartCertAutoRenewWorker()
	// Start Cert Issue Worker
	// go services.StartCertIssueWorker()
	// Start Site Create Worker
	services.StartSiteCreateWorker()
	// Start User Package Expiration Worker
	services.StartUserPackageExpirationWorker()
	// Start Package Auto Renew Worker
	services.StartPackageAutoRenewWorker()
	// Start User Package Traffic Worker
	services.StartUserPackageTrafficWorker()
	// Start Cleanup & Backup Worker
	services.StartCleanupAndBackupWorker()

	// 4. Start Server
	// Recommend running behind Nginx Load Balancer for HA
	log.Printf("Starting CDN Core API on :%s", config.App.Port)
	addr := strings.TrimSpace(config.App.Port)
	if addr == "" {
		addr = "8080"
	}
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	for {
		if err := r.Run(addr); err != nil {
			log.Printf("HTTP server stopped unexpectedly: %v, retrying in 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return
	}
}

func resolveAllowedCORSOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	allowed := parseAllowedCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(allowed) == 0 {
		allowed = parseAllowedCORSOrigins(os.Getenv("APP_ALLOWED_ORIGINS"))
	}
	if len(allowed) == 0 {
		allowed = parseAllowedCORSOrigins(config.App.CORSAllowedOrigins)
	}
	if len(allowed) == 0 {
		return ""
	}
	for _, item := range allowed {
		if item == origin {
			return origin
		}
	}
	return ""
}

func parseAllowedCORSOrigins(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimRight(strings.TrimSpace(part), "/")
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func runMigrationsWithRetry() {
	for {
		if err := db.DB.AutoMigrate(&models.UserPackage{}); err != nil {
			if db.RecoverIfConnectionError(err) {
				time.Sleep(2 * time.Second)
				continue
			}
			log.Printf("Failed to migrate schemas (UserPackage): %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err := db.DB.AutoMigrate(&models.BalanceLedger{}); err != nil {
			if db.RecoverIfConnectionError(err) {
				time.Sleep(2 * time.Second)
				continue
			}
			log.Printf("Failed to migrate schemas (BalanceLedger): %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return
	}
}

func safeGo(name string, fn func()) {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[%s] panic recovered: %v\n%s", name, r, string(debug.Stack()))
					}
				}()
				fn()
			}()
			log.Printf("[%s] exited unexpectedly, restarting in 2s", name)
			time.Sleep(2 * time.Second)
		}
	}()
}

func ensureNodeColumns() {
	if !db.DB.Migrator().HasTable(&models.Node{}) {
		return
	}
	m := db.DB.Migrator()
	ensureColumn(m, &models.Node{}, "Token")
	ensureColumn(m, &models.Node{}, "SSHHost")
	ensureColumn(m, &models.Node{}, "SSHPort")
	ensureColumn(m, &models.Node{}, "SSHUser")
	ensureColumn(m, &models.Node{}, "SSHAuthType")
	ensureColumn(m, &models.Node{}, "SSHPassword")
	ensureColumn(m, &models.Node{}, "SSHKey")
	ensureColumn(m, &models.Node{}, "WorkDir")
	ensureColumn(m, &models.Node{}, "AutoInstall")
	ensureColumn(m, &models.Node{}, "InstallStatus")
	ensureColumn(m, &models.Node{}, "InstallError")
	ensureColumn(m, &models.Node{}, "InstallAt")
}

func ensureSiteColumns() {
	if !db.DB.Migrator().HasTable(&models.Site{}) {
		return
	}
	m := db.DB.Migrator()
	ensureColumn(m, &models.Site{}, "SettingsRaw")
}

func ensureColumn(m gorm.Migrator, model interface{}, name string) {
	if m.HasColumn(model, name) {
		return
	}
	if err := m.AddColumn(model, name); err != nil {
		log.Printf("Failed to add column %s: %v", name, err)
	}
}
