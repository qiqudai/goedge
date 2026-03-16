//go:build linux

package main

import (
	"cdn-agent/geneva"
	"log"
	"os"
	osSignal "os/signal"
	"sync"
	"syscall"
)

var genevaSignalOnce sync.Once

func startGenevaIfEnabled() {
	if !GenevaEnable {
		return
	}

	svc := geneva.New(geneva.Config{
		WindowSize: GenevaWindowSize,
		Debug:      DebugMode,
	})
	if err := svc.Start(); err != nil {
		log.Printf("[GENEVA] Start failed: %v", err)
		return
	}

	genevaSignalOnce.Do(func() {
		sigCh := make(chan os.Signal, 1)
		osSignal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			log.Printf("[GENEVA] Shutdown signal received: %v", sig)
			if err := svc.Stop(); err != nil {
				log.Printf("[GENEVA] Stop failed: %v", err)
			}
			os.Exit(0)
		}()
	})
}
