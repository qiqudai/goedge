package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/acme"
)

type IssueCertItem struct {
	CertID  int64    `json:"cert_id"`
	Domains []string `json:"domains"`
}

type IssueCertTaskPayload struct {
	CA       string          `json:"ca"`
	CADirURL string          `json:"ca_dir_url"`
	Email    string          `json:"email"`
	Items    []IssueCertItem `json:"items"`
}

type issueTaskMeta struct {
	TargetNodeID int64 `json:"target_node_id"`
}

const maxCertIssueAttempts = 3

// IssueCertsAsync creates tasks and starts processing
func IssueCertsAsync(batchID int64, ids []int64) {
	if len(ids) == 0 {
		return
	}
	go func() {
		certs, err := loadCertsForIssue(ids)
		if err != nil {
			log.Printf("[CertIssue] load certs failed: %v", err)
			return
		}
		if len(certs) == 0 {
			return
		}

		localCerts := make([]models.Cert, 0, len(certs))
		nodeCerts := make([]models.Cert, 0, len(certs))
		for _, cert := range certs {
			if requiresDNSChallenge(cert) {
				localCerts = append(localCerts, cert)
			} else {
				nodeCerts = append(nodeCerts, cert)
			}
		}

		if len(nodeCerts) > 0 {
			if err := dispatchCertsToNodes(batchID, nodeCerts); err != nil {
				log.Printf("[CertIssue] dispatch to nodes failed: %v", err)
				markCertsIssueFailed(nodeCerts, err.Error())
			}
		}

		for _, cert := range localCerts {
			certID := int64(cert.ID)
			go processUniqueIssueTask(batchID, certID)
		}
	}()
}

func processUniqueIssueTask(batchID int64, certID int64) {
	var task models.Task
	var cert models.Cert
	log.Printf("[CertIssue] start batch=%d cert_id=%d", batchID, certID)

	defer func() {
		if r := recover(); r != nil {
			errReason := fmt.Sprintf("Panic: %v", r)
			if task.ID != 0 {
				failTask(&task, errReason)
			}
			// Update cert state if cert found
			if cert.ID != 0 {
				db.DB.Model(&models.Cert{ID: cert.ID}).Update("state", "fail")
			}
		}
	}()

	// 1. Load Cert
	if err := db.DB.First(&cert, certID).Error; err != nil {
		log.Printf("[CertIssue] cert not found cert_id=%d err=%v", certID, err)
		return
	}
	if isManualUploadCert(cert.Type) {
		log.Printf("[CertIssue] skip manual-upload cert cert_id=%d", certID)
		return
	}

	// 2. Create Task with payload data
	payload, payloadErr := buildIssuePayload(cert)
	if payloadErr != nil {
		log.Printf("[CertIssue] build payload failed cert_id=%d err=%v", certID, payloadErr)
		markCertIssueFailed(certID, payloadErr.Error())
		return
	}
	payloadRaw, _ := json.Marshal(payload)
	task = models.Task{
		Type:     "issue_cert",
		Name:     "Issue Cert " + strings.TrimSpace(strconv.FormatInt(certID, 10)),
		State:    "waiting",
		PID:      batchID,
		Enable:   true,
		Data:     string(payloadRaw),
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		log.Printf("[CertIssue] create task failed cert_id=%d err=%v", certID, err)
		markCertIssueFailed(certID, err.Error())
		return
	}

	// 3. Associate with Cert
	db.DB.Model(&cert).Updates(map[string]interface{}{
		"issue_task_id": task.ID,
		"task_id":       task.ID,
		"state":         "waiting",
		"ret":           "",
	})

	// 4. Issue with retry
	started := false
	for {
		if err := db.DB.First(&cert, certID).Error; err != nil {
			failTask(&task, "Cert not found")
			return
		}
		if isManualUploadCert(cert.Type) {
			markIssueTaskStoppedByManual(task.ID)
			log.Printf("[CertIssue] stop task=%d cert_id=%d because cert switched to upload", task.ID, certID)
			return
		}
		if !started {
			now := time.Now()
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":    "running",
				"start_at": &now,
			})
			db.DB.Model(&models.Cert{ID: cert.ID}).Update("state", "issuing")
			started = true
		} else {
			db.DB.Model(&task).Update("state", "running")
		}

		if err := issueCertLocal(cert); err != nil {
			errMsg := err.Error()
			log.Printf("[CertIssue] issue failed cert_id=%d err=%v", certID, err)
			db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
				"state": "fail",
				"ret":   errMsg,
			})
			if isFatalIssueError(err) {
				failTask(&task, errMsg)
				db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
					"state": "fail",
					"ret":   errMsg,
				})
				return
			}

			delay := nextCertRetryDelay(task.ErrTimes)
			retryAt := time.Now().Add(delay)
			task.ErrTimes++
			if task.ErrTimes >= maxCertIssueAttempts {
				failTask(&task, errMsg)
				db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
					"state": "fail",
					"ret":   errMsg,
				})
				return
			}
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":     "retrying",
				"ret":       errMsg,
				"retry_at":  retryAt,
				"err_times": task.ErrTimes,
			})
			time.Sleep(delay)
			continue
		}

		// Success
		completeTask(&task)
		log.Printf("[CertIssue] success cert_id=%d", certID)
		return
	}
}

func isManualUploadCert(certType string) bool {
	return strings.EqualFold(strings.TrimSpace(certType), "upload")
}

func markIssueTaskStoppedByManual(taskID int64) {
	if taskID == 0 {
		return
	}
	_ = db.DB.Model(&models.Task{}).
		Where("id = ? AND type = ?", taskID, "issue_cert").
		Updates(map[string]interface{}{
			"enable":   false,
			"state":    "fail",
			"retry_at": nil,
			"ret":      "manual upload overrides auto-issue",
			"end_at":   time.Now(),
		}).Error
}

func failTask(task *models.Task, reason string) {
	db.DB.Model(task).Updates(map[string]interface{}{
		"state":  "fail",
		"ret":    reason,
		"end_at": time.Now(),
	})
}

func completeTask(task *models.Task) {
	db.DB.Model(task).Updates(map[string]interface{}{
		"state":  "success",
		"ret":    "",
		"end_at": time.Now(),
	})
}

func issueCertLocal(cert models.Cert) error {
	domains := splitCertDomains(cert.Domain)
	if len(domains) == 0 {
		return errors.New("cert domain is empty")
	}
	candidates := buildIssueCAFallbackList(cert.Type)
	attemptLogs := make([]string, 0, len(candidates))
	var provider acme.ChallengeProvider
	var providerErr error
	if requiresDNSChallenge(cert) {
		provider, providerErr = BuildDNSChallengeProvider(cert)
		if providerErr != nil {
			return providerErr
		}
	}

	for idx, ca := range candidates {
		var issuer *acme.Issuer
		if provider != nil {
			issuer = NewDNS01Issuer(ca, provider)
		} else {
			issuer = NewHTTP01Issuer(ca)
		}

		result, err := issuer.Issue(domains)
		if err != nil {
			attemptLogs = append(attemptLogs, fmt.Sprintf("ca=%s failed: %s", ca, err.Error()))
			if idx+1 < len(candidates) {
				attemptLogs = append(attemptLogs, fmt.Sprintf("switch_ca: %s -> %s", ca, candidates[idx+1]))
			}
			continue
		}

		if ca != normalizeIssueCA(cert.Type) {
			log.Printf("[CertIssue] cert_id=%d switched ca %s -> %s", cert.ID, normalizeIssueCA(cert.Type), ca)
		}
		return UpdateIssuedCert(int64(cert.ID), result.CertPEM, result.KeyPEM, result.NotBefore, result.NotAfter, 0)
	}

	return errors.New(strings.Join(attemptLogs, " | "))
}

func splitCertDomains(raw string) []string {
	raw = strings.NewReplacer(";", ",", "\n", ",", "\r", ",", "\t", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		item := strings.TrimSpace(strings.ToLower(part))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func requiresDNSChallenge(cert models.Cert) bool {
	if cert.DNSAPI != nil && *cert.DNSAPI > 0 {
		return true
	}
	return hasWildcardDomain(splitCertDomains(cert.Domain))
}

func hasWildcardDomain(domains []string) bool {
	for _, domain := range domains {
		for _, item := range splitCertDomains(domain) {
			if strings.HasPrefix(strings.TrimSpace(item), "*.") {
				return true
			}
		}
	}
	return false
}

func UpdateIssuedCert(certID int64, certPEM string, keyPEM string, notBefore time.Time, notAfter time.Time, issueTaskID int64) error {
	updates := map[string]interface{}{
		"cert":        certPEM,
		"key":         keyPEM,
		"start_time":  notBefore,
		"expire_time": notAfter,
		"enable":      true,
		"state":       "ready",
		"ret":         "",
	}
	var current models.Cert
	if err := db.DB.Select("id", "type", "auto_renew").Where("id = ?", certID).First(&current).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(current.Type)) != "upload" && !current.AutoRenew {
			updates["auto_renew"] = true
		}
	}
	if issueTaskID != 0 {
		updates["issue_task_id"] = issueTaskID
	} else {
		updates["issue_task_id"] = nil
	}
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(updates).Error; err != nil {
		return err
	}
	ActivatePendingHTTPSForCert(certID, certPEM)
	BumpConfigVersion("cert", []int64{certID})
	return nil
}

func ActivatePendingHTTPSForCert(certID int64, certPEM string) {
	if certID == 0 {
		return
	}
	var sites []models.Site
	if err := db.DB.Find(&sites).Error; err != nil {
		log.Printf("[CertIssue] load pending https sites failed cert_id=%d err=%v", certID, err)
		return
	}
	changedSiteIDs := make([]int64, 0)
	for _, site := range sites {
		httpsCfg := getMap(site.Settings, "https")
		if httpsCfg == nil {
			continue
		}
		pendingID := int64(parseIntValue(httpsCfg["pending_certificate_id"], 0))
		if pendingID != certID {
			continue
		}
		settings := site.Settings
		if settings == nil {
			settings = map[string]interface{}{}
		}
		httpsCfg = getMap(settings, "https")
		if httpsCfg == nil {
			httpsCfg = map[string]interface{}{}
			settings["https"] = httpsCfg
		}
		errText := ""
		for _, domain := range site.Domains {
			if result := CertificateCoversDomain(certPEM, domain); !result.OK {
				errText = FormatCertCoverageError(normalizeDomainHostForEdge(domain), result)
				break
			}
		}
		if errText != "" {
			httpsCfg["enable"] = false
			httpsCfg["state"] = "failed"
			httpsCfg["last_error"] = errText
		} else {
			httpsCfg["enable"] = true
			httpsCfg["state"] = "probing"
			httpsCfg["certificate_id"] = certID
			httpsCfg["active_certificate_id"] = 0
			httpsCfg["pending_certificate_id"] = certID
			httpsCfg["last_error"] = ""
			httpsCfg["probe_at"] = time.Now().Format(time.RFC3339)
		}
		raw, _ := json.Marshal(settings)
		updates := map[string]interface{}{
			"settings":  string(raw),
			"update_at": time.Now(),
		}
		if errText == "" {
			if db.DB.Migrator().HasColumn(&models.Site{}, "cert_id") {
				updates["cert_id"] = certID
			}
			if len(site.HttpsListen) == 0 && strings.TrimSpace(site.HttpsListenRaw) == "" {
				updates["https_listen"] = "[\"443\"]"
			}
		}
		if err := db.DB.Model(&models.Site{}).Where("id = ?", site.ID).Updates(updates).Error; err != nil {
			log.Printf("[CertIssue] update pending https site failed site_id=%d cert_id=%d err=%v", site.ID, certID, err)
			continue
		}
		changedSiteIDs = append(changedSiteIDs, site.ID)
	}
	if len(changedSiteIDs) > 0 {
		BumpConfigVersion("site", changedSiteIDs)
		CreateHTTPSProbeTasksForSites(changedSiteIDs, certID)
	}
}

func ParseCertTimes(certPEM string) (time.Time, time.Time, error) {
	return acme.ParseCertTimes(certPEM)
}

func buildIssuePayload(cert models.Cert) (IssueCertTaskPayload, error) {
	email := strings.TrimSpace(config.App.AcmeEmail)
	if email == "" {
		return IssueCertTaskPayload{}, errors.New("acme_email is required")
	}
	domains := splitCertDomains(cert.Domain)
	if len(domains) == 0 {
		return IssueCertTaskPayload{}, errors.New("cert domain is empty")
	}
	ca := normalizeIssueCA(cert.Type)
	return IssueCertTaskPayload{
		CA:       ca,
		CADirURL: BuildCADirURL(ca),
		Email:    email,
		Items: []IssueCertItem{
			{
				CertID:  int64(cert.ID),
				Domains: domains,
			},
		},
	}, nil
}

func dispatchCertsToNodes(batchID int64, certs []models.Cert) error {
	email := strings.TrimSpace(config.App.AcmeEmail)
	if email == "" {
		return errors.New("acme_email is required")
	}

	for _, cert := range certs {
		domains := splitCertDomains(cert.Domain)
		if len(domains) == 0 {
			markCertIssueFailed(int64(cert.ID), "cert domain is empty")
			continue
		}
		nodes, scoped, err := loadIssueCandidateNodesForCert(cert, domains)
		if err != nil {
			markCertIssueFailed(int64(cert.ID), err.Error())
			continue
		}
		if len(nodes) == 0 {
			reason := "no available nodes for cert issue"
			if scoped {
				reason = "no available package nodes for cert domains"
			}
			markCertIssueFailed(int64(cert.ID), reason)
			continue
		}
		if scoped {
			if err := validateCertHTTP01DNSForDomains(int64(cert.UserID), domains); err != nil {
				markCertIssueFailed(int64(cert.ID), err.Error())
				continue
			}
		}
		ca := strings.ToLower(strings.TrimSpace(cert.Type))
		ca = normalizeIssueCA(ca)
		payload := IssueCertTaskPayload{
			CA:       ca,
			CADirURL: BuildCADirURL(ca),
			Email:    email,
			Items: []IssueCertItem{
				{
					CertID:  int64(cert.ID),
					Domains: domains,
				},
			},
		}
		target := nodes[rand.Intn(len(nodes))]

		taskName := fmt.Sprintf("Issue Cert %d", cert.ID)
		task, err := createIssueTask(batchID, target.ID, taskName, payload)
		if err != nil {
			markCertIssueFailed(int64(cert.ID), err.Error())
			continue
		}
		if err := db.DB.Model(&models.Cert{}).Where("id = ?", cert.ID).Updates(map[string]interface{}{
			"issue_task_id": task.ID,
			"task_id":       task.ID,
			"state":         "waiting",
			"ret":           "",
		}).Error; err != nil {
			log.Printf("[CertIssue] update cert task id failed cert_id=%d err=%v", cert.ID, err)
		}
	}
	return nil
}

func loadIssueCandidateNodesForCert(cert models.Cert, domains []string) ([]models.Node, bool, error) {
	nodeIDs, scoped, err := resolveCertIssueTaskNodeIDs(int64(cert.UserID), domains)
	if err != nil {
		return nil, scoped, err
	}
	if !scoped {
		nodes, err := loadAvailableNodes()
		return nodes, false, err
	}
	nodes, err := loadAvailableNodesByIDs(nodeIDs)
	return nodes, true, err
}

func loadIssueSitesForUser(userID int64) ([]models.Site, error) {
	var sites []models.Site
	query := db.DB.Model(&models.Site{})
	if userID > 0 {
		query = query.Where("uid = ?", userID)
	}
	if err := query.Find(&sites).Error; err != nil {
		return nil, err
	}
	return sites, nil
}

func resolveCertIssueTaskNodeIDs(userID int64, domains []string) ([]int64, bool, error) {
	normalizedDomains := normalizeIssueDomains(domains)
	if len(normalizedDomains) == 0 {
		return nil, false, nil
	}
	sites, err := loadIssueSitesForUser(userID)
	if err != nil {
		return nil, false, err
	}

	scoped := false
	missingDomains := make([]string, 0)
	var common []int64
	for _, domain := range normalizedDomains {
		matched := make([]models.Site, 0)
		for _, site := range sites {
			if siteMatchesCertDomain(site, domain) {
				matched = append(matched, site)
			}
		}
		if len(matched) == 0 {
			missingDomains = append(missingDomains, domain)
			continue
		}
		scoped = true
		groupIDs, err := collectIssueNodeGroupIDs(matched)
		if err != nil {
			return nil, true, err
		}
		nodeIDs, err := loadTaskNodeIDsForGroups(groupIDs)
		if err != nil {
			return nil, true, err
		}
		if len(nodeIDs) == 0 {
			return nil, true, nil
		}
		if common == nil {
			common = nodeIDs
			continue
		}
		common = intersectInt64s(common, nodeIDs)
		if len(common) == 0 {
			return nil, true, nil
		}
	}
	if !scoped {
		return nil, false, nil
	}
	if len(missingDomains) > 0 {
		return nil, true, fmt.Errorf("cert domains not bound to any site: %s", strings.Join(missingDomains, ","))
	}
	return uniqueInt64List(common), true, nil
}

func validateCertHTTP01DNSForDomains(userID int64, domains []string) error {
	normalizedDomains := normalizeIssueDomains(domains)
	if len(normalizedDomains) == 0 {
		return nil
	}
	sites, err := loadIssueSitesForUser(userID)
	if err != nil {
		return err
	}
	for _, domain := range normalizedDomains {
		if strings.HasPrefix(domain, "*.") {
			continue
		}
		matched := make([]models.Site, 0)
		for _, site := range sites {
			if siteMatchesCertDomain(site, domain) {
				matched = append(matched, site)
			}
		}
		if len(matched) == 0 {
			continue
		}
		groupIDs, err := collectIssueNodeGroupIDs(matched)
		if err != nil {
			return err
		}
		expectedIPs, err := loadIssueLineIPsForGroups(groupIDs)
		if err != nil {
			return err
		}
		if len(expectedIPs) == 0 {
			return fmt.Errorf("cert domain has no enabled package line IPs: domain=%s groups=%s", domain, joinInt64s(groupIDs))
		}
		resolvedIPs, err := resolveIssueDomainIPs(domain)
		if err != nil {
			return fmt.Errorf("cert domain DNS preflight failed: domain=%s err=%v", domain, err)
		}
		if len(resolvedIPs) == 0 {
			return fmt.Errorf("cert domain DNS preflight failed: domain=%s has no A/AAAA records", domain)
		}
		if !issueResolvedIPsAllowed(resolvedIPs, expectedIPs) {
			return fmt.Errorf("cert domain DNS mismatch: domain=%s resolved=[%s] expected_package_ips=[%s]", domain, strings.Join(resolvedIPs, ","), strings.Join(expectedIPs, ","))
		}
	}
	return nil
}

func loadIssueLineIPsForGroups(groupIDs []int64) ([]string, error) {
	groupIDs = uniqueInt64List(groupIDs)
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var lines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id IN ? AND enable = ?", groupIDs, true).
		Find(&lines).Error; err != nil {
		return nil, err
	}
	nodeIDs := make([]int64, 0, len(lines))
	for _, line := range lines {
		id := line.NodeIPID
		if id == 0 {
			id = line.NodeID
		}
		if id != 0 {
			nodeIDs = append(nodeIDs, id)
		}
	}
	nodeIDs = uniqueInt64List(nodeIDs)
	if len(nodeIDs) == 0 {
		return nil, nil
	}
	var nodes []models.Node
	if err := db.DB.Select("id", "ip").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if ip := normalizeIssueIP(node.IP); ip != "" {
			ips = append(ips, ip)
		}
	}
	return uniqueSortedStrings(ips), nil
}

func resolveIssueDomainIPs(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	values, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(values))
	for _, value := range values {
		if ip := normalizeIssueIP(value); ip != "" {
			ips = append(ips, ip)
		}
	}
	return uniqueSortedStrings(ips), nil
}

func issueResolvedIPsAllowed(resolved []string, expected []string) bool {
	resolved = uniqueSortedStrings(resolved)
	expected = uniqueSortedStrings(expected)
	if len(resolved) == 0 || len(expected) == 0 {
		return false
	}
	expectedSet := make(map[string]struct{}, len(expected))
	for _, ip := range expected {
		expectedSet[ip] = struct{}{}
	}
	for _, ip := range resolved {
		if _, ok := expectedSet[ip]; !ok {
			return false
		}
	}
	return true
}

func normalizeIssueIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func uniqueSortedStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func joinInt64s(items []int64) string {
	items = uniqueInt64List(items)
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, strconv.FormatInt(item, 10))
	}
	return strings.Join(parts, ",")
}

func normalizeIssueDomains(domains []string) []string {
	out := make([]string, 0, len(domains))
	seen := map[string]struct{}{}
	for _, domain := range domains {
		host := normalizeDomainHostForEdge(domain)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out
}

func siteMatchesCertDomain(site models.Site, certDomain string) bool {
	for _, siteDomain := range site.Domains {
		if certDomainMatchesSiteDomain(certDomain, siteDomain) {
			return true
		}
	}
	return false
}

func certDomainMatchesSiteDomain(certDomain string, siteDomain string) bool {
	certHost := normalizeDomainHostForEdge(certDomain)
	if certHost == "" {
		return false
	}
	certWildcard := false
	if strings.HasPrefix(certHost, "*.") {
		certWildcard = true
		certHost = strings.TrimPrefix(certHost, "*.")
	}
	exact, wildcard := splitHostPattern(siteDomain)
	if exact != "" {
		return !certWildcard && certHost == exact
	}
	if wildcard == "" {
		return false
	}
	if certWildcard {
		return certHost == wildcard
	}
	return certHost != wildcard && strings.HasSuffix(certHost, "."+wildcard)
}

func collectIssueNodeGroupIDs(sites []models.Site) ([]int64, error) {
	packageIDs := make([]int64, 0)
	seenPackages := map[int64]struct{}{}
	for _, site := range sites {
		if site.UserPackageID == 0 {
			continue
		}
		if _, ok := seenPackages[site.UserPackageID]; ok {
			continue
		}
		seenPackages[site.UserPackageID] = struct{}{}
		packageIDs = append(packageIDs, site.UserPackageID)
	}

	packages := map[int64]models.UserPackage{}
	if len(packageIDs) > 0 {
		var rows []models.UserPackage
		if err := db.DB.Where("id IN ?", packageIDs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, pkg := range rows {
			packages[pkg.ID] = pkg
		}
	}

	groupIDs := make([]int64, 0)
	for _, site := range sites {
		if site.NodeGroupID > 0 {
			groupIDs = append(groupIDs, site.NodeGroupID)
		}
		if pkg, ok := packages[site.UserPackageID]; ok {
			if site.NodeGroupID == 0 && pkg.NodeGroupID > 0 {
				groupIDs = append(groupIDs, pkg.NodeGroupID)
			}
		}
	}
	return uniqueInt64List(groupIDs), nil
}

func loadTaskNodeIDsForGroups(groupIDs []int64) ([]int64, error) {
	groupIDs = uniqueInt64List(groupIDs)
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var nodeIDs []int64
	if err := db.DB.Model(&models.Line{}).
		Select("distinct node_id").
		Where("node_group_id IN ? AND enable = ? AND node_id <> 0", groupIDs, true).
		Pluck("node_id", &nodeIDs).Error; err != nil {
		return nil, err
	}
	return uniqueInt64List(nodeIDs), nil
}

func loadAvailableNodesByIDs(nodeIDs []int64) ([]models.Node, error) {
	nodeIDs = uniqueInt64List(nodeIDs)
	if len(nodeIDs) == 0 {
		return nil, nil
	}
	var nodes []models.Node
	if err := db.DB.Where("id IN ? AND pid = 0 AND enable = ?", nodeIDs, true).Order("id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return filterAvailableIssueNodes(nodes), nil
}

func groupCertsByCA(certs []models.Cert) map[string][]IssueCertItem {
	result := map[string][]IssueCertItem{}
	for _, cert := range certs {
		ca := normalizeIssueCA(cert.Type)
		item := IssueCertItem{
			CertID:  int64(cert.ID),
			Domains: splitCertDomains(cert.Domain),
		}
		if len(item.Domains) == 0 {
			continue
		}
		result[ca] = append(result[ca], item)
	}
	return result
}

func normalizeIssueCA(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "lets", "let's encrypt", "lets encrypt":
		return "letsencrypt"
	case "letsencrypt", "zerossl", "google", "buypass":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "letsencrypt"
	}
}

func buildIssueCAFallbackList(primary string) []string {
	first := normalizeIssueCA(primary)
	if first == "letsencrypt" {
		return []string{"letsencrypt"}
	}
	return []string{first, "letsencrypt"}
}

func createIssueTask(batchID int64, nodeID int64, name string, payload IssueCertTaskPayload) (models.Task, error) {
	data, _ := json.Marshal(payload)
	meta, _ := json.Marshal(issueTaskMeta{TargetNodeID: nodeID})
	task := models.Task{
		PID:      batchID,
		Name:     name,
		Type:     "issue_cert",
		Data:     string(data),
		Res:      string(meta),
		Enable:   true,
		State:    "waiting",
		CreateAt: time.Now(),
	}
	if nodeID > 0 {
		targets := NewTaskTargets([]int64{nodeID})
		task.TargetsJSON = targets.Marshal()
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return task, err
	}
	for _, item := range payload.Items {
		_ = db.DB.Model(&models.Cert{}).Where("id = ?", item.CertID).Update("issue_task_id", task.ID).Error
	}
	TriggerDispatchPending()
	return task, nil
}

func loadAvailableNodes() ([]models.Node, error) {
	var nodes []models.Node
	if err := db.DB.Where("pid = 0 AND enable = ?", true).Order("id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return filterAvailableIssueNodes(nodes), nil
}

func filterAvailableIssueNodes(nodes []models.Node) []models.Node {
	candidates := make([]models.Node, 0, len(nodes))
	online := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		if IsNodeRateLimited(node.ID) {
			continue
		}
		candidates = append(candidates, node)
		if IsNodeOnline(node.ID, 90*time.Second) {
			online = append(online, node)
		}
	}
	if len(online) > 0 {
		return online
	}
	return candidates
}

func intersectInt64s(a []int64, b []int64) []int64 {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := make(map[int64]struct{}, len(b))
	for _, id := range b {
		if id != 0 {
			set[id] = struct{}{}
		}
	}
	out := make([]int64, 0, len(a))
	seen := map[int64]struct{}{}
	for _, id := range a {
		if id == 0 {
			continue
		}
		if _, ok := set[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func loadCertsForIssue(ids []int64) ([]models.Cert, error) {
	var certs []models.Cert
	if err := db.DB.Where("id IN ?", ids).Order("id asc").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

func markCertsIssueFailed(certs []models.Cert, reason string) {
	for _, cert := range certs {
		markCertIssueFailed(int64(cert.ID), reason)
	}
}

func markCertIssueFailed(certID int64, reason string) {
	if certID == 0 {
		return
	}
	updates := map[string]interface{}{
		"state": "fail",
		"ret":   strings.TrimSpace(reason),
	}
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(updates).Error; err != nil {
		log.Printf("[CertIssue] mark cert failed cert_id=%d err=%v", certID, err)
	}
	MarkPendingHTTPSFailedForCert(certID, reason)
}

func MarkPendingHTTPSFailedForCert(certID int64, reason string) {
	if certID == 0 {
		return
	}
	var sites []models.Site
	if err := db.DB.Find(&sites).Error; err != nil {
		log.Printf("[CertIssue] load pending https failed sites failed cert_id=%d err=%v", certID, err)
		return
	}
	changedSiteIDs := make([]int64, 0)
	for _, site := range sites {
		httpsCfg := getMap(site.Settings, "https")
		if httpsCfg == nil {
			continue
		}
		pendingID := int64(parseIntValue(httpsCfg["pending_certificate_id"], 0))
		if pendingID != certID {
			continue
		}
		settings := site.Settings
		if settings == nil {
			settings = map[string]interface{}{}
		}
		httpsCfg = getMap(settings, "https")
		if httpsCfg == nil {
			httpsCfg = map[string]interface{}{}
			settings["https"] = httpsCfg
		}
		httpsCfg["enable"] = false
		httpsCfg["state"] = "failed"
		httpsCfg["last_error"] = strings.TrimSpace(reason)
		raw, _ := json.Marshal(settings)
		if err := db.DB.Model(&models.Site{}).Where("id = ?", site.ID).Updates(map[string]interface{}{
			"settings":  string(raw),
			"update_at": time.Now(),
		}).Error; err != nil {
			log.Printf("[CertIssue] mark pending https failed site_id=%d cert_id=%d err=%v", site.ID, certID, err)
			continue
		}
		changedSiteIDs = append(changedSiteIDs, site.ID)
	}
	if len(changedSiteIDs) > 0 {
		BumpConfigVersion("site", changedSiteIDs)
	}
}

func MarkIssueTaskFailed(taskID int64, reason string) {
	if taskID == 0 {
		return
	}
	updates := map[string]interface{}{
		"state": "fail",
		"ret":   strings.TrimSpace(reason),
	}
	if err := db.DB.Model(&models.Cert{}).
		Where("issue_task_id = ? AND state <> ?", taskID, "ready").
		Updates(updates).Error; err != nil {
		log.Printf("[CertIssue] mark task certs failed task_id=%d err=%v", taskID, err)
	}
}

func nextCertRetryDelay(errTimes int) time.Duration {
	delays := []int{5, 10, 20, 30, 60, 60, 60}
	if errTimes < len(delays) {
		return time.Duration(delays[errTimes]) * time.Minute
	}
	return 60 * time.Minute
}

func isFatalIssueError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "cert not found") ||
		strings.Contains(msg, "domain is empty") ||
		strings.Contains(msg, "acme:error") ||
		strings.Contains(msg, "dns problem") ||
		strings.Contains(msg, "nxdomain") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "timeout")
}
