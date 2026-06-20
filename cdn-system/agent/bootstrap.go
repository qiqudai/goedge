package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func resetWorkDirContents() bool {
	targetDir := runtimeRoot()
	if strings.TrimSpace(targetDir) == "" {
		log.Printf("[Warn] Reset skipped: empty WorkDir")
		return false
	}
	if !isSafeWorkDir(targetDir) {
		log.Printf("[Warn] Reset skipped: unsafe WorkDir=%s (expect base name 'edge-node')", targetDir)
		return false
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Printf("[Error] Reset mkdir failed: %v", err)
		return false
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		log.Printf("[Error] Reset read dir failed: %v", err)
		return false
	}
	for _, entry := range entries {
		fp := filepath.Join(targetDir, entry.Name())
		if err := os.RemoveAll(fp); err != nil {
			log.Printf("[Error] Reset remove failed: %v", err)
			return false
		}
	}
	log.Printf("[Init] WorkDir reset: %s", targetDir)
	return true
}

func isSafeWorkDir(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if abs == string(os.PathSeparator) {
		return false
	}
	if runtime.GOOS == "windows" {
		if len(abs) == 2 && abs[1] == ':' {
			return false
		}
		if len(abs) == 3 && abs[1] == ':' && (abs[2] == '\\' || abs[2] == '/') {
			return false
		}
	}
	base := filepath.Base(abs)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(base, "edge-node")
	}
	return base == "edge-node"
}

func bootstrapSyncAndStart() {
	if !BootstrapSync {
		return
	}
	if err := pullConfigBootstrap(); err != nil {
		log.Printf("[Error] Bootstrap sync failed: %v", err)
		return
	}
	if BootstrapStart {
		if err := startOrRestartManagedNginx(); err != nil {
			log.Printf("[Error] Bootstrap start nginx failed: %v", err)
			return
		}
	}
}

func pullConfigBootstrap() error {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/config?node_id="+NodeID, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	body, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		log.Printf("[Error] Bootstrap config pull failed: %v", err)
		return err
	}

	if status == 200 {
		debugLogInteraction("GET", req.URL.String(), status, nil, body)
		// Agent upgrades can change generated nginx/Lua output without changing the
		// API config version. Always rebuild on bootstrap, then let startup launch
		// nginx once with the newly generated configuration.
		if _, err := applyConfigPayloadWithOptionsAndReload(body, true, true); err != nil {
			return err
		}
		return nil
	}

	debugLogInteraction("GET", req.URL.String(), status, nil, nil)
	return fmt.Errorf("config pull status: %d", status)
}
