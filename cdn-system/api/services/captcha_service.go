package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"strings"
	"time"

	"crypto/rand"
)

const loginCaptchaTTL = 5 * time.Minute

func GenerateCaptchaCode(length int) string {
	if length <= 0 {
		length = 6
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		now := time.Now().UnixNano()
		for i := 0; i < length; i++ {
			b[i] = byte('0' + (now % 10))
			now /= 10
			if now == 0 {
				now = time.Now().UnixNano()
			}
		}
		return string(b)
	}
	for i := 0; i < length; i++ {
		b[i] = byte('0' + (b[i] % 10))
	}
	return string(b)
}

func StoreCaptcha(email, phone, ip, code string) error {
	record := models.Captcha{
		Email:     strings.TrimSpace(email),
		Phone:     strings.TrimSpace(phone),
		Code:      strings.TrimSpace(code),
		IP:        strings.TrimSpace(ip),
		CreatedAt: time.Now(),
	}
	return db.DB.Create(&record).Error
}

func VerifyCaptcha(email, phone, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	deadline := time.Now().Add(-loginCaptchaTTL)
	query := db.DB.Model(&models.Captcha{}).Where("captcha = ? AND create_at >= ?", code, deadline)
	if email = strings.TrimSpace(email); email != "" {
		query = query.Where("email = ?", email)
	} else if phone = strings.TrimSpace(phone); phone != "" {
		query = query.Where("phone = ?", phone)
	} else {
		return false
	}
	var count int64
	if err := query.Count(&count).Error; err != nil || count == 0 {
		return false
	}
	_ = query.Delete(&models.Captcha{}).Error
	return true
}
