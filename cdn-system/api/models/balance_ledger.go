package models

import "time"

// BalanceLedger records every balance mutation for audit and troubleshooting.
type BalanceLedger struct {
	ID int64 `json:"id" gorm:"primaryKey"`

	UserID int64 `json:"user_id" gorm:"column:uid;index"`
	OrderID int64 `json:"order_id" gorm:"column:order_id;index"`

	AmountBefore int64 `json:"amount_before" gorm:"column:amount_before"`
	AmountChange int64 `json:"amount_change" gorm:"column:amount_change"`
	AmountAfter  int64 `json:"amount_after" gorm:"column:amount_after"`

	Action string `json:"action" gorm:"column:action;type:varchar(20);index"` // credit/debit
	Source string `json:"source" gorm:"column:source;type:varchar(40);index"` // onchain/admin/manual/auto_renew/debug
	Reason string `json:"reason" gorm:"column:reason;type:text"`

	OperatorID   int64  `json:"operator_id" gorm:"column:operator_id"`
	OperatorRole string `json:"operator_role" gorm:"column:operator_role;type:varchar(20)"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at;index"`
}

func (BalanceLedger) TableName() string {
	return "balance_ledger"
}

