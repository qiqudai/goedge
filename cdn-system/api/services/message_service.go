package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

func LookupMessageSubscription(userID int64, msgType string) (bool, bool, bool) {
	if userID == 0 || msgType == "" {
		return false, false, false
	}
	var sub models.MessageSub
	err := db.DB.Where("uid = ? AND msg_type = ?", userID, msgType).First(&sub).Error
	if err == nil {
		return sub.Phone, sub.Email, true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, false, false
	}
	var count int64
	if err := db.DB.Model(&models.MessageSub{}).Where("uid = ?", userID).Count(&count).Error; err != nil {
		return false, false, false
	}
	if count == 0 {
		return true, true, true
	}
	return false, false, false
}

func CreateUserMessage(userID int64, msgType, title, content string, packageID, siteID int64) error {
	phone, email, ok := LookupMessageSubscription(userID, msgType)
	if !ok {
		return nil
	}
	now := time.Now()
	msg := models.Message{
		Type:          msgType,
		Receive:       userID,
		Title:         title,
		Content:       content,
		PhoneContent:  content,
		UserPackageID: packageID,
		SiteID:        siteID,
		IsShow:        true,
		EmailNeedSend: email,
		PhoneNeedSend: phone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return db.DB.Create(&msg).Error
}
