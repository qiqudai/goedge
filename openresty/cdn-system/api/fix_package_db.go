package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"fmt"
)

func main() {
	config.Load()
	db.Init()

	// 1. Drop FKs if they exist (to allow modifying column) regarding constraints
	// It's safer to just Modify column. MySQL usually allows modifying to NULLable without dropping FK,
	// unless strict mode issues.
	// But we must assume these columns are mapped to valid FKs.

	sqls := []string{
		// Make Package fields NULL-able
		"ALTER TABLE `package` MODIFY COLUMN `region_id` bigint(20) NULL;",
		"ALTER TABLE `package` MODIFY COLUMN `node_group_id` bigint(20) NULL;",
		"ALTER TABLE `package` MODIFY COLUMN `backup_node_group` bigint(20) NULL;",

		// Make UserPackage fields NULL-able
		"ALTER TABLE `user_package` MODIFY COLUMN `region_id` bigint(20) NULL;",
		"ALTER TABLE `user_package` MODIFY COLUMN `node_group_id` bigint(20) NULL;",
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Error executing: %s\n%v\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	fmt.Println("Database schema fix completed.")
}
