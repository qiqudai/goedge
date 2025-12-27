package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"fmt"
)

func main() {
	// 1. Load Config
	config.Load()

	// Initialize DB
	db.Init()

	// Add missing columns to node_group table
	sqls := []string{
		"ALTER TABLE `node_group` ADD COLUMN `sort_order` int(11) DEFAULT 100;",
		"ALTER TABLE `node_group` ADD COLUMN `ipv4_resolution` varchar(255) DEFAULT '';",
		"ALTER TABLE `node_group` ADD COLUMN `l2_config` varchar(255) DEFAULT '';",
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Exec failed (might already exist): %s, error: %v\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	fmt.Println("Migration finished.")
}
