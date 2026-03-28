package models

import "time"

// IPSwitchLog maps to the `ip_switch_log` table.
type IPSwitchLog struct {
	ID             int64      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time  `json:"create_at" gorm:"column:create_at"`
	Type           string     `json:"type" gorm:"column:type;size:30"`
	NodeGroupID    int64      `json:"node_group_id" gorm:"column:node_group_id"`
	NodeID         int64      `json:"node_id" gorm:"column:node_id"`
	LineID         int64      `json:"line_id" gorm:"column:line_id"`
	IP             string     `json:"ip" gorm:"column:ip;size:20"`
	Action         string     `json:"action" gorm:"column:action;size:20"`
	EmailNeedSend  bool       `json:"email_need_send" gorm:"column:email_need_send"`
	EmailIsSent    bool       `json:"email_is_sent" gorm:"column:email_is_sent"`
	EmailFailTimes int        `json:"email_fail_times" gorm:"column:email_fail_times"`
	EmailRet       string     `json:"email_ret" gorm:"column:email_ret;size:255"`
	EmailTime      *time.Time `json:"email_time" gorm:"column:email_time"`
	EmailSendState string     `json:"email_send_state" gorm:"column:email_send_state;size:10"`
	PhoneNeedSend  bool       `json:"phone_need_send" gorm:"column:phone_need_send"`
	PhoneIsSent    bool       `json:"phone_is_sent" gorm:"column:phone_is_sent"`
	PhoneFailTimes int        `json:"phone_fail_times" gorm:"column:phone_fail_times"`
	PhoneRet       string     `json:"phone_ret" gorm:"column:phone_ret;size:255"`
	PhoneTime      *time.Time `json:"phone_time" gorm:"column:phone_time"`
	PhoneSendState string     `json:"phone_send_state" gorm:"column:phone_send_state;size:10"`
	Content        string     `json:"content" gorm:"column:content;type:text"`
}

func (IPSwitchLog) TableName() string {
	return "ip_switch_log"
}
