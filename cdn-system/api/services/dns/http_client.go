package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const proxyCacheTTL = 5 * time.Second

var proxyCache struct {
	mu        sync.RWMutex
	raw       string
	parsed    *url.URL
	updatedAt time.Time
}

var proxyTransport *http.Transport

func init() {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		proxyTransport = base.Clone()
	} else {
		proxyTransport = &http.Transport{}
	}
	proxyTransport.Proxy = resolveProxy
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: proxyTransport,
	}
}

func resolveProxy(req *http.Request) (*url.URL, error) {
	proxy := getProxyURL()
	if proxy == nil {
		return http.ProxyFromEnvironment(req)
	}
	return proxy, nil
}

func getProxyURL() *url.URL {
	if db.DB == nil {
		return nil
	}
	now := time.Now()
	proxyCache.mu.RLock()
	if proxyCache.updatedAt.Add(proxyCacheTTL).After(now) {
		parsed := proxyCache.parsed
		proxyCache.mu.RUnlock()
		return parsed
	}
	proxyCache.mu.RUnlock()

	raw := ""
	var item models.ConfigItem
	err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", "system", "global", 0, "http_proxy").First(&item).Error
	if err == nil {
		raw = strings.TrimSpace(item.Value)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[DNS] load http_proxy config failed: %v", err)
	}

	parsed := parseProxyURL(raw)

	proxyCache.mu.Lock()
	proxyCache.raw = raw
	proxyCache.parsed = parsed
	proxyCache.updatedAt = now
	proxyCache.mu.Unlock()

	return parsed
}

func parseProxyURL(raw string) *url.URL {
	if raw == "" {
		return nil
	}
	value := raw
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		log.Printf("[DNS] invalid http_proxy value: %s", raw)
		return nil
	}
	return parsed
}
