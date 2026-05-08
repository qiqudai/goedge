package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
)

func DeleteBlockedLogsByIPs(ips []string) error {
	quoted := quoteValues(ips)
	if len(quoted) == 0 {
		return nil
	}
	query := fmt.Sprintf(
		"ALTER TABLE node_access_logs DELETE WHERE remote_addr IN (%s) AND status IN (403,410,418,429,451) SETTINGS mutations_sync = 1",
		strings.Join(quoted, ","),
	)
	if err := execClickHouseMutation(query); err != nil {
		return err
	}
	return verifyBlockedLogsDeleted("remote_addr", quoted)
}

func DeleteBlockedLogsByHosts(hosts []string) error {
	quoted := quoteValues(hosts)
	if len(quoted) == 0 {
		return nil
	}
	query := fmt.Sprintf(
		"ALTER TABLE node_access_logs DELETE WHERE host IN (%s) AND status IN (403,410,418,429,451) SETTINGS mutations_sync = 1",
		strings.Join(quoted, ","),
	)
	if err := execClickHouseMutation(query); err != nil {
		return err
	}
	return verifyBlockedLogsDeleted("host", quoted)
}

func quoteValues(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, quoteClickHouseString(v))
	}
	return out
}

func execClickHouseMutation(query string) error {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	if db.ClickHouseEnabled() && db.CK != nil {
		_, err := db.CK.Exec(query)
		return err
	}
	if cfg := buildHTTPConfig(); cfg != nil {
		err := execClickHouseMutationHTTP(cfg, query)
		return err
	}
	return nil
}

func verifyBlockedLogsDeleted(field string, quotedValues []string) error {
	if strings.TrimSpace(field) == "" || len(quotedValues) == 0 {
		return nil
	}
	// Keep verification bounded to avoid long API tail latency.
	deadline := time.Now().Add(8 * time.Second)
	for {
		remaining, err := countBlockedLogs(field, quotedValues)
		if err != nil {
			return err
		}
		if remaining == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("unblock verify failed: %d records still present", remaining)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func countBlockedLogs(field string, quotedValues []string) (int64, error) {
	query := fmt.Sprintf(
		"SELECT count() FROM node_access_logs WHERE %s IN (%s) AND status IN (403,410,418,429,451)",
		field, strings.Join(quotedValues, ","),
	)
	if db.ClickHouseEnabled() && db.CK != nil {
		var count int64
		if err := db.CK.QueryRow(query).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}
	if cfg := buildHTTPConfig(); cfg != nil {
		body, err := queryClickHouseHTTP(cfg, query+" FORMAT JSONEachRow")
		if err != nil {
			return 0, err
		}
		line := strings.TrimSpace(string(body))
		if line == "" {
			return 0, nil
		}
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return 0, nil
		}
		var raw string
		for _, v := range row {
			raw = fmt.Sprintf("%v", v)
			break
		}
		n, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		return n, nil
	}
	return 0, nil
}

func execClickHouseMutationHTTP(cfg *httpCKConfig, query string) error {
	params := url.Values{}
	if cfg.database != "" {
		params.Set("database", cfg.database)
	}
	endpoint := cfg.baseURL + "/"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.user != "" {
		req.SetBasicAuth(cfg.user, cfg.pass)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 512 {
			detail = detail[:512]
		}
		if detail == "" {
			return fmt.Errorf("clickhouse mutation http status %s", resp.Status)
		}
		return fmt.Errorf("clickhouse mutation http status %s: %s", resp.Status, detail)
	}
	return nil
}
