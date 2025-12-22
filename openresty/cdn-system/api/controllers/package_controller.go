package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PackageController struct{}

// ListVersions
func (ctr *PackageController) ListVersions(c *gin.Context) {
    // Mock Versions
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "list": []gin.H{
                {"version": "v1.2.0", "status": "stable", "upload_time": "2023-10-01"},
                {"version": "v1.3.0-beta", "status": "gray", "gray_percent": 10, "upload_time": "2023-10-20"},
            },
        },
    })
}

// UploadVersion
func (ctr *PackageController) UploadVersion(c *gin.Context) {
    // File upload logic would go here
    c.JSON(http.StatusOK, gin.H{"msg": "Version Uploaded", "version": "v1.3.1"})
}

// UpdateGrayScale
func (ctr *PackageController) UpdateGrayScale(c *gin.Context) {
     c.JSON(http.StatusOK, gin.H{"msg": "Gray Scale Config Updated"})
}
