package db

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"cdn-api/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
var dbMu sync.Mutex

func Init() {
	const retryDelay = 2 * time.Second
	for attempt := 1; ; attempt++ {
		if err := connect(); err == nil {
			if attempt > 1 {
				log.Printf("Database connection recovered after %d attempt(s)", attempt)
			}
			return
		} else {
			log.Printf("Failed to connect to database (attempt %d): %v", attempt, err)
		}
		time.Sleep(retryDelay)
	}
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

func connect() error {
	dbConn, err := gorm.Open(mysql.Open(config.App.DBDSN), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		return err
	}

	// Connection Pooling
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	DB = dbConn
	log.Println("Database connection established")
	return nil
}

func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "driver: bad connection") ||
		strings.Contains(lower, "unexpected eof") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "invalid connection")
}

func RecoverIfConnectionError(err error) bool {
	if !IsConnectionError(err) {
		return false
	}
	dbMu.Lock()
	defer dbMu.Unlock()
	if reconnectErr := connect(); reconnectErr != nil {
		log.Printf("Database reconnect failed: %v", reconnectErr)
		return false
	}
	log.Printf("Database reconnected after connection error: %v", err)
	return true
}
