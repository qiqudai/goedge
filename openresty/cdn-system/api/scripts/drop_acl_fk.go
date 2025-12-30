package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/cdnfy?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Drop FK task_ibfk_5 from acl table
	err = db.Exec("ALTER TABLE acl DROP FOREIGN KEY task_ibfk_5").Error
	if err != nil {
		fmt.Println("Error dropping FK (might not exist):", err)
	} else {
		fmt.Println("Dropped FK task_ibfk_5 from acl")
	}
}
