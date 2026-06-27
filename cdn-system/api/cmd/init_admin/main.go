package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/utils"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	config.Load()
	args := flagArgs()
	if len(args) < 2 {
		fmt.Println("Usage: init_admin [-config path] [-db dsn] <username> <password> [email]")
		os.Exit(1)
	}

	username := strings.TrimSpace(args[0])
	password := strings.TrimSpace(args[1])
	email := ""
	if len(args) >= 3 {
		email = strings.TrimSpace(args[2])
	}

	if username == "" || password == "" {
		fmt.Println("username and password are required")
		os.Exit(1)
	}

	db.Init()

	// Map Role to Type (Assuming 1=Admin)
	adminType := 1

	var existing models.User
	if err := db.DB.Where("name = ?", username).First(&existing).Error; err == nil {
		if os.Getenv("FORCE") != "1" {
			fmt.Println("user already exists; set FORCE=1 to overwrite password and role")
			os.Exit(1)
		}
		hashed, err := utils.HashPasswordForStorage(password)
		if err != nil {
			fmt.Println("failed to hash password:", err)
			os.Exit(1)
		}
		existing.Password = string(hashed)
		existing.Type = adminType
		existing.Enable = true
		if email != "" {
			existing.Email = email
		}
		if err := db.DB.Save(&existing).Error; err != nil {
			fmt.Println("failed to update user:", err)
			os.Exit(1)
		}
		fmt.Println("admin user updated:", existing.Name)
		return
	}

	hashed, err := utils.HashPasswordForStorage(password)
	if err != nil {
		fmt.Println("failed to hash password:", err)
		os.Exit(1)
	}

	user := models.User{
		Name:     username,
		Password: string(hashed),
		Email:    email,
		Type:     adminType,
		Enable:   true,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		fmt.Println("failed to create user:", err)
		os.Exit(1)
	}

	fmt.Println("admin user created:", user.Name)
}

func flagArgs() []string {
	return flag.Args()
}
