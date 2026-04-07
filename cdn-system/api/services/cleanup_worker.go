package services

import (
	"bytes"
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

const cleanupWorkerInterval = 1 * time.Hour

var backupState struct {
	mu     sync.Mutex
	lastAt time.Time
}

func StartCleanupAndBackupWorker() {
	go func() {
		runCleanupAndBackup()
		ticker := time.NewTicker(cleanupWorkerInterval)
		for range ticker.C {
			runCleanupAndBackup()
		}
	}()
}

func runCleanupAndBackup() {
	cfg, err := LoadSystemConfig()
	if err != nil {
		log.Printf("[Cleanup] Load system config failed: %v", err)
		return
	}
	runCleanup(cfg)
	runBackup(cfg)
}

func runCleanup(cfg map[string]string) {
	cleanupTableByDays("login_log", "create_at", parseDays(cfg["keep-login-log-days"], 30))
	cleanupTableByDays("op_log", "create_at", parseDays(cfg["keep-op-log-days"], 30))
	cleanupTableByDays("task", "create_at", parseDays(cfg["keep-task-log-days"], 7))
	cleanupTableByDays("node_monitor_log", "create_at", parseDays(cfg["keep-node-log-days"], 7))
	cleanupIssueCertTasks(cfg)

	accessDays := minPositive(
		parseDays(cfg["keep-access-log-days"], 0),
		parseDays(cfg["keep-traffic-history-days"], 0),
	)
	cleanupClickHouseByDays("node_access_logs", accessDays)
	cleanupClickHouseByDays("node_events", parseDays(cfg["keep-node-log-days"], 0))
	cleanupClickHouseByDays("node_metrics", parseDays(cfg["keep-node-traffic-days"], 0))
}

func cleanupIssueCertTasks(cfg map[string]string) {
	timeoutMinutes := parseIntConfigOrDefault(cfg["cert_issue_timeout_minutes"], 120)
	if timeoutMinutes <= 0 {
		return
	}
	cutoff := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)
	var tasks []models.Task
	if err := db.DB.Where("type = ? AND state IN ? AND enable = ?", "issue_cert", []string{"waiting", "running", "retrying"}, true).
		Where("COALESCE(start_at, create_at) < ?", cutoff).
		Find(&tasks).Error; err != nil {
		log.Printf("[Cleanup] issue_cert task scan failed: %v", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	now := time.Now()
	reason := fmt.Sprintf("证书签发超时（超过 %d 分钟）", timeoutMinutes)
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	if err := db.DB.Model(&models.Task{}).Where("id IN ?", ids).Updates(map[string]interface{}{
		"state":    "fail",
		"enable":   false,
		"ret":      reason,
		"end_at":   now,
		"retry_at": nil,
	}).Error; err != nil {
		log.Printf("[Cleanup] issue_cert task update failed: %v", err)
	}
	for _, task := range tasks {
		MarkIssueTaskFailed(task.ID, reason)
	}
}

func cleanupTableByDays(table, column string, days int) {
	if days <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", table, column)
	if err := db.DB.Exec(query, cutoff).Error; err != nil {
		log.Printf("[Cleanup] %s failed: %v", table, err)
	}
}

func cleanupClickHouseByDays(table string, days int) {
	if days <= 0 || !db.ClickHouseEnabled() {
		return
	}
	cutoffUnix := time.Now().AddDate(0, 0, -days).UTC().Unix()
	query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE ts < toDateTime(%d, 'UTC')", table, cutoffUnix)
	if _, err := db.CK.Exec(query); err != nil {
		log.Printf("[Cleanup] ClickHouse %s failed: %v", table, err)
	}
}

func runBackup(cfg map[string]string) {
	backupDir := strings.TrimSpace(cfg["backup_dir"])
	if backupDir == "" {
		return
	}
	interval := parseBackupInterval(cfg["backup_rate"])
	if interval <= 0 {
		return
	}
	backupState.mu.Lock()
	lastAt := backupState.lastAt
	backupState.mu.Unlock()
	if !lastAt.IsZero() && time.Since(lastAt) < interval {
		return
	}

	startAt := time.Now()
	path, err := runDatabaseBackup(backupDir)
	finishAt := time.Now()
	status := 1
	result := path
	if err != nil {
		status = 0
		result = err.Error()
		log.Printf("[Backup] Failed: %v", err)
	} else {
		log.Printf("[Backup] Saved: %s", path)
	}
	recordBackupTask(startAt, finishAt, status == 1, result)

	backupState.mu.Lock()
	backupState.lastAt = finishAt
	backupState.mu.Unlock()

	keepDays := parseDays(cfg["backup_keep_days"], 7)
	cleanupBackupFiles(backupDir, keepDays)
}

func recordBackupTask(startAt, finishAt time.Time, success bool, result string) {
	state := "done"
	errTimes := 0
	if !success {
		state = "fail"
		errTimes = 1
	}
	task := models.Task{
		Name:     "database_backup",
		Type:     "backup",
		CreateAt: startAt,
		StartAt:  &startAt,
		EndAt:    &finishAt,
		Ret:      result,
		State:    state,
		Enable:   true,
		ErrTimes: errTimes,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		log.Printf("[Backup] Failed to record task: %v", err)
	}
}

func runDatabaseBackup(backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	dsn, err := mysql.ParseDSN(config.App.DBDSN)
	if err != nil {
		return "", err
	}
	host, port := splitMySQLAddr(dsn.Addr)
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "3306"
	}
	filename := fmt.Sprintf("backup-%s.sql", time.Now().Format("20060102-150405"))
	path := filepath.Join(backupDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	args := []string{"-h", host, "-P", port, "-u", dsn.User, dsn.DBName}
	cmd := exec.Command("mysqldump", args...)
	if dsn.Passwd != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+dsn.Passwd)
	}
	var stderr bytes.Buffer
	cmd.Stdout = file
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(stderr.String()))
	}
	return path, nil
}

func cleanupBackupFiles(dir string, keepDays int) {
	if keepDays <= 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

func parseDays(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func parseBackupInterval(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.HasSuffix(raw, "d") {
		raw = strings.TrimSuffix(raw, "d")
		if days, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	if dur, err := time.ParseDuration(raw); err == nil {
		return dur
	}
	if days, err := strconv.Atoi(raw); err == nil && days > 0 {
		return time.Duration(days) * 24 * time.Hour
	}
	return 0
}

func splitMySQLAddr(addr string) (string, string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", ""
	}
	if strings.Contains(addr, "/") {
		return addr, ""
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, ""
	}
	return host, port
}

func minPositive(values ...int) int {
	min := 0
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if min == 0 || value < min {
			min = value
		}
	}
	return min
}
