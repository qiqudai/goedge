package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	packageAutoRenewInterval = 30 * time.Minute
)

// StartPackageAutoRenewWorker checks packages close to expiration and renews with balance.
func StartPackageAutoRenewWorker() {
	go func() {
		runPackageAutoRenew()
		ticker := time.NewTicker(packageAutoRenewInterval)
		defer ticker.Stop()
		for range ticker.C {
			runPackageAutoRenew()
		}
	}()
}

func runPackageAutoRenew() {
	systemCfg, err := LoadSystemConfig()
	if err != nil {
		log.Printf("[AutoRenew] load system config failed: %v", err)
		return
	}
	enabled := true
	if val, ok := systemCfg["package_auto_renew_enable"]; ok {
		enabled = ParseBoolFlag(val)
	}
	if !enabled {
		return
	}

	now := time.Now()
	var rows []models.UserPackage
	if err := db.DB.Select("id").Where("end_at > ? AND (is_expired = ? OR is_expired IS NULL)", now, false).Find(&rows).Error; err != nil {
		log.Printf("[AutoRenew] query user packages failed: %v", err)
		return
	}
	for _, row := range rows {
		processPackageAutoRenew(row.ID, now)
	}
}

func processPackageAutoRenew(userPackageID int64, now time.Time) {
	var syncID int64
	var userID int64
	var packageName string
	var newEndAt time.Time

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var up models.UserPackage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userPackageID).First(&up).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if up.EndAt.IsZero() || !up.EndAt.After(now) {
			return nil
		}

		var pkg models.Package
		if err := tx.Select("id", "before_exp_days_renew").Where("id = ?", up.PackageID).First(&pkg).Error; err != nil {
			return nil
		}
		renewDays := int(pkg.BeforeExpDaysRenew)
		if renewDays <= 0 {
			return nil
		}
		renewAt := up.EndAt.AddDate(0, 0, -renewDays)
		if now.Before(renewAt) {
			return nil
		}

		amount := calcAutoRenewAmount(up.MonthPrice, up.QuarterPrice, up.YearPrice)
		if amount <= 0 {
			return nil
		}

		orderData := map[string]interface{}{
			"user_package_id": up.ID,
			"months":          1,
			"auto_renew":      true,
			"source":          "worker",
		}

		pattern := fmt.Sprintf("\"user_package_id\":%d", up.ID)
		var pendingCount int64
		if err := tx.Model(&models.Order{}).
			Where("uid = ? AND type = ? AND state = ? AND pay_type = ?", up.UserID, "renew", "pending", "balance").
			Where("data LIKE ?", "%"+pattern+"%").
			Count(&pendingCount).Error; err == nil && pendingCount > 0 {
			return nil
		}

		order := models.Order{
			UserID:        int64(up.UserID),
			Type:          "renew",
			Description:   "auto renew by balance",
			Data:          toJSON(orderData),
			CreatedAt:     now,
			Amount:        amount,
			PayType:       "balance",
			MerchantOrder: autoRenewMerchantOrder(),
			State:         "pending",
		}
		if err := tx.Omit("pay_at").Create(&order).Error; err != nil {
			return err
		}

		if _, err := AdjustUserBalanceWithLedger(tx, BalanceAdjustInput{
			UserID:       int64(up.UserID),
			OrderID:      order.ID,
			AmountChange: -amount,
			Reason:       "auto renew by balance",
			Source:       "auto_renew",
			OperatorID:   0,
			OperatorRole: "system",
		}); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "insufficient balance") {
				return nil
			}
			return err
		}

		newEnd := up.EndAt
		if newEnd.Before(now) {
			newEnd = now
		}
		newEnd = newEnd.AddDate(0, 1, 0)
		if err := tx.Model(&models.UserPackage{}).Where("id = ?", up.ID).Updates(map[string]interface{}{
			"end_at":     newEnd,
			"is_expired": false,
		}).Error; err != nil {
			return err
		}

		orderData["renewed_end_at"] = newEnd.Format("2006-01-02 15:04:05")
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"state":  "paid",
			"pay_at": now,
			"data":   toJSON(orderData),
		}).Error; err != nil {
			return err
		}

		syncID = up.ID
		userID = int64(up.UserID)
		packageName = up.Name
		newEndAt = newEnd
		return nil
	})
	if err != nil {
		log.Printf("[AutoRenew] process package %d failed: %v", userPackageID, err)
		return
	}
	if syncID <= 0 {
		return
	}
	if err := NewUserPackageService().SyncUserPackage(syncID, "renew"); err != nil {
		log.Printf("[AutoRenew] sync package %d failed: %v", syncID, err)
	}
	title := "Package auto renewed"
	content := fmt.Sprintf("Package %s auto-renewed. New expiry: %s", packageName, newEndAt.Format("2006-01-02 15:04:05"))
	_ = CreateUserMessage(userID, "package-auto-renew", title, content, syncID, 0)
}

func calcAutoRenewAmount(monthPrice, quarterPrice, yearPrice int64) int64 {
	if monthPrice > 0 {
		return monthPrice
	}
	if quarterPrice > 0 {
		return int64(math.Round(float64(quarterPrice) / 3.0))
	}
	if yearPrice > 0 {
		return int64(math.Round(float64(yearPrice) / 12.0))
	}
	return 0
}

func autoRenewMerchantOrder() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("renew-%d-random", time.Now().Unix())
	}
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return fmt.Sprintf("renew-%s-%s", time.Now().Format("20060102150405"), string(buf))
}

func toJSON(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}
