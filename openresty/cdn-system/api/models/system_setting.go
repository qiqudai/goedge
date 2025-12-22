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

    // 2. Package Related
    ExpireCloseSite     bool `json:"expire_close_site"`
    TrafficCloseSite    bool `json:"traffic_close_site"`
    AllowUpgrade        bool `json:"allow_upgrade"`
    AllowDowngrade      bool `json:"allow_downgrade"`

    // 3. Maintenance
    MaintenanceStatus   bool   `json:"maintenance_status"`
    MaintenanceMsg      string `json:"maintenance_msg"`
    AutoUpgradeNode     bool   `json:"auto_upgrade_node"`

    // 4. Data Cleaning (Days)
    CleanCacheDays      int `json:"clean_cache_days"`      // 30
    CleanLoginLogDays   int `json:"clean_login_log_days"`  // 30
    CleanOpLogDays      int `json:"clean_op_log_days"`     // 365
    CleanSiteLogDays    int `json:"clean_site_log_days"`   // 7
    CleanNodeMonitorDays int `json:"clean_node_monitor_days"` // 7
    CleanTrafficDays    int `json:"clean_traffic_days"`    // 90
    CleanBlacklistDays  int `json:"clean_blacklist_days"`  // 7
    CleanNodeTrafficDays int `json:"clean_node_traffic_days"` // 45
    
    // Backup
    BackupFrequency     int    `json:"backup_frequency"` // Days
    BackupRetention     int    `json:"backup_retention"` // Days
    BackupDir           string `json:"backup_dir"`

    // 5. User Config
    SessionLife         int    `json:"session_life"` // Seconds
    LimitUserLoginDomain bool  `json:"limit_user_login_domain"`
    LimitAdminLoginDomain bool `json:"limit_admin_login_domain"`
    EnableEmailLogin    bool   `json:"enable_email_login"` 
    EnableSMSLogin      bool   `json:"enable_sms_login"`
    OpenRegister        bool   `json:"open_register"`
    
    // Templates
    RegisterMailTitle   string `json:"register_mail_title"`
    RegisterMailContent string `json:"register_mail_content"`
    ResetPwdmailTitle   string `json:"reset_pwd_mail_title"`
    ResetPwdmailContent string `json:"reset_pwd_mail_content"`
    VerifyMailTitle     string `json:"verify_mail_title"`
    VerifyMailContent   string `json:"verify_mail_content"`

    // 6. Notifications
    // Traffic Exceeded
    NotifyTrafficExceeded       bool     `json:"notify_traffic_exceeded"`
    NotifyTrafficExceededMethod []string `json:"notify_traffic_exceeded_method"` // email, sms
    NotifyTrafficExceededCount  int      `json:"notify_traffic_exceeded_count"`
    NotifyTrafficExceededInterval int    `json:"notify_traffic_exceeded_interval"`
    NotifyTrafficExceededTitle  string   `json:"notify_traffic_exceeded_title"`
    NotifyTrafficExceededContent string  `json:"notify_traffic_exceeded_content"`
    
    // Traffic Low
    NotifyTrafficLow            bool     `json:"notify_traffic_low"`
    NotifyTrafficLowThreshold   int      `json:"notify_traffic_low_threshold"`
    NotifyTrafficLowTitle       string   `json:"notify_traffic_low_title"`
    NotifyTrafficLowContent     string   `json:"notify_traffic_low_content"`
    
    // Expire
    NotifyExpire                bool     `json:"notify_expire"`
    NotifyExpireTitle           string   `json:"notify_expire_title"`
    NotifyExpireContent         string   `json:"notify_expire_content"`
    
    // Expire Soon
    NotifyExpireSoon            bool     `json:"notify_expire_soon"`
    NotifyExpireSoonDays        int      `json:"notify_expire_soon_days"`
    NotifyExpireSoonTitle       string   `json:"notify_expire_soon_title"`
    NotifyExpireSoonContent     string   `json:"notify_expire_soon_content"`
    
    // Domain CC Switch
    NotifyCCSwitch              bool     `json:"notify_cc_switch"`
    NotifyCCSwitchTitle         string   `json:"notify_cc_switch_title"`
    NotifyCCSwitchContent       string   `json:"notify_cc_switch_content"`
    
    // Cert Expire
    NotifyCertExpire            bool     `json:"notify_cert_expire"`
    NotifyCertExpireTitle       string   `json:"notify_cert_expire_title"`
    NotifyCertExpireContent     string   `json:"notify_cert_expire_content"`

    // 7. HTTPS
    CertContent string `json:"cert_content"` // PEM
    KeyContent  string `json:"key_content"`  // PEM
    ForceSSL    bool   `json:"force_ssl"`
}

type Link struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}
