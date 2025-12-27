package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"fmt"
)

type CreateTableResult struct {
	Table           string `gorm:"column:Table"`
	CreateStatement string `gorm:"column:Create Table"`
}

func main() {
	config.Load()
	db.Init()

	var result CreateTableResult
	err := db.DB.Raw("SHOW CREATE TABLE `node_group`").Scan(&result).Error
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("--- SCHEMA START ---")
	fmt.Println(result.CreateStatement)
	fmt.Println("--- SCHEMA END ---")
}
