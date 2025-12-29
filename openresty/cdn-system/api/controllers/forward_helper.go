package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseForwardCreateRequest(c *gin.Context, admin bool) (*models.Forward, int64, error) {
	var req struct {
		UserID           int64    `json:"user_id"`
		UserPackageID    int64    `json:"user_package_id"`
		GroupID          int64    `json:"group_id"`
		ListenPorts      []string `json:"listen_ports"`
		ListenPortsInput string   `json:"listen_ports_input"`
		OriginInput      string   `json:"origin_input"`
		Remark           string   `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, 0, errors.New("invalid request")
	}

	userID := req.UserID
	if !admin {
		userID = parseInt64(mustGet(c, "userID"))
	}
	if userID == 0 {
		return nil, 0, errors.New("user_id is required")
	}

	listenPorts := req.ListenPorts
	if len(listenPorts) == 0 && strings.TrimSpace(req.ListenPortsInput) != "" {
		listenPorts = splitFields(req.ListenPortsInput)
	}
	if len(listenPorts) == 0 {
		return nil, 0, errors.New("listen_ports is required")
	}

	origins := parseOrigins(req.OriginInput)
	if len(origins) == 0 {
		return nil, 0, errors.New("origin is required")
	}

	nodeGroupID, _ := resolveNodeGroupFromPackage(req.UserPackageID, 0)

	forward := &models.Forward{
		UserID:        userID,
		UserPackageID: req.UserPackageID,
		NodeGroupID:   nodeGroupID,
		ListenPorts:   listenPorts,
		Origins:       origins,
		Remark:        req.Remark,
		BackendPort:   extractBackendPort(origins),
		Enable:        true,
		State:         "running",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if forward.Cname == "" && len(listenPorts) > 0 {
		forward.Cname = listenPorts[0] + ".cdn.node.com"
	}

	defaults, err := services.GetStreamDefaultMap(userID)
	if err != nil {
		return nil, 0, err
	}
	services.ApplyForwardDefaults(forward, defaults)

	return forward, req.GroupID, nil
}

func createForwardWithGroup(forward *models.Forward, groupID int64) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(forward).Error; err != nil {
			return err
		}
		if groupID != 0 {
			rel := models.ForwardGroupRelation{ForwardID: forward.ID, GroupID: groupID}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func queryForwards(c *gin.Context) (*forwardQueryResult, error) {
	query := db.DB.Model(&models.Forward{})
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid != 0 {
			query = query.Where("uid = ?", uid)
		}
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	searchField := strings.TrimSpace(c.DefaultQuery("search_field", "all"))
	like := "%" + keyword + "%"

	if keyword != "" {
		switch searchField {
		case "forward_id":
			if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
				query = query.Where("id = ?", id)
			} else {
				return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
			}
		case "listen":
			query = query.Where("listen LIKE ?", like)
		case "origin":
			query = query.Where("backend LIKE ?", like)
		case "cname":
			query = query.Where("cname_hostname LIKE ?", like)
		case "package":
			ids, err := findUserPackageIDsByName(keyword)
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
			}
			query = query.Where("user_package IN ?", ids)
		case "group":
			ids, err := findForwardIDsByGroupName(keyword)
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
			}
			query = query.Where("id IN ?", ids)
		case "user":
			userIDs, err := findUserIDsByKeyword(keyword)
			if err != nil {
				return nil, err
			}
			if len(userIDs) == 0 {
				return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
			}
			query = query.Where("uid IN ?", userIDs)
		default:
			cond := db.DB.Where("listen LIKE ? OR backend LIKE ? OR cname_hostname LIKE ?", like, like, like)
			if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
				cond = cond.Or("id = ?", id)
			}
			if userIDs, err := findUserIDsByKeyword(keyword); err == nil && len(userIDs) > 0 {
				cond = cond.Or("uid IN ?", userIDs)
			}
			if pkgIDs, err := findUserPackageIDsByName(keyword); err == nil && len(pkgIDs) > 0 {
				cond = cond.Or("user_package IN ?", pkgIDs)
			}
			if forwardIDs, err := findForwardIDsByGroupName(keyword); err == nil && len(forwardIDs) > 0 {
				cond = cond.Or("id IN ?", forwardIDs)
			}
			query = query.Where(cond)
		}
	}

	if !isUserRequest(c) {
		if uidStr := c.Query("user_id"); uidStr != "" {
			if uid, err := strconv.Atoi(uidStr); err == nil {
				query = query.Where("uid = ?", uid)
			}
		}
	}
	if pkgStr := c.Query("user_package_id"); pkgStr != "" {
		if id, err := strconv.Atoi(pkgStr); err == nil {
			query = query.Where("user_package = ?", id)
		}
	}
	if groupStr := c.Query("group_id"); groupStr != "" {
		if id, err := strconv.Atoi(groupStr); err == nil {
			ids, err := findForwardIDsByGroupID(int64(id))
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return &forwardQueryResult{Forwards: []models.Forward{}, Total: 0}, nil
			}
			query = query.Where("id IN ?", ids)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var forwards []models.Forward
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&forwards).Error; err != nil {
		return nil, err
	}

	return &forwardQueryResult{Forwards: forwards, Total: total}, nil
}

func buildForwardListItems(forwards []models.Forward) ([]forwardListItem, error) {
	userMap, err := loadUsersForForward(forwards)
	if err != nil {
		return nil, err
	}
	pkgMap, err := loadUserPackagesForForward(forwards)
	if err != nil {
		return nil, err
	}
	groupMap, relMap, err := loadForwardGroups(forwards)
	if err != nil {
		return nil, err
	}
	nodeGroupMap, err := loadNodeGroupsForForward(forwards)
	if err != nil {
		return nil, err
	}

	items := make([]forwardListItem, 0, len(forwards))
	for _, forward := range forwards {
		originDisplay := ""
		if len(forward.Origins) > 0 {
			parts := make([]string, 0, len(forward.Origins))
			for _, o := range forward.Origins {
				parts = append(parts, o.Address)
			}
			originDisplay = strings.Join(parts, ",")
		} else if forward.OriginsRaw != "" {
			originDisplay = forward.OriginsRaw
		}

		cname := strings.TrimSpace(forward.Cname)
		if cname == "" {
			cname = "-"
		}

		item := forwardListItem{
			ID:              forward.ID,
			UserID:          forward.UserID,
			UserName:        userMap[forward.UserID],
			ListenPorts:     strings.Join(forward.ListenPorts, " "),
			OriginDisplay:   originDisplay,
			UserPackageID:   forward.UserPackageID,
			UserPackageName: pkgMap[forward.UserPackageID],
			GroupID:         relMap[forward.ID],
			GroupName:       groupMap[relMap[forward.ID]],
			NodeGroupID:     forward.NodeGroupID,
			NodeGroupName:   nodeGroupMap[forward.NodeGroupID],
			CNAME:           cname,
			Status:          forward.Enable,
			Remark:          forward.Remark,
			CreatedAt:       forward.CreatedAt,
		}
		items = append(items, item)
	}
	return items, nil
}
