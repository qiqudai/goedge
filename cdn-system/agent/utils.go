package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func doRequest(req *http.Request, timeout time.Duration, readBody bool) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if readBody {
		body, err := ioutil.ReadAll(resp.Body)
		return body, resp.StatusCode, err
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil, resp.StatusCode, nil
}

func debugLogInteraction(method, url string, status int, reqBody, respBody []byte) {
	if !DebugMode {
		return
	}
	reqText := strings.TrimSpace(string(reqBody))
	if len(reqText) > 1024 {
		reqText = reqText[:1024] + "...(truncated)"
	}
	respText := strings.TrimSpace(string(respBody))
	if len(respText) > 1024 {
		respText = respText[:1024] + "...(truncated)"
	}
	log.Printf("[Debug] agent->api %s %s status=%d req=%s resp=%s", method, url, status, reqText, respText)
}

func splitLines(input string) []string {
	parts := strings.Split(input, "\n")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
