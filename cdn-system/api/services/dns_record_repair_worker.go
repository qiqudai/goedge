package services

import (
	"log"
	"strconv"
	"strings"
	"time"
)

const dnsRecordRepairInterval = 1 * time.Hour

// StartDNSRecordRepairWorker runs periodic DNS repair/cleanup based on system config.
func StartDNSRecordRepairWorker() {
	go func() {
		ticker := time.NewTicker(dnsRecordRepairInterval)
		for range ticker.C {
			runDNSRecordRepair()
		}
	}()
}

func runDNSRecordRepair() {
	cfg, err := LoadSystemConfig()
	if err != nil {
		log.Printf("[DNS Repair] load config failed: %v", err)
		return
	}
	mode := parseRecordRepairMode(cfg["record-repair-enable"])
	if mode <= 0 {
		return
	}

	if errs := RepairDNSRecords(); len(errs) > 0 {
		log.Printf("[DNS Repair] repair errors: %s", strings.Join(errs, "; "))
	}
	if mode >= 2 {
		if errs := CleanupInvalidDNSRecords(); len(errs) > 0 {
			log.Printf("[DNS Repair] cleanup errors: %s", strings.Join(errs, "; "))
		}
	}
}

func parseRecordRepairMode(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if raw == "true" || raw == "on" || raw == "yes" {
		return 1
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return val
}
