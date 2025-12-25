package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"fmt"
)

func main() {
	config.Load()
	db.Init()

	sqls := []string{
		// Drop FK from node_group table
		"ALTER TABLE `node_group` DROP FOREIGN KEY `region_ibfk_2`;",

		// Ensure region_id is bigint(20) NULL to match others (schema showed int(11), let's upgrade to bigint for consistency if possible, or key fails)
		// But region.id is likely int(11) or bigint. Package uses bigint.
		// If region.id is int(11), changing this to bigint might break future FKs.
		// But we are removing FKs, so consistency with Go code (int64) is better.
		"ALTER TABLE `node_group` MODIFY COLUMN `region_id` bigint(20) NULL;",
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Note: %s -> %v\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	fmt.Println("Drop node_group constraints script finished.")
}
