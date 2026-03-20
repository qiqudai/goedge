package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func ensureManagedNginxOnStartup() {
	if err := startOrRestartManagedNginx(); err != nil {
		log.Printf("[Error] Managed nginx startup failed: %v", err)
	}
}

func waitForShutdownAndCleanup() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	sig := <-sigCh
	log.Printf("[Info] Received shutdown signal: %s", sig)
	if err := stopNginx(); err != nil {
		log.Printf("[Warn] Managed nginx stop on shutdown failed: %v", err)
	}
}
