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
	nodeIDFlag := flag.String("node-id", "", "Node ID (string, usually numeric node id)")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// 2. Load from Config File
	if fileData, err := ioutil.ReadFile(*configFile); err == nil {
		var fileConfig struct {
			API    string `json:"api"`
			Token  string `json:"token"`
			NodeID string `json:"node_id"`
			Debug  bool   `json:"debug"`
		}
		if err := json.Unmarshal(fileData, &fileConfig); err == nil {
			if fileConfig.API != "" {
				API_BaseURL = fileConfig.API
			}
			if fileConfig.Token != "" {
				AuthToken = fileConfig.Token
			}
			if fileConfig.NodeID != "" {
				NodeID = fileConfig.NodeID
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
	if *nodeIDFlag != "" {
		NodeID = *nodeIDFlag
	}
	if *debugFlag {
		DebugMode = true
	}

	if AuthToken == "" {
		log.Fatal("Error: Token is required in either agent.json or -token flag.")
	}

	// Default NodeID to hostname if not provided
	if NodeID == "" {
		hostname, _ := os.Hostname()
		NodeID = hostname
	}

	log.Printf("Starting Edge Agent...")
	log.Printf("Target Master: %s", API_BaseURL)
	log.Printf("Node ID:       %s", NodeID)
	log.Printf("Debug Mode:    %v", DebugMode)

	// Initialize Environment (Unpack Assets)
	initEnvironment()

	// 2. Start Tickers
	go startWebSocketClient() // Persistent Connection
	log.Printf("[Info] Access log ship enabled")
	go startAccessLogShip()
	go startMetricsShip()
	go startLogCleanup()
	go startL2Monitor()

	// 3. Keep Alive
	select {}
}
