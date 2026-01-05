package main

import (
	"cdn-api/config"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	config.Load()
	dsn := config.App.DBDSN
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("DSN:", redactDSN(dsn))

	printOneRow(db, "SELECT DATABASE()")
	printVars(db, []string{
		"@@version",
		"@@character_set_server",
		"@@collation_server",
		"@@character_set_database",
		"@@collation_database",
		"@@character_set_client",
		"@@collation_connection",
		"@@character_set_connection",
	})

	fmt.Println("\n-- table status --")
	printTableStatus(db, "config")

	fmt.Println("\n-- columns: config --")
	printColumns(db, "config")

	fmt.Println("\n-- sample values --")
	printConfigItem(db, "error-page", "error_page", "global", 0)
	printConfigItem(db, "global_config", "system", "global", 0)
	printGlobalErrorPagesFromGlobalConfig(db)
}

func redactDSN(dsn string) string {
	// crude best-effort redaction: "user:pass@tcp(...)/db?..."
	at := strings.Index(dsn, "@")
	colon := strings.Index(dsn, ":")
	if at > 0 && colon > 0 && colon < at {
		return dsn[:colon+1] + "****" + dsn[at:]
	}
	return dsn
}

func printOneRow(db *sql.DB, query string) {
	var v any
	if err := db.QueryRow(query).Scan(&v); err != nil {
		fmt.Printf("%s -> error: %v\n", query, err)
		return
	}
	fmt.Printf("%s -> %v\n", query, v)
}

func printVars(db *sql.DB, exprs []string) {
	for _, e := range exprs {
		printOneRow(db, "SELECT "+e)
	}
}

func printTableStatus(db *sql.DB, table string) {
	rows, err := db.Query(fmt.Sprintf("SHOW TABLE STATUS LIKE %q", table))
	if err != nil {
		fmt.Printf("SHOW TABLE STATUS %s -> error: %v\n", table, err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	if rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err == nil {
			for i, c := range cols {
				switch strings.ToLower(c) {
				case "name", "engine", "collation", "comment", "row_format":
					fmt.Printf("%s.%s = %v\n", table, c, values[i])
				}
			}
		}
	}
}

func printColumns(db *sql.DB, table string) {
	rows, err := db.Query("SHOW FULL COLUMNS FROM " + table)
	if err != nil {
		fmt.Printf("SHOW FULL COLUMNS FROM %s -> error: %v\n", table, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var field, colType, collation, nullStr, key, defaultVal, extra, privileges, comment sql.NullString
		if err := rows.Scan(&field, &colType, &collation, &nullStr, &key, &defaultVal, &extra, &privileges, &comment); err != nil {
			fmt.Printf("scan error: %v\n", err)
			return
		}
		fmt.Printf("%s: type=%s collation=%s\n", field.String, colType.String, collation.String)
	}
}

func printConfigItem(db *sql.DB, name, typ, scopeName string, scopeID int64) {
	var value sql.NullString
	err := db.QueryRow(
		"SELECT value FROM config WHERE name=? AND type=? AND scope_name=? AND scope_id=? LIMIT 1",
		name, typ, scopeName, scopeID,
	).Scan(&value)
	if err != nil {
		fmt.Printf("config(name=%s,type=%s,scope=%s/%d) -> error: %v\n", name, typ, scopeName, scopeID, err)
		return
	}
	s := value.String
	fmt.Printf("config(name=%s,type=%s,scope=%s/%d) len=%d qmarks=%d hasCJK=%v\n",
		name, typ, scopeName, scopeID, len(s), strings.Count(s, "?"), containsCJK(s),
	)
	fmt.Println("prefix:", preview(s, 200))
	fmt.Println("prefix(hex):", hexPreview(s, 200))

	// If value looks like JSON, try to parse and show a title sample.
	var pages map[string]string
	if json.Unmarshal([]byte(s), &pages) == nil && len(pages) > 0 {
		for k, v := range pages {
			fmt.Printf("page key=%s len=%d qmarks=%d hasCJK=%v title=%q\n", k, len(v), strings.Count(v, "?"), containsCJK(v), extractTitle(v))
			break
		}
	}
}

func printGlobalErrorPagesFromGlobalConfig(db *sql.DB) {
	var raw sql.NullString
	err := db.QueryRow(
		"SELECT value FROM config WHERE name=? AND type=? AND scope_name=? AND scope_id=? LIMIT 1",
		"global_config", "system", "global", 0,
	).Scan(&raw)
	if err != nil {
		fmt.Printf("global_config read -> error: %v\n", err)
		return
	}

	var decoded map[string]any
	if json.Unmarshal([]byte(raw.String), &decoded) != nil {
		fmt.Println("global_config JSON parse failed")
		return
	}

	ep, ok := decoded["error_pages"].(map[string]any)
	if !ok || len(ep) == 0 {
		fmt.Println("global_config.error_pages missing/empty")
		return
	}

	fmt.Printf("global_config.error_pages keys=%d\n", len(ep))
	for _, k := range []string{"403", "traffic_limit", "timeout", "ip"} {
		vAny, exists := ep[k]
		if !exists {
			continue
		}
		v, _ := vAny.(string)
		fmt.Printf("global_config.error_pages[%s] len=%d qmarks=%d hasCJK=%v title=%q\n",
			k, len(v), strings.Count(v, "?"), containsCJK(v), extractTitle(v),
		)
	}
}

func preview(s string, n int) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func hexPreview(s string, n int) string {
	b := []byte(s)
	if len(b) > n {
		b = b[:n]
	}
	return hex.EncodeToString(b)
}

var cjkRe = regexp.MustCompile(`[\p{Han}]`)

func containsCJK(s string) bool {
	return cjkRe.MatchString(s)
}

var titleRe = regexp.MustCompile(`(?is)<title>(.*?)</title>`)

func extractTitle(html string) string {
	m := titleRe.FindStringSubmatch(html)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
