package services

import (
	"cdn-api/models"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BalanceAdjustInput struct {
	UserID       int64
	OrderID      int64
	AmountChange int64 // cents; positive=credit, negative=debit
	Reason       string
	Source       string
	OperatorID   int64
	OperatorRole string
}

type BalanceAdjustResult struct {
	Before int64
	After  int64
}

// AdjustUserBalanceWithLedger updates user balance atomically and writes one ledger row.
func AdjustUserBalanceWithLedger(tx *gorm.DB, in BalanceAdjustInput) (*BalanceAdjustResult, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	if in.UserID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	if in.AmountChange == 0 {
		return nil, errors.New("amount_change must not be zero")
	}

	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", in.UserID).
		First(&user).Error; err != nil {
		return nil, err
	}

	before := user.Balance
	after := before + in.AmountChange
	if after < 0 {
		return nil, errors.New("insufficient balance")
	}

	if err := tx.Model(&models.User{}).
		Where("id = ?", in.UserID).
		Update("balance", after).Error; err != nil {
		return nil, err
	}

	action := "credit"
	if in.AmountChange < 0 {
		action = "debit"
	}

	ledger := models.BalanceLedger{
		UserID:       in.UserID,
		OrderID:      in.OrderID,
		AmountBefore: before,
		AmountChange: in.AmountChange,
		AmountAfter:  after,
		Action:       action,
		Source:       in.Source,
		Reason:       in.Reason,
		OperatorID:   in.OperatorID,
		OperatorRole: in.OperatorRole,
		CreatedAt:    time.Now(),
	}
	if err := tx.Create(&ledger).Error; err != nil {
		return nil, err
	}

	return &BalanceAdjustResult{Before: before, After: after}, nil
}

