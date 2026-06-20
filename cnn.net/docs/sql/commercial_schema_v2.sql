SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS async_task_log;
DROP TABLE IF EXISTS async_task;
DROP TABLE IF EXISTS user_default_config;
DROP TABLE IF EXISTS site_security_binding;
DROP TABLE IF EXISTS security_rule;
DROP TABLE IF EXISTS site_certificate_binding;
DROP TABLE IF EXISTS certificate;
DROP TABLE IF EXISTS stream_app;
DROP TABLE IF EXISTS site_group_member;
DROP TABLE IF EXISTS site_group;
DROP TABLE IF EXISTS site;
DROP TABLE IF EXISTS subscription;
DROP TABLE IF EXISTS product_plan;
DROP TABLE IF EXISTS line_group_node;
DROP TABLE IF EXISTS line_group;
DROP TABLE IF EXISTS node;
DROP TABLE IF EXISTS user_account;

SET FOREIGN_KEY_CHECKS = 1;

CREATE TABLE user_account (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  email VARCHAR(128) NULL,
  phone VARCHAR(32) NULL,
  password_hash VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  user_type VARCHAR(32) NOT NULL DEFAULT 'customer',
  display_name VARCHAR(128) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_account_username (username),
  UNIQUE KEY uk_user_account_email (email),
  UNIQUE KEY uk_user_account_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE node (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  node_code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  region_code VARCHAR(64) NULL,
  public_ip VARCHAR(64) NULL,
  private_ip VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_node_code (node_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE line_group (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  group_code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_line_group_code (group_code),
  UNIQUE KEY uk_line_group_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE line_group_node (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  line_group_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  line_code VARCHAR(64) NOT NULL,
  line_name VARCHAR(128) NOT NULL,
  weight INT NOT NULL DEFAULT 100,
  is_backup TINYINT(1) NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_line_group_node_line (line_group_id, node_id, line_code, is_backup),
  KEY idx_line_group_node_node (node_id),
  CONSTRAINT fk_line_group_node_group
    FOREIGN KEY (line_group_id) REFERENCES line_group(id)
    ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT fk_line_group_node_node
    FOREIGN KEY (node_id) REFERENCES node(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product_plan (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  plan_code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  default_line_group_id BIGINT NULL,
  traffic_limit_gb BIGINT NULL,
  bandwidth_limit_mbps BIGINT NULL,
  domain_limit INT NULL,
  http_port_limit INT NULL,
  stream_port_limit INT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_product_plan_code (plan_code),
  UNIQUE KEY uk_product_plan_name (name),
  KEY idx_product_plan_default_line_group (default_line_group_id),
  CONSTRAINT fk_product_plan_default_line_group
    FOREIGN KEY (default_line_group_id) REFERENCES line_group(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE subscription (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  plan_id BIGINT NOT NULL,
  subscription_no VARCHAR(64) NOT NULL,
  display_name VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  start_at DATETIME NULL,
  end_at DATETIME NULL,
  snapshot_json JSON NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_subscription_no (subscription_no),
  KEY idx_subscription_user (user_id),
  KEY idx_subscription_plan (plan_id),
  CONSTRAINT fk_subscription_user
    FOREIGN KEY (user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_subscription_plan
    FOREIGN KEY (plan_id) REFERENCES product_plan(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE site (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  subscription_id BIGINT NOT NULL,
  effective_line_group_id BIGINT NOT NULL,
  name VARCHAR(128) NOT NULL,
  primary_domain VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  line_group_source VARCHAR(32) NOT NULL DEFAULT 'subscription',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_site_user_domain (user_id, primary_domain),
  KEY idx_site_subscription (subscription_id),
  KEY idx_site_line_group (effective_line_group_id),
  CONSTRAINT fk_site_user
    FOREIGN KEY (user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_site_subscription
    FOREIGN KEY (subscription_id) REFERENCES subscription(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_site_effective_line_group
    FOREIGN KEY (effective_line_group_id) REFERENCES line_group(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE site_group (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  owner_user_id BIGINT NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_site_group_owner_name (owner_user_id, name),
  CONSTRAINT fk_site_group_owner
    FOREIGN KEY (owner_user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE site_group_member (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  site_id BIGINT NOT NULL,
  site_group_id BIGINT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_site_group_member (site_id, site_group_id),
  KEY idx_site_group_member_group (site_group_id, site_id),
  CONSTRAINT fk_site_group_member_site
    FOREIGN KEY (site_id) REFERENCES site(id)
    ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT fk_site_group_member_group
    FOREIGN KEY (site_group_id) REFERENCES site_group(id)
    ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE stream_app (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  subscription_id BIGINT NOT NULL,
  effective_line_group_id BIGINT NOT NULL,
  name VARCHAR(128) NOT NULL,
  protocol VARCHAR(16) NOT NULL,
  listen_port INT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_stream_user_protocol_port (user_id, protocol, listen_port),
  KEY idx_stream_subscription (subscription_id),
  KEY idx_stream_line_group (effective_line_group_id),
  CONSTRAINT fk_stream_user
    FOREIGN KEY (user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_stream_subscription
    FOREIGN KEY (subscription_id) REFERENCES subscription(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_stream_effective_line_group
    FOREIGN KEY (effective_line_group_id) REFERENCES line_group(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE certificate (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  cert_no VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  cert_type VARCHAR(32) NOT NULL,
  domain_main VARCHAR(255) NOT NULL,
  cert_pem LONGTEXT NULL,
  key_pem LONGTEXT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  auto_renew TINYINT(1) NOT NULL DEFAULT 0,
  not_before DATETIME NULL,
  not_after DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_certificate_no (cert_no),
  KEY idx_certificate_user (user_id),
  CONSTRAINT fk_certificate_user
    FOREIGN KEY (user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE site_certificate_binding (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  site_id BIGINT NOT NULL,
  certificate_id BIGINT NOT NULL,
  binding_type VARCHAR(32) NOT NULL DEFAULT 'primary',
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_site_certificate_binding (site_id, certificate_id, binding_type),
  KEY idx_site_certificate_cert (certificate_id),
  CONSTRAINT fk_site_certificate_binding_site
    FOREIGN KEY (site_id) REFERENCES site(id)
    ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT fk_site_certificate_binding_cert
    FOREIGN KEY (certificate_id) REFERENCES certificate(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE security_rule (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  owner_user_id BIGINT NOT NULL,
  rule_kind VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  rule_payload_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_security_rule_owner_kind (owner_user_id, rule_kind),
  UNIQUE KEY uk_security_rule_owner_kind_name (owner_user_id, rule_kind, name),
  CONSTRAINT fk_security_rule_owner
    FOREIGN KEY (owner_user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE site_security_binding (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  site_id BIGINT NOT NULL,
  rule_id BIGINT NOT NULL,
  usage_type VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_site_security_binding (site_id, rule_id, usage_type),
  KEY idx_site_security_binding_rule (rule_id),
  CONSTRAINT fk_site_security_binding_site
    FOREIGN KEY (site_id) REFERENCES site(id)
    ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT fk_site_security_binding_rule
    FOREIGN KEY (rule_id) REFERENCES security_rule(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE user_default_config (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  config_key VARCHAR(64) NOT NULL,
  config_value_json JSON NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_default_config (user_id, config_key),
  CONSTRAINT fk_user_default_config_user
    FOREIGN KEY (user_id) REFERENCES user_account(id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE async_task (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  task_no VARCHAR(64) NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  state VARCHAR(32) NOT NULL DEFAULT 'pending',
  owner_user_id BIGINT NULL,
  operator_user_id BIGINT NULL,
  resource_type VARCHAR(32) NULL,
  resource_id BIGINT NULL,
  dedupe_key VARCHAR(128) NULL,
  payload_json JSON NOT NULL,
  result_json JSON NULL,
  error_code VARCHAR(64) NULL,
  error_message TEXT NULL,
  progress_percent INT NULL,
  scheduled_at DATETIME NULL,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_async_task_no (task_no),
  UNIQUE KEY uk_async_task_dedupe_key (dedupe_key),
  KEY idx_async_task_state_type (state, task_type, created_at),
  KEY idx_async_task_owner (owner_user_id, created_at),
  CONSTRAINT fk_async_task_owner
    FOREIGN KEY (owner_user_id) REFERENCES user_account(id)
    ON DELETE SET NULL ON UPDATE RESTRICT,
  CONSTRAINT fk_async_task_operator
    FOREIGN KEY (operator_user_id) REFERENCES user_account(id)
    ON DELETE SET NULL ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE async_task_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  task_id BIGINT NOT NULL,
  stage VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  message TEXT NULL,
  payload_json JSON NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_async_task_log_task (task_id, created_at),
  CONSTRAINT fk_async_task_log_task
    FOREIGN KEY (task_id) REFERENCES async_task(id)
    ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
