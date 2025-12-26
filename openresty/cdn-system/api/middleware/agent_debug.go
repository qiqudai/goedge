package middleware

import (
	"bytes"
	"io"
	"log"
	"strings"
	"time"

	"cdn-api/config"

	"github.com/gin-gonic/gin"
)

type agentDebugWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *agentDebugWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func AgentDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.App.Debug {
			c.Next()
			return
		}

		var reqBody string
		if c.Request.Body != nil {
			if bodyBytes, err := io.ReadAll(c.Request.Body); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				reqBody = strings.TrimSpace(string(bodyBytes))
				if len(reqBody) > 2048 {
					reqBody = reqBody[:2048] + "...(truncated)"
				}
			}
		}

		start := time.Now()
		writer := &agentDebugWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = writer
		c.Next()

		respBody := strings.TrimSpace(writer.body.String())
		if len(respBody) > 2048 {
			respBody = respBody[:2048] + "...(truncated)"
		}

		log.Printf(
			"[AgentDebug] %s %s status=%d duration=%s req=%s resp=%s",
			c.Request.Method,
			c.Request.URL.RequestURI(),
			writer.Status(),
			time.Since(start).String(),
			reqBody,
			respBody,
		)
	}
}
