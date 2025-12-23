package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FinanceController struct{}

type adminOrderRow struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    int     `json:"status"`
	CreatedAt string  `json:"created_at"`
	PayType   string  `json:"pay_type"`
	OrderNo   string  `json:"order_no"`
	Type      string  `json:"type"`
	Remark    string  `json:"remark"`
}

type userOrderRow struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	TypeLabel string `json:"type_label"`
	Remark    string `json:"remark"`
	Price     string `json:"price"`
	Pay       string `json:"pay"`
	More      string `json:"more"`
	PayType   string `json:"pay_type"`
	OrderNo   string `json:"order_no"`
	CreatedAt string `json:"created_at"`
	Paid      bool   `json:"paid"`
}

func isPaid(state string) bool {
	switch strings.ToLower(state) {
	case "paid", "success", "done":
		return true
	default:
		return false
	}
}

func formatAmount(amount int64) float64 {
	return float64(amount) / 100.0
}

func orderTypeLabel(t string) string {
	switch strings.ToLower(t) {
	case "purchase":
		return "??"
	case "renew":
		return "??"
	case "recharge":
		return "??"
	default:
		return "??"
	}
}

// ListOrders
// GET /api/v1/admin/orders
func (ctr *FinanceController) ListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))

	query := db.DB.Model(&models.Order{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("mch_order_no LIKE ? OR des LIKE ?", like, like)
		if uid, err := strconv.Atoi(keyword); err == nil {
			query = query.Or("uid = ?", uid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var orders []models.Order
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	list := make([]adminOrderRow, 0, len(orders))
	for _, o := range orders {
		list = append(list, adminOrderRow{
			ID:        o.ID,
			UserID:    o.UserID,
			Amount:    formatAmount(o.Amount),
			Status:    map[bool]int{true: 1, false: 0}[isPaid(o.State)],
			CreatedAt: o.CreatedAt.Format("2006-01-02 15:04:05"),
			PayType:   o.PayType,
			OrderNo:   o.MerchantOrder,
			Type:      o.Type,
			Remark:    o.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// ListUserOrders
// GET /api/v1/user/orders
func (ctr *FinanceController) ListUserOrders(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	orderType := strings.TrimSpace(c.Query("type"))

	query := db.DB.Model(&models.Order{}).Where("uid = ?", userID)
	if orderType != "" {
		query = query.Where("type = ?", orderType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("mch_order_no LIKE ? OR des LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var orders []models.Order
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	list := make([]userOrderRow, 0, len(orders))
	for _, o := range orders {
		amount := formatAmount(o.Amount)
		amountText := strconv.FormatFloat(amount, 'f', 2, 64) + "?"
		list = append(list, userOrderRow{
			ID:        o.ID,
			Type:      o.Type,
			TypeLabel: orderTypeLabel(o.Type),
			Remark:    o.Description,
			Price:     amountText,
			Pay:       amountText,
			More:      o.Data,
			PayType:   o.PayType,
			OrderNo:   o.MerchantOrder,
			CreatedAt: o.CreatedAt.Format("2006-01-02 15:04:05"),
			Paid:      isPaid(o.State),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// ManualRecharge
func (ctr *FinanceController) Recharge(c *gin.Context) {
	var req struct {
		UserID int64   `json:"user_id"`
		Amount float64 `json:"amount"`
		Remark string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}
	if req.UserID <= 0 || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user_id or amount"})
		return
	}

	amountCents := int64(req.Amount * 100)
	now := time.Now()

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Where("id = ?", req.UserID).First(&user).Error; err != nil {
			return err
		}
		user.Balance += amountCents
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		order := models.Order{
			UserID:        req.UserID,
			Type:          "recharge",
			Description:   req.Remark,
			Data:          "",
			CreatedAt:     now,
			PaidAt:        now,
			Amount:        amountCents,
			PayType:       "manual",
			MerchantOrder: "manual-" + now.Format("20060102150405"),
			TransactionID: "",
			State:         "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Recharge Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Recharge Successful"})
}
