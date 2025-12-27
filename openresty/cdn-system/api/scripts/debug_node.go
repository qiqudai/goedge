package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"fmt"
	"log"
)

func main() {
	config.Load()
	db.Init()

	var node models.Node
	if err := db.DB.First(&node, 1).Error; err != nil {
		log.Fatalf("Error finding node 1: %v", err)
	}

	data, _ := json.MarshalIndent(node, "", "  ")
	fmt.Printf("Node 1 Data:\n%s\n", string(data))
}
