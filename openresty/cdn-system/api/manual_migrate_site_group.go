package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"log"
)

func main() {
	// Load config
	config.Load()

	// Initialize DB
	db.Init()

	// Add missing columns to site_group
	if !db.DB.Migrator().HasColumn(&models.SiteGroup{}, "CreatedAt") {
		log.Println("Adding column create_at to site_group...")
		if err := db.DB.Migrator().AddColumn(&models.SiteGroup{}, "CreatedAt"); err != nil {
			log.Printf("Failed to add create_at: %v", err)
		} else {
			log.Println("Added create_at successfully")
		}
	}

	if !db.DB.Migrator().HasColumn(&models.SiteGroup{}, "UpdatedAt") {
		log.Println("Adding column update_at to site_group...")
		if err := db.DB.Migrator().AddColumn(&models.SiteGroup{}, "UpdatedAt"); err != nil {
			log.Printf("Failed to add update_at: %v", err)
		} else {
			log.Println("Added update_at successfully")
		}
	}

	log.Println("Migration check completed")
}
