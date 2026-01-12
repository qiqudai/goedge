package main

import (
	"bytes"
	"cdn-common/acme"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type apiTokenStore struct {
	apiBase string
}

type acmeTokenPayload struct {
	Token string `json:"token"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

func newAPITokenStore() acme.TokenStore {
	apiBase := strings.TrimRight(strings.TrimSpace(API_BaseURL), "/")
	if apiBase == "" || strings.TrimSpace(AuthToken) == "" {
		return nil
	}
	return &apiTokenStore{apiBase: apiBase}
}

func (s *apiTokenStore) Put(token, value string, ttl time.Duration) {
	if s == nil || strings.TrimSpace(token) == "" || strings.TrimSpace(value) == "" {
		return
	}
	payload := acmeTokenPayload{
		Token: token,
		Value: value,
		TTL:   int64(ttl.Seconds()),
	}
	body, _ := json.Marshal(payload)
	endpoint := s.apiBase + "/api/v1/agent/acme/tokens"
	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")
	respBody, status, err := doRequest(req, 10*time.Second, true)
	if err != nil || status != http.StatusOK {
		log.Printf("[Warn] acme token sync failed status=%d err=%v resp=%s", status, err, strings.TrimSpace(string(respBody)))
		return
	}
	debugLogInteraction("POST", endpoint, status, body, respBody)
}

func (s *apiTokenStore) Delete(token string) {
	if s == nil || strings.TrimSpace(token) == "" {
		return
	}
	endpoint := s.apiBase + "/api/v1/agent/acme/tokens/" + url.PathEscape(token)
	req, _ := http.NewRequest("DELETE", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	respBody, status, err := doRequest(req, 10*time.Second, true)
	if err != nil || status != http.StatusOK {
		log.Printf("[Warn] acme token delete failed status=%d err=%v resp=%s", status, err, strings.TrimSpace(string(respBody)))
		return
	}
	debugLogInteraction("DELETE", endpoint, status, nil, respBody)
}

func (s *apiTokenStore) Get(_ string) (string, bool) {
	return "", false
}
