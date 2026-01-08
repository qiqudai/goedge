package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Task struct {
	ID       int64      `gorm:"primaryKey"`
	Type     string     `gorm:"column:type"`
	State    string     `gorm:"column:state"`
	Enable   bool       `gorm:"column:enable"`
	Ret      string     `gorm:"column:ret"`
	ErrTimes int        `gorm:"column:err_times"`
	RetryAt  *time.Time `gorm:"column:retry_at"`
	EndAt    *time.Time `gorm:"column:end_at"`
}

func (Task) TableName() string {
	return "task"
}

func main() {
	var (
		dsn       string
		config    string
		idsRaw    string
		retPrefix string
	)
	flag.StringVar(&dsn, "dsn", "", "MySQL DSN. If empty, reads db_dsn from config file.")
	flag.StringVar(&config, "config", "config.yaml", "Path to api config.yaml")
	flag.StringVar(&idsRaw, "ids", "1655,1656,1657", "Comma-separated task IDs")
	flag.StringVar(&retPrefix, "ret", "archived by cleanup script", "Ret message prefix")
	flag.Parse()

	if dsn == "" {
		path := config
		if !filepath.IsAbs(path) {
			if cwd, err := os.Getwd(); err == nil {
				path = filepath.Join(cwd, path)
			}
		}
		var err error
		dsn, err = readDSN(path)
		if err != nil {
			log.Fatalf("failed to read db_dsn from %s: %v", path, err)
		}
	}

	ids, err := parseIDs(idsRaw)
	if err != nil {
		log.Fatalf("invalid ids: %v", err)
	}
	if len(ids) == 0 {
		log.Fatal("no ids provided")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	now := time.Now()
	retMessage := fmt.Sprintf("%s at %s", retPrefix, now.Format("2006-01-02 15:04:05"))
	updates := map[string]interface{}{
		"state":    "fail",
		"enable":   false,
		"end_at":   now,
		"retry_at": nil,
		"ret":      retMessage,
	}

	res := db.Model(&Task{}).Where("id IN ? AND type = ?", ids, "issue_cert").Updates(updates)
	if res.Error != nil {
		log.Fatalf("update failed: %v", res.Error)
	}
	log.Printf("updated %d rows", res.RowsAffected)

	var tasks []Task
	if err := db.Select("id,type,state,enable,ret,err_times,retry_at,end_at").Where("id IN ?", ids).Find(&tasks).Error; err != nil {
		log.Fatalf("readback failed: %v", err)
	}
	for _, t := range tasks {
		fmt.Printf("id=%d type=%s state=%s enable=%v err_times=%d retry_at=%v end_at=%v ret=%s\n",
			t.ID, t.Type, t.State, t.Enable, t.ErrTimes, t.RetryAt, t.EndAt, t.Ret)
	}
}

func parseIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad id %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func readDSN(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?m)^\s*db_dsn:\s*"?([^"\r\n]+)"?\s*$`)
	match := re.FindSubmatch(data)
	if match == nil {
		return "", fmt.Errorf("db_dsn not found")
	}
	return string(match[1]), nil
}
