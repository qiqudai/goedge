package services

import (
	"strings"
	"sync"
	"time"
)

type limitEntry struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

type InMemoryLimiter struct {
	mu       sync.Mutex
	entries  map[string]*limitEntry
	max      int
	window   time.Duration
	cooldown time.Duration
}

func NewInMemoryLimiter(max int, window, cooldown time.Duration) *InMemoryLimiter {
	if max <= 0 {
		max = 5
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	return &InMemoryLimiter{
		entries:  make(map[string]*limitEntry),
		max:      max,
		window:   window,
		cooldown: cooldown,
	}
}

func (l *InMemoryLimiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	if entry == nil {
		entry = &limitEntry{windowStart: now, lastSeen: now}
		l.entries[key] = entry
	}
	entry.lastSeen = now
	if now.Before(entry.blockedUntil) {
		return false, entry.blockedUntil.Sub(now)
	}
	if now.Sub(entry.windowStart) > l.window {
		entry.windowStart = now
		entry.count = 0
	}
	entry.count++
	if entry.count > l.max {
		entry.blockedUntil = now.Add(l.cooldown)
		return false, l.cooldown
	}
	return true, 0
}

var (
	loginLimiter   = NewInMemoryLimiter(5, 5*time.Minute, 10*time.Minute)
	captchaLimiter = NewInMemoryLimiter(3, 10*time.Minute, 30*time.Minute)
)

func AllowLoginAttempt(username, ip string) (bool, time.Duration) {
	return loginLimiter.Allow(composeLimiterKey("login", username, ip))
}

func AllowLoginCaptcha(username, ip string) (bool, time.Duration) {
	return captchaLimiter.Allow(composeLimiterKey("captcha", username, ip))
}

func composeLimiterKey(prefix, username, ip string) string {
	userPart := strings.ToLower(strings.TrimSpace(username))
	ipPart := strings.TrimSpace(ip)
	return prefix + "|" + ipPart + "|" + userPart
}
