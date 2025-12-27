package controllers

import (
	"net/http"

	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type AcmeController struct{}

func (ctr *AcmeController) ServeChallenge(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.Status(http.StatusNotFound)
		return
	}
	if value, ok := services.AcmeTokens.Get(token); ok {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, value)
		return
	}
	c.Status(http.StatusNotFound)
}
