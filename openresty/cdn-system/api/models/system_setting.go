package models

// Detailed System Configuration
type SystemSettings struct {
    // 1. System Config
    SystemName          string   `json:"system_name"`           // cdn 4.0
    UserTitle           string   `json:"user_title"`            // cdn用户控制台
    AdminTitle          string   `json:"admin_title"`           // cdn管理员控制台
    FooterLinks         []Link   `json:"footer_links"`          // [Name, URL]
    FooterText          string   `json:"footer_text"`           // Copyright...
    GlobalJS            string   `json:"global_js"`             // Console.log etc
    AdminDomains        []string `json:"admin_domains"`         // Bind domains
    FaviconFile         string   `json:"favicon_file"`
    LogoFile            string   `json:"logo_file"`
    LoginAdFile         string   `json:"login_ad_file"`

    // 2. Package Related (Updated Keys)
    ExpireCloseSite     bool `json:"package_expire_close_site"`
    TrafficCloseSite    bool `json:"traffic_excceed_close_site"` // NOTE: user dump has typo "excceed"
    AllowUpgrade        bool `json:"package_allow_upgrade"`
    AllowDowngrade      bool `json:"package_allow_downgrade"`

    // 3. Maintenance
    MaintenanceStatus   bool   `json:"maintenance_status"` // or 'maintain' JSON? Dump has 'maintain' {"enable":0...}
    MaintenanceMsg      string `json:"maintenance_msg"`
    AutoUpgradeNode     bool   `json:"auto_upgrade_node"`
    AutoUpgradeAgent    bool   `json:"auto_upgrade_agent"` // New

    // 4. Data Cleaning (Days) -> Keep Keys
    CleanCacheDays      int `json:"keep-job-days"`
    CleanLoginLogDays   int `json:"keep-login-log-days"`
    CleanOpLogDays      int `json:"keep-op-log-days"`
    CleanSiteLogDays    int `json:"keep-access-log-days"`
    CleanNodeMonitorDays int `json:"keep-node-log-days"`
    CleanTrafficDays    int `json:"keep-traffic-history-days"`
    CleanBlacklistDays  int `json:"keep-blacklist-days"`
    CleanNodeTrafficDays int `json:"keep-node-traffic-days"`
    
    // Backup
    BackupFrequency     string `json:"backup_rate"` // User dump has "2h", string?
    BackupRetention     int    `json:"backup_keep_days"`
    BackupDir           string `json:"backup_dir"`

    // 5. User Config
    SessionLife         int    `json:"login_session_valid_time"`
    LimitUserLoginDomain bool  `json:"limit_user_login_domain"` 
    LimitAdminLoginDomain bool `json:"limit_admin_login_domain"`
    EnableEmailLogin    bool   `json:"allow-enable-email-captcha-login"` 
    EnableSMSLogin      bool   `json:"allow-enable-sms-captcha-login"`
    OpenRegister        bool   `json:"allow_register"`
    
    // Templates
    RegisterMailTitle   string `json:"-"` // Use JSON structs
    RegisterMailContent string `json:"-"`
    ResetPwdmailTitle   string `json:"-"`
    ResetPwdmailContent string `json:"-"`
    VerifyMailTitle     string `json:"-"`
    VerifyMailContent   string `json:"-"`
    
    RegisterSuccessTempl   string `json:"register_success_templ"` // JSON
    ForgetPasswordTempl    string `json:"forget_password_templ"`  // JSON
    EmailCaptchaTempl      string `json:"email_captcha_templ"`    // JSON
    
    // SMS Templates
    PhoneCaptchaTemplId string `json:"phone_captcha_templ_id"`
    PhoneCaptchaTempl   string `json:"phone_captcha_templ"`

    // 6. Notifications
    NotificationPeriod string `json:"notification-period"` // all, custom

    // JSON configs
    NotifyTrafficExceedInfo    string `json:"traffic-exceed-notify"`
    NotifyTrafficLowInfo       string `json:"traffic-exceeding-notify"`
    NotifyPackageExpireInfo    string `json:"package-expire-notify"`
    NotifyPackageExpiringInfo  string `json:"package-expiring-notify"`
    NotifyCCSwitchInfo         string `json:"cc-switch-notify"`
    NotifyBandwidthExceedInfo  string `json:"bandwidth-exceed-notify"`
    NotifyCertExpireInfo       string `json:"cert-expire-notify"`
    NotifyCertExpiringInfo     string `json:"cert-expiring-notify"`
    NotifyConnExceedInfo       string `json:"conn-exceed-notify"`
    NotifyAccountAuth2Info     string `json:"account-auth2-notify"`

    // 7. HTTPS
    CertContent string `json:"https_cert"`
    KeyContent  string `json:"https_key"`
    ForceSSL    bool   `json:"force_ssl"`

    // 8. Other Config
    MasterClientIpHeader        string  `json:"master_client_ip_header"`
    RecordRepairEnable          int     `json:"record-repair-enable"`
    DnsRsProtect                string  `json:"dns_rs_protect"`
    MaxSiteStreamSyncOneTime    int     `json:"max_site_stream_sync_one_time"`
    SyncSiteConfigScope         string  `json:"sync-site-config-scope"`
    ResRankSize                 int     `json:"res_rank_size"`
    HttpProxy                   string  `json:"http_proxy"`
    ApiKeyStatus                bool    `json:"api_key_status"`
    TcpTrafficFactor            float64 `json:"tcp_traffic_factor"`
    
    // Node
    NodeHealthCheck   bool `json:"node_health_check"`
    NodeMaxFailed     int  `json:"node_max_failed"`
}

type Link struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}
