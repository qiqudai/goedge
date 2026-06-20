#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DB_PATH="${1:-$ROOT_DIR/.runtime/cnn-local.db}"

mkdir -p "$(dirname "$DB_PATH")"

sqlite3 "$DB_PATH" <<'SQL'
PRAGMA journal_mode=WAL;

CREATE TABLE IF NOT EXISTS user (
  id INTEGER PRIMARY KEY,
  email TEXT NULL,
  name TEXT NULL,
  des TEXT NULL,
  phone TEXT NULL,
  qq TEXT NULL,
  cert_id TEXT NULL,
  cert_name TEXT NULL,
  cert_no TEXT NULL,
  cert_verified INTEGER NULL,
  white_ip TEXT NULL,
  login_captcha TEXT NULL,
  balance INTEGER NULL,
  freeze INTEGER NULL,
  create_at TEXT NULL,
  password TEXT NULL,
  enable INTEGER NULL,
  type INTEGER NULL
);

CREATE TABLE IF NOT EXISTS login_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NULL,
  ip TEXT NULL,
  create_at TEXT NULL,
  success INTEGER NULL,
  post_content TEXT NULL
);

CREATE TABLE IF NOT EXISTS config (
  name TEXT NULL,
  value TEXT NULL,
  type TEXT NULL,
  scope_id INTEGER NULL,
  scope_name TEXT NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  enable INTEGER NULL,
  task_id INTEGER NULL
);

CREATE TABLE IF NOT EXISTS package (
  id INTEGER PRIMARY KEY,
  name TEXT NULL,
  des TEXT NULL,
  region_id INTEGER NULL,
  node_group_id INTEGER NULL,
  backup_node_group INTEGER NULL,
  cname_domain TEXT NULL,
  cname_hostname2 TEXT NULL,
  cname_mode TEXT NULL,
  traffic INTEGER NULL,
  bandwidth TEXT NULL,
  connection INTEGER NULL,
  domain INTEGER NULL,
  http_port INTEGER NULL,
  stream_port INTEGER NULL,
  custom_cc_rule INTEGER NULL,
  websocket INTEGER NULL,
  l2_origin INTEGER NULL,
  expire TEXT NULL,
  buy_num_limit INTEGER NULL,
  backend_ip_limit TEXT NULL,
  id_verify INTEGER NULL,
  before_exp_days_renew INTEGER NULL,
  month_price INTEGER NULL,
  quarter_price INTEGER NULL,
  year_price INTEGER NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  sort INTEGER NULL,
  owner TEXT NULL,
  enable INTEGER NULL
);

CREATE TABLE IF NOT EXISTS user_package (
  id INTEGER PRIMARY KEY,
  uid INTEGER NULL,
  name TEXT NULL,
  package INTEGER NULL,
  region_id INTEGER NULL,
  node_group_id INTEGER NULL,
  backup_node_group INTEGER NULL,
  enable_backup_group INTEGER NULL,
  cname_domain TEXT NULL,
  cname_hostname2 TEXT NULL,
  cname_hostname TEXT NULL,
  cname_mode TEXT NULL,
  record_id TEXT NULL,
  traffic INTEGER NULL,
  bandwidth TEXT NULL,
  connection INTEGER NULL,
  domain INTEGER NULL,
  main_domain_limit INTEGER NULL,
  http_port INTEGER NULL,
  stream_port INTEGER NULL,
  custom_cc_rule INTEGER NULL,
  websocket INTEGER NULL,
  l2_origin INTEGER NULL,
  month_price INTEGER NULL,
  quarter_price INTEGER NULL,
  year_price INTEGER NULL,
  create_at TEXT NULL,
  start_at TEXT NULL,
  end_at TEXT NULL,
  task_id INTEGER NULL,
  version INTEGER NULL,
  is_expired INTEGER NULL
);

CREATE TABLE IF NOT EXISTS site (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NULL,
  user_package INTEGER NULL,
  region_id INTEGER NULL,
  node_group_id INTEGER NULL,
  backup_node_group INTEGER NULL,
  enable_backup_group INTEGER NULL,
  dns_provider_id INTEGER NULL,
  platform_dns_record_id TEXT NULL,
  user_dns_record_id TEXT NULL,
  cname_domain TEXT NULL,
  cname_hostname2 TEXT NULL,
  cname_mode TEXT NULL,
  cname_hostname TEXT NULL,
  domain TEXT NULL,
  http_listen TEXT NULL,
  https_listen TEXT NULL,
  balance_way TEXT NULL,
  backend TEXT NULL,
  backend_protocol TEXT NULL,
  backend_https_port TEXT NULL,
  backend_http_port TEXT NULL,
  proxy_timeout TEXT NULL,
  backend_port_mapping INTEGER NULL,
  health_check TEXT NULL,
  ups_keepalive INTEGER NULL,
  ups_keepalive_conn INTEGER NULL,
  ups_keepalive_timeout INTEGER NULL,
  proxy_http_version TEXT NULL,
  proxy_ssl_protocols TEXT NULL,
  backend_host TEXT NULL,
  range INTEGER NULL,
  proxy_cache TEXT NULL,
  cc_default_rule INTEGER NULL,
  cc_switch TEXT NULL,
  extra_cc_rule TEXT NULL,
  block_proxy INTEGER NULL,
  block_region TEXT NULL,
  black_ip TEXT NULL,
  white_ip TEXT NULL,
  spider_allow TEXT NULL,
  acl INTEGER NULL,
  hotlink TEXT NULL,
  cors TEXT NULL,
  resp_header TEXT NULL,
  req_header TEXT NULL,
  page_404 TEXT NULL,
  page_50x TEXT NULL,
  url_rewrite TEXT NULL,
  gzip_enable INTEGER NULL,
  gzip_types TEXT NULL,
  websocket_enable INTEGER NULL,
  acme_proxy_to_orgin INTEGER NULL,
  post_size_limit INTEGER NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  version INTEGER NULL,
  enable INTEGER NULL,
  task_id INTEGER NULL,
  cname_task_id INTEGER NULL,
  record_id TEXT NULL,
  state TEXT NULL
);

CREATE TABLE IF NOT EXISTS merge_site_group (
  site_id INTEGER NULL,
  group_id INTEGER NULL
);

CREATE TABLE IF NOT EXISTS site_group (
  id INTEGER PRIMARY KEY,
  uid INTEGER NULL,
  name TEXT NULL,
  des TEXT NULL
);

CREATE TABLE IF NOT EXISTS node_group (
  id INTEGER PRIMARY KEY,
  region_id INTEGER NULL,
  cname_hostname TEXT NULL,
  cname_domain TEXT NULL,
  name TEXT NULL,
  des TEXT NULL,
  backup_switch_type TEXT NULL,
  backup_switch_policy TEXT NULL,
  create_at TEXT NULL,
  update_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS region (
  id INTEGER PRIMARY KEY,
  name TEXT NULL,
  des TEXT NULL,
  create_at TEXT NULL,
  update_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS cert (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NULL,
  name TEXT NULL,
  des TEXT NULL,
  type TEXT NULL,
  domain TEXT NULL,
  dnsapi INTEGER NULL,
  cert TEXT NULL,
  key TEXT NULL,
  start_time TEXT NULL,
  expire_time TEXT NULL,
  auto_renew INTEGER NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  enable INTEGER NULL,
  task_id INTEGER NULL,
  issue_task_id INTEGER NULL,
  state TEXT NULL,
  ret TEXT NULL,
  version INTEGER NULL
);

CREATE TABLE IF NOT EXISTS line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_group_id INTEGER NULL,
  node_id INTEGER NULL,
  node_ip_id INTEGER NULL,
  line_id TEXT NULL,
  line_name TEXT NULL,
  weight TEXT NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  record_id TEXT NULL,
  task_id INTEGER NULL,
  enable INTEGER NULL,
  is_backup INTEGER NULL,
  enable_backup INTEGER NULL,
  is_backup_default_line INTEGER NULL,
  enable_backup_default_line INTEGER NULL,
  switch_at TEXT NULL,
  disable_by TEXT NULL
);

CREATE TABLE IF NOT EXISTS node (
  id INTEGER PRIMARY KEY,
  pid INTEGER NULL,
  region_id INTEGER NULL,
  name TEXT NULL,
  des TEXT NULL,
  ip TEXT NULL,
  token TEXT NULL,
  host TEXT NULL,
  port INTEGER NULL,
  http_proxy TEXT NULL,
  is_mgmt INTEGER NULL,
  create_at TEXT NULL,
  update_at TEXT NULL,
  enable INTEGER NULL,
  disable_by TEXT NULL,
  config_task TEXT NULL,
  check_on INTEGER NULL,
  check_protocol TEXT NULL,
  check_timeout INTEGER NULL,
  check_port INTEGER NULL,
  check_host TEXT NULL,
  check_path TEXT NULL,
  check_node_group TEXT NULL,
  check_action TEXT NULL,
  bw_limit TEXT NULL,
  level INTEGER NULL,
  sort INTEGER NULL,
  cache_dir TEXT NULL,
  max_cache_size INTEGER NULL,
  log_dir TEXT NULL,
  ssh_host TEXT NULL,
  ssh_port INTEGER NULL,
  ssh_user TEXT NULL,
  ssh_auth_type TEXT NULL,
  ssh_password TEXT NULL,
  ssh_key TEXT NULL,
  work_dir TEXT NULL,
  auto_install INTEGER NULL,
  install_status TEXT NULL,
  install_error TEXT NULL,
  install_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS dnsapi (
  id INTEGER PRIMARY KEY,
  uid INTEGER NULL,
  name TEXT NULL,
  des TEXT NULL,
  type TEXT NULL,
  auth TEXT NULL
);

CREATE TABLE IF NOT EXISTS cname_domains (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  domain TEXT NOT NULL,
  dns_provider_id INTEGER NOT NULL DEFAULT 0,
  note TEXT NULL,
  create_at TEXT NULL,
  update_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS task (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pid INTEGER NULL,
  pry INTEGER NULL,
  name TEXT NULL,
  type TEXT NULL,
  res TEXT NULL,
  data TEXT NULL,
  targets_json TEXT NULL,
  depend TEXT NULL,
  create_at TEXT NULL,
  start_at TEXT NULL,
  end_at TEXT NULL,
  ret TEXT NULL,
  enable INTEGER NULL,
  state TEXT NULL,
  err_times INTEGER NULL,
  retry_at TEXT NULL,
  progress TEXT NULL
);

CREATE TABLE IF NOT EXISTS balance_ledger (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NOT NULL DEFAULT 0,
  order_id INTEGER NOT NULL DEFAULT 0,
  amount_before INTEGER NOT NULL DEFAULT 0,
  amount_change INTEGER NOT NULL DEFAULT 0,
  amount_after INTEGER NOT NULL DEFAULT 0,
  action TEXT NULL,
  source TEXT NULL,
  reason TEXT NULL,
  operator_id INTEGER NOT NULL DEFAULT 0,
  operator_role TEXT NULL,
  create_at TEXT NOT NULL
);

INSERT OR REPLACE INTO user (id, email, name, password, enable, type, create_at)
VALUES
  (1, 'admin@test.local', 'admin', 'admin123', 1, 1, datetime('now')),
  (2, 'user2@test.local', 'user2', 'user123', 1, 0, datetime('now'));

INSERT OR REPLACE INTO region (id, name, create_at, update_at)
VALUES (1, 'default-region', datetime('now'), datetime('now'));

INSERT OR REPLACE INTO node_group (id, region_id, cname_hostname, cname_domain, name, create_at, update_at)
VALUES (1, 1, 'edge', 'cdn.default.test', 'default-group', datetime('now'), datetime('now'));

INSERT OR REPLACE INTO package (id, name, region_id, node_group_id, enable, create_at, update_at)
VALUES (1, 'default-package', 1, 1, 1, datetime('now'), datetime('now'));

INSERT OR REPLACE INTO user_package (
  id, uid, name, package, region_id, node_group_id, cname_domain, cname_mode, enable_backup_group, create_at, start_at, end_at, is_expired
) VALUES
  (1, 1, 'user-package-1', 1, 1, 1, 'cdn.default.test', 'site', 0, datetime('now'), datetime('now'), datetime('now', '+365 day'), 0),
  (2, 2, 'user-package-2', 1, 1, 1, 'cdn.default.test', 'site', 0, datetime('now'), datetime('now'), datetime('now', '+365 day'), 0);

INSERT OR REPLACE INTO dnsapi (id, uid, name, type, auth)
VALUES (1, 1, 'smoke-dns', 'huawei', '{}');

INSERT OR IGNORE INTO cname_domains (domain, dns_provider_id, note, create_at, update_at)
VALUES
  ('smoke-cname-b.test', 1, 'default smoke cname', datetime('now'), datetime('now')),
  ('cdn.default.test', 1, 'default domain', datetime('now'), datetime('now'));

INSERT OR REPLACE INTO config (name, value, type, scope_name, scope_id, enable, create_at, update_at)
VALUES
  ('edge_config_version', '1', 'system', 'global', 0, 1, datetime('now'), datetime('now')),
  ('system_info', '{"sys_name":"CNN.NET Local","admin_console_title":"CNN.NET Admin","user_console_title":"CNN.NET User"}', 'system', 'global', 0, 1, datetime('now'), datetime('now'));
SQL

echo "SQLite bootstrap completed: $DB_PATH"
echo "Admin: admin / admin123"
echo "User:  user2 / user123"
