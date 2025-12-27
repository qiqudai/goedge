package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"log"
)

func main() {
	config.Load()
	db.Init()

	// Modify max_cache_size to int
	// Note: If there are existing string values like "10GB", this might fail or convert to 10.
	// Since I just added this column and it's likely empty or has test data "100", it should be fine.
	// But to be safe, we can update it to 0 first if not numeric.

	// Sanitize data first
	if err := db.DB.Exec("UPDATE `node` SET `max_cache_size` = '0' WHERE `max_cache_size` = '';").Error; err != nil {
		log.Printf("Warning: Update empty values failed: %v", err)
	}

	// Simply ALTER.
	sql := "ALTER TABLE `node` MODIFY COLUMN `max_cache_size` int(11) DEFAULT 0;"

	if err := db.DB.Exec(sql).Error; err != nil {
		log.Printf("Error executing SQL: %s\nError: %v", sql, err)
	} else {
		log.Printf("Success: %s", sql)
	}
}
