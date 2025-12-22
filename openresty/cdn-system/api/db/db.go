package db

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var err error
	// DB, err = gorm.Open(mysql.Open(config.App.DBDSN), &gorm.Config{})
	DB, err = gorm.Open(sqlite.Open("cdn_system.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection established")
}
