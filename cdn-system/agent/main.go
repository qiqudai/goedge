package main

import (
	"cdn-common/i18n"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func resolveWorkDir(configPath string) {
	baseDir := ""
	if strings.TrimSpace(configPath) != "" {
		if abs, err := filepath.Abs(configPath); err == nil {
			baseDir = filepath.Dir(abs)
		}
	}
	if baseDir == "" {
		if exePath, err := os.Executable(); err == nil {
			baseDir = filepath.Dir(exePath)
		}
	}
	if WorkDir == "" || WorkDir == "." {
		WorkDir = baseDir
		return
	}
	if filepath.IsAbs(WorkDir) {
		return
	}
	if baseDir != "" {
		WorkDir = filepath.Join(baseDir, WorkDir)
	}
}

func main() {
	// 1. Argument Parsing
	configFile := flag.String("config", "agent.json", "Path to config file")
	apiFlag := flag.String("api", "", "API Server URL")
	tokenFlag := flag.String("token", "", "Node Auth Token")
	nodeIDFlag := flag.String("node-id", "", "Node ID (string, usually numeric node id)")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if err := i18n.Load(""); err != nil {
		log.Printf("i18n load failed: %v", err)
	}

	configPath := *configFile
	if abs, err := filepath.Abs(configPath); err == nil {
		configPath = abs
	}
	// 2. Load from Config File
	if fileData, err := ioutil.ReadFile(configPath); err == nil {
		var fileConfig struct {
			API            string `json:"api"`
			Token          string `json:"token"`
			NodeID         string `json:"node_id"`
			Debug          bool   `json:"debug"`
			WorkDir        string `json:"work_dir"`
			ResetResources bool   `json:"reset_resources"`
			BootstrapSync  bool   `json:"bootstrap_sync"`
			BootstrapStart bool   `json:"bootstrap_start"`
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
			if strings.TrimSpace(fileConfig.WorkDir) != "" {
				WorkDir = strings.TrimSpace(fileConfig.WorkDir)
			}
			ResetResources = fileConfig.ResetResources
			BootstrapSync = fileConfig.BootstrapSync
			BootstrapStart = fileConfig.BootstrapStart
			log.Printf("[Info] Loaded config from %s", configPath)
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
	resolveWorkDir(configPath)

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
	if ResetResources {
		if !resetWorkDirContents() {
			ResetResources = false
		}
	}
	initEnvironment()
	bootstrapSyncAndStart()

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
