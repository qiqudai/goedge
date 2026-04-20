package main

import (
	"embed"
	"net/http"
	"sync"
	"time"
)

//go:embed assets
var assetsFS embed.FS

// Global Configuration
var (
	API_BaseURL   = "http://127.0.0.1:8080"
	HEARTBEAT_INT = 3 * time.Second
	LOG_SHIP_INT  = 5 * time.Second
	METRICS_INT   = 10 * time.Second
	L2_CHECK_INT  = 10 * time.Second

	// Dynamic Paths (will be set in initEnvironment)
	WorkDir      = "."
	CONFIG_PATH  = "" // e.g. ./edge-node/conf/cdn_config.json
	CONFIG_BAK   = ""
	NginxBinPath = "" // e.g. ./edge-node/nginx

	NodeID    = "" // Unique Node ID
	AuthToken = "" // Token from install parameter
	DebugMode = false

	GenevaEnable     = false
	GenevaWindowSize = uint16(512)

	// Bootstrap behavior
	ResetResources      = false // Clear WorkDir contents before init (requires safe path)
	BootstrapSync       = false // Pull full config from API on startup
	BootstrapStart      = false // Start OpenResty after bootstrap sync
	AutoInstallService  = true  // Linux: auto register systemd service on first start
	AutoDisableFirewall = true  // Linux: disable host firewall on startup to avoid blocked ports

	localConfigMu    sync.RWMutex
	LocalResources   *edgeResources
	LocalErrorPages  map[string]string
	LocalDefaultConf *edgeDefaultConfig
	LocalWAFConfig   *edgeWAFConfig
	LocalCCRules     map[int64][]edgeCCRuleItem
	LocalCCMatchers  map[int64]edgeCCMatcher
	LocalCCFilters   map[int64]edgeCCFilter
	LocalNginxConfig *edgeNginxConfig
	LocalPackages    map[int64]AgentPackageConfig
	pendingNodeSyncs []nodeSyncAck
)

type nodeSyncAck struct {
	Action    string
	Success   bool
	Attempts  int
	LastError string
	LastAt    time.Time
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}
