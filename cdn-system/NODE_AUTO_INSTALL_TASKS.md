# 节点自动安装与 SSH 配置任务清单

> 目标：在创建新节点时支持 SSH 认证与自动安装，自动生成 agent.json（含 token/node_id/work_dir），并立即启动最新 cdn-agent。

## 1. 数据模型扩展（P0）

**模块**：`api/models/node.go` / `db.sql`

**新增字段**：
- `ssh_host`：可选，未填则默认节点 IP
- `ssh_port`：默认 22
- `ssh_user`：SSH 用户名
- `ssh_auth_type`：`password`/`key`
- `ssh_password`：密码（不在响应中回传）
- `ssh_key`：私钥（不在响应中回传）
- `work_dir`：工作目录（默认 `/www`，安装时标准化为 `${work_dir}/edge-node`）
- `auto_install`：是否创建后自动安装

**风险**：中（涉及数据库列扩展）。

---

## 2. API：创建节点 + 自动安装（P0）

**模块**：`api/controllers/node_controller.go`

**改动点**：
- Create/Update 使用请求结构体接收 SSH/工作目录字段
- 生成 token（优先 `config.App.AgentToken`，否则随机 32 位 hex）
- Create 后如 `auto_install=true`，触发安装流程

**伪代码**：
```
req := bindNodeRequest()
token := config.App.AgentToken or randomToken()
node := map req -> model (token, ssh, work_dir, auto_install)
db.Create(node)
if req.auto_install:
  InstallNodeAgent(node, resolveAPIBaseURL(request))
```

---

## 3. 安装服务（P0）

**模块**：`api/services/node_install_service.go`

**职责**：
- 通过 SSH 连接节点
- 上传最新 `cdn-agent` 到 work_dir
- 写入 `agent.json`（reset_resources/bootstrap_sync/bootstrap_start）
- 启动 agent（nohup + 后台）

**关键数据结构**：
```
type SSHInstallConfig struct {
  Host, User, AuthType, Password, Key string
  Port int
  WorkDir string
  NodeID int64
  Token, APIBase string
}
```

**伪代码**：
```
workDir := normalizeWorkDir(req.WorkDir)
mkdir -p workDir
scp cdn-agent -> workDir/cdn-agent (chmod 755)
write agent.json -> workDir/agent.json
nohup workDir/cdn-agent -config workDir/agent.json &
```

---

## 4. 前端：创建节点表单扩展（P1）

**模块**：`web/admin/src/views/nodes/list/NodeEditDialog.vue`

**新增表单**：
- SSH 主机、端口、用户名
- 认证方式（密码/密钥）
- 密码或密钥输入
- 工作目录（默认 `/www`）
- 自动安装开关

**常量**：`web/admin/src/views/nodes/list/constants.js` 增加 label 文案。

---

## 5. 安装状态与重装入口（P1）

**模块**：`web/admin/src/views/nodes/list/NodeTable.vue`、`web/admin/src/views/nodes/List.vue`、`api/controllers/node_controller.go`

**改动点**：
- 节点列表展示安装状态与最近错误（hover 提示）。
- 行操作新增“重新安装”入口，调用后端安装接口。
- 后端新增 `POST /nodes/:id/install` 直接重装。

---

## 6. 迁移与兼容（P1）

**模块**：`api/main.go` + `db.sql`

**改动点**：
- 运行时补齐 node 表新增字段（HasColumn -> AddColumn）
- db.sql 补充列定义，保持初始化一致

---

## 测试计划（WSL）

1. `go test ./...`（API 侧）
2. 创建节点（auto_install=true），确认：
   - 远端 `/www/edge-node/agent.json` 存在且包含 token/node_id
   - 远端 `cdn-agent` 可执行且进程启动
3. Agent 拉取配置成功（/api/v1/agent/config 200）
