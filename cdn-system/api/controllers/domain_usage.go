package controllers

import (
	"cdn-api/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (ctrl *SiteController) DomainUsage(c *gin.Context) {
	userID := int64(0)
	if isUserRequest(c) {
		userID = parseInt64(mustGet(c, "userID"))
	} else {
		if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
			if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
				userID = uid
			}
		}
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
		return
	}

	userPackageID := int64(0)
	if pidStr := strings.TrimSpace(c.Query("user_package_id")); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			userPackageID = pid
		}
	}
	if userPackageID == 0 {
		defaultID, err := findDefaultUserPackageID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(err.Error())})
			return
		}
		userPackageID = defaultID
	}

	usage, err := services.GetDomainUsage(userID, userPackageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Load failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": usage})
}
