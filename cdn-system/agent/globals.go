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
	L2_CHECK_INT  = 30 * time.Second

	// Dynamic Paths (will be set in initEnvironment)
	WorkDir      = "./edge-node"
	CONFIG_PATH  = "" // e.g. ./edge-node/conf/cdn_config.json
	CONFIG_BAK   = ""
	NginxBinPath = "" // e.g. ./edge-node/nginx

	NodeID    = "" // Unique Node ID
	AuthToken = "" // Token from install parameter
	DebugMode = false

	localConfigMu    sync.RWMutex
	LocalResources   *edgeResources
	LocalErrorPages  map[string]string
	LocalDefaultConf *edgeDefaultConfig
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
