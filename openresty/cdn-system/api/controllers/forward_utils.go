package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"strings"
)

func parseForwardBatchLine(line string) ([]string, []models.ForwardOrigin, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, nil, errors.New("invalid batch data format")
	}
	listenPorts := splitFields(parts[0])
	if len(listenPorts) == 0 {
		return nil, nil, errors.New("listen ports required")
	}
	origins := parseOrigins(parts[1])
	if len(origins) == 0 {
		return nil, nil, errors.New("origin required")
	}
	return listenPorts, origins, nil
}

func parseOrigins(input string) []models.ForwardOrigin {
	origins := []models.ForwardOrigin{}
	for _, item := range splitFields(input) {
		origins = append(origins, models.ForwardOrigin{Address: item, Weight: 1, Enable: true})
	}
	return origins
}

func loadUsersForForward(items []models.Forward) (map[int64]string, error) {
	ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.UserID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var users []models.User
	if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		result[u.ID] = u.Name
	}
	return result, nil
}

func loadUserPackagesForForward(items []models.Forward) (map[int64]string, error) {
	ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.UserPackageID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var pkgs []models.UserPackage
	if err := db.DB.Where("id IN ?", ids).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		result[p.ID] = p.Name
	}
	return result, nil
}

func loadForwardGroups(items []models.Forward) (map[int64]string, map[int64]int64, error) {
	ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.ID })
	groupMap := map[int64]string{}
	relMap := map[int64]int64{}
	if len(ids) == 0 {
		return groupMap, relMap, nil
	}
	var relations []models.ForwardGroupRelation
	if err := db.DB.Where("stream_id IN ?", ids).Find(&relations).Error; err != nil {
		return nil, nil, err
	}
	groupIDs := make([]int64, 0, len(relations))
	for _, rel := range relations {
		relMap[rel.ForwardID] = rel.GroupID
		groupIDs = append(groupIDs, rel.GroupID)
	}
	if len(groupIDs) == 0 {
		return groupMap, relMap, nil
	}
	var groups []models.ForwardGroup
	if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}
	return groupMap, relMap, nil
}

func loadNodeGroupsForForward(items []models.Forward) (map[int64]string, error) {
	ids := uniqueIDsForward(items, func(f models.Forward) int64 { return f.NodeGroupID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var groups []models.NodeGroup
	if err := db.DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, g := range groups {
		result[g.ID] = g.Name
	}
	return result, nil
}

func uniqueIDsForward(items []models.Forward, fn func(models.Forward) int64) []int64 {
	seen := map[int64]struct{}{}
	for _, item := range items {
		id := fn(item)
		if id != 0 {
			seen[id] = struct{}{}
		}
	}
	result := make([]int64, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result
}

func findForwardIDsByGroupName(keyword string) ([]int64, error) {
	var groups []models.ForwardGroup
	if err := db.DB.Where("name LIKE ?", "%"+keyword+"%").Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []int64{}, nil
	}
	ids := make([]int64, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	return findForwardIDsByGroupIDs(ids)
}

func findForwardIDsByGroupID(groupID int64) ([]int64, error) {
	return findForwardIDsByGroupIDs([]int64{groupID})
}

func findForwardIDsByGroupIDs(groupIDs []int64) ([]int64, error) {
	if len(groupIDs) == 0 {
		return []int64{}, nil
	}
	var relations []models.ForwardGroupRelation
	if err := db.DB.Where("group_id IN ?", groupIDs).Find(&relations).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(relations))
	for _, rel := range relations {
		ids = append(ids, rel.ForwardID)
	}
	return ids, nil
}

func filterForwardIDsForUser(ids []int64, userID int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	var allowed []int64
	if err := db.DB.Model(&models.Forward{}).Where("uid = ? AND id IN ?", userID, ids).Pluck("id", &allowed).Error; err != nil {
		return nil, err
	}
	return allowed, nil
}
