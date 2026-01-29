package main

// CDN Core API - Stateless & Scalable
// Developed by Antigravity

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/routers"
	"cdn-api/services"
	"cdn-common/i18n"

	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	// 1. Load Config (Env or File)
	config.Load()

	if err := i18n.Load(""); err != nil {
		log.Printf("i18n load failed: %v", err)
	}

	// 2. Connect Database (MySQL)
	db.Init()
	db.InitClickHouse()

	if err := db.DB.AutoMigrate(&models.UserPackage{}); err != nil {
		log.Fatal("Failed to migrate schemas:", err)
	}
	if db.DB.Migrator().HasTable(&models.Task{}) {
		if !db.DB.Migrator().HasColumn(&models.Task{}, "targets_json") {
			if err := db.DB.Migrator().AddColumn(&models.Task{}, "TargetsJSON"); err != nil {
				log.Printf("Failed to add task.targets_json: %v", err)
			}
		}
	}
	ensureNodeColumns()
	if db.DB.Migrator().HasTable("job") {
		if err := db.DB.Migrator().DropTable("job"); err != nil {
			log.Printf("Failed to drop job table: %v", err)
		}
	}

	// 3. Initialize Router (Gin)
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, Pragma, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Auth-Token")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	routers.Setup(r)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			enabled, maxFails := services.ResolveNodeHealthConfig()
			if !enabled {
				continue
			}
			offlineIDs := services.EvaluateNodeHealth(3*time.Second, maxFails)
			for _, nodeID := range offlineIDs {
				services.HandleNodeOffline(nodeID)
			}
		}
	}()

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
	// Start User Package Traffic Worker
	services.StartUserPackageTrafficWorker()
	// Start Cleanup & Backup Worker
	services.StartCleanupAndBackupWorker()

	// 4. Start Server
	// Recommend running behind Nginx Load Balancer for HA
	log.Printf("Starting CDN Core API on :%s", config.App.Port)
	r.Run(":" + config.App.Port)
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

func ensureColumn(m gorm.Migrator, model interface{}, name string) {
	if m.HasColumn(model, name) {
		return
	}
	if err := m.AddColumn(model, name); err != nil {
		log.Printf("Failed to add column %s: %v", name, err)
	}
}
