use cdn;

create table `task` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `pid` int(11),
    `pry` int(11),
    `name` varchar(255),
    `type` varchar(255),
    `res` longtext,
    `data` longtext,
    `depend` text,
    `create_at` datetime,
    `start_at` datetime,
    `end_at` datetime,
    `ret` text,
    `enable` boolean,
    `state` varchar(255),
    `err_times` int(11) default 0,
    `retry_at` datetime,
    `progress` varchar(255),
    KEY `idx_pid` (`pid`),
    KEY `idx_type` (`type`),
    KEY `idx_create_at` (`create_at`),
    KEY `idx_enable` (`enable`),
    KEY `idx_state` (`state`),
    KEY `idx_pry` (`pry`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `site_conf_cache` (
    `site_id` int(11),
    `templ_md5` varchar(32),
    `version` int(11),
    `data` MEDIUMTEXT,
    KEY `idx_site_id` (`site_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `user` (
    `id` int(11) not null AUTO_INCREMENT,
    `email` varchar(255),
    `name` varchar(255),
    `des` varchar(255),
    `phone` varchar(255),
    `qq` varchar(255),
    `cert_id` varchar(32),
    `cert_name` varchar(32),
    `cert_no` varchar(32),
    `cert_verified` boolean,
    `white_ip` varchar(255),
    `login_captcha` varchar(10),
    `balance` bigint default 0,
    `freeze` bigint default 0,
    `create_at` datetime,
    `password` varchar(255),
    `enable` boolean,
    `type` int(11),
    KEY `idx_name` (`name`),
    KEY `idx_email` (`email`),
    KEY `idx_enable` (`enable`),
    KEY `idx_type` (`type`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `login_log` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `ip` varchar(255),
    `create_at` datetime,
    `success` boolean,
    `post_content` text,
    primary KEY `id` (`id`),
    KEY `idx_ip` (`ip`),
    KEY `idx_success` (`success`),
    KEY `idx_create_at` (`create_at`),
    CONSTRAINT `user_ibfk_1` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `region` (
    `id` int(11) not null AUTO_INCREMENT,
    `name` varchar(255),
    `des` varchar(255),
    `create_at` datetime,
    `update_at` datetime,
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `node` (
    `id` int(11) not null AUTO_INCREMENT,
    `pid` int(11) not null,
    `region_id` int(11),
    `name` varchar(255),
    `des` varchar(255),
    `ip` varchar(255),
    `host` varchar(255),
    `port` int(11),
    `http_proxy` varchar(255),
    `is_mgmt` boolean,
    `create_at` datetime,
    `update_at` datetime,
    `enable` boolean,
    `disable_by` varchar(20),
    `config_task` varchar(255),
    `check_on` boolean,
    `check_protocol` varchar(10),
    `check_timeout` int(11),
    `check_port` int(11),
    `check_host` varchar(255),
    `check_path` varchar(255),
    `check_node_group` varchar(255),
    `check_action` varchar(10),
    `bw_limit` varchar(50),

    KEY `idx_enable` (`enable`),
    KEY `idx_ip` (`ip`),
    CONSTRAINT `region_ibfk_1` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `node_monitor_log` (
  `create_at` datetime DEFAULT NULL,
  `type` varchar(10),
  `event_id` varchar(10) DEFAULT NULL,
  `ip` varchar(50) DEFAULT NULL,
  `success` varchar(2) DEFAULT NULL,
  `node_id` int(11) DEFAULT NULL,
  KEY `idx_create_at` (`create_at`),
  KEY `idx_event_id` (`event_id`),
  KEY `idx_ip` (`ip`),
  KEY `idx_type` (`type`),
  KEY `idx_success` (`success`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `node_group` (
    `id` int(11) not null AUTO_INCREMENT,
    `region_id` int(11),
    `cname_hostname` varchar(8),
    `name` varchar(255),
    `des` varchar(255),
    `backup_switch_type` varchar(20),
    `backup_switch_policy` varchar(80),
    `create_at` datetime,
    `update_at` datetime,
    CONSTRAINT `region_ibfk_2` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `line` (
    `id` int(11) not null AUTO_INCREMENT,
    `node_group_id` int(11),
    `node_id` int(11),
    `node_ip_id` int(11),
    `line_id` varchar(255),
    `line_name` varchar(255),    
    `weight` varchar(4),
    `create_at` datetime,
    `update_at` datetime,
    `record_id` varchar(255),
    `task_id` bigint,
    `enable` boolean,
    `is_backup` boolean,
    `enable_backup` boolean,
    `is_backup_default_line` boolean,
    `enable_backup_default_line` boolean,
    `switch_at` datetime,
    `disable_by` varchar(20),
    primary KEY `id` (`id`),
    CONSTRAINT `node_group_ibfk_3` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`),
    CONSTRAINT `node_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `node` (`id`),
    CONSTRAINT `node_ibfk_3` FOREIGN KEY (`node_ip_id`) REFERENCES `node` (`id`),
    CONSTRAINT `task_ibfk_1` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `op_log` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `type` varchar(255),
    `action` varchar(255),
    `content` text,
    `diff` text,
    `ip` varchar(255),
    `create_at` datetime,
    `process` varchar(255),
    primary KEY `id` (`id`),
    KEY `idx_type` (`type`),
    KEY `idx_action` (`action`),
    KEY `idx_ip` (`ip`),
    KEY `idx_create_at` (`create_at`),
    KEY `idx_process` (`process`),
    CONSTRAINT `user_ibfk_2` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `dnsapi` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),    
    `type` varchar(255),
    `auth` varchar(255),
    `is_default` tinyint(1) default 0,
    primary KEY `id` (`id`),
    KEY `idx_type` (`type`),
    CONSTRAINT `user_ibfk_3` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `cert` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),       
    `type` varchar(255),
    `domain` text,
    `dnsapi` int(11),
    `cert` text,
    `key` text,
    `start_time` datetime,
    `expire_time` datetime,
    `auto_renew` boolean,
    `create_at` datetime,
    `update_at` datetime,
    `enable` boolean,
    `task_id` bigint,
    `issue_task_id` bigint,
    `version` int(11),
    KEY `idx_type` (`type`),
    KEY `idx_expire_time` (`expire_time`),
    KEY `idx_enable` (`enable`),
    primary KEY `id` (`id`),
    CONSTRAINT `user_ibfk_4` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `dnsapi_ibfk_1` FOREIGN KEY (`dnsapi`) REFERENCES `dnsapi` (`id`),
    CONSTRAINT `task_ibfk_3` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
    CONSTRAINT `task_ibfk_4` FOREIGN KEY (`issue_task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `acl` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),       
    `default_action` varchar(255),
    `data` MEDIUMTEXT,
    `create_at` datetime,
    `update_at` datetime,
    `enable` boolean,
    `task_id` bigint,
    `version` int(11),
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `user_ibfk_5` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `task_ibfk_5` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `cc_rule` (
    `id` int(11) not null AUTO_INCREMENT,
    `sort` int(11),
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),       
    `data` text,
    `create_at` datetime,
    `update_at` datetime,
    `internal` boolean,
    `enable` boolean,
    `is_show` boolean,
    `task_id` bigint,
    `version` int(11),
    primary KEY `id` (`id`),
    KEY `idx_internal` (`internal`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `user_ibfk_6` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `task_ibfk_8` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `cc_match` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),       
    `data` MEDIUMTEXT,
    `create_at` datetime,
    `update_at` datetime,
    `internal` boolean,
    `enable` boolean,
    `task_id` bigint,    
    `version` int(11),
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    KEY `idx_internal` (`internal`),
    CONSTRAINT `user_ibfk_7` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `task_ibfk_6` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `cc_filter` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),  
    `type` varchar(255),  
    `within_second` int(11),  
    `max_req` int(11),  
    `max_req_per_uri` int(11),  
    `extra` varchar(255),  
    `create_at` datetime,
    `update_at` datetime,
    `internal` boolean,
    `enable` boolean,
    `task_id` bigint,       
    `version` int(11),
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    KEY `idx_internal` (`internal`),
    CONSTRAINT `user_ibfk_8` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `task_ibfk_7` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `package` (
    `id` int(11) not null AUTO_INCREMENT,
    `name` varchar(255),
    `des` varchar(255),  
    `region_id` int(11),
    `node_group_id` int(11),
    `backup_node_group` int(11),
    `cname_domain` varchar(255),
    `cname_hostname2` varchar(255),
    `cname_mode` varchar(10),
    `traffic` int(11),
    `bandwidth` varchar(20),
    `connection` int(11),    
    `domain` int(11),  
    `http_port` int(11),  
    `stream_port` int(11),  
    `custom_cc_rule` boolean,
    `websocket` boolean,
    `expire` datetime,
    `buy_num_limit` int(11),
    `backend_ip_limit` text,
    `id_verify` boolean,
    `before_exp_days_renew` int(11),
    `month_price` bigint,
    `quarter_price` bigint,
    `year_price` bigint,
    `create_at` datetime,
    `update_at` datetime,
    `sort` int(11),
    `owner` varchar(255),
    `enable` boolean,
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `region_ibfk_3` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`),
    CONSTRAINT `node_group_2` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`),
    CONSTRAINT `node_group_ibfk_4` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `package_group` (
    `id` int(11) not null AUTO_INCREMENT,
    `name` varchar(255),
    `des` varchar(255),
    `sort` int(11),
    `create_at` datetime,
    `update_at` datetime,
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `merge_package_group` (
    `package_id` int(11),
    `package_group_id` int(11),
    CONSTRAINT `package_ibfk_1` FOREIGN KEY (`package_id`) REFERENCES `package` (`id`),
    CONSTRAINT `package_group_ibfk_1` FOREIGN KEY (`package_group_id`) REFERENCES `package_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `package_up` (
    `id` int(11) not null AUTO_INCREMENT,
    `name` varchar(255),
    `des` varchar(255),  
    `type` varchar(255),  
    `amount` int(11),
    `bind_package` varchar(255),
    `price` bigint,
    `create_at` datetime,
    `update_at` datetime,    
    `enable` boolean,
    KEY `idx_type` (`type`),
    KEY `idx_enable` (`enable`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `user_package` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255), 
    `package` int(11),
    `region_id` int(11),
    `node_group_id` int(11),
    `backup_node_group` int(11),
    `enable_backup_group` boolean,
    `cname_domain` varchar(255),
    `cname_hostname2` varchar(255),
    `cname_hostname` varchar(255),
    `cname_mode` varchar(10),
    `record_id` varchar(255),
    `traffic` int(11) DEFAULT NULL,
    `bandwidth` varchar(20),
    `connection` int(11),
    `domain` int(11) DEFAULT NULL,
    `http_port` int(11) DEFAULT NULL,
    `stream_port` int(11) DEFAULT NULL,
    `custom_cc_rule` tinyint(1) DEFAULT NULL,
    `websocket` tinyint(1),
    `month_price` bigint(20) DEFAULT NULL,
    `quarter_price` bigint(20) DEFAULT NULL,
    `year_price` bigint(20) DEFAULT NULL,
    `create_at` datetime,
    `start_at` datetime,
    `end_at` datetime,    
    `task_id` bigint,
    primary KEY `id` (`id`),
    CONSTRAINT `user_ibfk_14` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `package_ibfk_3` FOREIGN KEY (`package`) REFERENCES `package` (`id`),
    CONSTRAINT `region_ibfk_6` foreign key(`region_id`) REFERENCES `region`(`id`),
    CONSTRAINT `node_group_ibfk_1` foreign key(`node_group_id`) REFERENCES `node_group`(`id`),
    CONSTRAINT `node_group_ibfk_5` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`),
    CONSTRAINT `task_ibfk_21` foreign key(`task_id`) REFERENCES `task`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `user_package_up` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `package_up` int(11),
    `user_package` int(11),
    `amount` int(11),
    primary KEY `id` (`id`),
    CONSTRAINT `user_ibfk_15` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `package_up_ibfk_1` FOREIGN KEY (`package_up`) REFERENCES `package_up` (`id`),
    CONSTRAINT `user_package_ibfk_1` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `config` (
    `name` varchar(50),
    `value` MEDIUMTEXT,  
    `type` varchar(30),  
    `scope_id` int(11),
    `scope_name` varchar(10),
    `create_at` datetime,
    `update_at` datetime,    
    `enable` boolean,
    `task_id` bigint,
    UNIQUE KEY `name` (`name`,`type`,`scope_id`,`scope_name`),
    KEY `idx_type` (`type`),
    KEY `idx_name` (`name`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `task_ibfk_14` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `site` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `user_package` int(11),
    `region_id` int(11),
    `node_group_id` int(11),
    `backup_node_group` int(11),
    `enable_backup_group` boolean,
    `cname_domain` varchar(255),
    `cname_hostname2` varchar(255),
    `cname_mode` varchar(10),
    `cname_hostname` varchar(255),
    `domain` text,  
    `http_listen` text,  
    `https_listen` text,  
    `balance_way` varchar(255),  
    `backend` text,  
    `backend_protocol` varchar(8),
    `backend_https_port` varchar(5),
    `backend_http_port` varchar(5),
    `proxy_timeout` varchar(3),
    `backend_port_mapping` boolean,
    `health_check` varchar(255),
    `ups_keepalive` boolean,
    `ups_keepalive_conn` int(3),
    `ups_keepalive_timeout` int(4),
    `proxy_http_version` varchar(3),
    `proxy_ssl_protocols` varchar(255),
    `backend_host` varchar(255),
    `range` boolean,
    `proxy_cache` text,  
    `cc_default_rule` int(11),
    `cc_switch` text,
    `extra_cc_rule` text,
    `block_proxy` boolean,
    `block_region` text,
    `black_ip` text,  
    `white_ip` text,  
    `spider_allow` varchar(255),
    `acl` int(11),
    `hotlink` text,
    `cors` text,
    `resp_header` text,  
    `req_header` text,  
    `page_404` text,  
    `page_50x` text,  
    `url_rewrite` text,  
    `gzip_enable` boolean,
    `gzip_types` varchar(255),
    `websocket_enable` boolean,
    `acme_proxy_to_orgin` boolean,
    `post_size_limit` int(11),
    `create_at` datetime,
    `update_at` datetime,   
    `version` int(11),
    `enable` boolean,
    `task_id` bigint,
    `cname_task_id` bigint,
    `record_id` varchar(255),
    `state` varchar(255),
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `user_ibfk_9` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `user_package_ibfk_3` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`),
    CONSTRAINT `acl_ibfk_2` FOREIGN KEY (`acl`) REFERENCES `acl` (`id`),
    CONSTRAINT `task_ibfk_9` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
    CONSTRAINT `region_ibfk_4` foreign key(`region_id`) REFERENCES `region`(`id`),
    CONSTRAINT `task_ibfk_19` foreign key(`cname_task_id`) REFERENCES `task`(`id`),
    CONSTRAINT `node_group_ibfk_6` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`),
    CONSTRAINT `node_group_ibfk_8` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `stream` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `user_package` int(11),
    `region_id` int(11),
    `node_group_id` int(11),
    `backup_node_group` int(11),
    `enable_backup_group` boolean,
    `cname_domain` varchar(255),
    `cname_hostname2` varchar(255),
    `cname_mode` varchar(10),    
    `cname_hostname` varchar(255),
    `listen` text,  
    `balance_way` varchar(255),  
    `proxy_protocol` boolean,
    `backend_port` varchar(255),  
    `backend` text,  
    `conn_limit` varchar(255),
    `acl` text,  
    `create_at` datetime,
    `update_at` datetime,   
    `version` int(11),
    `enable` boolean,
    `task_id` bigint,
    `cname_task_id` bigint,
    `record_id` varchar(255),
    `state` varchar(255),
    primary KEY `id` (`id`),
    KEY `idx_enable` (`enable`),
    CONSTRAINT `user_ibfk_12` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `user_package_ibfk_2` FOREIGN KEY (`user_package`) REFERENCES `user_package` (`id`),
    CONSTRAINT `task_ibfk_11` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`),
    CONSTRAINT `region_ibfk_5` foreign key(`region_id`) REFERENCES `region`(`id`),
    CONSTRAINT `task_ibfk_20` foreign key(`cname_task_id`) REFERENCES `task`(`id`),
    CONSTRAINT `node_group_ibfk_7` FOREIGN KEY (`backup_node_group`) REFERENCES `node_group` (`id`),
    CONSTRAINT `node_group_ibfk_9` FOREIGN KEY (`node_group_id`) REFERENCES `node_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `site_group` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),
    primary KEY `id` (`id`),
    CONSTRAINT `user_ibfk_11` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `merge_site_group` (
    `site_id` int(11) ,
    `group_id` int(11),
    CONSTRAINT `site_ibfk_1` FOREIGN KEY (`site_id`) REFERENCES `site` (`id`),
    CONSTRAINT `site_group_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `site_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `stream_group` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `name` varchar(255),
    `des` varchar(255),
    primary KEY `id` (`id`),
    CONSTRAINT `user_ibfk_13` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `merge_stream_group` (
    `stream_id` int(11) ,
    `group_id` int(11),
    CONSTRAINT `stream_ibfk_1` FOREIGN KEY (`stream_id`) REFERENCES `stream` (`id`),
    CONSTRAINT `stream_group_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `stream_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `order` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `type` varchar(255),
    `des` varchar(255),
    `data` text,
    `create_at` datetime,
    `pay_at` datetime,
    `amount` bigint,
    `pay_type` varchar(20),
    `mch_order_no` varchar(40),
    `transaction_id` varchar(255),
    `state` varchar(255),
    primary KEY `id` (`id`),
    KEY `idx_type` (`type`),
    KEY `idx_state` (`state`),
    KEY `idx_mch_order_no` (`mch_order_no`),
    CONSTRAINT `user_ibfk_16` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `job` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `type` varchar(255),
    `key1`varchar(255),
    `key2`varchar(255),
    `data` text,
    `create_at` datetime,
    `task_id` bigint,
    primary KEY `id` (`id`),
    KEY type_idx (type),
    KEY key1_idx (key1),
    KEY key2_idx (key2),
    CONSTRAINT `user_ibfk_17` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    CONSTRAINT `task_ibfk_18` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `tlock` (
    `name` varchar(30),
    primary KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table `res_count` (
    `id` int(11) not null AUTO_INCREMENT,
    `time` datetime,
    `user_package` int(11),
    `uid` int(11),
    `cate` varchar(255),
    `type` varchar(255),
    `res` varchar(255),
    `value` bigint,
    KEY `idx_time` (`time`),
    KEY `idx_user_package` (`user_package`),
    KEY `idx_uid` (`uid`),
    KEY `idx_cate` (`cate`),
    KEY `idx_type` (`type`),
    KEY `idx_res` (`res`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `captcha` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `email` varchar(50) DEFAULT NULL,
  `phone` varchar(15) DEFAULT NULL,
  `captcha` varchar(10) DEFAULT NULL,
  `ip` varchar(18) DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_email` (`email`),
  KEY `idx_phone` (`phone`),
  KEY `idx_ip` (`ip`),
  KEY `idx_create_at` (`create_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `api_key` (
    `id` int(11) not null AUTO_INCREMENT,
    `uid` int(11),
    `api_key` varchar(16),
    `api_secret` varchar(30),
    `api_ip` text,
    KEY `idx_api_key` (`api_key`),
    CONSTRAINT `user_ibfk_18` FOREIGN KEY (`uid`) REFERENCES `user` (`id`),
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table `message` (
    `id` bigint not null AUTO_INCREMENT,
    `type` varchar(20),
    `pub_user` int(11),
    `receive` int(11),
    `title` varchar(255),
    `content` text,
    `phone_content` text,
    `event_id` varchar(32),
    `user_package_id` int(11),
    `site_id` int(11),
    `is_show` boolean,
    `is_red` boolean,
    `is_bold` boolean,
    `is_external` boolean,
    `is_popup` boolean,
    `email_need_send` boolean,
    `phone_need_send` boolean,
    `email_is_sent` boolean,
    `phone_is_sent` boolean,
    `url` varchar(255),
    `sort` int(11),
    `create_at` datetime,
    `update_at` datetime,
    primary KEY `id` (`id`),
    KEY `type_idx` (`type`),
    KEY `receive_idx` (`receive`),
    KEY `is_show_idx` (`is_show`),
    KEY `create_at_idx` (`create_at`),
    KEY `user_package_id_idx` (`user_package_id`),
    KEY `site_id_idx` (`site_id`),
    index event_id_idx(event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table message_read (
    `uid` int(11),
    `msg_id` bigint,
    `create_at` datetime,
    CONSTRAINT `message_ibfk_1` FOREIGN KEY (`msg_id`) REFERENCES `message` (`id`),
    CONSTRAINT `user_ibfk_10` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table message_sub (
    `uid` int(11),
    `msg_type` varchar(50),
    `phone` boolean,
    `email` boolean,
    CONSTRAINT `user_ibfk_19` FOREIGN KEY (`uid`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

create table message_send (
    `id` bigint not null AUTO_INCREMENT,
    `uid` int(11),
    `msg_id` int(11),
    `media` varchar(10),
    `failed_times` int(11),
    `state` varchar(10),
    `ret` text,
    `create_at` datetime,
    primary KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ip_switch_log` (
  `id` bigint not null auto_increment,
  `create_at` datetime DEFAULT NULL,
  `type` varchar(30) DEFAULT NULL,
  `node_group_id` int(11),
  `node_id` int(11),
  `line_id` int(11),
  `ip` varchar(20) DEFAULT NULL,
  `action` varchar(20) DEFAULT NULL,
  `email_need_send` boolean,
  `email_is_sent` boolean,
  `email_fail_times` int(11),
  `email_ret` varchar(255),
  `email_time` datetime,
  `email_send_state` varchar(10),
  `phone_need_send` boolean,
  `phone_is_sent` boolean,
  `phone_fail_times` int(11),
  `phone_ret` varchar(255),
  `phone_time` datetime,
  `phone_send_state` varchar(10),
  `content` text,
  primary KEY `id` (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_node_id` (`node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table lets_account (
	`id` int(11) NOT NULL AUTO_INCREMENT,
	`enable` boolean,
	`invalid_date` datetime,
    `is_created` boolean,
    `create_failed_at` datetime,
	 PRIMARY KEY (`id`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

# resouce config

insert into  lets_account values (1,1,null,0,null);
insert into  lets_account values (2,1,null,0,null);
insert into  lets_account values (3,1,null,0,null);
insert into  lets_account values (4,1,null,0,null);
insert into  lets_account values (5,1,null,0,null);
insert into  lets_account values (6,1,null,0,null);
insert into  lets_account values (7,1,null,0,null);
insert into  lets_account values (8,1,null,0,null);
insert into  lets_account values (9,1,null,0,null);
insert into  lets_account values (10,1,null,0,null);
insert into  lets_account values (11,1,null,0,null);
insert into  lets_account values (12,1,null,0,null);
insert into  lets_account values (13,1,null,0,null);
insert into  lets_account values (14,1,null,0,null);
insert into  lets_account values (15,1,null,0,null);
insert into  lets_account values (16,1,null,0,null);
insert into  lets_account values (17,1,null,0,null);
insert into  lets_account values (18,1,null,0,null);
insert into  lets_account values (19,1,null,0,null);
insert into  lets_account values (20,1,null,0,null);
insert into  lets_account values (21,1,null,0,null);
insert into  lets_account values (22,1,null,0,null);
insert into  lets_account values (23,1,null,0,null);
insert into  lets_account values (24,1,null,0,null);
insert into  lets_account values (25,1,null,0,null);
insert into  lets_account values (26,1,null,0,null);
insert into  lets_account values (27,1,null,0,null);
insert into  lets_account values (28,1,null,0,null);
insert into  lets_account values (29,1,null,0,null);
insert into  lets_account values (30,1,null,0,null);
insert into  lets_account values (31,1,null,0,null);
insert into  lets_account values (32,1,null,0,null);
insert into  lets_account values (33,1,null,0,null);
insert into  lets_account values (34,1,null,0,null);
insert into  lets_account values (35,1,null,0,null);
insert into  lets_account values (36,1,null,0,null);
insert into  lets_account values (37,1,null,0,null);
insert into  lets_account values (38,1,null,0,null);
insert into  lets_account values (39,1,null,0,null);
insert into  lets_account values (40,1,null,0,null);
insert into  lets_account values (41,1,null,0,null);
insert into  lets_account values (42,1,null,0,null);
insert into  lets_account values (43,1,null,0,null);
insert into  lets_account values (44,1,null,0,null);
insert into  lets_account values (45,1,null,0,null);
insert into  lets_account values (46,1,null,0,null);
insert into  lets_account values (47,1,null,0,null);
insert into  lets_account values (48,1,null,0,null);
insert into  lets_account values (49,1,null,0,null);
insert into  lets_account values (50,1,null,0,null);


# site
insert into config values ('related-config-min-limit',50,'site','0','global', now(),now(),1,null);
insert into config values ('related-config-max-times-limit',3,'site','0','global', now(),now(),1,null); 
insert into config values ('black-ip-limit',50,'site','0','global', now(),now(),1,null);  
insert into config values ('white-ip-limit',50,'site','0','global', now(),now(),1,null);
insert into config values ('max-domain-persite-limit',100,'site','0','global', now(),now(),1,null);
insert into config values ('listen-default-http-80',1,'site','0','global', now(),now(),1,null);
insert into config values ('clean_url',2000,'site','0','global', now(),now(),1,null);
insert into config values ('clean_dir',500,'site','0','global', now(),now(),1,null);
insert into config values ('pre_cache_url',2000,'site','0','global', now(),now(),1,null);
insert into config values ('pre_cache_timeout',120,'site','0','global', now(),now(),1,null);
insert into config values ('ip-unlock-max-limit',1000,'site','0','global', now(),now(),1,null);
insert into config values ('ip-unlock-max-per-limit',50,'site','0','global', now(),now(),1,null);
insert into config values ('cc-rule-max-limit',5,'site','0','global', now(),now(),1,null); 
insert into config values ('acl-max-limit',5,'site','0','global', now(),now(),1,null); 
insert into config values ('download-access-log-limit',10,'site','0','global', now(),now(),1,null);
insert into config values ('download-access-log-tmp-dir','/data/download-temp/','site','0','global', now(),now(),1,null);
insert into config values ('download-access-log-retain','12','site','0','global', now(),now(),1,null);

# stream
insert into config values ('custom-port-not-allow','80 443','stream','0','global', now(),now(),1,null);
insert into config values ('related-config-min-limit',10,'stream','0','global', now(),now(),1,null);
insert into config values ('related-config-max-times-limit',2,'stream','0','global', now(),now(),1,null); 
insert into config values ('acl-max-limit',10,'stream','0','global', now(),now(),1,null); 

# 公共    
insert into config values ('custom-port-not-allow','22 5000 9001 6379','site_stream','0','global', now(),now(),1,null);     
insert into config values ('custom-port-allow','1-65535','site_stream','0','global', now(),now(),1,null);


# nginx config
insert into config values ('nginx-config-file','{"logs_dir":"/usr/local/openresty/nginx/logs/", "worker_rlimit_nofile":51200,"worker_shutdown_timeout":"60s","worker_connections":51200,"worker_processes":"auto","resolver":"223.5.5.5 8.8.8.8","resolver_timeout":"5s","http":{"proxy_request_buffering":"on", "server_addr_outgoing":"1", "large_client_header_buffers":"4 32k", "proxy_cache_dir":"/data/nginx/cache/", "proxy_cache_max_size":"100g", "proxy_cache_keys_zone_size":"1000M", "gzip_comp_level":1,"gzip_http_version":"1.0","gzip_min_length":"1k","gzip_vary":"on","proxy_buffering":"on","proxy_cache_methods":"GET HEAD","proxy_http_version":"1.0","proxy_max_temp_file_size":"1024m","proxy_next_upstream":"error timeout","proxy_connect_timeout":"60s","proxy_send_timeout":"60s","proxy_read_timeout":"60s","server":"cdn","client_max_body_size":"10m","default_type":"text/plain","keepalive_requests":100,"keepalive_timeout":"60s","log_not_found":"off","server_names_hash_max_size":512,"server_names_hash_bucket_size":128, "server_tokens":"off"},"stream":{"proxy_connect_timeout":"60s","proxy_timeout":"10m"}}','nginx_config','0','global', now(),now(),1,null);

# openresty config
set @slider_html = "<!DOCTYPE html>\\n<html>\\n<head>\\n<title>拖动验证</title>\\n<meta charset='utf-8'/>\\n<meta name='viewport' content='width=device-width, initial-scale=1, user-scalable=no'>\\n<meta name='apple-mobile-web-app-capable' content='yes'>\\n<meta name='apple-mobile-web-app-status-bar-style' content='black'>\\n<meta name='format-detection' content='telephone=no'>\\n<link rel='stylesheet' href='https://cdn.staticfile.org/twitter-bootstrap/3.3.4/css/bootstrap.min.css'>\\n<style type='text/css'>\\n.stage{position:relative;padding: 0 15px;height:55px;}\\n.slider{position:absolute;height:52px;box-shadow:0 0 3px #999;background-color:#ddd;left:15px;right:15px;}\\n.tips {\\n    background: -webkit-gradient(linear, left top, right top, color-stop(0, #4d4d4d), color-stop(.4, #4d4d4d), color-stop(.5, white), color-stop(.6, #4d4d4d), color-stop(1, #4d4d4d));\\n    -webkit-background-clip: text;\\n    -webkit-text-fill-color: transparent;\\n    -webkit-animation: slidetounlock 3s infinite;\\n    -webkit-text-size-adjust: none;\\n    line-height: 52px;\\n    height: 52px;\\n    text-align: center;\\n    font-size: 16px;\\n    width: 100%;\\n    color: #aaa;\\n}\\n\\n@media screen and (max-width: 560px) { \\n.main {max-width:100%;font-size: 16px;} \\n} \\n\\n@keyframes slidetounlock\\n{\\n    0%     {background-position:-200px 0;}\\n    100%   {background-position:200px 0;}\\n}\\n@-webkit-keyframes slidetounlock\\n{\\n    0%     {background-position:-200px 0;}\\n    100%   {background-position:200px 0;}\\n}\\n.button{\\n    position: absolute;\\n    left: 0;\\n    top: 0;\\n    width: 52px;\\n    height: 52px;\\n    background-color: #fff;\\n    transition: left 0s;\\n    -webkit-transition: left 0s;\\n}\\n.button-on{\\n    position: absolute;\\n    left: 0;\\n    top: 0;\\n    width: 52px;\\n    height: 52px;\\n    background-color: #fff;\\n    transition: left 1s;\\n    -webkit-transition: left .5s;\\n}\\n.track{\\n    position: absolute;\\n    left: 0;\\n    top: 0;\\n    height: 100%;\\n    width: 0;\\n    overflow: hidden;\\n    transition: width 0s;\\n    -webkit-transition: width 0s;\\n}\\n.track-on{\\n    position: absolute;\\n    left: 0;\\n    top: 0;\\n    height: 100%;\\n    width: 0;\\n    overflow: hidden;\\n    transition: width 1s;\\n    -webkit-transition: width .5s;\\n}\\n.icon  {\\n    width: 32px;\\n    height: 32px;\\n    position: relative;\\n    top:10px;\\n    left:20px;\\n    font-family: sans-serif;\\n}\\n.icon:before{\\n    content:'>>';\\n    color:#ccc;\\n    line-height:32px;\\n}\\n.spinner {\\n    width: 32px;\\n    height: 32px;\\n        background: url('data:image/png;base64,jwv8YQUAAAAJcEhZcwAADsQAAA7EAZUrDhsAAAQMSURBVFhHzZdLbFRlFMf/82g7LYPQDlDQaZTQR7QlERVoN6Q1UXdqVChrWUDCohtDUqJViQFW+FywAV0ykGhi4sImWuqmDTXaGDChlABhkBlK26lMpzPMy/P/vnsvc+fFtCPiL2nvfN+99/zPPed7nM+RFfAYcRrXx8ayInB36QYuhL7FlcgYQrEriKcWVb/HvQobG9rQ1tiDHc1vYV3906q/Eipy4NLsTwhMvY+ZpevwuFbB5ayFy+GWOw79ALJIZ1NIZe4jkV7E+vrN6G//BJ2+l437pSnrAG8dm3gVwegleGua4BRRh8MULQ7fyYgz0eQc/N5ODG4fLvtOSQfCsav4eHwXGtxPoMblMXqXRzIdx1Lqbwx1/4Lmhi1Gr52igzAcm8bQWA9W1/pWLE74rlds0BZtFqMgAmweHPErcafDZfRWRyabVin5qvdmQToKIsCcM+zVivNDkpmEutJWvWu12H7NuPsAmwMc7Rxw1YSdUDSanMWzjbsQSy2oNm0GoxdF42fjKY3NgcDUB2q0VwPFIonbeLv1I+zrOon3Xvwei6l5dY+2z8p0zsVygHN8ZumammorRYv/hf6Oo+hr2af6YsmIrBZahrbviAa1TCwHuMJxkXnYPC+FKb634zj6/Fr8z9kRnPjtTRlTa1SbtutEYyL8nWoTy4HpyLha4VaCGfb+jmPo9b+r+pjrzyb3oMnTYvsot2hMz48brRwHQjJP9fKqoVFOnUUJIX+XwhJvl7AbX07xLyb70VT3VEFEqXFb9hETy4F4+p781w/TaCwVwaGXfsCBrV9jLh4s6oQWt+ecYf9cvryxiLjGYWhpLAdyScn8fWHD62jxduE5Xy8Gtp3DXOKWzQlTPD/nn/7+jny5v+KxZDngkYVCzKrfbmcdfpWBMhI8pdqdvj4MPB+QSGgntHjxnPvycl5I1tDSWA5sbGhVWyqhAW+ND4HLgzgfPK36uLUObDur0jHPsFeY83yosUlqBxPLgba1PUjLfm5CQ2vrnsQZmxN92L/1FN7YMriMnNthzdDa2G20REfCqeLOxYG71praZpshM9cMt/nFJmbOHx52DW0t3A/jSM+YFC3PqD4rAuzgH4uJXMxIBC4ftsYEqTznD6DtDVItmeLENgv2SBnFuZ+PdmKTcmI0+A3+uDuMLyf3VpTzXGibGrkU1ANHL7yiBlqxHZGPxtNRuWZQL1v2csRZHXFVPLxj2OjR/P8KEj7wYfeoDLyQerFaaGMhEcbQzvNFI/boi1JZdod2jpYsSks6QHirurK8S8ryH8u+U9YBk4sy5VjJsGDhfs4ttdzBhFNtt4z2rmoPJvnMyNFsIixHM9nPQ7GpvKNZuxzNurFdjmbr/+2j2aOkYBb8twD/APBmN4ba5Vu7AAAAAElFTkSuQmCC') no-repeat;\\n    position: relative;\\n    top:10px;\\n    left:10px;\\n    display: none;\\n}\\n\\n@-webkit-keyframes bouncedelay {\\n    0%, 80%, 100% { -webkit-transform: scale(0.0) }\\n    40% { -webkit-transform: scale(1.0) }\\n}\\n@keyframes bouncedelay {\\n    0%, 80%, 100% {\\n        transform: scale(0.0);\\n        -webkit-transform: scale(0.0);\\n    } 40% {\\n          transform: scale(1.0);\\n          -webkit-transform: scale(1.0);\\n      }\\n}\\n.bg-green {\\n    line-height: 52px;\\n    height: 52px;\\n    text-align: center;\\n    font-size: 16px;\\n    background-color: #78c430;\\n}\\nbody{ margin:auto; padding:0;font-family: 'Microsoft Yahei',Hiragino Sans GB, WenQuanYi Micro Hei, sans-serif; background:#f9f9f9}\\n.main{width:560px;margin:auto; margin-top:140px}\\n.panel-footer{ text-align: center}\\n.txts{ text-align:center; margin-top:40px}\\n.bds{ line-height:40px; border-left:#CCC 1px solid; padding-left:20px}\\n.panel{ margin-top:30px}\\n</style>\\n</head>\\n<body>\\n  <div class='main'>\\n<div class='alert alert-success' role='alert'>\\n  <span class='glyphicon glyphicon-exclamation-sign' aria-hidden='true'></span>\\n  <span class='sr-only'>Error:</span>\\n   网站当前访问量较大，请拖动滑块后继续访问\\n</div>\\n<form class='form-inline'>\\n<div class='panel panel-success'>\\n  <div class='panel-body'>\\n  <div class='row'>\\n<div class='stage'>\\n    <div class='slider' id='slider'>\\n      <div class='tips'>向右滑动验证</div>\\n      <div class='track' id='track'>\\n        <div class='bg-green'></div>\\n      </div>\\n      <div class='button' id='btn'>\\n        <div class='icon' id='icon'></div>\\n        <div class='spinner' id='spinner'></div>\\n      </div>\\n    </div>\\n</div>\\n</div>\\n</div>\\n</div>\\n</form>\\n</div>\\n<script type='text/javascript' src='/_guard/encrypt.js'></script>\\n<script type='text/javascript' src='/_guard/slide.js'></script>\\n\\n</script>\\n";
set @captcha_html = "<!doctype html>\\n<html>\\n<head>\\n<html lang='zh-CN'>\\n<meta charset='utf-8'>\\n<meta name='viewport' content='width=device-width, initial-scale=1, user-scalable=no'>\\n<meta name='apple-mobile-web-app-capable' content='yes'>\\n<meta name='apple-mobile-web-app-status-bar-style' content='black'>\\n<meta name='format-detection' content='telephone=no'>\\n<title>CC LOCK</title>\\n<link rel='stylesheet' href='//apps.bdimg.com/libs/bootstrap/3.3.4/css/bootstrap.min.css'>\\n<script type='text/javascript' src='//apps.bdimg.com/libs/jquery/1.7.2/jquery.min.js'></script>\\n<style>\\nbody{ margin:auto; padding:0;font-family: 'Microsoft Yahei',Hiragino Sans GB, WenQuanYi Micro Hei, sans-serif; background:#f9f9f9}\\n.main{width:560px;margin:auto; margin-top:140px}\\n@media screen and (max-width: 560px) { \\n.main {max-width:100%;} \\n} \\n.panel-footer{ text-align: center}\\n.txts{ text-align:center; margin-top:40px}\\n.bds{ line-height:40px; border-left:#CCC 1px solid; padding-left:20px}\\n.panel{ margin-top:30px}\\n</style>\\n<!--[if lt IE 9]>\\n<style>\\n.row\\n{\\n    height: 100%;\\n    display: table-row;\\n}\\n.col-md-3\\n{\\n    display: table-cell;\\n}\\n.col-md-9\\n{\\n    display: table-cell;\\n}\\n</style>\\n<![endif]-->\\n</head>\\n<body>\\n<div class='main'>\\n<div class='alert alert-success' role='alert'>\\n  <span class='glyphicon glyphicon-exclamation-sign' aria-hidden='true'></span>\\n  <span class='sr-only'>Error:</span>\\n  &nbsp;网站当前访问量较大，请输入验证码后继续访问\\n</div>\\n<form class='form-inline'>\\n<div class='panel panel-success'>\\n  <div class='panel-body'>\\n  <div class='row'>\\n  <div class='col-md-3'><div class='txts'>请输入验证码</div></div>\\n  <div class='col-md-9'>\\n  <div class='bds row'>\\n  请输入图片中的验证码，不区分大小写<br>\\n  <input type='text' name='response' class='form-control' id='response'  style='width:40%;display:inline;'>&nbsp;\\n  <span style='width:60px' id='captcha' class='yz'  alt='Captcha image'><img class='captcha-code' src='/_guard/captcha.png'></span>&nbsp;<span><a class='refresh-captcha-code'>换一个</a></span>\\n  <p><span style='color:red' id='notice'></span></p>\\n  </div>\\n  </div>\\n  </div> \\n  </div>\\n   <div class='panel-footer'><input id='access' type='button' class='btn btn-success' value='进入网站' /></div>\\n</div>\\n</form>\\n</div>\\n<script language='javascript' type='text/javascript'>\\n    $('.refresh-captcha-code').click(function() {\\n        $('.captcha-code').attr('src','/_guard/captcha.png?r=' + Math.random());\\n    });\\n    $('#access').click(function(e){\\n      var response = $('#response').val();\\n      document.cookie = 'guardret='+response\\n      window.location.reload();\\n    });\\n</script>\\n</body>\\n</html>";
set @click_html = "<!doctype html>\\n<html>\\n<head>\\n<html lang='zh-CN'>\\n<meta charset='utf-8'>\\n<meta name='viewport' content='width=device-width, initial-scale=1, user-scalable=no'>\\n<meta name='apple-mobile-web-app-capable' content='yes'>\\n<meta name='apple-mobile-web-app-status-bar-style' content='black'>\\n<meta name='format-detection' content='telephone=no'>\\n<title>CC LOCK</title>\\n<style>\\nbody{ margin:auto; padding:0;font-family: 'Microsoft Yahei',Hiragino Sans GB, WenQuanYi Micro Hei, sans-serif; background:#f9f9f9}\\n.main{width:460px;margin:auto; margin-top:140px}\\n@media screen and (max-width: 560px) { \\n.main {max-width:100%;} \\n} \\n.alert {text-align:center}\\n.panel-footer{ text-align: center}\\n.txts{ text-align:center; margin-top:40px}\\n.bds{ line-height:40px; border-left:#CCC 1px solid; padding-left:20px}\\n.panel{ margin-top:30px}\\n.alert-success {\\n    color: #3c763d;\\n    background-color: #dff0d8;\\n    border-color: #d6e9c6;\\n}\\n.alert {\\n    padding: 15px;\\n    margin-bottom: 20px;\\n    border: 1px solid transparent;\\n    border-radius: 4px;\\n}\\n.glyphicon {\\n    position: relative;\\n    top: 1px;\\n    display: inline-block;\\n    font-family: 'Glyphicons Halflings';\\n    font-style: normal;\\n    font-weight: 400;\\n    line-height: 1;\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.btn-success {\\n    color: #fff;\\n    background-color: #5cb85c;\\n    border-color: #4cae4c;\\n}\\n.btn {\\n    display: inline-block;\\n    padding: 6px 12px;\\n    margin-bottom: 0;\\n    font-size: 14px;\\n    font-weight: 400;\\n    line-height: 1.42857143;\\n    text-align: center;\\n    white-space: nowrap;\\n    vertical-align: middle;\\n    -ms-touch-action: manipulation;\\n    touch-action: manipulation;\\n    cursor: pointer;\\n    -webkit-user-select: none;\\n    -moz-user-select: none;\\n    -ms-user-select: none;\\n    user-select: none;\\n    background-image: none;\\n    border: 1px solid transparent;\\n    border-radius: 4px;\\n}\\n</style>\\n<!--[if lt IE 9]>\\n<style>\\n.row\\n{\\n    height: 100%;\\n    display: table-row;\\n}\\n.col-md-3\\n{\\n    display: table-cell;\\n}\\n.col-md-9\\n{\\n    display: table-cell;\\n}\\n</style>\\n<![endif]-->\\n</head>\\n<body>\\n<div class='main'>\\n<div class='alert alert-success' role='alert'>\\n  <span class='glyphicon glyphicon-exclamation-sign' aria-hidden='true'></span>\\n  <span style='font-size: 15px;'> 网站当前访问量较大，请点击按钮继续访问</span><br>\\n    <input style='margin-top: 20px;' id='access' type='button' class='btn btn-success' value='进入网站'>\\n</div>\\n</div>\\n<script type='text/javascript' src='/_guard/encrypt.js'></script>\\n<script type='text/javascript' src='/_guard/click.js'></script>\\n</body>\\n</html>";
set @delay_jump_html = "<!doctype html>\\n<html>\\n<head>\\n<html lang='zh-CN'>\\n<meta charset='utf-8'>\\n<meta name='viewport' content='width=device-width, initial-scale=1, user-scalable=no'>\\n<meta name='apple-mobile-web-app-capable' content='yes'>\\n<meta name='apple-mobile-web-app-status-bar-style' content='black'>\\n<meta name='format-detection' content='telephone=no'>\\n<title>CC LOCK</title>\\n<style>\\nbody{ margin:auto; padding:0;font-family: 'Microsoft Yahei',Hiragino Sans GB, WenQuanYi Micro Hei, sans-serif; background:#f9f9f9}\\n.main{width:460px;margin:auto; margin-top:140px}\\n@media screen and (max-width: 560px) { \\n.main {max-width:100%;} \\n} \\n#second {color:red;}\\n.alert {text-align:center}\\n.panel-footer{ text-align: center}\\n.txts{ text-align:center; margin-top:40px}\\n.bds{ line-height:40px; border-left:#CCC 1px solid; padding-left:20px}\\n.panel{ margin-top:30px}\\n.alert-success {\\n    color: #3c763d;\\n    background-color: #dff0d8;\\n    border-color: #d6e9c6;\\n}\\n.alert {\\n    padding: 15px;\\n    margin-bottom: 20px;\\n    border: 1px solid transparent;\\n    border-radius: 4px;\\n}\\n.glyphicon {\\n    position: relative;\\n    top: 1px;\\n    display: inline-block;\\n    font-family: 'Glyphicons Halflings';\\n    font-style: normal;\\n    font-weight: 400;\\n    line-height: 1;\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n</style>\\n<!--[if lt IE 9]>\\n<style>\\n.row\\n{\\n    height: 100%;\\n    display: table-row;\\n}\\n.col-md-3\\n{\\n    display: table-cell;\\n}\\n.col-md-9\\n{\\n    display: table-cell;\\n}\\n</style>\\n<![endif]-->\\n</head>\\n<body>\\n<div class='main'>\\n<div class='alert alert-success' role='alert'>\\n  <span class='glyphicon glyphicon-exclamation-sign' aria-hidden='true'></span>\\n  <span style='font-size: 15px;'>浏览器安全检查中，系统将在<span id='second'>5</span>秒后返回网站</span>\\n</div>\\n</div>\\n<script type='text/javascript' src='/_guard/encrypt.js'></script>\\n<script type='text/javascript' src='/_guard/delay_jump.js'></script>\\n</body>\\n</html>";
set @rotate_html = "<html>\\n<head>\\n<meta http-equiv='Content-Type' content='text/html; charset=utf-8' />\\n<meta name='viewport' content='width=device-width, initial-scale=1, user-scalable=no'>\\n<title>安全验证</title>\\n</head>\\n<body>\\n<style>\\n@media screen and (max-width: 305px) { \\n.captcha-root {max-width:100%;font-size: 16px;} \\n} \\n\\n</style>\\n\\n<div class='J__captcha__'></div>\\n<script src='/_guard/rotate.js'></script>\\n<script>\\nlet myCaptcha = document.querySelectorAll('.J__captcha__').item(0).captcha({\\n  // 验证成功时显示\\n  timerProgressBar: !0, // 是否启用进度条\\n  timerProgressBarColor: '#07f', // 进度条颜色\\n        title: '安全验证',\\n  desc: '拖动滑块，使图片角度为正'  \\n});\\n\\n</script>\\n</body>\\n\\n</html>\\n\\n\\n";
insert into config values ('openresty-config',concat('{"icmp_drop":"0", "default_page_refuse":"0", "default_page_rule":"1", "cc_enable":true,"custom_white":"","custom_black":"", "key":"__OPENRESTY_KEY__", "rnd_url":{"enable":true,"rnd_url_qps":100, "uptime":120,"in_seconds":60,"max_req":10,"last_seconds":300},"auto_switch":{"enable":false, "qps_50x":10,"qps_total":500,"rule":"2","seconds":300}, "log":{"port":514,"host":"127.0.0.1","log_level":"info","debug_ip":"127.0.0.1"}, "rotate_html":"',@rotate_html,'", "delay_jump_html":"',@delay_jump_html,'", "slider_html":"',@slider_html,'","captcha_html":"',@captcha_html,'","click_html":"',@click_html,'","block_time":3600,"white_time":43200,"built_in_white":["49.232.166.154","119.91.114.81","139.162.76.111","192.46.226.104","1.117.156.236","45.79.81.228","129.28.14.61","139.155.250.52","111.206.198.0/24", "111.206.221.0/24", "116.179.32.0/24", "116.179.37.0/24", "123.125.66.0/24", "123.125.71.0/24", "124.166.232.0/24", "158.247.209.0/24", "180.149.133.0/24", "180.76.15.0/24", "180.76.5.0/24", "220.181.108.0/24", "220.181.32.0/24", "61.135.165.0/24", "61.135.168.0/24", "61.135.169.0/24", "61.135.186.0/24", "65.49.194.0/24", "106.120.173.0/24", "106.120.188.0/24", "106.38.241.0/24", "111.202.100.0/24", "111.202.101.0/24", "111.202.103.0/24", "118.184.177.0/24", "123.125.125.0/24", "123.126.113.0/24", "123.126.68.0/24", "123.183.224.0/24", "134.195.209.0/24", "218.30.103.0/24", "220.181.124.0/24", "220.181.125.0/24", "36.110.147.0/24", "43.231.99.0/24", "49.7.116.0/24", "49.7.117.0/24", "49.7.20.0/24", "49.7.21.0/24", "58.250.125.0/24", "61.135.189.0/24", "106.11.152.0/24", "106.11.153.0/24", "106.11.154.0/24", "106.11.155.0/24", "106.11.156.0/24", "106.11.157.0/24", "106.11.158.0/24", "106.11.159.0/24", "42.120.160.0/24", "42.120.161.0/24", "42.120.234.0/24", "42.120.235.0/24", "42.120.236.0/24", "42.156.136.0/24", "42.156.137.0/24", "42.156.138.0/24", "42.156.139.0/24", "42.156.254.0/24", "42.156.255.0/24", "103.25.156.0/24", "103.255.141.0/24", "104.44.253.0/24", "104.44.91.0/24", "104.44.92.0/24", "104.44.93.0/24", "104.47.224.0/24", "111.221.28.0/24", "111.221.31.0/24", "131.253.24.0/24", "131.253.25.0/24", "131.253.26.0/24", "131.253.27.0/24", "131.253.35.0/24", "131.253.36.0/24", "131.253.38.0/24", "131.253.46.0/24", "131.253.47.0/24", "13.66.139.0/24", "13.66.144.0/24", "157.55.10.0/24", "157.55.103.0/24", "157.55.106.0/24", "157.55.107.0/24", "157.55.12.0/24", "157.55.13.0/24", "157.55.154.0/24", "157.55.2.0/24", "157.55.21.0/24", "157.55.22.0/24", "157.55.23.0/24", "157.55.34.0/24", "157.55.39.0/24", "157.55.50.0/24", "157.55.7.0/24", "157.56.0.0/24", "157.56.1.0/24", "157.56.2.0/24", "157.56.3.0/24", "157.56.71.0/24", "157.56.92.0/24", "157.56.93.0/24", "185.209.30.0/24", "191.232.136.0/24", "199.30.17.0/24", "199.30.18.0/24", "199.30.19.0/24", "199.30.20.0/24", "199.30.21.0/24", "199.30.22.0/24", "199.30.23.0/24", "199.30.24.0/24", "199.30.25.0/24", "199.30.26.0/24", "199.30.27.0/24", "199.30.28.0/24", "199.30.29.0/24", "199.30.30.0/24", "199.30.31.0/24", "202.101.96.0/24", "202.89.224.0/24", "202.89.235.0/24", "202.89.236.0/24", "207.46.102.0/24", "207.46.12.0/24", "207.46.126.0/24", "207.46.13.0/24", "207.46.199.0/24", "207.68.146.0/24", "207.68.155.0/24", "207.68.176.0/24", "207.68.185.0/24", "213.199.160.0/24", "219.136.255.0/24", "23.103.64.0/24", "40.66.1.0/24", "40.66.4.0/24", "40.73.148.0/24", "40.77.160.0/24", "40.77.161.0/24", "40.77.162.0/24", "40.77.163.0/24", "40.77.164.0/24", "40.77.165.0/24", "40.77.166.0/24", "40.77.167.0/24", "40.77.168.0/24", "40.77.169.0/24", "40.77.170.0/24", "40.77.171.0/24", "40.77.172.0/24", "40.77.173.0/24", "40.77.174.0/24", "40.77.175.0/24", "40.77.176.0/24", "40.77.177.0/24", "40.77.178.0/24", "40.77.179.0/24", "40.77.180.0/24", "40.77.181.0/24", "40.77.182.0/24", "40.77.183.0/24", "40.77.184.0/24", "40.77.185.0/24", "40.77.186.0/24", "40.77.187.0/24", "40.77.188.0/24", "40.77.189.0/24", "40.77.190.0/24", "40.77.191.0/24", "40.77.192.0/24", "40.77.193.0/24", "40.77.194.0/24", "40.77.195.0/24", "40.77.208.0/24", "40.77.209.0/24", "40.77.210.0/24", "40.77.211.0/24", "40.77.212.0/24", "40.77.213.0/24", "40.77.214.0/24", "40.77.215.0/24", "40.77.216.0/24", "40.77.217.0/24", "40.77.218.0/24", "40.77.219.0/24", "40.77.220.0/24", "40.77.221.0/24", "40.77.222.0/24", "40.77.223.0/24", "40.77.248.0/24", "40.77.250.0/24", "40.77.251.0/24", "40.77.252.0/24", "40.77.253.0/24", "40.77.254.0/24", "40.77.255.0/24", "40.90.11.0/24", "40.90.144.0/24", "40.90.145.0/24", "40.90.146.0/24", "40.90.147.0/24", "40.90.148.0/24", "40.90.149.0/24", "40.90.150.0/24", "40.90.151.0/24", "40.90.152.0/24", "40.90.153.0/24", "40.90.154.0/24", "40.90.155.0/24", "40.90.156.0/24", "40.90.157.0/24", "40.90.158.0/24", "40.90.159.0/24", "40.90.8.0/24", "42.159.176.0/24", "42.159.48.0/24", "51.4.84.0/24", "51.5.84.0/24", "52.167.144.0/24", "61.131.4.0/24", "62.109.1.0/24", "64.4.22.0/24", "65.52.109.0/24", "65.52.110.0/24", "65.54.164.0/24", "65.54.247.0/24", "65.55.1.0/24", "65.55.107.0/24", "65.55.146.0/24", "65.55.189.0/24", "65.55.208.0/24", "65.55.209.0/24", "65.55.210.0/24", "65.55.211.0/24", "65.55.212.0/24", "65.55.213.0/24", "65.55.214.0/24", "65.55.215.0/24", "65.55.216.0/24", "65.55.217.0/24", "65.55.218.0/24", "65.55.219.0/24", "65.55.230.0/24", "65.55.25.0/24", "65.55.44.0/24", "65.55.54.0/24", "65.55.60.0/24", "104.248.26.0/24", "108.61.163.0/24", "109.228.12.0/24", "109.238.6.0/24", "134.195.209.0/24", "139.180.178.0/24", "142.147.250.0/24", "143.198.137.0/24", "144.76.92.0/24", "162.221.189.0/24", "173.212.206.0/24", "173.212.237.0/24", "173.249.20.0/24", "173.249.22.0/24", "173.249.31.0/24", "173.82.94.0/24", "174.34.149.0/24", "175.45.118.0/24", "178.20.236.0/24", "185.164.4.0/24", "194.48.168.0/24", "194.67.218.0/24", "195.201.22.0/24", "202.222.14.0/24", "203.208.60.0/24", "212.162.12.0/24", "213.136.87.0/24", "213.136.91.0/24", "217.156.87.0/24", "217.20.115.0/24", "23.105.51.0/24", "45.77.69.0/24", "5.189.166.0/24", "64.68.88.0/24", "64.68.90.0/24", "64.68.91.0/24", "64.68.92.0/24", "66.249.64.0/24", "66.249.65.0/24", "66.249.66.0/24", "66.249.68.0/24", "66.249.69.0/24", "66.249.70.0/24", "66.249.71.0/24", "66.249.72.0/24", "66.249.73.0/24", "66.249.74.0/24", "66.249.75.0/24", "66.249.76.0/24", "66.249.77.0/24", "66.249.79.0/24", "78.47.203.0/24", "79.174.79.0/24", "89.46.100.0/24", "91.144.154.0/24", "93.104.213.0/24", "108.177.64.0/24", "108.177.65.0/24", "108.177.66.0/24", "108.177.67.0/24", "108.177.68.0/24", "108.177.69.0/24", "108.177.70.0/24", "108.177.71.0/24", "108.177.72.0/24", "108.177.73.0/24", "108.177.74.0/24", "108.177.75.0/24", "108.177.76.0/24", "108.177.77.0/24", "108.177.78.0/24", "108.177.79.0/24", "203.208.38.0/24", "209.85.238.0/24", "66.249.87.0/24", "66.249.89.0/24", "66.249.90.0/24", "66.249.91.0/24", "66.249.92.0/24", "72.14.199.0/24", "74.125.148.0/24", "74.125.149.0/24", "74.125.150.0/24", "74.125.151.0/24", "74.125.216.0/24", "74.125.217.0/24", "74.125.218.0/24", "110.249.201.0/24", "110.249.202.0/24", "111.225.148.0/24", "111.225.149.0/24", "220.243.135.0/24", "220.243.136.0/24", "220.243.188.0/24", "220.243.189.0/24", "60.8.123.0/24", "60.8.151.0/24", "180.153.234.0/24", "180.153.236.0/24", "180.163.220.0/24", "42.236.101.0/24", "42.236.102.0/24", "42.236.103.0/24", "42.236.10.0/24", "42.236.12.0/24", "42.236.13.0/24", "42.236.14.0/24", "42.236.15.0/24", "42.236.16.0/24", "42.236.17.0/24", "42.236.46.0/24", "42.236.48.0/24", "42.236.49.0/24", "42.236.50.0/24", "42.236.51.0/24", "42.236.52.0/24", "42.236.53.0/24", "42.236.54.0/24", "42.236.55.0/24", "42.236.99.0/24"]}'),'openresty_config','0','global', now(),now(),1,null);

# error page
set @p403 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>请求被禁止访问</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>403</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">当前请求已被禁止访问</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您的请求被网站管理员禁止访问</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>可以联系网站管理员咨询原因</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p502 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>网站请求出错</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>502</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">回源请求被中断</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>CDN节点请求源服务器时，请求被源服务器防火墙中断。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>如果您是网站用户，请稍候重试，或者联系管理员。</p>\\n            <p>如果您是网站管理员，请检查您的源服务器防火墙是否拉黑了CDN的节点，检查CDN网站配置的回源协议和回源端口是否正确。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p504 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>网站请求超时</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>504</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">回源请求超时</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>CDN节点请求源服务器时，等待时间过长。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>如果您是网站用户，请稍候重试，或者联系管理员。</p>\\n            <p>如果您是网站管理员，请检查您的源服务器防火墙是否拉黑了CDN的节点，检查CDN节点与源服务器之间的链路是否畅通。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p512 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>套餐到期</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>512</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">套餐到期</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您网站使用的套餐已到期。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请登录CDN后台续费套餐恢复。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p513 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>流量超限</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>513</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">流量超限</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您网站使用的套餐流量已用完。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请登录CDN后台升级流量恢复。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p514 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>网站被锁定</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>514</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">网站被锁定</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您的网站已被管理员锁定。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请联系管理员。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @p515 = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>套餐连接数超限</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>515</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">套餐连接数超限</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您的套餐连接数超限。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请联系管理员。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @host_not_found = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>域名未配置</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>530</span>\\n         </h1>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">您的IP: {client_ip} &bull;</span>\\n         <span class=\\"inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed\\">节点IP: {node_ip}</span>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">域名未配置</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>您的域名指向了CDN节点，但配置未生效或者未在CDN配置此域名。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请到CDN后台添加此域名，或联系管理员处理。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';

set @access_ip_not_allow = '\\n<!DOCTYPE html>\\n<!--[if lt IE 7]> <html class=\\"no-js ie6 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 7]>    <html class=\\"no-js ie7 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if IE 8]>    <html class=\\"no-js ie8 oldie\\" lang=\\"en-US\\"> <![endif]-->\\n<!--[if gt IE 8]><!--> <html class=\\"no-js\\" lang=\\"en-US\\"> <!--<![endif]-->\\n<head>\\n<title>请使用域名访问</title>\\n<meta charset=\\"UTF-8\\" />\\n<meta http-equiv=\\"Content-Type\\" content=\\"text/html; charset=UTF-8\\" />\\n<meta http-equiv=\\"X-UA-Compatible\\" content=\\"IE=Edge,chrome=1\\" />\\n<meta name=\\"robots\\" content=\\"noindex, nofollow\\" />\\n<meta name=\\"viewport\\" content=\\"width=device-width,initial-scale=1\\" />\\n<style>\\n*, body, html {\\n    margin: 0;\\n    padding: 0;\\n}\\n\\nbody, html {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;\\n    font-size: 16px;\\n}\\n* {\\n    box-sizing: border-box;\\n}\\nhtml[Attributes Style] {\\n    -webkit-locale: \\"en-US\\";\\n}\\n.p-0 {\\n    padding: 0;\\n}\\n\\n\\n.w-240 {\\n    width: 60rem;\\n}\\n\\n.antialiased {\\n    -webkit-font-smoothing: antialiased;\\n    -moz-osx-font-smoothing: grayscale;\\n}\\n.pt-10 {\\n    padding-top: 2.5rem;\\n}\\n.mb-15 {\\n    margin-bottom: 3.75rem;\\n}\\n.mx-auto {\\n    margin-left: auto;\\n    margin-right: auto;\\n}\\n\\n.text-black-dark {\\n    --text-opacity: 1;\\n    color: #404040;\\n    color: rgba(64,64,64,var(--text-opacity));\\n}\\n\\n.mr-2 {\\n    margin-right: .5rem;\\n}\\n.leading-tight {\\n    line-height: 1.25;\\n}\\n.text-60 {\\n    font-size: 60px;\\n}\\n.font-light {\\n    font-weight: 300;\\n}\\n.inline-block {\\n    display: inline-block;\\n}\\n\\n.text-15 {\\n    font-size: 15px;\\n}\\n.font-mono {\\n    font-family: monaco,courier,monospace;\\n}\\n.text-gray-600 {\\n    --text-opacity: 1;\\n    color: #999;\\n    color: rgba(153,153,153,var(--text-opacity));\\n}\\n.leading-1\\\\.3 {\\n    line-height: 1.3;\\n}\\n.text-3xl {\\n    font-size: 1.875rem;\\n}\\n\\n.mb-8 {\\n    margin-bottom: 2rem;\\n}\\n\\n.w-1\\\\/2 {\\n    width: 50%;\\n}\\n\\n.mt-6 {\\n    margin-top: 1.5rem;\\n}\\n\\n.mb-4 {\\n    margin-bottom: 1rem;\\n}\\n\\n\\n.font-normal {\\n    font-weight: 400;\\n}\\n\\n#what-happened-section p {\\n    font-size: 15px;\\n    line-height: 1.5;\\n}\\n\\n</style>\\n\\n</head>\\n<body>\\n  <div id=\\"cf-wrapper\\">\\n    <div id=\\"cf-error-details\\" class=\\"p-0\\">\\n      <header class=\\"mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased\\">\\n         <h1 class=\\"inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight\\">\\n           <span data-translate=\\"error\\">Error</span>\\n           <span>1003</span>\\n         </h1>\\n        <h2 class=\\"text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light\\">请使用域名访问</h2>\\n      </header>\\n\\n      <section class=\\"w-240 lg:w-full mx-auto mb-8 lg:px-8\\">\\n          <div id=\\"what-happened-section\\" class=\\"w-1/2 md:w-full\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_happened\\">什么问题?</h2>\\n            <p>当前是直接访问的节点IP。</p>\\n            \\n          </div>\\n\\n          \\n          <div id=\\"resolution-copy-section\\" class=\\"w-1/2 mt-6 text-15 leading-normal\\">\\n            <h2 class=\\"text-3xl leading-tight font-normal mb-4 text-black-dark antialiased\\" data-translate=\\"what_can_i_do\\">如何解决?</h2>\\n            <p>请使用域名访问。</p>\\n          </div>\\n          \\n      </section>\\n\\n      <div class=\\"cf-error-footer cf-wrapper w-240 lg:w-full py-10 sm:py-4 sm:px-8 mx-auto text-center sm:text-left border-solid border-0 border-t border-gray-300\\">\\n\\n</div><!-- /.error-footer -->\\n\\n\\n    </div><!-- /#cf-error-details -->\\n  </div><!-- /#cf-wrapper -->\\n\\n\\n</body>\\n</html>\\n\\n';



insert into config values ('error-page',concat('{"p403":"',@p403,'","p502":"',@p502,'","p504":"',@p504,'","p512":"',@p512,'","p513":"',@p513,'","p514":"',@p514,'","p515":"',@p515,'","host_not_found":"',@host_not_found,'","access_ip_not_allow":"',@access_ip_not_allow,'"}'),'error_page','0','global', now(),now(),1,null);    

# user package config
insert into config values ('user-package-config','','user_package_config','0','global', now(),now(),1,null);


# site default config
insert into config values ('http_listen-port','80','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('https_listen-port','443','site_default_config','0','global', now(),now(),1,null);     
insert into config values ('https_listen-hsts','0','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('https_listen-http2','0','site_default_config','0','global', now(),now(),1,null); 

insert into config values ('https_listen-force_ssl_enable','0','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('https_listen-ssl_protocols','TLSv1 TLSv1.1 TLSv1.2 TLSv1.3','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('https_listen-ssl_ciphers','ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES256-SHA256:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA','site_default_config','0','global', now(),now(),1,null);
insert into config values ('https_listen-ssl_prefer_server_ciphers','on','site_default_config','0','global', now(),now(),1,null);

insert into config values ('balance_way','rr','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('cc_default_rule','10002','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('gzip_enable','1','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('gzip_types','text/plain text/css text/xml text/javascript application/javascript application/x-javascript application/json','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('websocket_enable','0','site_default_config','0','global', now(),now(),1,null);    
insert into config values ('backend_protocol','http','site_default_config','0','global', now(),now(),1,null);    
insert into config values ('backend_http_port','80','site_default_config','0','global', now(),now(),1,null);
insert into config values ('backend_https_port','443','site_default_config','0','global', now(),now(),1,null);
insert into config values ('proxy_timeout','60','site_default_config','0','global', now(),now(),1,null);
insert into config values ('range','0','site_default_config','0','global', now(),now(),1,null);
insert into config values ('proxy_cache','[]','site_default_config','0','global', now(),now(),1,null);
insert into config values ('proxy_http_version','1.0','site_default_config','0','global', now(),now(),1,null);
insert into config values ('post_size_limit',32,'site_default_config',0,'global', now(),now(),1,null);
insert into config values ('proxy_ssl_protocols','TLSv1 TLSv1.1 TLSv1.2','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('ups_keepalive','0','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('ups_keepalive_conn','5','site_default_config','0','global', now(),now(),1,null); 
insert into config values ('ups_keepalive_timeout','50','site_default_config','0','global', now(),now(),1,null); 

# stream default config
insert into config values ('listen_protocol','tcp','stream_default_config','0','global', now(),now(),1,null); 
insert into config values ('balance_way','rr','stream_default_config','0','global', now(),now(),1,null); 
insert into config values ('proxy_protocol','0','stream_default_config','0','global', now(),now(),1,null); 

# cert default config
insert into config values ('cert_default_type','lets','cert_default_config','0','global', now(),now(),1,null); 


# system config
insert into config values ('keep-job-days','30','system','0','global', now(),now(),1,null); 
insert into config values ('keep-login-log-days','30','system','0','global', now(),now(),1,null); 
insert into config values ('keep-op-log-days','30','system','0','global', now(),now(),1,null); 
insert into config values ('keep-task-log-days','7','system','0','global', now(),now(),1,null); 
insert into config values ('keep-access-log-days','7','system','0','global', now(),now(),1,null); 
insert into config values ('keep-node-log-days','7','system','0','global', now(),now(),1,null); 
insert into config values ('keep-traffic-history-days','90','system','0','global', now(),now(),1,null); 

insert into config values ('backup_rate','2h','system','0','global', now(),now(),1,null); 
insert into config values ('backup_keep_days','7','system','0','global', now(),now(),1,null); 
insert into config values ('backup_dir','/data/backup/cdn/','system','0','global', now(),now(),1,null); 

insert into config values ('max_site_stream_sync_one_time','5000','system','0','global', now(),now(),1,null); 
insert into config values ('login_session_valid_time','3600','system','0','global', now(),now(),1,null); 
insert into config values ('dns_config','{}','system','0','global', now(),now(),1,null); 
insert into config values ('admin_domain','','system','0','global', now(),now(),1,null); 
insert into config values ('user_domain','','system','0','global', now(),now(),1,null); 
insert into config values ('allow_register','0','system','0','global', now(),now(),1,null); 
insert into config values ('smtp','{}','system','0','global', now(),now(),1,null); 
insert into config values ('register_success_templ','{"title":"cdn用户注册成功","data":"<p>尊敬的{{username}}:</p>\\n<p>您好！感谢您注册cdn。</p>\\n<p>您的注册邮箱{{email}}，用户名{{username}}，密码{{password}}，请妥善保管！</p>"}','system','0','global', now(),now(),1,null); 
insert into config values ('forget_password_templ','{"title":"cdn用户密码重置","data":"<p>尊敬的{{username}}:</p>\\n<p>您好！\\b您已重置密码，新的密码为{{password}}</p>\\n"}','system','0','global', now(),now(),1,null); 
insert into config values ('email_captcha_templ','{"title":"cdn用户验证码","data":"<p>尊敬的用户:</p>\\n<p>您好！您的验证码为{{captcha}}，请在注册页面输入此验证码继续注册</p>\\n"}','system','0','global', now(),now(),1,null); 
insert into config values ('register_require','{"username":{"need":1},"email":{"need":1,"verify":1},"phone":{"need":0,"verify":0},"qq":{"need":0}}','system','0','global', now(),now(),1,null); 
insert into config values ('user_agreement','{"title":"用户协议、法律声明和隐私政策","data":"这里填写协议内容"}','system','0','global', now(),now(),1,null); 
insert into config values ('sms_config','{}','system','0','global', now(),now(),1,null); 
insert into config values ('phone_captcha_templ','【cdn】您的验证码为{{captcha}}，在5分钟内有效。','system','0','global', now(),now(),1,null); 
insert into config values ('alipay_id_auth','{}','system','0','global', now(),now(),1,null); 
insert into config values ('dns_rs_protect','','system','0','global', now(),now(),1,null); 

insert into config values ('node_health_check','1','system','0','global', now(),now(),1,null); 
insert into config values ('node_max_failed','2','system','0','global', now(),now(),1,null); 
insert into config values ('record_repair','0','system','0','global', now(),now(),1,null); 
insert into config values ('record_sync','1','system','0','global', now(),now(),1,null);

insert into config values ('package_expire_close_site','1','system','0','global', now(),now(),1,null); 
insert into config values ('traffic_excceed_close_site','1','system','0','global', now(),now(),1,null); 
insert into config values ('package_allow_upgrade','1','system','0','global', now(),now(),1,null); 
insert into config values ('package_allow_downgrade','1','system','0','global', now(),now(),1,null); 

insert into config values ('system_info','{"user_console_title":"cdn用户控制台","admin_console_title":"cdn管理员控制台","sys_name":"cdn 4.0"}','system','0','global', now(),now(),1,null);
insert into config values ('auth_code','','system','0','global', now(),now(),1,null);
insert into config values ('maintain','{"enable":0,"msg":"维护中,请稍候重试!"}','system','0','global', now(),now(),1,null);
insert into config values ('master_client_ip_header','X-Real-IP','system','0','global', now(),now(),1,null);
insert into config values ('recharge','{"wxpay":{"state":false,"subtype":"native", "app_id":"","mch_id":"","mch_key":"","notify_url":"xxx"},"alipay":{"state":false,"subtype":"pc", "app_id":"","app_key":"","public_key":"","notify_url":""},"transfer":{"state":false,"data":""},"default-pay":""}','system','0','global', now(),now(),1,null);
insert into config values ('auto_upgrade_agent','1','system','0','global', now(),now(),1,null);
# 从线路组移除节点时，延后删除节点上配置文件
insert into config values ('delete_config_delayed','','system','0','global', now(),now(),1,null);
insert into config values ('record-repair-enable','1','system','0','global',now(),now(),1,null);

insert into config values ("notification-period","8-22","system","0", "global",now(),now(),1,null);
insert into config values ("traffic-exceed-notify",' {"state":true,"notify-times":"2","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的套餐流量（ID: {{package_id}}，名称:{{package_name}}）已用尽，系统已暂停您的服务。您可随时升级恢复服务。","email-templ":"cdn套餐流量用尽提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐流量（ID: {{package_id}}，名称:{{package_name}}）已用尽，系统已暂停您的服务。您可随时升级恢复服务。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("traffic-exceeding-notify",'{"state":true,"notify-times":"2","less":"10","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的套餐流量（ID: {{package_id}}，名称:{{package_name}}）仅剩余{{traffic_remain}}GB，为避免影响您的服务，请及时升级。","email-templ":"cdn套餐流量不足提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐流量（ID: {{package_id}}，名称:{{package_name}}）仅剩余{{traffic_remain}}GB，为避免影响您的服务，请及时升级。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("package-expire-notify",'{"state":true,"notify-times":"2","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的套餐（ID: {{package_id}}，名称:{{package_name}}）已过期，系统已暂停您的服务。您可随时续费恢复服务。","email-templ":"cdn套餐过期提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐（ID: {{package_id}}，名称:{{package_name}}）已过期，系统已暂停您的服务。您可随时续费恢复服务。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("package-expiring-notify",'{"state":true,"notify-times":"2","less":"7","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的套餐（ID: {{package_id}}，名称:{{package_name}}）即期过期，仅剩余{{remain_days}}天，为避免影响您的服务，请及时续费。","email-templ":"cdn套餐即将过期提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐（ID: {{package_id}}，名称:{{package_name}}）即期过期，仅剩余{{remain_days}}天，为避免影响您的服务，请及时续费。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("cert-expire-notify",'{"state":true,"notify-times":"2","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的证书（ID: {{cert_id}}，名称:{{cert_name}}，域名:{{domain}} ）已过期，为避免影响业务，请尽快处理。","email-templ":"cdn证书过期提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的证书（ID: {{cert_id}}，名称:{{cert_name}}，域名:{{domain}} ）已过期，为避免影响业务，请尽快处理。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("cert-expiring-notify",'{"state":true,"notify-times":"3","less":"7","interval":"24","phone-templ":"【cdn】尊敬的{{username}}，您的证书（ID: {{cert_id}}，名称:{{cert_name}}，域名:{{domain}} ）即期过期，仅剩余{{remain_days}}天，为避免影响您的业务，请及时处理。","email-templ":"cdn证书即将过期提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的证书（ID: {{cert_id}}，名称:{{cert_name}}，域名:{{domain}} ）即期过期，仅剩余{{remain_days}}天，为避免影响您的业务，请及时处理。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("notify-method",'{"email":true,"phone":true}',"system","0", "global",now(),now(),1,null);

insert into config values ('sync-site-config-scope','line_group','system','0','global',now(),now(),1,null);
insert into config values ('node_monitor_config','{"notification_period":"8-22", "notify_method":"email sms","notify_msg_type": "节点IP解析 带宽监控 备用IP 备用默认解析 备用线路组","email":"","phone":"","bw_exceed_times":2, "monitor_api":"", "interval":30,"failed_times":3,"failed_rate":"50"}','system','0','global', now(),now(),1,null);
INSERT INTO `config` VALUES (19, 'monitor_config', '{"interval": 60, "enable_email": false, "email": ""}', '2023-10-27 10:00:00', '2023-10-27 10:00:00',1,null);

DROP TABLE IF EXISTS `cname_domains`;
CREATE TABLE `cname_domains` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) NOT NULL,
  `note` varchar(255) DEFAULT '',
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
insert into config values ('https_cert','-----BEGIN CERTIFICATE-----\nMIIFljCCBH6gAwIBAgIQBL/rK+7F4CpsSmlMHSvE6TANBgkqhkiG9w0BAQsFADBy\nMQswCQYDVQQGEwJDTjElMCMGA1UEChMcVHJ1c3RBc2lhIFRlY2hub2xvZ2llcywg\nSW5jLjEdMBsGA1UECxMURG9tYWluIFZhbGlkYXRlZCBTU0wxHTAbBgNVBAMTFFRy\ndXN0QXNpYSBUTFMgUlNBIENBMB4XDTE5MDgyOTAwMDAwMFoXDTIwMDgyODEyMDAw\nMFowFzEVMBMGA1UEAxMMdi54bW15YnV5LmNuMIIBIjANBgkqhkiG9w0BAQEFAAOC\nAQ8AMIIBCgKCAQEAvklonZm10SOgrFkH8ftzzLmcqRts+GwZthSpqC6iVuKrbJ8P\nwUpuW7NeK1bqzBN6Dfq+M2wvqwnjreUPD8+yrh1SM942wAEoMh2V4ozTZ3j1a99E\nzVxF1XB5Lj0mz49/0Xx0cUBnP9gCS3QZFixvxDLYcOKar43FC3nRxzA9kkyqB1t+\ndTCjnag7txFV38ta0rCGFMZBP4k8Uv36Lbjmy6vYSqqyV7nbwba9YhdfWQRdHU2k\nNxl3WB23V9jzH8vXvT8ZdLJhL78Xa1NE6riD7dMOWQ5PAafUBJVHS5QZpDwQ57s9\nV5izozmOkxtore8oh00JmDZRSIrVWhxUPc3gxwIDAQABo4ICgTCCAn0wHwYDVR0j\nBBgwFoAUf9OZ86BHDjEAVlYijrfMnt3KAYowHQYDVR0OBBYEFNC7hqpkdbYsSxAH\n8gwqcDRX87KpMBcGA1UdEQQQMA6CDHYueG1teWJ1eS5jbjAOBgNVHQ8BAf8EBAMC\nBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMEwGA1UdIARFMEMwNwYJ\nYIZIAYb9bAECMCowKAYIKwYBBQUHAgEWHGh0dHBzOi8vd3d3LmRpZ2ljZXJ0LmNv\nbS9DUFMwCAYGZ4EMAQIBMIGSBggrBgEFBQcBAQSBhTCBgjA0BggrBgEFBQcwAYYo\naHR0cDovL3N0YXR1c2UuZGlnaXRhbGNlcnR2YWxpZGF0aW9uLmNvbTBKBggrBgEF\nBQcwAoY+aHR0cDovL2NhY2VydHMuZGlnaXRhbGNlcnR2YWxpZGF0aW9uLmNvbS9U\ncnVzdEFzaWFUTFNSU0FDQS5jcnQwCQYDVR0TBAIwADCCAQMGCisGAQQB1nkCBAIE\ngfQEgfEA7wB1AKS5CZC0GFgUh7sTosxncAo8NZgE+RvfuON3zQ7IDdwQAAABbN1d\n8dIAAAQDAEYwRAIgKjX5TCYDTKSldG2mUsiG4WnxImZOsSaVkxU1+CLPBOcCIAQi\nusi7GvFs4xrtrhvOwTGFxGDXY+6S0SUz8zJmrKzcAHYAXqdz+d9WwOe1Nkh90Eng\nMnqRmgyEoRIShBh1loFxRVgAAAFs3V3xYgAABAMARzBFAiBtc5xBBPtKTdOFKva1\nJRaE8J5NGd92sSRPi/wxmfUeBAIhAOiLc4fRh9GW1SCc+JCkdqZ5siLUy3n6e87u\npMDtKJpbMA0GCSqGSIb3DQEBCwUAA4IBAQBz/ONP/OqmV4FZe3ealUfOwYk0Y2lr\noB2IrO5pLl+hUBIaoTxcxa8prfWL3658b+l+fe5q/oeA9y5mFaH6SBHFCDMnpJqq\n1dLQ3HtcdQI65uKXLmNDUpA/t1VGRexTqpAp2tpPyGcYf2MDMztpdwh/ap67pgfd\najY+++Pfl3uJW8SODFUiR9mnX20o6X1gPXAYI6Oo8NauM5/Uw/W5cDe8lqEa0T7J\nGxh9ytzM1LZbWTpdsnDcpV6yMRuJ7Z2Kkz74m5ljoDSU3Wj5xbLG8HSG3DwjEEd+\nj5Seof4jYe3eHrOqm/y0GumtFQ26RLcFq4092SipfO7BVFX0qfbmEzor\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIErjCCA5agAwIBAgIQBYAmfwbylVM0jhwYWl7uLjANBgkqhkiG9w0BAQsFADBh\nMQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3\nd3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBD\nQTAeFw0xNzEyMDgxMjI4MjZaFw0yNzEyMDgxMjI4MjZaMHIxCzAJBgNVBAYTAkNO\nMSUwIwYDVQQKExxUcnVzdEFzaWEgVGVjaG5vbG9naWVzLCBJbmMuMR0wGwYDVQQL\nExREb21haW4gVmFsaWRhdGVkIFNTTDEdMBsGA1UEAxMUVHJ1c3RBc2lhIFRMUyBS\nU0EgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCgWa9X+ph+wAm8\nYh1Fk1MjKbQ5QwBOOKVaZR/OfCh+F6f93u7vZHGcUU/lvVGgUQnbzJhR1UV2epJa\ne+m7cxnXIKdD0/VS9btAgwJszGFvwoqXeaCqFoP71wPmXjjUwLT70+qvX4hdyYfO\nJcjeTz5QKtg8zQwxaK9x4JT9CoOmoVdVhEBAiD3DwR5fFgOHDwwGxdJWVBvktnoA\nzjdTLXDdbSVC5jZ0u8oq9BiTDv7jAlsB5F8aZgvSZDOQeFrwaOTbKWSEInEhnchK\nZTD1dz6aBlk1xGEI5PZWAnVAba/ofH33ktymaTDsE6xRDnW97pDkimCRak6CEbfe\n3dXw6OV5AgMBAAGjggFPMIIBSzAdBgNVHQ4EFgQUf9OZ86BHDjEAVlYijrfMnt3K\nAYowHwYDVR0jBBgwFoAUA95QNVbRTLtm8KPiGxvDl7I90VUwDgYDVR0PAQH/BAQD\nAgGGMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjASBgNVHRMBAf8ECDAG\nAQH/AgEAMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3Au\nZGlnaWNlcnQuY29tMEIGA1UdHwQ7MDkwN6A1oDOGMWh0dHA6Ly9jcmwzLmRpZ2lj\nZXJ0LmNvbS9EaWdpQ2VydEdsb2JhbFJvb3RDQS5jcmwwTAYDVR0gBEUwQzA3Bglg\nhkgBhv1sAQIwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cuZGlnaWNlcnQuY29t\nL0NQUzAIBgZngQwBAgEwDQYJKoZIhvcNAQELBQADggEBAK3dVOj5dlv4MzK2i233\nlDYvyJ3slFY2X2HKTYGte8nbK6i5/fsDImMYihAkp6VaNY/en8WZ5qcrQPVLuJrJ\nDSXT04NnMeZOQDUoj/NHAmdfCBB/h1bZ5OGK6Sf1h5Yx/5wR4f3TUoPgGlnU7EuP\nISLNdMRiDrXntcImDAiRvkh5GJuH4YCVE6XEntqaNIgGkRwxKSgnU3Id3iuFbW9F\nUQ9Qqtb1GX91AJ7i4153TikGgYCdwYkBURD8gSVe8OAco6IfZOYt/TEwii1Ivi1C\nqnuUlWpsF1LdQNIdfbW3TSe0BhQa7ifbVIfvPWHYOu3rkg1ZeMo6XRU9B4n5VyJY\nRmE=\n-----END CERTIFICATE-----\n','system','0','global', now(),now(),1,null);
insert into config values ('https_key','-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC+SWidmbXRI6Cs\nWQfx+3PMuZypG2z4bBm2FKmoLqJW4qtsnw/BSm5bs14rVurME3oN+r4zbC+rCeOt\n5Q8Pz7KuHVIz3jbAASgyHZXijNNnePVr30TNXEXVcHkuPSbPj3/RfHRxQGc/2AJL\ndBkWLG/EMthw4pqvjcULedHHMD2STKoHW351MKOdqDu3EVXfy1rSsIYUxkE/iTxS\n/fotuObLq9hKqrJXudvBtr1iF19ZBF0dTaQ3GXdYHbdX2PMfy9e9Pxl0smEvvxdr\nU0TquIPt0w5ZDk8Bp9QElUdLlBmkPBDnuz1XmLOjOY6TG2it7yiHTQmYNlFIitVa\nHFQ9zeDHAgMBAAECggEAHQwPVwYondvJQks27hvwgd8QSIo3Y2FRDXKbBrbTsN1a\nG13ZOwGAgNQL0HmB+b68DNfqt+Z7DP9t9392Afe2cri8HHnT3rxueolPS4LWyelK\nLCTWwl3PCoq9nNmDNqpU8dFZ5GiB5QTgLiIeADyun62+oitylHuDiZcXdvITZrpf\nbKGe1d2O0tbz1razZTzDR4E50ZS7+ck/TUS+Ciyq76WFaFY+Mhd3TbQS21tHVHId\nBr3F7eoXrZvXZ9J/C/2pmCbiK6rPzTlR1sLDFN2wH2ARUICfVCSctampGRxSKZQW\nQrGMmGInaueqmQqHEBl02WHb6WPhXnIeicCSB1PGYQKBgQDnM6YIqsfJC6qMECL3\nglj1dseWRY3Y1HVQ49gkwk7MFqnx4ZQKdMI3T88+MnnoUK6YsJA9imX1nXaTCUsy\n4umRd+i608RyXLrtwuezU4ejKdAK47Y/SFGaF18pvsNiQWWb2FMyOSWqYFj8e1Mx\nicYnlZiGNzoiM4V/RsFU7tf/YQKBgQDSslBH3d99WddEiP2XawSYjnLfP+OLLTW6\nvdRhEj48st8utVHOAqoFEOGVikHxtwN5AhfHOy6cRMVWPYFUpdVMa9qqsz5J2mmp\nRi0GHNU3n297Wv0UHlkVUGP00VQcUTi7pCUh+ejxs1K6mYrbp/dRrJMQjrBPJU8K\n3zBWQmWZJwKBgF9dFmcMyktK3JXZMhMVWMwmqjx5hACj4Z/z2vuOiiH0VzTF7uJB\nNrrJ2Jm3CEGixeGFMnmv1E5zHK2Zb8MVhXHTG9Oz9ZuWVCQt+JQnKBNM89sKAeoo\nUkBU05PMc5rbjqWxnN9iYv7brti1paMRSQKa2cbCkN/6kF3nOWdm/QEBAoGAQNau\n7e7Rf/nNzUF7CMXePDRaFWnL1GCtUDJq0RSUIonJNM6HxiX7vGNdiG9rq77uSqbi\nOmV0CpL/R3LWAf6mjUYDnNRcLs4QBg+ae28UDnH6FLQDfdV5BJ4gpI5mm/BCzTvO\nUY5eqULOCq6FlOMzsOayuz2t9C0/DdFxRppYObECgYEAofQzjUVlXo9oI00L2d9l\nSl8MWSv/vMV+wlB4/eqw3zOhhCMWv78e/diREHJTWPcbitILE245BBrRHyESUwbu\nsVK/EcJODLumVfH5b8hGbobHAjBTaZ18+w0+Ei58Px3B2BBZCfiwEJBNk6b5KUan\n7+zsTxUmCSpgtqOlvMIwSoM=\n-----END PRIVATE KEY-----\n','system','0','global', now(),now(),1,null);

insert into config values ("cc-switch-notify",'{"state":true,"phone-templ":"【cdn】尊敬的{{username}}，您的域名:{{domain}}）当前QPS为{{curr_qps}}，已超过设置的{{qps_limit}}，疑似被攻击，现系统已自动切换到规则组{{rule_name}}来防御。","email-templ":"网站CC规则组自动切换提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的域名:{{domain}}）当前QPS为{{curr_qps}}，已超过设置的{{qps_limit}}，疑似被攻击，现系统已自动切换到规则组{{rule_name}}来防御。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("bandwidth-exceed-notify",'{"state":true,"phone-templ":"【cdn】尊敬的{{username}}，您的套餐（ID: {{package_id}}，名称:{{package_name}}）当前带宽为{{curr_bandwidth}}，已超过限制的{{bandwidth_limit}}，现系统已开启限速。","email-templ":"cdn套餐带宽超限提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐（ID: {{package_id}}，名称:{{package_name}}）当前带宽为{{curr_bandwidth}}，已超过限制的{{bandwidth_limit}}，现系统已开启限速。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ("conn-exceed-notify",'{"state":true,"phone-templ":"【cdn】尊敬的{{username}}，您的套餐（ID: {{package_id}}，名称:{{package_name}}）当前连接数为{{curr_conn}}，已超过限制的{{conn_limit}}，现系统已开启限速。","email-templ":"cdn套餐连接数超限提醒！\\n<p>尊敬的{{username}}:</p>\\n<p>您的套餐（ID: {{package_id}}，名称:{{package_name}}）当前连接数为{{curr_conn}}，已超过限制的{{conn_limit}}，现系统已开启限速。</p>"}',"system","0", "global",now(),now(),1,null);
insert into config values ('allow-enable-email-captcha-login','1','system','0','global',now(),now(),1,null);
insert into config values ('allow-enable-sms-captcha-login','0','system','0','global',now(),now(),1,null);


# internal cc rule
insert into cc_match values (3, null, '匹配所有资源', '内置匹配器', '{}',now(),now(),1,1,null,1);
insert into cc_match values (4, null, '匹配静态资源', '内置匹配器', '{"req_uri":{"operator":"contain","value":".js\\n.css\\n.png\\n.jpg\\n.jpeg\\n.gif" }}',now(),now(),1,1,null,1);
insert into cc_match values (5, null, '匹配内置资源', '内置匹配器', '{"uri":{"operator":"contain","value":"/_guard/click.js\\n/_guard/slide.js\\n/_guard/captcha.png\\n/_guard/verify-captcha\\n/_guard/encrypt.js\\nfavicon.ico"}}',now(),now(),1,1,null,1);
insert into cc_match values (10020, null, null, null, null,null,null,null,null,null,null);

insert into cc_filter VALUES (1,NULL,'302跳转60-5','内置过滤器','302_challenge',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (2,NULL,'滑动过滤60-5','内置过滤器','slide_filter',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (3,NULL,'验证码60-5','内置过滤器','captcha_filter',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (4,NULL,'内置请求保护60-20-15','内置过滤器','req_rate',60,20,15,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (5,NULL,'请求速率5-20-10','内置过滤器','req_rate',5,20,10,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (6,NULL,'请求速率5-30-20','内置过滤器','req_rate',5,30,20,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (7,NULL,'请求速率5-100-10','内置过滤器','req_rate',5,100,10,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (8,NULL,'浏览器识别60-5','内置过滤器','browser_verify_auto',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (9,NULL,'请求速率5-300-50','内置过滤器','req_rate',5,300,50,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (10,NULL,'请求速率5-200-50','内置过滤器','req_rate',5,200,50,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (11,NULL,'临时白名单专用1','内置过滤器','req_rate',5,300,50,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (12,NULL,'临时白名单专用2','内置过滤器','req_rate',5,200,50,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (10000,NULL,'点击过滤60-5','内置过滤器','click_filter',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (10001,NULL,'5秒盾60-5','内置过滤器','delay_jump_filter',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (10002,NULL,'旋转图片60-5','内置过滤器','rotate_filter',60,5,0,'{}',now(),now(),1,1,NULL,1);
insert into cc_filter VALUES (10020,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);

insert into cc_rule VALUES (1,3,NULL,'JS验证','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "8", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "浏览器识别60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (2,6,NULL,'滑块验证','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "2", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "滑动过滤60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (3,100,NULL,'临时白名单','内置规则','[{"matcher": "3", "matcher_name": "匹配所有资源", "state": true, "filter1": "11", "filter2_name": "", "filter2": "", "action": "ipset", "filter1_name": "临时白名单专用1"}]',now(),now(),1,1,0,NULL,1);
insert into cc_rule VALUES (4,7,NULL,'验证码','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "3", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "验证码60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (6,2,NULL,'宽松','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "9", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "请求速率5-300-50"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (10000,5,NULL,'点击验证','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "10000", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "点击过滤60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (10001,4,NULL,'5秒盾','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "10001", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "5秒盾60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (10002,1,NULL,'关闭','内置规则','[]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (10003,8,NULL,'旋转图片','内置规则','[{"matcher": "3", "action": "ipset", "state": true, "filter1": "10002", "filter2_name": "", "filter2": "", "matcher_name": "匹配所有资源", "filter1_name": "旋转图片60-5"}]',now(),now(),1,1,1,NULL,1);
insert into cc_rule VALUES (10020,100,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);

insert into package_group values (1,'默认','',100,now(),now());
insert into region values (1,'默认','',now(),now());

# lock
insert into `tlock` values ("user");
insert into `tlock` values ("dns");
insert into `tlock` values ("region");
insert into `tlock` values ("node");
insert into `tlock` values ("node_line");
insert into `tlock` values ("line");
insert into `tlock` values ("node_group");
insert into `tlock` values ("dnsapi");
insert into `tlock` values ("cert");
insert into `tlock` values ("acl");
insert into `tlock` values ("cc_rule");
insert into `tlock` values ("cc_match");
insert into `tlock` values ("cc_filter");
insert into `tlock` values ("package");
insert into `tlock` values ("package_up");
insert into `tlock` values ("user_package");
insert into `tlock` values ("user_package_up");
insert into `tlock` values ("config");
insert into `tlock` values ("site");
insert into `tlock` values ("record");
insert into `tlock` values ("site_group");
insert into `tlock` values ("merge_site_group");
insert into `tlock` values ("stream");
insert into `tlock` values ("stream_group");
insert into `tlock` values ("merge_stream_group");
insert into `tlock` values ("order");
insert into `tlock` values ("task");
insert into `tlock` values ("package_group");    
insert into `tlock` values ("merge_package_group");   
insert into `tlock` values ("message");   

# user
insert into user values (1, 'admin@cdn.cn', 'wenjian', '',null,null,null,null,null,0,null,null,0,0,now(),'$2b$12$hzRCWeN2vpX8NLyifoqpVOhAWkBbsNKZnEp7iBWpuSCNPAEUc7rom',1,1);
insert into user values (2, 'ceshi@cdn.cn', 'ceshi', '',null,null,null,null,null,0,null,null,0,0,now(),'$2b$12$DhHKbdFN2CANH11zRBCNTOt6Ci4VmBCOG4cQzSXjS85A2mYi5qwiS',1,2);
insert into message_sub values (1,'package-expire',1,1);
insert into message_sub values (1,'traffic-exceed',1,1);

insert into message_sub values (1,'cc-switch',1,1);
insert into message_sub values (1,'bandwidth-exceed',1,1);
insert into message_sub values (1,'connection-exceed',1,1);
insert into message_sub values (1,'cert-expire',1,1);

insert into message_sub values (2,'package-expire',1,1);
insert into message_sub values (2,'traffic-exceed',1,1);

insert into message_sub values (2,'cc-switch',1,1);
insert into message_sub values (2,'bandwidth-exceed',1,1);
insert into message_sub values (2,'connection-exceed',1,1);
insert into message_sub values (2,'cert-expire',1,1);
