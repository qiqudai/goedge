package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageController struct{}

var defaultMsgTypes = []string{
	"package-expire",
	"traffic-exceed",
	"connection-exceed",
	"bandwidth-exceed",
	"cc-switch",
	"cert-expire",
	"refresh_url",
	"refresh_dir",
	"preheat",
}

type messageRow struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	TypeLabel string `json:"type_label"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Phone     string `json:"phone"`
	SiteID    int64  `json:"site_id"`
	CreatedAt string `json:"created_at"`
	IsRead    bool   `json:"is_read"`
}

func typeLabel(t string) string {
	switch t {
	case "package-expire":
		return i18n.T("message.package_expire")
	case "traffic-exceed":
		return i18n.T("message.traffic_exceed")
	case "connection-exceed":
		return i18n.T("message.conn_exceed")
	case "bandwidth-exceed":
		return i18n.T("message.bandwidth_exceed")
	case "cc-switch":
		return i18n.T("message.rule_switch")
	case "cert-expire":
		return i18n.T("message.cert_expire")
	case "refresh_url":
		return i18n.T("message.refresh_url")
	case "refresh_dir":
		return i18n.T("message.refresh_dir")
	case "preheat":
		return i18n.T("message.preheat")
	default:
		return i18n.T("message.other")
	}
}

func normalizeMessageTitle(title, msgType string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return typeLabel(msgType)
	}
	for _, r := range trimmed {
		if r > 127 {
			return trimmed
		}
	}
	return typeLabel(msgType)
}

// AdminList
// GET /api/v1/admin/messages
func (ctr *MessageController) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	msgType := strings.TrimSpace(c.Query("type"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	query := db.DB.Model(&models.Message{})
	if msgType != "" {
		query = query.Where("type = ?", msgType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
		if siteID, err := strconv.Atoi(keyword); err == nil {
			query = query.Or("site_id = ?", siteID)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	var msgs []models.Message
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	list := make([]messageRow, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, messageRow{
			ID:        m.ID,
			Type:      m.Type,
			TypeLabel: typeLabel(m.Type),
			Title:     normalizeMessageTitle(m.Title, m.Type),
			Content:   m.Content,
			Phone:     m.PhoneContent,
			SiteID:    m.SiteID,
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
			IsRead:    false,
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

// UserList
// GET /api/v1/user/messages
func (ctr *MessageController) UserList(c *gin.Context) {
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

	msgType := strings.TrimSpace(c.Query("type"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	query := db.DB.Model(&models.Message{}).Where("receive = ?", userID)
	if msgType != "" {
		query = query.Where("type = ?", msgType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
		if siteID, err := strconv.Atoi(keyword); err == nil {
			query = query.Or("site_id = ?", siteID)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	var msgs []models.Message
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	readSet := map[int64]bool{}
	var reads []models.MessageRead
	_ = db.DB.Where("uid = ?", userID).Find(&reads).Error
	for _, r := range reads {
		readSet[r.MessageID] = true
	}

	list := make([]messageRow, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, messageRow{
			ID:        m.ID,
			Type:      m.Type,
			TypeLabel: typeLabel(m.Type),
			Title:     normalizeMessageTitle(m.Title, m.Type),
			Content:   m.Content,
			Phone:     m.PhoneContent,
			SiteID:    m.SiteID,
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
			IsRead:    readSet[m.ID],
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

// UserUnread
// GET /api/v1/user/messages/unread
func (ctr *MessageController) UserUnread(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid user")})
		return
	}

	var total int64
	baseQuery := db.DB.Table("message as m").
		Joins("left join message_read r on r.msg_id = m.id and r.uid = ?", userID).
		Where("m.receive = ? AND r.msg_id IS NULL", userID)
	if err := baseQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	var latest messageRow
	if total > 0 {
		var msg struct {
			ID        int64     `gorm:"column:id"`
			Type      string    `gorm:"column:type"`
			Title     string    `gorm:"column:title"`
			Content   string    `gorm:"column:content"`
			Phone     string    `gorm:"column:phone_content"`
			SiteID    int64     `gorm:"column:site_id"`
			CreatedAt time.Time `gorm:"column:create_at"`
		}
		latestQuery := db.DB.Table("message as m").
			Joins("left join message_read r on r.msg_id = m.id and r.uid = ?", userID).
			Where("m.receive = ? AND r.msg_id IS NULL", userID)
		if err := latestQuery.Select("m.id, m.type, m.title, m.content, m.phone_content, m.site_id, m.create_at").
			Order("m.id desc").Limit(1).Scan(&msg).Error; err == nil && msg.ID > 0 {
			latest = messageRow{
				ID:        msg.ID,
				Type:      msg.Type,
				TypeLabel: typeLabel(msg.Type),
				Title:     normalizeMessageTitle(msg.Title, msg.Type),
				Content:   msg.Content,
				Phone:     msg.Phone,
				SiteID:    msg.SiteID,
				CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
				IsRead:    false,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"count":  total,
			"latest": latest,
		},
	})
}

// MarkRead
// POST /api/v1/user/messages/:id/read
func (ctr *MessageController) MarkRead(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	record := models.MessageRead{
		UserID:    userID,
		MessageID: id,
		CreatedAt: time.Now(),
	}
	_ = db.DB.Create(&record).Error

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("ok")})
}

// GetSubscriptions
// GET /api/v1/user/message_sub
func (ctr *MessageController) GetSubscriptions(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var subs []models.MessageSub
	_ = db.DB.Where("uid = ?", userID).Find(&subs).Error

	list := make([]gin.H, 0, len(subs))
	if len(subs) == 0 {
		for _, t := range defaultMsgTypes {
			list = append(list, gin.H{
				"msg_type": t,
				"name":     typeLabel(t),
				"phone":    true,
				"email":    true,
			})
		}
	} else {
		for _, s := range subs {
			list = append(list, gin.H{
				"msg_type": s.MsgType,
				"name":     typeLabel(s.MsgType),
				"phone":    s.Phone,
				"email":    s.Email,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

// UpdateSubscriptions
// PUT /api/v1/user/message_sub
func (ctr *MessageController) UpdateSubscriptions(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var req struct {
		List []models.MessageSub `json:"list"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid JSON")})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("uid = ?", userID).Delete(&models.MessageSub{}).Error; err != nil {
			return err
		}
		for _, item := range req.List {
			item.UserID = userID
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}
