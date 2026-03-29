package db

import (
	"errors"
	"log"
	"sync"
	"time"

	"cdn-api/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
var dbMu sync.Mutex

func Init() {
	var err error
	DB, err = gorm.Open(mysql.Open(config.App.DBDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get generic database object:", err)
	}

	// Connection Pooling
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)

	log.Println("Database connection established")
}

func Ensure() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if DB == nil {
		return errors.New("db not initialized")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Ping(); err != nil {
		return err
	}
	return nil
}
