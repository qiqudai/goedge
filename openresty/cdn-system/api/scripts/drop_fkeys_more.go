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
		// Try to drop other potential constraints
		"ALTER TABLE `package` DROP FOREIGN KEY `node_group_2`;",
		"ALTER TABLE `package` DROP FOREIGN KEY `node_group_ibfk_4`;",

		// Ensure columns are nullable (though schema said they are)
		"ALTER TABLE `package` MODIFY COLUMN `node_group_id` bigint(20) NULL;",
		"ALTER TABLE `package` MODIFY COLUMN `backup_node_group` bigint(20) NULL;",
	}

	for _, sql := range sqls {
		if err := db.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Note: %s -> %v\n", sql, err)
		} else {
			fmt.Printf("Success: %s\n", sql)
		}
	}

	fmt.Println("Drop additional constraints script finished.")
}
