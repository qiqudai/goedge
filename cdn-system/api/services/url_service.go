package services

import (
	"net/http"
	"os"
	"strings"
)

func ResolveAPIBaseURL(req *http.Request) string {
	if env := strings.TrimSpace(os.Getenv("API_BASE_URL")); env != "" {
		return strings.TrimRight(env, "/")
	}
	if req == nil {
		return ""
	}
	proto := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if req.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(req.Host)
	}
	if host == "" {
		return ""
	}
	return strings.TrimRight(proto+"://"+host, "/")
}
