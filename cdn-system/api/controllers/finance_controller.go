package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type balanceLedgerRow struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	OrderID      int64  `json:"order_id"`
	Action       string `json:"action"`
	Source       string `json:"source"`
	Reason       string `json:"reason"`
	Before       int64  `json:"amount_before"`
	Change       int64  `json:"amount_change"`
	After        int64  `json:"amount_after"`
	OperatorID   int64  `json:"operator_id"`
	OperatorRole string `json:"operator_role"`
	CreatedAt    string `json:"created_at"`
}

type packageOrderData struct {
	PackageID     int64 `json:"package_id,omitempty"`
	UserPackageID int64 `json:"user_package_id,omitempty"`
	Months        int   `json:"months"`
	AutoRenew     bool  `json:"auto_renew"`
}

func periodToMonths(period string, months int) int {
	if months > 0 {
		return months
	}
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "month":
		return 1
	case "quarter":
		return 3
	case "year":
		return 12
	default:
		return 0
	}
}

func packageAmountByMonths(monthPrice, quarterPrice, yearPrice int64, months int) (int64, error) {
	if months <= 0 {
		return 0, errors.New("invalid months")
	}
	switch months {
	case 1:
		if monthPrice > 0 {
			return monthPrice, nil
		}
	case 3:
		if quarterPrice > 0 {
			return quarterPrice, nil
		}
	case 12:
		if yearPrice > 0 {
			return yearPrice, nil
		}
	}
	if monthPrice > 0 {
		return monthPrice * int64(months), nil
	}
	if quarterPrice > 0 {
		return int64(math.Round(float64(quarterPrice) * float64(months) / 3.0)), nil
	}
	if yearPrice > 0 {
		return int64(math.Round(float64(yearPrice) * float64(months) / 12.0)), nil
	}
	return 0, errors.New("no valid price for selected period")
}

func parsePackageOrderData(raw string) (*packageOrderData, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("order data is empty")
	}
	var data packageOrderData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	if data.Months <= 0 {
		return nil, errors.New("order data months is invalid")
	}
	return &data, nil
}

func createUserPackageFromPlanTx(tx *gorm.DB, userID int64, packageID int64, months int) (int64, error) {
	var pkg models.Package
	if err := tx.Where("id = ?", packageID).First(&pkg).Error; err != nil {
		return 0, err
	}
	now := time.Now()
	endAt := now.AddDate(0, months, 0)
	recordID, err := generateUniqueRecordIDTx(tx)
	if err != nil {
		return 0, err
	}
	userPkg := models.UserPackage{
		UserID:          int32(userID),
		Name:            pkg.Name,
		PackageID:       int32(pkg.ID),
		RegionID:        pkg.RegionID,
		NodeGroupID:     pkg.NodeGroupID,
		BackupNodeGroup: pkg.BackupNode,
		EnableBackup:    false,
		CnameDomain:     pkg.CnameDomain,
		CnameHostname2:  pkg.CnameHost2,
		CnameMode:       pkg.CnameMode,
		RecordID:        recordID,
		Traffic:         int32(pkg.Traffic),
		Bandwidth:       pkg.Bandwidth,
		Connection:      int32(pkg.Connection),
		DomainLimit:     int32(pkg.DomainLimit),
		HTTPPortLimit:   int32(pkg.HttpPort),
		StreamPortLimit: int32(pkg.StreamPort),
		CustomCCRule:    pkg.CustomCCRule,
		Websocket:       pkg.Websocket,
		L2Origin:        pkg.L2Origin,
		MonthPrice:      pkg.MonthPrice,
		QuarterPrice:    pkg.QuarterPrice,
		YearPrice:       pkg.YearPrice,
		StartAt:         now,
		EndAt:           endAt,
		CreatedAt:       now,
	}
	if err := tx.Create(&userPkg).Error; err != nil {
		return 0, err
	}
	return userPkg.ID, nil
}

func renewUserPackageTx(tx *gorm.DB, userID int64, userPackageID int64, months int) (int64, error) {
	var pack models.UserPackage
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND uid = ?", userPackageID, userID).
		First(&pack).Error; err != nil {
		return 0, err
	}
	now := time.Now()
	base := pack.EndAt
	if base.IsZero() || base.Before(now) {
		base = now
	}
	newEnd := base.AddDate(0, months, 0)
	if err := tx.Model(&models.UserPackage{}).Where("id = ?", pack.ID).Updates(map[string]interface{}{
		"end_at":     newEnd,
		"is_expired": false,
	}).Error; err != nil {
		return 0, err
	}
	return pack.ID, nil
}

func generateUniqueRecordIDTx(tx *gorm.DB) (string, error) {
	for i := 0; i < 5; i++ {
		id, err := randomOrderToken(8)
		if err != nil {
			return "", err
		}
		var count int64
		if err := tx.Model(&models.UserPackage{}).Where("record_id = ?", id).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return id, nil
		}
	}
	return "", errors.New("failed to allocate unique record id")
}

func isPaidState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "paid", "success", "done":
		return true
	default:
		return false
	}
}

func formatAmount(amount int64) float64 {
	return float64(amount) / 100.0
}

func formatAmountText(amount int64) string {
	return strconv.FormatFloat(float64(amount)/100.0, 'f', 2, 64)
}

func toCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func orderTypeLabel(orderType string) string {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "purchase":
		return "Purchase"
	case "renew":
		return "Renew"
	case "recharge":
		return "Recharge"
	case "adjust":
		return "Adjust"
	default:
		return "Order"
	}
}

func normalizePayType(raw string) string {
	val := strings.ToLower(strings.TrimSpace(raw))
	switch val {
	case "", "usdt", "trc20", "usdt_trc20", "usdt-trc20", "shkeeper", "shkeeper_trc20":
		return "usdt_trc20"
	default:
		return val
	}
}

func isShkeeperPayType(payType string) bool {
	return normalizePayType(payType) == "usdt_trc20"
}

func randomOrderToken(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf), nil
}

func generateMerchantOrder(prefix string) string {
	now := time.Now().Format("20060102150405")
	token, err := randomOrderToken(6)
	if err != nil {
		token = "random"
	}
	return fmt.Sprintf("%s-%s-%s", prefix, now, token)
}

func deriveCallbackURL(c *gin.Context, configured string) string {
	cfg := strings.TrimSpace(configured)
	if cfg != "" {
		return cfg
	}
	if c == nil || c.Request == nil {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if fwd := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); fwd != "" {
		scheme = strings.ToLower(strings.Split(fwd, ",")[0])
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s/api/v1/pay/shkeeper/callback", scheme, host)
}

func createShkeeperPayInfo(c *gin.Context, merchantOrder string, amountCents int64) (map[string]interface{}, error) {
	settings, err := services.LoadShkeeperSettings()
	if err != nil {
		return nil, err
	}
	if !settings.Enable {
		return nil, errors.New("SHKeeper is disabled")
	}
	callbackURL := deriveCallbackURL(c, settings.CallbackURL)
	if callbackURL == "" {
		return nil, errors.New("callback url is empty")
	}
	invoiceResp, err := services.ShkeeperCreateInvoice(settings, services.ShkeeperInvoiceCreateRequest{
		ExternalID:  merchantOrder,
		Fiat:        settings.Fiat,
		Amount:      formatAmountText(amountCents),
		CallbackURL: callbackURL,
	})
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{
		"channel":         "shkeeper",
		"crypto":          settings.CryptoName,
		"fiat":            settings.Fiat,
		"invoice_id":      invoiceResp.ID,
		"wallet":          invoiceResp.Wallet,
		"expected_amount": invoiceResp.Amount,
		"exchange_rate":   invoiceResp.ExchangeRate,
		"display_name":    invoiceResp.DisplayName,
		"status":          invoiceResp.Status,
	}
	return data, nil
}

func marshalJSON(raw interface{}) string {
	b, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	return string(b)
}

func summarizeOrderMore(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return raw
	}
	parts := make([]string, 0, 5)
	if channel, ok := data["channel"].(string); ok && channel != "" {
		parts = append(parts, "channel="+channel)
	}
	if crypto, ok := data["crypto"].(string); ok && crypto != "" {
		parts = append(parts, "crypto="+crypto)
	}
	if amount, ok := data["expected_amount"].(string); ok && amount != "" {
		parts = append(parts, "crypto_amount="+amount)
	}
	if wallet, ok := data["wallet"].(string); ok && wallet != "" {
		parts = append(parts, "wallet="+wallet)
	}
	if len(parts) == 0 {
		return raw
	}
	return strings.Join(parts, ", ")
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

// ListOrders
// GET /api/v1/admin/orders
func (ctr *FinanceController) ListOrders(c *gin.Context) {
	page, pageSize := parsePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	orderType := strings.TrimSpace(c.Query("type"))
	state := strings.TrimSpace(c.Query("state"))

	query := db.DB.Model(&models.Order{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("mch_order_no LIKE ? OR des LIKE ?", like, like)
		if uid, err := strconv.ParseInt(keyword, 10, 64); err == nil && uid > 0 {
			query = query.Or("uid = ?", uid)
		}
	}
	if orderType != "" {
		query = query.Where("type = ?", orderType)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	var orders []models.Order
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	list := make([]adminOrderRow, 0, len(orders))
	for _, o := range orders {
		list = append(list, adminOrderRow{
			ID:        o.ID,
			UserID:    o.UserID,
			Amount:    formatAmount(o.Amount),
			Status:    map[bool]int{true: 1, false: 0}[isPaidState(o.State)],
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
	userID := parseUserID(mustGet(c, "userID"))
	if userID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
		return
	}

	page, pageSize := parsePagination(c)
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	var orders []models.Order
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	list := make([]userOrderRow, 0, len(orders))
	for _, o := range orders {
		amountText := formatAmountText(o.Amount)
		list = append(list, userOrderRow{
			ID:        o.ID,
			Type:      o.Type,
			TypeLabel: orderTypeLabel(o.Type),
			Remark:    o.Description,
			Price:     amountText,
			Pay:       amountText,
			More:      summarizeOrderMore(o.Data),
			PayType:   o.PayType,
			OrderNo:   o.MerchantOrder,
			CreatedAt: o.CreatedAt.Format("2006-01-02 15:04:05"),
			Paid:      isPaidState(o.State),
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

// Recharge keeps compatibility for existing admin "manual recharge" UI.
// POST /api/v1/admin/recharge
func (ctr *FinanceController) Recharge(c *gin.Context) {
	var req struct {
		UserID int64   `json:"user_id"`
		Amount float64 `json:"amount"`
		Remark string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.UserID <= 0 || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid user_id or amount")})
		return
	}
	if err := adjustBalanceWithOrder(req.UserID, req.Amount, "credit", req.Remark, "admin_manual", parseUserID(mustGet(c, "userID")), "admin"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Recharge Successful")})
}

// AdminAdjustBalance
// POST /api/v1/admin/balance/adjust
func (ctr *FinanceController) AdminAdjustBalance(c *gin.Context) {
	var req struct {
		UserID int64   `json:"user_id"`
		Amount float64 `json:"amount"`
		Action string  `json:"action"`
		Reason string  `json:"reason"`
		Source string  `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.UserID <= 0 || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid user_id or amount")})
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "debit" {
		action = "credit"
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		if action == "debit" {
			source = "admin_deduct"
		} else {
			source = "admin_recharge"
		}
	}
	if err := adjustBalanceWithOrder(req.UserID, req.Amount, action, req.Reason, source, parseUserID(mustGet(c, "userID")), "admin"); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "insufficient balance") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Adjusted")})
}

func adjustBalanceWithOrder(userID int64, amount float64, action string, reason string, source string, operatorID int64, operatorRole string) error {
	amountCents := toCents(amount)
	if amountCents <= 0 {
		return errors.New("invalid amount")
	}
	change := amountCents
	if strings.ToLower(strings.TrimSpace(action)) == "debit" {
		change = -amountCents
	}
	now := time.Now()
	orderType := "adjust"
	if change > 0 {
		orderType = "recharge"
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		order := models.Order{
			UserID:        userID,
			Type:          orderType,
			Description:   strings.TrimSpace(reason),
			Data:          "",
			CreatedAt:     now,
			PaidAt:        now,
			Amount:        change,
			PayType:       source,
			MerchantOrder: generateMerchantOrder("adj"),
			TransactionID: "",
			State:         "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		_, err := services.AdjustUserBalanceWithLedger(tx, services.BalanceAdjustInput{
			UserID:       userID,
			OrderID:      order.ID,
			AmountChange: change,
			Reason:       strings.TrimSpace(reason),
			Source:       source,
			OperatorID:   operatorID,
			OperatorRole: operatorRole,
		})
		return err
	})
}

// UserRecharge
// POST /api/v1/user/recharge
func (ctr *FinanceController) UserRecharge(c *gin.Context) {
	userID := parseUserID(mustGet(c, "userID"))
	if userID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
		return
	}

	var req struct {
		Amount  float64 `json:"amount"`
		Remark  string  `json:"remark"`
		PayType string  `json:"pay_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid amount")})
		return
	}

	amountCents := toCents(req.Amount)
	if amountCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid amount")})
		return
	}
	payType := normalizePayType(req.PayType)
	now := time.Now()

	if isShkeeperPayType(payType) {
		merchantOrder := generateMerchantOrder("recharge")
		data, err := createShkeeperPayInfo(c, merchantOrder, amountCents)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "msg": err.Error()})
			return
		}
		order := models.Order{
			UserID:        userID,
			Type:          "recharge",
			Description:   strings.TrimSpace(req.Remark),
			Data:          marshalJSON(data),
			CreatedAt:     now,
			Amount:        amountCents,
			PayType:       payType,
			MerchantOrder: merchantOrder,
			TransactionID: "",
			State:         "pending",
		}
		if err := db.DB.Omit("pay_at").Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  T("Order Created"),
			"data": gin.H{
				"order_id":   order.ID,
				"order_no":   order.MerchantOrder,
				"pay_type":   order.PayType,
				"amount":     formatAmount(order.Amount),
				"pay_info":   data,
				"created_at": order.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		})
		return
	}

	order := models.Order{
		UserID:        userID,
		Type:          "recharge",
		Description:   strings.TrimSpace(req.Remark),
		Data:          "",
		CreatedAt:     now,
		Amount:        amountCents,
		PayType:       payType,
		MerchantOrder: generateMerchantOrder("recharge"),
		TransactionID: "",
		State:         "pending",
	}
	if err := db.DB.Omit("pay_at").Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Order Created"), "data": gin.H{"order_id": order.ID, "order_no": order.MerchantOrder}})
}

// UserCreatePackageOpenOrder
// POST /api/v1/user/orders/package/open
func (ctr *FinanceController) UserCreatePackageOpenOrder(c *gin.Context) {
	userID := parseUserID(mustGet(c, "userID"))
	if userID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
		return
	}
	var req struct {
		PackageID int64  `json:"package_id"`
		Period    string `json:"period"`
		Months    int    `json:"months"`
		PayType   string `json:"pay_type"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "package_id is required"})
		return
	}
	var pkg models.Package
	if err := db.DB.Where("id = ? AND enable = ?", req.PackageID, true).First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "package not found"})
		return
	}
	months := periodToMonths(req.Period, req.Months)
	if months <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid period"})
		return
	}
	amountCents, err := packageAmountByMonths(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice, months)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	payType := normalizePayType(req.PayType)
	if payType == "" {
		payType = "balance"
	}
	now := time.Now()
	merchantOrder := generateMerchantOrder("purchase")
	data := map[string]interface{}{
		"package_id": req.PackageID,
		"months":     months,
		"auto_renew": true,
	}
	order := models.Order{
		UserID:        userID,
		Type:          "purchase",
		Description:   strings.TrimSpace(req.Remark),
		Data:          marshalJSON(data),
		CreatedAt:     now,
		Amount:        amountCents,
		PayType:       payType,
		MerchantOrder: merchantOrder,
		State:         "pending",
	}
	var syncIDs []int64
	if payType == "balance" {
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			var err error
			syncIDs, err = applyOrderPaidTx(tx, order.ID, "", "balance_pay", "purchase by balance", userID, "user", nil)
			return err
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		syncUserPackages(syncIDs, "purchase")
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "purchase success",
			"data": gin.H{"order_id": order.ID, "order_no": order.MerchantOrder, "paid": true},
		})
		return
	}
	if isShkeeperPayType(payType) {
		payInfo, err := createShkeeperPayInfo(c, merchantOrder, amountCents)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "msg": err.Error()})
			return
		}
		data["channel"] = payInfo["channel"]
		data["crypto"] = payInfo["crypto"]
		data["fiat"] = payInfo["fiat"]
		data["invoice_id"] = payInfo["invoice_id"]
		data["wallet"] = payInfo["wallet"]
		data["expected_amount"] = payInfo["expected_amount"]
		data["exchange_rate"] = payInfo["exchange_rate"]
		data["display_name"] = payInfo["display_name"]
		data["status"] = payInfo["status"]
		order.Data = marshalJSON(data)
	}
	if err := db.DB.Omit("pay_at").Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}
	resp := gin.H{"order_id": order.ID, "order_no": order.MerchantOrder, "paid": false}
	if isShkeeperPayType(payType) {
		resp["pay_info"] = data
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "order created", "data": resp})
}

// UserCreatePackageRenewOrder
// POST /api/v1/user/orders/package/renew
func (ctr *FinanceController) UserCreatePackageRenewOrder(c *gin.Context) {
	userID := parseUserID(mustGet(c, "userID"))
	if userID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
		return
	}
	var req struct {
		UserPackageID int64  `json:"user_package_id"`
		Period        string `json:"period"`
		Months        int    `json:"months"`
		PayType       string `json:"pay_type"`
		Remark        string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.UserPackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "user_package_id is required"})
		return
	}
	var up models.UserPackage
	if err := db.DB.Where("id = ? AND uid = ?", req.UserPackageID, userID).First(&up).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "package not found"})
		return
	}
	months := periodToMonths(req.Period, req.Months)
	if months <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid period"})
		return
	}
	amountCents, err := packageAmountByMonths(up.MonthPrice, up.QuarterPrice, up.YearPrice, months)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	payType := normalizePayType(req.PayType)
	if payType == "" {
		payType = "balance"
	}
	now := time.Now()
	merchantOrder := generateMerchantOrder("renew")
	data := map[string]interface{}{
		"user_package_id": req.UserPackageID,
		"months":          months,
		"auto_renew":      true,
	}
	order := models.Order{
		UserID:        userID,
		Type:          "renew",
		Description:   strings.TrimSpace(req.Remark),
		Data:          marshalJSON(data),
		CreatedAt:     now,
		Amount:        amountCents,
		PayType:       payType,
		MerchantOrder: merchantOrder,
		State:         "pending",
	}
	var syncIDs []int64
	if payType == "balance" {
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			var err error
			syncIDs, err = applyOrderPaidTx(tx, order.ID, "", "balance_pay", "renew by balance", userID, "user", nil)
			return err
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		syncUserPackages(syncIDs, "renew")
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "renew success",
			"data": gin.H{"order_id": order.ID, "order_no": order.MerchantOrder, "paid": true},
		})
		return
	}
	if isShkeeperPayType(payType) {
		payInfo, err := createShkeeperPayInfo(c, merchantOrder, amountCents)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "msg": err.Error()})
			return
		}
		data["channel"] = payInfo["channel"]
		data["crypto"] = payInfo["crypto"]
		data["fiat"] = payInfo["fiat"]
		data["invoice_id"] = payInfo["invoice_id"]
		data["wallet"] = payInfo["wallet"]
		data["expected_amount"] = payInfo["expected_amount"]
		data["exchange_rate"] = payInfo["exchange_rate"]
		data["display_name"] = payInfo["display_name"]
		data["status"] = payInfo["status"]
		order.Data = marshalJSON(data)
	}
	if err := db.DB.Omit("pay_at").Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}
	resp := gin.H{"order_id": order.ID, "order_no": order.MerchantOrder, "paid": false}
	if isShkeeperPayType(payType) {
		resp["pay_info"] = data
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "order created", "data": resp})
}

// ShkeeperCallback
// POST /api/v1/pay/shkeeper/callback
func (ctr *FinanceController) ShkeeperCallback(c *gin.Context) {
	settings, err := services.LoadShkeeperSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	headerKey := c.GetHeader("X-Shkeeper-Api-Key")
	if !settings.IsValidCallbackKey(headerKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid callback key"})
		return
	}

	var req services.ShkeeperCallbackPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid callback payload"})
		return
	}
	externalID := strings.TrimSpace(req.ExternalID)
	if externalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "external_id is required"})
		return
	}
	paid := req.Paid
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "PAID" || status == "OVERPAID" {
		paid = true
	}
	if !paid {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ignored: unpaid status"})
		return
	}

	var order models.Order
	if err := db.DB.Where("mch_order_no = ?", externalID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "query order failed"})
		return
	}

	transactionID := ""
	for _, tx := range req.Transactions {
		if strings.TrimSpace(tx.TxID) != "" {
			transactionID = strings.TrimSpace(tx.TxID)
			if tx.Trigger {
				break
			}
		}
	}

	var syncIDs []int64
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		syncIDs, txErr = applyOrderPaidTx(tx, order.ID, transactionID, "shkeeper_callback", "shkeeper callback paid", 0, "system", req)
		return txErr
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	syncUserPackages(syncIDs, "payment")
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// AdminMarkOrderPaid
// POST /api/v1/admin/orders/:id/mark_paid
func (ctr *FinanceController) AdminMarkOrderPaid(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if orderID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid id"})
		return
	}
	var req struct {
		TransactionID string `json:"transaction_id"`
		Reason        string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	operatorID := parseUserID(mustGet(c, "userID"))
	var syncIDs []int64
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		reason := strings.TrimSpace(req.Reason)
		if reason == "" {
			reason = "admin debug mark paid"
		}
		var txErr error
		syncIDs, txErr = applyOrderPaidTx(tx, orderID, strings.TrimSpace(req.TransactionID), "admin_mark_paid", reason, operatorID, "admin", nil)
		return txErr
	}); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "unsupported") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	syncUserPackages(syncIDs, "payment")
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

func applyOrderPaidTx(tx *gorm.DB, orderID int64, transactionID string, source string, reason string, operatorID int64, operatorRole string, callbackPayload interface{}) ([]int64, error) {
	var order models.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	if isPaidState(order.State) {
		return nil, nil
	}

	orderType := strings.ToLower(strings.TrimSpace(order.Type))
	syncIDs := make([]int64, 0, 1)
	needBalanceDebit := normalizePayType(order.PayType) == "balance"
	var orderData map[string]interface{}
	if strings.TrimSpace(order.Data) != "" {
		_ = json.Unmarshal([]byte(order.Data), &orderData)
	}
	if orderData == nil {
		orderData = map[string]interface{}{}
	}

	switch strings.ToLower(strings.TrimSpace(order.Type)) {
	case "recharge":
		if _, err := services.AdjustUserBalanceWithLedger(tx, services.BalanceAdjustInput{
			UserID:       order.UserID,
			OrderID:      order.ID,
			AmountChange: order.Amount,
			Reason:       reason,
			Source:       source,
			OperatorID:   operatorID,
			OperatorRole: operatorRole,
		}); err != nil {
			return nil, err
		}
	case "adjust":
		if _, err := services.AdjustUserBalanceWithLedger(tx, services.BalanceAdjustInput{
			UserID:       order.UserID,
			OrderID:      order.ID,
			AmountChange: order.Amount,
			Reason:       reason,
			Source:       source,
			OperatorID:   operatorID,
			OperatorRole: operatorRole,
		}); err != nil {
			return nil, err
		}
	case "purchase":
		data, err := parsePackageOrderData(order.Data)
		if err != nil {
			return nil, err
		}
		if data.PackageID <= 0 {
			return nil, errors.New("order data package_id is invalid")
		}
		if needBalanceDebit {
			if _, err := services.AdjustUserBalanceWithLedger(tx, services.BalanceAdjustInput{
				UserID:       order.UserID,
				OrderID:      order.ID,
				AmountChange: -order.Amount,
				Reason:       reason,
				Source:       source,
				OperatorID:   operatorID,
				OperatorRole: operatorRole,
			}); err != nil {
				return nil, err
			}
		}
		userPackageID, err := createUserPackageFromPlanTx(tx, order.UserID, data.PackageID, data.Months)
		if err != nil {
			return nil, err
		}
		syncIDs = append(syncIDs, userPackageID)
		orderData["package_id"] = data.PackageID
		orderData["months"] = data.Months
		orderData["auto_renew"] = data.AutoRenew
		orderData["user_package_id"] = userPackageID
	case "renew":
		data, err := parsePackageOrderData(order.Data)
		if err != nil {
			return nil, err
		}
		if data.UserPackageID <= 0 {
			return nil, errors.New("order data user_package_id is invalid")
		}
		if needBalanceDebit {
			if _, err := services.AdjustUserBalanceWithLedger(tx, services.BalanceAdjustInput{
				UserID:       order.UserID,
				OrderID:      order.ID,
				AmountChange: -order.Amount,
				Reason:       reason,
				Source:       source,
				OperatorID:   operatorID,
				OperatorRole: operatorRole,
			}); err != nil {
				return nil, err
			}
		}
		userPackageID, err := renewUserPackageTx(tx, order.UserID, data.UserPackageID, data.Months)
		if err != nil {
			return nil, err
		}
		syncIDs = append(syncIDs, userPackageID)
		orderData["user_package_id"] = userPackageID
		orderData["months"] = data.Months
		orderData["auto_renew"] = data.AutoRenew
	default:
		return nil, fmt.Errorf("unsupported order type for mark paid: %s", order.Type)
	}

	updateData := map[string]interface{}{
		"state":  "paid",
		"pay_at": time.Now(),
	}
	if strings.TrimSpace(transactionID) != "" {
		updateData["transaction_id"] = strings.TrimSpace(transactionID)
	}
	if orderType == "purchase" || orderType == "renew" {
		updateData["data"] = marshalJSON(orderData)
	}
	if callbackPayload != nil {
		orderData["callback_payload"] = callbackPayload
		updateData["data"] = marshalJSON(orderData)
	}
	if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updateData).Error; err != nil {
		return nil, err
	}
	return syncIDs, nil
}

func syncUserPackages(ids []int64, trigger string) {
	if len(ids) == 0 {
		return
	}
	seen := make(map[int64]struct{}, len(ids))
	svc := services.NewUserPackageService()
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := svc.SyncUserPackage(id, trigger); err != nil {
			fmt.Printf("[WARN] SyncUserPackage Failed, id=%d trigger=%s err=%v\n", id, trigger, err)
		}
	}
}

// ListBalanceLogs
// GET /api/v1/admin/balance_logs
func (ctr *FinanceController) ListBalanceLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)

	query := db.DB.Model(&models.BalanceLedger{})
	if userID > 0 {
		query = query.Where("uid = ?", userID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var rows []models.BalanceLedger
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}
	list := make([]balanceLedgerRow, 0, len(rows))
	for _, row := range rows {
		list = append(list, balanceLedgerRow{
			ID:           row.ID,
			UserID:       row.UserID,
			OrderID:      row.OrderID,
			Action:       row.Action,
			Source:       row.Source,
			Reason:       row.Reason,
			Before:       row.AmountBefore,
			Change:       row.AmountChange,
			After:        row.AmountAfter,
			OperatorID:   row.OperatorID,
			OperatorRole: row.OperatorRole,
			CreatedAt:    row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": total}})
}

// ListUserBalanceLogs
// GET /api/v1/user/balance_logs
func (ctr *FinanceController) ListUserBalanceLogs(c *gin.Context) {
	userID := parseUserID(mustGet(c, "userID"))
	if userID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
		return
	}
	page, pageSize := parsePagination(c)
	query := db.DB.Model(&models.BalanceLedger{}).Where("uid = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}
	var rows []models.BalanceLedger
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}
	list := make([]balanceLedgerRow, 0, len(rows))
	for _, row := range rows {
		list = append(list, balanceLedgerRow{
			ID:           row.ID,
			UserID:       row.UserID,
			OrderID:      row.OrderID,
			Action:       row.Action,
			Source:       row.Source,
			Reason:       row.Reason,
			Before:       row.AmountBefore,
			Change:       row.AmountChange,
			After:        row.AmountAfter,
			OperatorID:   row.OperatorID,
			OperatorRole: row.OperatorRole,
			CreatedAt:    row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": total}})
}
