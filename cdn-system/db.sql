-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- 主机： mysql_b5r2-mysql_B5R2-1
-- 生成日期： 2026-02-06 05:02:37
-- 服务器版本： 9.0.1
-- PHP 版本： 8.2.27

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- 数据库： `ck`
--

-- --------------------------------------------------------

--
-- 表的结构 `acl`
--

CREATE TABLE `acl` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `default_action` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `data` mediumtext COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `version` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `api_key`
--

CREATE TABLE `api_key` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `api_key` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `api_secret` varchar(30) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `api_ip` text COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `captcha`
--

CREATE TABLE `captcha` (
  `id` int NOT NULL,
  `email` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` varchar(15) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `captcha` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ip` varchar(18) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `cc_filter`
--

CREATE TABLE `cc_filter` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `within_second` int DEFAULT NULL,
  `max_req` int DEFAULT NULL,
  `max_req_per_uri` int DEFAULT NULL,
  `extra` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `internal` tinyint(1) DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `version` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `cc_match`
--

CREATE TABLE `cc_match` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `data` mediumtext COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `internal` tinyint(1) DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `version` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `cc_rule`
--

CREATE TABLE `cc_rule` (
  `id` int NOT NULL,
  `sort` int DEFAULT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `data` text COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `internal` tinyint(1) DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `is_show` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `version` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `cert`
--

CREATE TABLE `cert` (
  `id` int NOT NULL,
  `uid` bigint DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_general_ci,
  `des` longtext COLLATE utf8mb4_general_ci,
  `type` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `domain` text COLLATE utf8mb4_general_ci,
  `dnsapi` int DEFAULT NULL,
  `cert` text COLLATE utf8mb4_general_ci,
  `key` text COLLATE utf8mb4_general_ci,
  `start_time` datetime DEFAULT NULL,
  `expire_time` datetime DEFAULT NULL,
  `auto_renew` tinyint(1) NOT NULL DEFAULT '0',
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `enable` tinyint(1) NOT NULL DEFAULT '1',
  `task_id` bigint DEFAULT NULL,
  `issue_task_id` bigint DEFAULT NULL,
  `version` int DEFAULT NULL,
  `state` longtext COLLATE utf8mb4_general_ci,
  `last_acme_type` longtext COLLATE utf8mb4_general_ci,
  `last_acme_node_id` bigint DEFAULT NULL,
  `ret` longtext COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `cname_domains`
--

CREATE TABLE `cname_domains` (
  `id` int NOT NULL,
  `domain` varchar(255) COLLATE utf8mb4_general_ci NOT NULL,
  `note` varchar(255) COLLATE utf8mb4_general_ci DEFAULT '',
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `dns_provider_id` bigint DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `config`
--

CREATE TABLE `config` (
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `value` longtext COLLATE utf8mb4_general_ci,
  `type` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `scope_id` bigint DEFAULT NULL,
  `scope_name` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `update_at` datetime(3) DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `dnsapi`
--

CREATE TABLE `dnsapi` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `auth` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ip_switch_log`
--

CREATE TABLE `ip_switch_log` (
  `id` bigint NOT NULL,
  `create_at` datetime DEFAULT NULL,
  `type` varchar(30) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `node_group_id` int DEFAULT NULL,
  `node_id` int DEFAULT NULL,
  `line_id` int DEFAULT NULL,
  `ip` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `action` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `email_need_send` tinyint(1) DEFAULT NULL,
  `email_is_sent` tinyint(1) DEFAULT NULL,
  `email_fail_times` int DEFAULT NULL,
  `email_ret` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `email_time` datetime DEFAULT NULL,
  `email_send_state` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone_need_send` tinyint(1) DEFAULT NULL,
  `phone_is_sent` tinyint(1) DEFAULT NULL,
  `phone_fail_times` int DEFAULT NULL,
  `phone_ret` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone_time` datetime DEFAULT NULL,
  `phone_send_state` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `content` text COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `lets_account`
--

CREATE TABLE `lets_account` (
  `id` int NOT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `invalid_date` datetime DEFAULT NULL,
  `is_created` tinyint(1) DEFAULT NULL,
  `create_failed_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `line`
--

CREATE TABLE `line` (
  `id` int NOT NULL,
  `node_group_id` int DEFAULT NULL,
  `node_id` int DEFAULT NULL,
  `node_ip_id` int DEFAULT NULL,
  `line_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `line_name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `weight` varchar(4) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `record_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `is_backup` tinyint(1) DEFAULT NULL,
  `enable_backup` tinyint(1) DEFAULT NULL,
  `is_backup_default_line` tinyint(1) DEFAULT NULL,
  `enable_backup_default_line` tinyint(1) DEFAULT NULL,
  `switch_at` datetime DEFAULT NULL,
  `disable_by` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `line_delete_queue`
--

CREATE TABLE `line_delete_queue` (
  `id` bigint NOT NULL,
  `node_id` bigint DEFAULT NULL,
  `node_group_id` bigint DEFAULT NULL,
  `line_id` longtext COLLATE utf8mb4_general_ci,
  `line_name` longtext COLLATE utf8mb4_general_ci,
  `delete_at` datetime(3) DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `login_log`
--

CREATE TABLE `login_log` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `ip` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `success` tinyint(1) DEFAULT NULL,
  `post_content` text COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `merge_package_group`
--

CREATE TABLE `merge_package_group` (
  `package_id` int DEFAULT NULL,
  `package_group_id` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `merge_site_group`
--

CREATE TABLE `merge_site_group` (
  `site_id` int DEFAULT NULL,
  `group_id` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `merge_stream_group`
--

CREATE TABLE `merge_stream_group` (
  `stream_id` int DEFAULT NULL,
  `group_id` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `message`
--

CREATE TABLE `message` (
  `id` bigint NOT NULL,
  `type` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `pub_user` int DEFAULT NULL,
  `receive` int DEFAULT NULL,
  `title` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `content` text COLLATE utf8mb4_general_ci,
  `phone_content` text COLLATE utf8mb4_general_ci,
  `event_id` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `user_package_id` int DEFAULT NULL,
  `site_id` int DEFAULT NULL,
  `is_show` tinyint(1) DEFAULT NULL,
  `is_red` tinyint(1) DEFAULT NULL,
  `is_bold` tinyint(1) DEFAULT NULL,
  `is_external` tinyint(1) DEFAULT NULL,
  `is_popup` tinyint(1) DEFAULT NULL,
  `email_need_send` tinyint(1) DEFAULT NULL,
  `phone_need_send` tinyint(1) DEFAULT NULL,
  `email_is_sent` tinyint(1) DEFAULT NULL,
  `phone_is_sent` tinyint(1) DEFAULT NULL,
  `url` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `sort` int DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `message_read`
--

CREATE TABLE `message_read` (
  `uid` int DEFAULT NULL,
  `msg_id` bigint DEFAULT NULL,
  `create_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `message_send`
--

CREATE TABLE `message_send` (
  `id` bigint NOT NULL,
  `uid` int DEFAULT NULL,
  `msg_id` int DEFAULT NULL,
  `media` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `failed_times` int DEFAULT NULL,
  `state` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ret` text COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `message_sub`
--

CREATE TABLE `message_sub` (
  `uid` int DEFAULT NULL,
  `msg_type` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` tinyint(1) DEFAULT NULL,
  `email` tinyint(1) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `node`
--

CREATE TABLE `node` (
  `id` int NOT NULL,
  `pid` bigint DEFAULT NULL,
  `region_id` bigint DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_general_ci,
  `des` longtext COLLATE utf8mb4_general_ci,
  `ip` varchar(191) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `host` longtext COLLATE utf8mb4_general_ci,
  `port` bigint DEFAULT NULL,
  `http_proxy` longtext COLLATE utf8mb4_general_ci,
  `is_mgmt` tinyint(1) DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `update_at` datetime(3) DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `disable_by` longtext COLLATE utf8mb4_general_ci,
  `config_task` longtext COLLATE utf8mb4_general_ci,
  `check_on` tinyint(1) DEFAULT NULL,
  `check_protocol` longtext COLLATE utf8mb4_general_ci,
  `check_timeout` bigint DEFAULT NULL,
  `check_port` bigint DEFAULT NULL,
  `check_host` longtext COLLATE utf8mb4_general_ci,
  `check_path` longtext COLLATE utf8mb4_general_ci,
  `check_node_group` longtext COLLATE utf8mb4_general_ci,
  `check_action` longtext COLLATE utf8mb4_general_ci,
  `bw_limit` longtext COLLATE utf8mb4_general_ci,
  `p_id` bigint DEFAULT NULL,
  `token` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `level` int DEFAULT '1',
  `sort` int DEFAULT '0',
  `cache_dir` longtext COLLATE utf8mb4_general_ci,
  `max_cache_size` bigint DEFAULT NULL,
  `log_dir` longtext COLLATE utf8mb4_general_ci,
  `ssh_host` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ssh_port` int DEFAULT '22',
  `ssh_user` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ssh_auth_type` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ssh_password` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ssh_key` longtext COLLATE utf8mb4_general_ci,
  `work_dir` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `auto_install` tinyint(1) DEFAULT '0',
  `install_status` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'idle',
  `install_error` text COLLATE utf8mb4_general_ci,
  `install_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `node_group`
--

CREATE TABLE `node_group` (
  `id` int NOT NULL,
  `region_id` bigint DEFAULT NULL,
  `cname_hostname` varchar(8) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backup_switch_type` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backup_switch_policy` varchar(80) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `sort_order` int DEFAULT '100',
  `ipv4_resolution` varchar(255) COLLATE utf8mb4_general_ci DEFAULT '',
  `l2_config` varchar(255) COLLATE utf8mb4_general_ci DEFAULT '',
  `cname_domain` longtext COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `node_monitor_log`
--

CREATE TABLE `node_monitor_log` (
  `create_at` datetime DEFAULT NULL,
  `type` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `event_id` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ip` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `success` varchar(2) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `node_id` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `op_log`
--

CREATE TABLE `op_log` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `action` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `content` text COLLATE utf8mb4_general_ci,
  `diff` text COLLATE utf8mb4_general_ci,
  `ip` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `process` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `order`
--

CREATE TABLE `order` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `data` text COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `pay_at` datetime DEFAULT NULL,
  `amount` bigint DEFAULT NULL,
  `pay_type` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `mch_order_no` varchar(40) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `transaction_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `package`
--

CREATE TABLE `package` (
  `id` int NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `region_id` bigint DEFAULT NULL,
  `node_group_id` bigint DEFAULT NULL,
  `backup_node_group` bigint DEFAULT NULL,
  `cname_domain` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_hostname2` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_mode` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `traffic` int DEFAULT NULL,
  `bandwidth` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `connection` int DEFAULT NULL,
  `domain` int DEFAULT NULL,
  `http_port` int DEFAULT NULL,
  `stream_port` int DEFAULT NULL,
  `custom_cc_rule` tinyint(1) DEFAULT NULL,
  `websocket` tinyint(1) DEFAULT NULL,
  `expire` datetime DEFAULT NULL,
  `buy_num_limit` int DEFAULT NULL,
  `backend_ip_limit` text COLLATE utf8mb4_general_ci,
  `id_verify` tinyint(1) DEFAULT NULL,
  `before_exp_days_renew` int DEFAULT NULL,
  `month_price` bigint DEFAULT NULL,
  `quarter_price` bigint DEFAULT NULL,
  `year_price` bigint DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `sort` int DEFAULT NULL,
  `owner` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `l2_origin` tinyint(1) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `package_group`
--

CREATE TABLE `package_group` (
  `id` int NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `sort` int DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `package_up`
--

CREATE TABLE `package_up` (
  `id` int NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `amount` int DEFAULT NULL,
  `bind_package` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `price` bigint DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `region`
--

CREATE TABLE `region` (
  `id` int NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `res_count`
--

CREATE TABLE `res_count` (
  `id` int NOT NULL,
  `time` datetime DEFAULT NULL,
  `user_package` int DEFAULT NULL,
  `uid` int DEFAULT NULL,
  `cate` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `res` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `value` bigint DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `site`
--

CREATE TABLE `site` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `user_package` int DEFAULT NULL,
  `region_id` int DEFAULT NULL,
  `node_group_id` int DEFAULT NULL,
  `backup_node_group` int DEFAULT NULL,
  `enable_backup_group` tinyint(1) DEFAULT NULL,
  `cname_domain` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_hostname2` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_mode` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_hostname` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `domain` text COLLATE utf8mb4_general_ci,
  `http_listen` text COLLATE utf8mb4_general_ci,
  `https_listen` text COLLATE utf8mb4_general_ci,
  `balance_way` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backend` text COLLATE utf8mb4_general_ci,
  `backend_protocol` varchar(32) COLLATE utf8mb4_general_ci DEFAULT 'follow',
  `backend_https_port` varchar(5) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backend_http_port` varchar(5) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `proxy_timeout` varchar(3) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backend_port_mapping` tinyint(1) DEFAULT NULL,
  `health_check` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ups_keepalive` tinyint(1) DEFAULT NULL,
  `ups_keepalive_conn` int DEFAULT NULL,
  `ups_keepalive_timeout` int DEFAULT NULL,
  `proxy_http_version` varchar(3) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `proxy_ssl_protocols` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backend_host` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `range` tinyint(1) DEFAULT NULL,
  `proxy_cache` text COLLATE utf8mb4_general_ci,
  `cc_default_rule` int DEFAULT NULL,
  `cc_switch` text COLLATE utf8mb4_general_ci,
  `extra_cc_rule` text COLLATE utf8mb4_general_ci,
  `block_proxy` tinyint(1) DEFAULT NULL,
  `block_region` text COLLATE utf8mb4_general_ci,
  `black_ip` text COLLATE utf8mb4_general_ci,
  `white_ip` text COLLATE utf8mb4_general_ci,
  `spider_allow` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `acl` int DEFAULT NULL,
  `hotlink` text COLLATE utf8mb4_general_ci,
  `cors` text COLLATE utf8mb4_general_ci,
  `resp_header` text COLLATE utf8mb4_general_ci,
  `req_header` text COLLATE utf8mb4_general_ci,
  `page_404` text COLLATE utf8mb4_general_ci,
  `page_50x` text COLLATE utf8mb4_general_ci,
  `url_rewrite` text COLLATE utf8mb4_general_ci,
  `gzip_enable` tinyint(1) DEFAULT NULL,
  `gzip_types` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `websocket_enable` tinyint(1) DEFAULT NULL,
  `acme_proxy_to_orgin` tinyint(1) DEFAULT NULL,
  `post_size_limit` int DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `version` int DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `cname_task_id` bigint DEFAULT NULL,
  `record_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `dns_provider_id` bigint DEFAULT NULL,
  `settings` longtext COLLATE utf8mb4_general_ci,
  `platform_dns_record_id` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `user_dns_record_id` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `site_conf_cache`
--

CREATE TABLE `site_conf_cache` (
  `site_id` int DEFAULT NULL,
  `templ_md5` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `version` int DEFAULT NULL,
  `data` mediumtext COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `site_group`
--

CREATE TABLE `site_group` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `update_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `stream`
--

CREATE TABLE `stream` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `user_package` int DEFAULT NULL,
  `region_id` int DEFAULT NULL,
  `node_group_id` int DEFAULT NULL,
  `backup_node_group` int DEFAULT NULL,
  `enable_backup_group` tinyint(1) DEFAULT NULL,
  `cname_domain` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_hostname2` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_mode` varchar(10) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `cname_hostname` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `listen` text COLLATE utf8mb4_general_ci,
  `balance_way` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `proxy_protocol` tinyint(1) DEFAULT NULL,
  `backend_port` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `backend` text COLLATE utf8mb4_general_ci,
  `conn_limit` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `acl` text COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `version` int DEFAULT NULL,
  `enable` tinyint(1) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `cname_task_id` bigint DEFAULT NULL,
  `record_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `stream_group`
--

CREATE TABLE `stream_group` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `sys_configs`
--

CREATE TABLE `sys_configs` (
  `key` varchar(191) CHARACTER SET utf8mb3 COLLATE utf8mb3_unicode_ci NOT NULL,
  `value` longtext CHARACTER SET utf8mb3 COLLATE utf8mb3_unicode_ci,
  `version` bigint DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;

-- --------------------------------------------------------

--
-- 表的结构 `task`
--

CREATE TABLE `task` (
  `id` bigint NOT NULL,
  `pid` bigint DEFAULT NULL,
  `pry` bigint DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_general_ci,
  `type` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `res` longtext COLLATE utf8mb4_general_ci,
  `data` longtext COLLATE utf8mb4_general_ci,
  `depend` text COLLATE utf8mb4_general_ci,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `start_at` datetime DEFAULT NULL,
  `end_at` datetime DEFAULT NULL,
  `ret` text COLLATE utf8mb4_general_ci,
  `enable` tinyint(1) DEFAULT NULL,
  `state` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `err_times` int DEFAULT '0',
  `retry_at` datetime DEFAULT NULL,
  `progress` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `targets_json` longtext COLLATE utf8mb4_general_ci
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `tlock`
--

CREATE TABLE `tlock` (
  `name` varchar(30) COLLATE utf8mb4_general_ci NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `user`
--

CREATE TABLE `user` (
  `id` int NOT NULL,
  `email` varchar(191) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `name` varchar(191) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `des` longtext COLLATE utf8mb4_general_ci,
  `phone` longtext COLLATE utf8mb4_general_ci,
  `qq` longtext COLLATE utf8mb4_general_ci,
  `cert_id` longtext COLLATE utf8mb4_general_ci,
  `cert_name` longtext COLLATE utf8mb4_general_ci,
  `cert_no` longtext COLLATE utf8mb4_general_ci,
  `cert_verified` tinyint(1) DEFAULT NULL,
  `white_ip` longtext COLLATE utf8mb4_general_ci,
  `login_captcha` longtext COLLATE utf8mb4_general_ci,
  `balance` bigint DEFAULT NULL,
  `freeze` bigint DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `password` longtext COLLATE utf8mb4_general_ci,
  `enable` tinyint(1) DEFAULT NULL,
  `type` bigint DEFAULT NULL,
  `company` longtext COLLATE utf8mb4_general_ci,
  `tea_code` longtext COLLATE utf8mb4_general_ci,
  `secondary_auth` tinyint(1) DEFAULT NULL,
  `secondary_auth_deadline` longtext COLLATE utf8mb4_general_ci,
  `secondary_auth_action` longtext COLLATE utf8mb4_general_ci,
  `secondary_auth_status` longtext COLLATE utf8mb4_general_ci,
  `group_id` bigint DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `user_package`
--

CREATE TABLE `user_package` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_general_ci,
  `package` int DEFAULT NULL,
  `region_id` bigint DEFAULT NULL,
  `node_group_id` bigint DEFAULT NULL,
  `backup_node_group` bigint DEFAULT NULL,
  `enable_backup_group` tinyint(1) DEFAULT NULL,
  `cname_domain` longtext COLLATE utf8mb4_general_ci,
  `cname_hostname2` longtext COLLATE utf8mb4_general_ci,
  `cname_hostname` longtext COLLATE utf8mb4_general_ci,
  `cname_mode` longtext COLLATE utf8mb4_general_ci,
  `record_id` longtext COLLATE utf8mb4_general_ci,
  `traffic` int DEFAULT NULL,
  `bandwidth` longtext COLLATE utf8mb4_general_ci,
  `connection` int DEFAULT NULL,
  `domain` int DEFAULT NULL,
  `http_port` int DEFAULT NULL,
  `stream_port` int DEFAULT NULL,
  `custom_cc_rule` tinyint(1) DEFAULT NULL,
  `websocket` tinyint(1) DEFAULT NULL,
  `month_price` bigint DEFAULT NULL,
  `quarter_price` bigint DEFAULT NULL,
  `year_price` bigint DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `start_at` datetime(3) DEFAULT NULL,
  `end_at` datetime(3) DEFAULT NULL,
  `task_id` bigint DEFAULT NULL,
  `version` bigint DEFAULT '1',
  `is_expired` tinyint(1) DEFAULT NULL,
  `main_domain_limit` int DEFAULT NULL,
  `l2_origin` tinyint(1) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `user_package_up`
--

CREATE TABLE `user_package_up` (
  `id` int NOT NULL,
  `uid` int DEFAULT NULL,
  `package_up` int DEFAULT NULL,
  `user_package` int DEFAULT NULL,
  `amount` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- 转储表的索引
--

--
-- 表的索引 `acl`
--
ALTER TABLE `acl`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `user_ibfk_5` (`uid`),
  ADD KEY `task_ibfk_5` (`task_id`);

--
-- 表的索引 `api_key`
--
ALTER TABLE `api_key`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_api_key` (`api_key`),
  ADD KEY `user_ibfk_18` (`uid`);

--
-- 表的索引 `captcha`
--
ALTER TABLE `captcha`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_email` (`email`),
  ADD KEY `idx_phone` (`phone`),
  ADD KEY `idx_ip` (`ip`),
  ADD KEY `idx_create_at` (`create_at`);

--
-- 表的索引 `cc_filter`
--
ALTER TABLE `cc_filter`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `idx_internal` (`internal`),
  ADD KEY `user_ibfk_8` (`uid`),
  ADD KEY `task_ibfk_7` (`task_id`);

--
-- 表的索引 `cc_match`
--
ALTER TABLE `cc_match`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `idx_internal` (`internal`),
  ADD KEY `user_ibfk_7` (`uid`),
  ADD KEY `task_ibfk_6` (`task_id`);

--
-- 表的索引 `cc_rule`
--
ALTER TABLE `cc_rule`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_internal` (`internal`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `user_ibfk_6` (`uid`),
  ADD KEY `task_ibfk_8` (`task_id`);

--
-- 表的索引 `cert`
--
ALTER TABLE `cert`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_expire_time` (`expire_time`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `user_ibfk_4` (`uid`),
  ADD KEY `dnsapi_ibfk_1` (`dnsapi`),
  ADD KEY `task_ibfk_3` (`task_id`),
  ADD KEY `task_ibfk_4` (`issue_task_id`);

--
-- 表的索引 `cname_domains`
--
ALTER TABLE `cname_domains`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `idx_cname_domains_domain` (`domain`);

--
-- 表的索引 `config`
--
ALTER TABLE `config`
  ADD UNIQUE KEY `name` (`name`,`type`,`scope_id`,`scope_name`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_name` (`name`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `task_ibfk_14` (`task_id`);

--
-- 表的索引 `dnsapi`
--
ALTER TABLE `dnsapi`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `user_ibfk_3` (`uid`);

--
-- 表的索引 `ip_switch_log`
--
ALTER TABLE `ip_switch_log`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_node_id` (`node_id`);

--
-- 表的索引 `lets_account`
--
ALTER TABLE `lets_account`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `line`
--
ALTER TABLE `line`
  ADD PRIMARY KEY (`id`),
  ADD KEY `node_group_ibfk_3` (`node_group_id`),
  ADD KEY `node_ibfk_1` (`node_id`),
  ADD KEY `node_ibfk_3` (`node_ip_id`),
  ADD KEY `task_ibfk_1` (`task_id`);

--
-- 表的索引 `line_delete_queue`
--
ALTER TABLE `line_delete_queue`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `login_log`
--
ALTER TABLE `login_log`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_ip` (`ip`),
  ADD KEY `idx_success` (`success`),
  ADD KEY `idx_create_at` (`create_at`),
  ADD KEY `user_ibfk_1` (`uid`);

--
-- 表的索引 `merge_package_group`
--
ALTER TABLE `merge_package_group`
  ADD KEY `package_ibfk_1` (`package_id`),
  ADD KEY `package_group_ibfk_1` (`package_group_id`);

--
-- 表的索引 `merge_site_group`
--
ALTER TABLE `merge_site_group`
  ADD KEY `site_ibfk_1` (`site_id`),
  ADD KEY `site_group_ibfk_1` (`group_id`);

--
-- 表的索引 `merge_stream_group`
--
ALTER TABLE `merge_stream_group`
  ADD KEY `stream_ibfk_1` (`stream_id`),
  ADD KEY `stream_group_ibfk_1` (`group_id`);

--
-- 表的索引 `message`
--
ALTER TABLE `message`
  ADD PRIMARY KEY (`id`),
  ADD KEY `type_idx` (`type`),
  ADD KEY `receive_idx` (`receive`),
  ADD KEY `is_show_idx` (`is_show`),
  ADD KEY `create_at_idx` (`create_at`),
  ADD KEY `user_package_id_idx` (`user_package_id`),
  ADD KEY `site_id_idx` (`site_id`),
  ADD KEY `event_id_idx` (`event_id`);

--
-- 表的索引 `message_read`
--
ALTER TABLE `message_read`
  ADD KEY `message_ibfk_1` (`msg_id`),
  ADD KEY `user_ibfk_10` (`uid`);

--
-- 表的索引 `message_send`
--
ALTER TABLE `message_send`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `message_sub`
--
ALTER TABLE `message_sub`
  ADD KEY `user_ibfk_19` (`uid`);

--
-- 表的索引 `node`
--
ALTER TABLE `node`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `idx_ip` (`ip`),
  ADD KEY `region_ibfk_1` (`region_id`),
  ADD KEY `idx_token` (`token`),
  ADD KEY `idx_node_ip` (`ip`);

--
-- 表的索引 `node_group`
--
ALTER TABLE `node_group`
  ADD PRIMARY KEY (`id`),
  ADD KEY `region_ibfk_2` (`region_id`);

--
-- 表的索引 `node_monitor_log`
--
ALTER TABLE `node_monitor_log`
  ADD KEY `idx_create_at` (`create_at`),
  ADD KEY `idx_event_id` (`event_id`),
  ADD KEY `idx_ip` (`ip`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_success` (`success`);

--
-- 表的索引 `op_log`
--
ALTER TABLE `op_log`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_action` (`action`),
  ADD KEY `idx_ip` (`ip`),
  ADD KEY `idx_create_at` (`create_at`),
  ADD KEY `idx_process` (`process`),
  ADD KEY `user_ibfk_2` (`uid`);

--
-- 表的索引 `order`
--
ALTER TABLE `order`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_state` (`state`),
  ADD KEY `idx_mch_order_no` (`mch_order_no`),
  ADD KEY `user_ibfk_16` (`uid`);

--
-- 表的索引 `package`
--
ALTER TABLE `package`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `region_ibfk_3` (`region_id`),
  ADD KEY `node_group_2` (`node_group_id`),
  ADD KEY `node_group_ibfk_4` (`backup_node_group`);

--
-- 表的索引 `package_group`
--
ALTER TABLE `package_group`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `package_up`
--
ALTER TABLE `package_up`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_enable` (`enable`);

--
-- 表的索引 `region`
--
ALTER TABLE `region`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `res_count`
--
ALTER TABLE `res_count`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_time` (`time`),
  ADD KEY `idx_user_package` (`user_package`),
  ADD KEY `idx_uid` (`uid`),
  ADD KEY `idx_cate` (`cate`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_res` (`res`);

--
-- 表的索引 `site`
--
ALTER TABLE `site`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `user_ibfk_9` (`uid`),
  ADD KEY `user_package_ibfk_3` (`user_package`),
  ADD KEY `acl_ibfk_2` (`acl`),
  ADD KEY `task_ibfk_9` (`task_id`),
  ADD KEY `region_ibfk_4` (`region_id`),
  ADD KEY `task_ibfk_19` (`cname_task_id`),
  ADD KEY `node_group_ibfk_6` (`backup_node_group`),
  ADD KEY `node_group_ibfk_8` (`node_group_id`);

--
-- 表的索引 `site_conf_cache`
--
ALTER TABLE `site_conf_cache`
  ADD KEY `idx_site_id` (`site_id`);

--
-- 表的索引 `site_group`
--
ALTER TABLE `site_group`
  ADD PRIMARY KEY (`id`),
  ADD KEY `user_ibfk_11` (`uid`);

--
-- 表的索引 `stream`
--
ALTER TABLE `stream`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `user_ibfk_12` (`uid`),
  ADD KEY `user_package_ibfk_2` (`user_package`),
  ADD KEY `task_ibfk_11` (`task_id`),
  ADD KEY `region_ibfk_5` (`region_id`),
  ADD KEY `task_ibfk_20` (`cname_task_id`),
  ADD KEY `node_group_ibfk_7` (`backup_node_group`),
  ADD KEY `node_group_ibfk_9` (`node_group_id`);

--
-- 表的索引 `stream_group`
--
ALTER TABLE `stream_group`
  ADD PRIMARY KEY (`id`),
  ADD KEY `user_ibfk_13` (`uid`);

--
-- 表的索引 `sys_configs`
--
ALTER TABLE `sys_configs`
  ADD PRIMARY KEY (`key`);

--
-- 表的索引 `task`
--
ALTER TABLE `task`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_pid` (`pid`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_create_at` (`create_at`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `idx_state` (`state`),
  ADD KEY `idx_pry` (`pry`);

--
-- 表的索引 `tlock`
--
ALTER TABLE `tlock`
  ADD PRIMARY KEY (`name`);

--
-- 表的索引 `user`
--
ALTER TABLE `user`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_name` (`name`),
  ADD KEY `idx_email` (`email`),
  ADD KEY `idx_enable` (`enable`),
  ADD KEY `idx_type` (`type`),
  ADD KEY `idx_user_email` (`email`),
  ADD KEY `idx_user_name` (`name`),
  ADD KEY `idx_user_type` (`type`);

--
-- 表的索引 `user_package`
--
ALTER TABLE `user_package`
  ADD PRIMARY KEY (`id`),
  ADD KEY `user_ibfk_14` (`uid`),
  ADD KEY `package_ibfk_3` (`package`),
  ADD KEY `region_ibfk_6` (`region_id`),
  ADD KEY `node_group_ibfk_1` (`node_group_id`),
  ADD KEY `node_group_ibfk_5` (`backup_node_group`),
  ADD KEY `task_ibfk_21` (`task_id`);

--
-- 表的索引 `user_package_up`
--
ALTER TABLE `user_package_up`
  ADD PRIMARY KEY (`id`),
  ADD KEY `user_ibfk_15` (`uid`),
  ADD KEY `package_up_ibfk_1` (`package_up`),
  ADD KEY `user_package_ibfk_1` (`user_package`);

--
-- 在导出的表使用AUTO_INCREMENT
--

--
-- 使用表AUTO_INCREMENT `acl`
--
ALTER TABLE `acl`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `api_key`
--
ALTER TABLE `api_key`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `captcha`
--
ALTER TABLE `captcha`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `cc_filter`
--
ALTER TABLE `cc_filter`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `cc_match`
--
ALTER TABLE `cc_match`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `cc_rule`
--
ALTER TABLE `cc_rule`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `cert`
--
ALTER TABLE `cert`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `cname_domains`
--
ALTER TABLE `cname_domains`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `dnsapi`
--
ALTER TABLE `dnsapi`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ip_switch_log`
--
ALTER TABLE `ip_switch_log`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `lets_account`
--
ALTER TABLE `lets_account`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `line`
--
ALTER TABLE `line`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `line_delete_queue`
--
ALTER TABLE `line_delete_queue`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `login_log`
--
ALTER TABLE `login_log`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `message`
--
ALTER TABLE `message`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `message_send`
--
ALTER TABLE `message_send`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `node`
--
ALTER TABLE `node`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `node_group`
--
ALTER TABLE `node_group`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `op_log`
--
ALTER TABLE `op_log`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `order`
--
ALTER TABLE `order`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `package`
--
ALTER TABLE `package`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `package_group`
--
ALTER TABLE `package_group`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `package_up`
--
ALTER TABLE `package_up`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `region`
--
ALTER TABLE `region`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `res_count`
--
ALTER TABLE `res_count`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `site`
--
ALTER TABLE `site`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `site_group`
--
ALTER TABLE `site_group`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `stream`
--
ALTER TABLE `stream`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `stream_group`
--
ALTER TABLE `stream_group`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `task`
--
ALTER TABLE `task`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `user`
--
ALTER TABLE `user`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `user_package`
--
ALTER TABLE `user_package`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `user_package_up`
--
ALTER TABLE `user_package_up`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 限制导出的表
--

--
-- 限制表 `acl`
--
ALTER TABLE `acl`
  ADD CONSTRAINT `user_ibfk_5` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `api_key`
--
ALTER TABLE `api_key`
  ADD CONSTRAINT `user_ibfk_18` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `cert`
--
ALTER TABLE `cert`
  ADD CONSTRAINT `dnsapi_ibfk_1` FOREIGN KEY (`dnsapi`) REFERENCES `dnsapi` (`id`),
  ADD CONSTRAINT `task_ibfk_3` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `task_ibfk_4` FOREIGN KEY (`issue_task_id`) REFERENCES `task` (`id`);

--
-- 限制表 `config`
--
ALTER TABLE `config`
  ADD CONSTRAINT `task_ibfk_14` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`);

--
-- 限制表 `dnsapi`
--
ALTER TABLE `dnsapi`
  ADD CONSTRAINT `user_ibfk_3` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `line`
--
ALTER TABLE `line`
  ADD CONSTRAINT `node_group_ibfk_3` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`),
  ADD CONSTRAINT `node_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `node` (`id`),
  ADD CONSTRAINT `node_ibfk_3` FOREIGN KEY (`node_ip_id`) REFERENCES `node` (`id`),
  ADD CONSTRAINT `task_ibfk_1` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`);

--
-- 限制表 `login_log`
--
ALTER TABLE `login_log`
  ADD CONSTRAINT `user_ibfk_1` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `merge_package_group`
--
ALTER TABLE `merge_package_group`
  ADD CONSTRAINT `package_group_ibfk_1` FOREIGN KEY (`package_group_id`) REFERENCES `package_group` (`id`),
  ADD CONSTRAINT `package_ibfk_1` FOREIGN KEY (`package_id`) REFERENCES `package` (`id`);

--
-- 限制表 `merge_site_group`
--
ALTER TABLE `merge_site_group`
  ADD CONSTRAINT `site_group_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `site_group` (`id`),
  ADD CONSTRAINT `site_ibfk_1` FOREIGN KEY (`site_id`) REFERENCES `site` (`id`);

--
-- 限制表 `merge_stream_group`
--
ALTER TABLE `merge_stream_group`
  ADD CONSTRAINT `stream_group_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `stream_group` (`id`),
  ADD CONSTRAINT `stream_ibfk_1` FOREIGN KEY (`stream_id`) REFERENCES `stream` (`id`);

--
-- 限制表 `message_read`
--
ALTER TABLE `message_read`
  ADD CONSTRAINT `message_ibfk_1` FOREIGN KEY (`msg_id`) REFERENCES `message` (`id`),
  ADD CONSTRAINT `user_ibfk_10` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `message_sub`
--
ALTER TABLE `message_sub`
  ADD CONSTRAINT `user_ibfk_19` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `op_log`
--
ALTER TABLE `op_log`
  ADD CONSTRAINT `user_ibfk_2` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `order`
--
ALTER TABLE `order`
  ADD CONSTRAINT `user_ibfk_16` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `site`
--
ALTER TABLE `site`
  ADD CONSTRAINT `acl_ibfk_2` FOREIGN KEY (`acl`) REFERENCES `acl` (`id`),
  ADD CONSTRAINT `node_group_ibfk_6` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`),
  ADD CONSTRAINT `task_ibfk_19` FOREIGN KEY (`cname_task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `task_ibfk_9` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `user_ibfk_9` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
  ADD CONSTRAINT `user_package_ibfk_3` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`);

--
-- 限制表 `site_group`
--
ALTER TABLE `site_group`
  ADD CONSTRAINT `user_ibfk_11` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `stream`
--
ALTER TABLE `stream`
  ADD CONSTRAINT `node_group_ibfk_7` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`),
  ADD CONSTRAINT `node_group_ibfk_9` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`),
  ADD CONSTRAINT `region_ibfk_5` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`),
  ADD CONSTRAINT `task_ibfk_11` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `task_ibfk_20` FOREIGN KEY (`cname_task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `user_ibfk_12` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
  ADD CONSTRAINT `user_package_ibfk_2` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`);

--
-- 限制表 `stream_group`
--
ALTER TABLE `stream_group`
  ADD CONSTRAINT `user_ibfk_13` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `user_package`
--
ALTER TABLE `user_package`
  ADD CONSTRAINT `package_ibfk_3` FOREIGN KEY (`package`) REFERENCES `package` (`id`),
  ADD CONSTRAINT `task_ibfk_21` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
  ADD CONSTRAINT `user_ibfk_14` FOREIGN KEY (`uid`) REFERENCES `user` (`id`);

--
-- 限制表 `user_package_up`
--
ALTER TABLE `user_package_up`
  ADD CONSTRAINT `package_up_ibfk_1` FOREIGN KEY (`package_up`) REFERENCES `package_up` (`id`),
  ADD CONSTRAINT `user_ibfk_15` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
  ADD CONSTRAINT `user_package_ibfk_1` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`);
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
