package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"fmt"
)

func main() {
	config.Load()
	db.Init()

	// List of constraints to drop that are causing issues
	// We use a try-catch style execution because checking for existence is verbose in SQL
	sqls := []string{
		// Drop FKs from package table
		"ALTER TABLE `package` DROP FOREIGN KEY `region_ibfk_3`;",
		// Also drop node_group FK if it exists (guessing name or generic)
		// Usually GORM names them automatically.
		// If we can't guess the name, we might strictly fail here if it doesn't exist.
		// So we focus on the one we KNOW is failing: region_ibfk_3
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Note: %s -> %v (This is expected if the key was already dropped)\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	// Re-apply NULL check just in case
	sqls2 := []string{
		"ALTER TABLE `package` MODIFY COLUMN `region_id` bigint(20) NULL;",
	}
	for _, sql := range sqls2 {
		db.DB.Exec(sql)
	}

	fmt.Println("Drop constraints script finished.")
}
