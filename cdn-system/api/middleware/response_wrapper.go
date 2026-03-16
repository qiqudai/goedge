package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"cdn-api/response"
	"cdn-common/i18n"

	"github.com/gin-gonic/gin"
)

type bodyWriter struct {
	gin.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *bodyWriter) WriteHeader(code int) {
	w.status = code
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

// ResponseWrapper normalizes API responses into {code,message,data,...} and localizes message.
func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		bw := &bodyWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = bw

		c.Next()

		// Restore writer for final output.
		c.Writer = bw.ResponseWriter

		rawBody := bw.body.Bytes()
		if len(rawBody) == 0 {
			return
		}

		if !shouldWrap(c, rawBody) {
			writeRaw(c, bw.status, rawBody)
			return
		}

		var payload interface{}
		if err := json.Unmarshal(rawBody, &payload); err != nil {
			writeRaw(c, bw.status, rawBody)
			return
		}

		lang := resolveLang(c)
		out := normalizePayload(payload, bw.status, lang)
		writeJSON(c, out)
	}
}

func shouldWrap(c *gin.Context, body []byte) bool {
	if c != nil && strings.HasPrefix(c.Request.URL.Path, "/api/v1/agent/") {
		return false
	}
	if ct := c.Writer.Header().Get("Content-Type"); ct != "" && !strings.Contains(ct, "json") {
		return false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed[0]
	return first == '{' || first == '['
}

func writeRaw(c *gin.Context, status int, body []byte) {
	if status == 0 {
		status = http.StatusOK
	}
	c.Writer.WriteHeader(status)
	_, _ = c.Writer.Write(body)
}

func writeJSON(c *gin.Context, payload interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(c.Writer)
	_ = enc.Encode(payload)
}

func resolveLang(c *gin.Context) string {
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = strings.TrimSpace(c.GetHeader("Accept-Language"))
	}
	if lang == "" {
		return ""
	}
	// Take the first language tag.
	if idx := strings.Index(lang, ","); idx != -1 {
		lang = lang[:idx]
	}
	if idx := strings.Index(lang, ";"); idx != -1 {
		lang = lang[:idx]
	}
	return strings.TrimSpace(lang)
}

func normalizePayload(payload interface{}, httpStatus int, lang string) map[string]interface{} {
	result := map[string]interface{}{}
	traceID := ""
	var errVal interface{}
	var msg string

	switch v := payload.(type) {
	case map[string]interface{}:
		if val, ok := v["trace_id"].(string); ok {
			traceID = val
		}
		if val, ok := v["error"]; ok {
			errVal = val
		}
		msg = extractMessage(v)
		code := response.NormalizeCode(v["code"], httpStatus)
		data, hasData := extractData(v)
		if !hasData {
			data = nil
		}
		if msg == "" {
			if code == response.CodeSuccess {
				msg = "Success"
			} else {
				msg = "Error"
			}
		}
		msg = i18n.Translate(lang, msg)
		result["code"] = code
		result["message"] = msg
		result["data"] = data
		if traceID != "" {
			result["trace_id"] = traceID
		}
		if errVal != nil {
			result["error"] = errVal
		}
		return result
	case []interface{}:
		result["code"] = response.CodeSuccess
		result["message"] = i18n.Translate(lang, "Success")
		result["data"] = v
		return result
	default:
		result["code"] = response.CodeSuccess
		result["message"] = i18n.Translate(lang, "Success")
		result["data"] = v
		return result
	}
}

func extractMessage(v map[string]interface{}) string {
	if val, ok := v["message"]; ok {
		return toString(val)
	}
	if val, ok := v["msg"]; ok {
		return toString(val)
	}
	if val, ok := v["error"]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func extractData(v map[string]interface{}) (interface{}, bool) {
	if val, ok := v["data"]; ok {
		return val, true
	}
	// Build data from remaining fields.
	skip := map[string]struct{}{
		"code":     {},
		"message":  {},
		"msg":      {},
		"error":    {},
		"trace_id": {},
	}
	data := map[string]interface{}{}
	for key, val := range v {
		if _, ok := skip[key]; ok {
			continue
		}
		data[key] = val
	}
	if len(data) == 0 {
		return nil, false
	}
	return data, true
}

func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toJSON(v)), "\n", " "))
	}
}

func toJSON(val interface{}) string {
	b, err := json.Marshal(val)
	if err != nil {
		return ""
	}
	return string(b)
}
