package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "regexp"
    "strings"

    _ "github.com/go-sql-driver/mysql"
)

func readDSN(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    re := regexp.MustCompile(`(?m)^\s*db_dsn:\s*"?([^"\r\n]+)"?\s*$`)
    matches := re.FindSubmatch(data)
    if len(matches) < 2 {
        return "", fmt.Errorf("db_dsn not found in %s", path)
    }
    return strings.TrimSpace(string(matches[1])), nil
}

func printRows(db *sql.DB, title, query string, args ...interface{}) {
    fmt.Println("==", title, "==")
    rows, err := db.Query(query, args...)
    if err != nil {
        log.Printf("query failed: %v", err)
        return
    }
    defer rows.Close()

    cols, err := rows.Columns()
    if err != nil {
        log.Printf("columns failed: %v", err)
        return
    }

    for rows.Next() {
        vals := make([]interface{}, len(cols))
        valPtrs := make([]interface{}, len(cols))
        for i := range vals {
            valPtrs[i] = &vals[i]
        }
        if err := rows.Scan(valPtrs...); err != nil {
            log.Printf("scan failed: %v", err)
            return
        }
        for i, col := range cols {
            var s string
            switch v := vals[i].(type) {
            case nil:
                s = "NULL"
            case []byte:
                s = string(v)
            default:
                s = fmt.Sprint(v)
            }
            fmt.Printf("%s=%s", col, s)
            if i < len(cols)-1 {
                fmt.Print(" | ")
            }
        }
        fmt.Println()
    }
    if err := rows.Err(); err != nil {
        log.Printf("rows error: %v", err)
    }
    fmt.Println()
}

func main() {
    dsn, err := readDSN("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }

    printRows(db, "site(323)", "SELECT id, user_package, node_group_id, region_id, domain, http_listen, https_listen, enable, state FROM site WHERE id = 323")
    printRows(db, "site(322)", "SELECT id, user_package, node_group_id, region_id, domain, http_listen, https_listen, enable, state FROM site WHERE id = 322")
    printRows(db, "user_package(7)", "SELECT id, package, node_group_id, region_id FROM user_package WHERE id = 7")
    printRows(db, "package(from user_package 7)", "SELECT p.id, p.node_group_id, p.region_id FROM package p WHERE p.id = (SELECT package FROM user_package WHERE id = 7)")
    printRows(db, "line(node_id=32)", "SELECT node_group_id, node_id, node_ip_id, enable FROM line WHERE node_id = 32 OR node_ip_id = 32")
}
