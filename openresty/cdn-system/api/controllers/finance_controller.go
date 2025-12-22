package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type FinanceController struct{}

// ListOrders
func (ctr *FinanceController) ListOrders(c *gin.Context) {
    // Mock Data
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "list": []gin.H{
                {"id": 1001, "user_id": 1, "amount": 100.00, "status": 1, "created_at": "2023-10-01 12:00:00"},
                {"id": 1002, "user_id": 2, "amount": 500.00, "status": 1, "created_at": "2023-10-02 14:00:00"},
            },
        },
    })
}

// ManualRecharge
func (ctr *FinanceController) Recharge(c *gin.Context) {
    var req struct {
        UserID int64 `json:"user_id"`
        Amount float64 `json:"amount"`
        Remark string `json:"remark"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
         return
    }
    // Logic: User.Balance += Amount
    c.JSON(http.StatusOK, gin.H{"msg": "Recharge Successful"})
}
