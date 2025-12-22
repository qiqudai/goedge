package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: init_admin <username> <password> [email]")
		os.Exit(1)
	}

	username := strings.TrimSpace(os.Args[1])
	password := strings.TrimSpace(os.Args[2])
	email := ""
	if len(os.Args) >= 4 {
		email = strings.TrimSpace(os.Args[3])
	}

	if username == "" || password == "" {
		fmt.Println("username and password are required")
		os.Exit(1)
	}

	config.Load()
	db.Init()

	// Map Role to Type (Assuming 1=Admin)
	adminType := 1

	var existing models.User
	if err := db.DB.Where("name = ?", username).First(&existing).Error; err == nil {
		if os.Getenv("FORCE") != "1" {
			fmt.Println("user already exists; set FORCE=1 to overwrite password and role")
			os.Exit(1)
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
