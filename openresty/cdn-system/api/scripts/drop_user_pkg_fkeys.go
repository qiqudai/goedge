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
		// Drop FKs from user_package table
		"ALTER TABLE `user_package` DROP FOREIGN KEY `region_ibfk_6`;",
		"ALTER TABLE `user_package` DROP FOREIGN KEY `node_group_ibfk_1`;",
		"ALTER TABLE `user_package` DROP FOREIGN KEY `node_group_ibfk_5`;",

		// Ensure columns are nullable/bigint (fixing type mismatch if any, though schema showed int(11))
		// Note: schema showed int(11) for region_id/node_group_id in user_package.
		// My earlier script tried converting to bigint(20). Let's force them to bigint(20) NULL to match Go code and Package table.
		"ALTER TABLE `user_package` MODIFY COLUMN `region_id` bigint(20) NULL;",
		"ALTER TABLE `user_package` MODIFY COLUMN `node_group_id` bigint(20) NULL;",
		"ALTER TABLE `user_package` MODIFY COLUMN `backup_node_group` bigint(20) NULL;",
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Note: %s -> %v\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	fmt.Println("Drop user_package constraints script finished.")
}
