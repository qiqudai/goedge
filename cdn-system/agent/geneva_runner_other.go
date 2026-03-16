//go:build !linux

package main

import (
	"log"
	"runtime"
)

func startGenevaIfEnabled() {
	if GenevaEnable {
		log.Printf("[GENEVA] geneva_enable ignored on unsupported platform: %s", runtime.GOOS)
	}
}
