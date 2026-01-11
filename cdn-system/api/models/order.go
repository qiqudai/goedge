package models

import "time"

type Order struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	UserID        int64     `json:"user_id" gorm:"column:uid;index"`
	Type          string    `json:"type"`
	Description   string    `json:"des" gorm:"column:des"`
	Data          string    `json:"data"`
	CreatedAt     time.Time `json:"create_at" gorm:"column:create_at"`
	PaidAt        time.Time `json:"pay_at" gorm:"column:pay_at"`
	Amount        int64     `json:"amount"`
	PayType       string    `json:"pay_type" gorm:"column:pay_type"`
	MerchantOrder string    `json:"mch_order_no" gorm:"column:mch_order_no"`
	TransactionID string    `json:"transaction_id" gorm:"column:transaction_id"`
	State         string    `json:"state"`
}

func (Order) TableName() string {
	return "order"
}
