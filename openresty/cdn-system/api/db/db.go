package db

import (
	"context"
	"log"

	"time"

	"cdn-api/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Redis *redis.Client

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
	sqlDB.SetConnMaxLifetime(time.Hour)

	Redis = redis.NewClient(&redis.Options{
		Addr:     config.App.RedisAddr,
		Password: config.App.RedisPassword,
		DB:       0,
	})

	if _, err := Redis.Ping(context.Background()).Result(); err != nil {
		log.Printf("[Warn] Redis connection failed: %v", err)
	} else {
		log.Println("Redis connection established")
	}

	log.Println("Database connection established")
}
