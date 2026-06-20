package services

import (
	"fmt"
	"strconv"
	"strings"

	"cdn-api/db"
	"cdn-api/models"
)

// LoadSiteDefaultGroupMap returns the first website-group ID for each site.
func LoadSiteDefaultGroupMap(siteIDs []int64) map[int64]int64 {
	result := map[int64]int64{}
	if len(siteIDs) == 0 {
		return result
	}
	var rels []models.SiteGroupRelation
	_ = db.DB.Where("site_id IN ?", siteIDs).Order("group_id asc").Find(&rels).Error
	for _, rel := range rels {
		if _, ok := result[rel.SiteID]; !ok {
			result[rel.SiteID] = rel.GroupID
		}
	}
	return result
}

func CountSiteGroupMembers(groupID int64) (int64, error) {
	if groupID == 0 {
		return 0, nil
	}
	var count int64
	err := db.DB.Model(&models.SiteGroupRelation{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func CountSitesReferencingCertIDs(certIDs []int64) (int64, error) {
	if len(certIDs) == 0 {
		return 0, nil
	}
	var siteCount int64
	if db.DB.Migrator().HasColumn(&models.Site{}, "cert_id") {
		err := db.DB.Table("site").Where("cert_id IN ?", certIDs).Count(&siteCount).Error
		return siteCount, err
	}
	err := db.DB.Table("site").
		Where("CAST(JSON_UNQUOTE(JSON_EXTRACT(settings, '$.https.certificate_id')) AS SIGNED) IN ?", certIDs).
		Count(&siteCount).Error
	return siteCount, err
}

func CountSitesReferencingCCRuleGroup(ruleID int64) (int64, error) {
	if ruleID == 0 {
		return 0, nil
	}
	var count int64
	query := db.DB.Table("site")
	if db.DB.Migrator().HasColumn(&models.Site{}, "cc_default_rule") {
		query = query.Where("cc_default_rule = ?", ruleID)
	} else {
		query = query.Where("CAST(JSON_UNQUOTE(JSON_EXTRACT(settings, '$.security.default_rule')) AS SIGNED) = ?", ruleID)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func CountCCRulesReferencingMatcher(matcherID int64) (int64, error) {
	return countCCRuleDataFieldRef("matcher_id", matcherID)
}

func CountCCRulesReferencingFilter(filterID int64) (int64, error) {
	n1, err := countCCRuleDataFieldRef("filter1_id", filterID)
	if err != nil {
		return 0, err
	}
	n2, err := countCCRuleDataFieldRef("filter2_id", filterID)
	if err != nil {
		return 0, err
	}
	return n1 + n2, nil
}

func countCCRuleDataFieldRef(field string, id int64) (int64, error) {
	if id == 0 {
		return 0, nil
	}
	patterns := []string{
		fmt.Sprintf(`"%s":%d`, field, id),
		fmt.Sprintf(`"%s": %d`, field, id),
	}
	var count int64
	query := db.DB.Model(&models.CCRule{})
	for i, pattern := range patterns {
		like := "%" + pattern + "%"
		if i == 0 {
			query = query.Where("data LIKE ?", like)
		} else {
			query = query.Or("data LIKE ?", like)
		}
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func CountSitesReferencingACL(aclID int64) (int64, error) {
	if aclID == 0 {
		return 0, nil
	}
	idStr := strconv.FormatInt(aclID, 10)
	var count int64
	err := db.DB.Table("site").
		Where("CAST(JSON_UNQUOTE(JSON_EXTRACT(settings, '$.access.acl')) AS CHAR) = ?", idStr).
		Count(&count).Error
	return count, err
}

func CountSitesReferencingUserPackages(userPackageIDs []int64) (int64, error) {
	if len(userPackageIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := db.DB.Model(&models.Site{}).Where("user_package IN ?", userPackageIDs).Count(&count).Error
	return count, err
}

func CountUserPackagesReferencingPlan(planID int64) (int64, error) {
	if planID == 0 {
		return 0, nil
	}
	var count int64
	err := db.DB.Model(&models.UserPackage{}).Where("package = ?", planID).Count(&count).Error
	return count, err
}

func HasEnabledLines(lineIDs []int64) (bool, error) {
	if len(lineIDs) == 0 {
		return false, nil
	}
	var count int64
	err := db.DB.Model(&models.Line{}).Where("id IN ? AND enable = ?", lineIDs, true).Count(&count).Error
	return count > 0, err
}

func IsCCRuleInternal(rule models.CCRule) bool {
	return rule.Internal || rule.UserID == 0
}

func IsCCMatcherInternal(item models.CCMatch) bool {
	return item.Internal || item.UserID == 0
}

func IsCCFilterInternal(item models.CCFilter) bool {
	return item.Internal || item.UserID == 0
}

func CountSitesReferencingMatcherInSettings(matcherID int64) (int64, error) {
	return countSiteSettingsFieldRef("matcher_id", matcherID)
}

func CountSitesReferencingFilterInSettings(filterID int64) (int64, error) {
	n1, err := countSiteSettingsFieldRef("filter1_id", filterID)
	if err != nil {
		return 0, err
	}
	n2, err := countSiteSettingsFieldRef("filter2_id", filterID)
	if err != nil {
		return 0, err
	}
	return n1 + n2, nil
}

func countSiteSettingsFieldRef(field string, id int64) (int64, error) {
	if id == 0 {
		return 0, nil
	}
	patterns := []string{
		fmt.Sprintf(`"%s":%d`, field, id),
		fmt.Sprintf(`"%s": %d`, field, id),
	}
	var count int64
	query := db.DB.Table("site")
	for i, pattern := range patterns {
		like := "%" + pattern + "%"
		if i == 0 {
			query = query.Where("settings LIKE ?", like)
		} else {
			query = query.Or("settings LIKE ?", like)
		}
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func GuardCCMatcherDelete(matcherID int64) (string, error) {
	if matcherID == 0 {
		return "invalid id", nil
	}
	ruleCount, err := CountCCRulesReferencingMatcher(matcherID)
	if err != nil {
		return "", err
	}
	if ruleCount > 0 {
		return "cc_match.in_use", nil
	}
	siteCount, err := CountSitesReferencingMatcherInSettings(matcherID)
	if err != nil {
		return "", err
	}
	if siteCount > 0 {
		return "cc_match.in_use", nil
	}
	return "", nil
}

func GuardCCFilterDelete(filterID int64) (string, error) {
	if filterID == 0 {
		return "invalid id", nil
	}
	ruleCount, err := CountCCRulesReferencingFilter(filterID)
	if err != nil {
		return "", err
	}
	if ruleCount > 0 {
		return "cc_filter.in_use", nil
	}
	siteCount, err := CountSitesReferencingFilterInSettings(filterID)
	if err != nil {
		return "", err
	}
	if siteCount > 0 {
		return "cc_filter.in_use", nil
	}
	return "", nil
}

func GuardCCRuleGroupDelete(rule models.CCRule) (string, error) {
	if IsCCRuleInternal(rule) {
		return "cc_rule.system_protected", nil
	}
	if !rule.Enable {
		// disabled groups may still be referenced; always check references
	}
	count, err := CountSitesReferencingCCRuleGroup(rule.ID)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "cc_rule.in_use", nil
	}
	return "", nil
}

func GuardACLDelete(item models.ACL) (string, error) {
	if item.Enable {
		return "acl.in_use_disable_first", nil
	}
	count, err := CountSitesReferencingACL(item.ID)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "acl.in_use", nil
	}
	return "", nil
}

func GuardCCRuleGroupModify(rule models.CCRule) string {
	if IsCCRuleInternal(rule) {
		return "cc_rule.system_readonly"
	}
	return ""
}

func GuardCCMatcherModify(matcher models.CCMatch) string {
	if IsCCMatcherInternal(matcher) {
		return "cc_match.system_readonly"
	}
	return ""
}

func GuardCCFilterModify(filter models.CCFilter) string {
	if IsCCFilterInternal(filter) {
		return "cc_filter.system_readonly"
	}
	return ""
}

func IsCCRuleGroupInUse(ruleID int64) (bool, error) {
	count, err := CountSitesReferencingCCRuleGroup(ruleID)
	return count > 0, err
}

func IsCCMatcherInUse(matcherID int64) (bool, error) {
	msg, err := GuardCCMatcherDelete(matcherID)
	return msg != "", err
}

func IsCCFilterInUse(filterID int64) (bool, error) {
	msg, err := GuardCCFilterDelete(filterID)
	return msg != "", err
}

func GuardCCRuleGroupDisable(ruleID int64, wasEnabled bool, nextEnabled bool) (string, error) {
	if !wasEnabled || nextEnabled {
		return "", nil
	}
	inUse, err := IsCCRuleGroupInUse(ruleID)
	if err != nil {
		return "", err
	}
	if inUse {
		return "cc_rule.in_use_disable", nil
	}
	return "", nil
}

func GuardCCMatcherDisable(matcherID int64, wasEnabled bool, nextEnabled bool) (string, error) {
	if !wasEnabled || nextEnabled {
		return "", nil
	}
	msg, err := GuardCCMatcherDelete(matcherID)
	return msgToDisableKey(msg, "cc_match.in_use_disable"), err
}

func GuardCCFilterDisable(filterID int64, wasEnabled bool, nextEnabled bool) (string, error) {
	if !wasEnabled || nextEnabled {
		return "", nil
	}
	msg, err := GuardCCFilterDelete(filterID)
	return msgToDisableKey(msg, "cc_filter.in_use_disable"), err
}

func msgToDisableKey(msg, disableKey string) string {
	if msg != "" {
		return disableKey
	}
	return ""
}

func NormalizeGuardMessage(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return key
}
