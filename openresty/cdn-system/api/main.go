package main

// CDN Core API - Stateless & Scalable
// Developed by Antigravity

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/routers"
	"cdn-api/utils"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Config (Env or File)
	config.Load()

	// 2. Connect Database (MySQL) & Cache (Redis)
	// 2. Connect Database (MySQL) & Cache (Redis)
	db.Init()
	// Auto Migrate
	db.DB.AutoMigrate(
		&models.Node{},
		&models.SysConfig{},
		&models.Cert{},
		// &models.Plan{}, // Deprecated? Check Package
		&models.Package{},
		&models.UserPackage{},
		&models.User{},
		// &models.Domain{}, // Deprecated -> Site
		&models.Site{},
		&models.SiteGroup{},
		&models.SiteGroupRelation{},
		// &models.DomainOrigin{}, // Deprecated
		// &models.NodeIP{}, // Deprecated
		&models.NodeGroup{},
		// &models.UserNodeGroup{}, // Deprecated
		&models.DNSAPI{},
		&models.DNSProvider{}, // Check if exists
		&models.UserLoginLog{},
		&models.UserOperationLog{},
		&models.CnameDomain{}, // Check if exists
		&models.CCRule{},
		&models.CCMatch{},
		&models.CCFilter{},
		&models.ACL{},
		&models.Order{},
		&models.APIKey{},
		&models.Message{},
		&models.MessageRead{},
		&models.MessageSub{},
	)

	// Ensure Admin Role / User Exists
	go func() {
		var u models.User
		// 1=Admin
		adminType := 1
		if err := db.DB.Where("name = ?", "admin").First(&u).Error; err != nil {
			// Create Admin
			u = models.User{
				Name:     "admin",
				Password: "123456", // Supported by fallback
				Type:     adminType,
				Enable:   true,
			}
			db.DB.Create(&u)
			log.Println("Created Admin User")
		} else {
			shouldSave := false
			if u.Type != adminType {
				u.Type = adminType
				shouldSave = true
			}
			if !u.Enable {
				u.Enable = true
				shouldSave = true
			}
			if shouldSave {
				db.DB.Save(&u)
				log.Println("Fixed Admin Type/Enable Status")
			}
		}

		// DEBUG: Print Admin Token
		token, _ := utils.GenerateToken(u.ID, "admin")
		log.Printf("[DEBUG] ADMIN TOKEN: %s", token)
	}()

	// 3. Initialize Router (Gin)
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	routers.Setup(r)

	// 4. Start Server
	// Recommend running behind Nginx Load Balancer for HA
	log.Printf("Starting CDN Core API on :%s", config.App.Port)
	r.Run(":" + config.App.Port)
}
