package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	// 1. Argument Parsing
	configFile := flag.String("config", "agent.json", "Path to config file")
	apiFlag := flag.String("api", "", "API Server URL")
	tokenFlag := flag.String("token", "", "Node Auth Token")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// 2. Load from Config File
	if fileData, err := ioutil.ReadFile(*configFile); err == nil {
		var fileConfig struct {
			API   string `json:"api"`
			Token string `json:"token"`
			Debug bool   `json:"debug"`
		}
		if err := json.Unmarshal(fileData, &fileConfig); err == nil {
			if fileConfig.API != "" {
				API_BaseURL = fileConfig.API
			}
			if fileConfig.Token != "" {
				AuthToken = fileConfig.Token
			}
			if fileConfig.Debug {
				DebugMode = true
			}
			log.Printf("[Info] Loaded config from %s", *configFile)
		}
	}

	// 3. Override with Flags
	if *apiFlag != "" {
		API_BaseURL = *apiFlag
	}
	if *tokenFlag != "" {
		AuthToken = *tokenFlag
	}
	if *debugFlag {
		DebugMode = true
	}

	if AuthToken == "" {
		log.Fatal("Error: Token is required in either agent.json or -token flag.")
	}

	// Assume hostname as NodeID for now
	hostname, _ := os.Hostname()
	NodeID = hostname

	log.Printf("Starting Edge Agent...")
	log.Printf("Target Master: %s", API_BaseURL)
	log.Printf("Node ID:       %s", NodeID)
	log.Printf("Debug Mode:    %v", DebugMode)

	// Initialize Environment (Unpack Assets)
	initEnvironment()

	// 2. Start Tickers
	go startHeartbeat()
	go startConfigPull()
	go startTaskPull()
	go startWebSocketClient() // Persistent Connection
	go startAccessLogShip()
	go startMetricsShip()
	go startL2Monitor()

	// 3. Keep Alive
	select {}
}
