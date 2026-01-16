# 部署说明（最简）

本文档面向最小化部署：只跑 API 与 Agent，依赖最少、步骤最短。

## 最小运行文件

API：
- `cdn-system/api/cdn-api`
- `cdn-system/api/config.yaml`
- 可写目录：`cdn-system/api/acme/accounts`（启用 ACME 时需要）

Agent：
- `cdn-system/agent/cdn-agent`
- `cdn-system/agent/agent.json`
- 运行时目录：`cdn-system/agent/edge-node`（首次启动自动创建并解包 OpenResty 与 Lua 资源）

## 前置条件
- Linux x86_64
- MySQL/MariaDB
- 可选：ClickHouse（仅在 `clickhouse_enabled: true` 时需要）

## 部署步骤

### 1. 初始化数据库
```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS CK DEFAULT CHARSET utf8mb4;"
mysql -u root -p CK < /www/server/go_project/openresty/cdn-system/db-src.sql
```

### 2. 配置 API
编辑 `cdn-system/api/config.yaml`，示例：
```yaml
port: "8080"
db_dsn: "root:密码@tcp(127.0.0.1:3306)/CK?charset=utf8mb4&parseTime=True&loc=Local"
debug: false
agent_token: "自定义随机串"
clickhouse_enabled: false
clickhouse_dsn: "http://default:123@127.0.0.1:8123/cdn_logs?compress=true"
acme_email: "admin@example.com"
acme_webroot: "/www/server/go_project/openresty/cdn-system/agent/edge-node/cert/acme"
acme_account_dir: "./acme/accounts"
```

### 3. 启动 API
```bash
nohup /www/server/go_project/openresty/cdn-system/api/cdn-api \
  -config /www/server/go_project/openresty/cdn-system/api/config.yaml \
  > /www/server/go_project/openresty/cdn-system/api/api.run.codex.out \
  2> /www/server/go_project/openresty/cdn-system/api/api.run.codex.err &
```

### 4. 初始化管理员（可选）
```bash
cd /www/server/go_project/openresty/cdn-system/api
go run ./cmd/init_admin/main.go admin [密码] admin@example.com
```

### 5. 配置 Agent
编辑 `cdn-system/agent/agent.json`，示例：
```json
{
  "api": "http://127.0.0.1:8080",
  "token": "与 config.yaml 的 agent_token 一致",
  "node_id": "后台创建的节点 ID",
  "debug": true
}
```

### 6. 启动 Agent
```bash
nohup /www/server/go_project/openresty/cdn-system/agent/cdn-agent \
  -config /www/server/go_project/openresty/cdn-system/agent/agent.json \
  > /www/server/go_project/openresty/cdn-system/agent/agent.run.codex.out \
  2> /www/server/go_project/openresty/cdn-system/agent/agent.run.codex.err &
```

## 端口与网络
- API：`config.yaml` 中 `port`（默认 8080）
- Agent：对外服务通常需要 80/443；四层转发按业务端口开放

## 从源码构建（二进制不存在时）
需要 Go 1.24（`go.mod` 指定 toolchain）：
```bash
cd /www/server/go_project/openresty/cdn-system/api
go build -o cdn-api .
cd /www/server/go_project/openresty/cdn-system/agent
go build -o cdn-agent .
```

## 常见配置提示
- 不用 ClickHouse 时：`clickhouse_enabled: false`
- 使用 ACME 自动签发时：确保 `acme_webroot` 与 `acme_account_dir` 可写
