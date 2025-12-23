package models

import "time"

type Message struct {
    ID            int64     `json:"id" gorm:"primaryKey"`
    Type          string    `json:"type"`
    PubUser       int64     `json:"pub_user" gorm:"column:pub_user"`
    Receive       int64     `json:"receive"`
    Title         string    `json:"title"`
    Content       string    `json:"content"`
    PhoneContent  string    `json:"phone_content"`
    EventID       string    `json:"event_id" gorm:"column:event_id"`
    UserPackageID int64     `json:"user_package_id" gorm:"column:user_package_id"`
    SiteID        int64     `json:"site_id" gorm:"column:site_id"`
    IsShow        bool      `json:"is_show" gorm:"column:is_show"`
    IsRed         bool      `json:"is_red" gorm:"column:is_red"`
    IsBold        bool      `json:"is_bold" gorm:"column:is_bold"`
    IsExternal    bool      `json:"is_external" gorm:"column:is_external"`
    IsPopup       bool      `json:"is_popup" gorm:"column:is_popup"`
    EmailNeedSend bool      `json:"email_need_send" gorm:"column:email_need_send"`
    PhoneNeedSend bool      `json:"phone_need_send" gorm:"column:phone_need_send"`
    EmailIsSent   bool      `json:"email_is_sent" gorm:"column:email_is_sent"`
    PhoneIsSent   bool      `json:"phone_is_sent" gorm:"column:phone_is_sent"`
    URL           string    `json:"url" gorm:"column:url"`
    Sort          int       `json:"sort"`
    CreatedAt     time.Time `json:"create_at" gorm:"column:create_at"`
    UpdatedAt     time.Time `json:"update_at" gorm:"column:update_at"`
}

func (Message) TableName() string {
    return "message"
}

type MessageRead struct {
    UserID   int64     `json:"user_id" gorm:"column:uid"`
    MessageID int64    `json:"msg_id" gorm:"column:msg_id"`
    CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
}

func (MessageRead) TableName() string {
    return "message_read"
}

type MessageSub struct {
    UserID  int64  `json:"user_id" gorm:"column:uid"`
    MsgType string `json:"msg_type" gorm:"column:msg_type"`
    Phone   bool   `json:"phone"`
    Email   bool   `json:"email"`
}

func (MessageSub) TableName() string {
    return "message_sub"
}
