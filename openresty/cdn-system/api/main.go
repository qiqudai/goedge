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
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Config (Env or File)
	config.Load()

	// 2. Connect Database (MySQL)
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

	// Custom Migration Fixes
	migrator := db.DB.Migrator()
	if !migrator.HasColumn(&models.Site{}, "dns_provider_id") {
		migrator.AddColumn(&models.Site{}, "DNSProviderID")
		log.Println("Added missing column: dns_provider_id")
	}
	// Note: HasColumn checks against DB column name usually, but GORM maps field names too.
	// Use explicit DB names for check, Struct field names for Add.
	if !migrator.HasColumn(&models.Site{}, "settings") {
		err := migrator.AddColumn(&models.Site{}, "SettingsRaw")
		if err != nil {
			log.Printf("Error adding settings column: %v", err)
		} else {
			log.Println("Added missing column: settings")
		}
	}
	if !migrator.HasColumn(&models.Site{}, "cname_hostname2") {
		migrator.AddColumn(&models.Site{}, "CnameHostname2")
		log.Println("Added missing column: cname_hostname2")
	}
	if !migrator.HasColumn(&models.DNSAPI{}, "is_default") {
		migrator.AddColumn(&models.DNSAPI{}, "IsDefault")
		log.Println("Added missing column: is_default")
	}

	// Drop troublesome FK constraint if exists to allow 0 value
	if migrator.HasConstraint(&models.Site{}, "region_ibfk_4") {
		if err := migrator.DropConstraint(&models.Site{}, "region_ibfk_4"); err == nil {
			log.Println("Dropped constraint: region_ibfk_4")
		} else {
			log.Printf("Failed to drop constraint region_ibfk_4: %v", err)
		}
	}

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

		// Seed Default Package
		var pkgCount int64
		db.DB.Model(&models.Package{}).Count(&pkgCount)
		if pkgCount == 0 {
			defaultPkg := models.Package{
				Name:         "Free Plan",
				Description:  "Default free plan",
				MonthPrice:   0,
				QuarterPrice: 0,
				YearPrice:    0,
				Bandwidth:    "10M",
				Traffic:      100,
				DomainLimit:  10,
				Enable:       true,
				Sort:         0,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := db.DB.Create(&defaultPkg).Error; err == nil {
				log.Println("Created Default Package: Free Plan")
			}
		}

		// Ensure Admin has a UserPackage
		var userPkg models.UserPackage
		if err := db.DB.Where("uid = ?", u.ID).First(&userPkg).Error; err != nil {
			var pkg models.Package
			if err := db.DB.Order("id asc").First(&pkg).Error; err == nil {
				userPkg = models.UserPackage{
					UserID:      u.ID,
					Name:        pkg.Name + " (Admin)",
					PackageID:   pkg.ID,
					Bandwidth:   pkg.Bandwidth,
					Traffic:     pkg.Traffic,
					Connection:  1000,
					DomainLimit: pkg.DomainLimit,
					StartAt:     time.Now(),
					EndAt:       time.Now().AddDate(10, 0, 0),
					CreatedAt:   time.Now(),
				}
				if err := db.DB.Create(&userPkg).Error; err == nil {
					log.Println("Assigned Default Package to Admin")
				}
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
