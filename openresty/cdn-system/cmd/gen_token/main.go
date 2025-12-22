package main

import (
	"cdn-api/utils"
	"fmt"
)

func main() {
	token, _ := utils.GenerateToken(1, "admin")
	fmt.Printf("TOKEN:%s", token)
}
