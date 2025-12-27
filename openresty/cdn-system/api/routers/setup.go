package routers

import (
	"cdn-api/controllers"
	"cdn-api/middleware"
	"cdn-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	// Health Check for Load Balancer (HA)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "node": "server-1"})
	})

	// 0. Public Routes
	authCtr := &controllers.AuthController{}
	// Shared login or specific login paths (Frontend requests /api/v1/admin/login etc due to baseURL)
	r.POST("/api/v1/login", authCtr.Login)
	r.POST("/api/v1/admin/login", authCtr.Login)
	r.POST("/api/v1/user/login", authCtr.Login)

	// V1 API Group
	v1 := r.Group("/api/v1")
	{
		// 1. Admin Routes (Require Admin Auth Middleware)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired("admin"))
		admin.Use(middleware.OperationLog())
		{
			nodeCtr := &controllers.NodeController{
				NodeService: services.NewNodeService(),
			}
			admin.GET("/nodes", nodeCtr.ListNodes)
			admin.POST("/nodes", nodeCtr.CreateNode)
			admin.PUT("/nodes/:id", nodeCtr.UpdateNode)
			admin.POST("/nodes/batch", nodeCtr.BatchAction)
			ngCtr := &controllers.NodeGroupController{}
			admin.GET("/node-groups", ngCtr.ListNodeGroups)
			admin.POST("/node-groups", ngCtr.CreateNodeGroup)
			admin.PUT("/node-groups/:id", ngCtr.UpdateNodeGroup)
			admin.DELETE("/node-groups/:id", ngCtr.DeleteNodeGroup)
			regionCtr := &controllers.RegionController{}
			admin.GET("/regions", regionCtr.ListRegions)

			dnsCtr := &controllers.DnsController{}
			admin.GET("/dns/providers", dnsCtr.ListProviders)
			admin.GET("/dns/providers/types", dnsCtr.GetProviderTypes)
			admin.POST("/dns/providers", dnsCtr.CreateProvider)
			admin.DELETE("/dns/providers/:id", dnsCtr.DeleteProvider)

			// CNAME Domains
			cnameCtr := &controllers.CnameController{}
			admin.GET("/cname_domains", cnameCtr.ListDomains)
			admin.POST("/cname_domains", cnameCtr.CreateDomain)
			admin.DELETE("/cname_domains/:id", cnameCtr.DeleteDomain)

			admin.GET("/monitor_config", (&controllers.MonitorController{}).GetConfig)
			admin.POST("/monitor_config", (&controllers.MonitorController{}).UpdateConfig)

			// Logs
			logCtr := &controllers.LogController{}
			admin.GET("/logs/login", logCtr.ListLoginLogs)
			admin.GET("/logs/operation", logCtr.ListOpLogs)
			admin.GET("/logs/access", logCtr.ListAccessLogs)

			blockLogCtr := &controllers.BlockLogController{}
			admin.GET("/logs/block/current", blockLogCtr.ListCurrent)
			admin.GET("/logs/block/stats", blockLogCtr.ListStats)
			admin.GET("/logs/block/history", blockLogCtr.ListHistory)

			// Messages
			admin.GET("/messages", (&controllers.MessageController{}).AdminList)

			// Statistics & Ranking
			statCtr := &controllers.StatController{}
			admin.GET("/stats/ranking", statCtr.ListRanking)
			admin.GET("/stats/basic", statCtr.ListBasic)
			admin.GET("/stats/quality", statCtr.ListQuality)
			admin.GET("/stats/origin", statCtr.ListOrigin)

			// Dashboard
			dashCtr := &controllers.DashboardController{}
			admin.GET("/dashboard", dashCtr.Index)

			// Global Config
			admin.GET("/global_config", (&controllers.GlobalConfigController{}).GetConfig)
			admin.POST("/global_config", (&controllers.GlobalConfigController{}).UpdateConfig)
			// Config Items (defaults)
			configItemCtr := &controllers.ConfigItemController{}
			admin.GET("/config_items", configItemCtr.List)
			admin.POST("/config_items", configItemCtr.Upsert)

			// Packages
			admin.GET("/packages", (&controllers.PackageController{}).ListVersions)
			admin.POST("/packages", (&controllers.PackageController{}).UploadVersion)
			admin.POST("/packages/grayscale", (&controllers.PackageController{}).UpdateGrayScale)

			// Plans (Packages)
			planCtr := &controllers.PlanController{}
			admin.GET("/plans", planCtr.ListPlans)
			admin.GET("/plans/:id", planCtr.GetPlan)
			admin.POST("/plans", planCtr.CreatePlan)
			admin.PUT("/plans/:id", planCtr.UpdatePlan)
			admin.DELETE("/plans/:id", planCtr.DeletePlan)

			// User Plans (Sold)
			admin.GET("/user_plans", planCtr.ListUserPlans)
			admin.POST("/user_plans/assign", planCtr.AssignUserPlan)
			admin.PUT("/user_plans/:id", planCtr.UpdateUserPlan)
			admin.DELETE("/user_plans", planCtr.DeleteUserPlans)

			// Finance
			admin.GET("/orders", (&controllers.FinanceController{}).ListOrders)
			admin.POST("/recharge", (&controllers.FinanceController{}).Recharge)

			// Announcements
			annCtr := &controllers.AnnouncementController{}
			admin.GET("/announcements", annCtr.List)
			admin.POST("/announcements", annCtr.Create)
			admin.PUT("/announcements/:id", annCtr.Update)
			admin.DELETE("/announcements/:id", annCtr.Delete)

			// System
			admin.GET("/system_info", (&controllers.SystemController{}).GetInfo)
			admin.POST("/system_info", (&controllers.SystemController{}).UpdateInfo)

			// Domain Management
			userDomainCtr := &controllers.UserDomainController{}
			admin.GET("/domains", userDomainCtr.AdminListDomains)

			// User Domain Management
			userCtr := &controllers.UserController{}
			admin.GET("/users", userCtr.ListUsers)
			admin.PUT("/users/:id/status", userCtr.ToggleStatus)
			admin.DELETE("/users/:id", userCtr.DeleteUser)
			admin.POST("/users/:id/purge/reset", userCtr.ResetPurgeUsage)
			admin.POST("/users/:id/impersonate", userCtr.Impersonate)

			// Site/Cert Management (Admin)
			// Site
			siteCtr := &controllers.SiteController{}
			admin.GET("/sites", siteCtr.AdminList)
			admin.POST("/sites", siteCtr.AdminCreate)
			admin.POST("/sites/batch", siteCtr.AdminBatchCreate)
			admin.POST("/sites/batch_update", siteCtr.AdminBatchUpdate)
			admin.POST("/sites/batch_action", siteCtr.AdminBatchAction)
			admin.POST("/sites/apply_cert", siteCtr.AdminApplyCert)
			admin.GET("/sites/export", siteCtr.AdminExport)
			admin.GET("/sites/resolve", siteCtr.AdminResolve)

			siteGroupCtr := &controllers.SiteGroupController{}
			admin.GET("/site_groups", siteGroupCtr.List)
			admin.POST("/site_groups", siteGroupCtr.Create)
			admin.PUT("/site_groups/:id", siteGroupCtr.Update)
			admin.DELETE("/site_groups/:id", siteGroupCtr.Delete)

			siteDefaultCtr := &controllers.SiteDefaultController{}
			admin.GET("/site_defaults", siteDefaultCtr.List)
			admin.POST("/site_defaults", siteDefaultCtr.Create)
			admin.PUT("/site_defaults/:name", siteDefaultCtr.Update)
			admin.DELETE("/site_defaults/:name", siteDefaultCtr.Delete)

			userPackageCtr := &controllers.UserPackageController{}
			admin.GET("/user_packages", userPackageCtr.ListUserPackages)

			// Cert
			certCtr := &controllers.CertController{}
			admin.GET("/certs", certCtr.AdminList)
			admin.POST("/certs", certCtr.Upload)
			admin.PUT("/certs/:id", certCtr.Update)
			admin.DELETE("/certs/:id", certCtr.Delete)
			admin.POST("/certs/batch_action", certCtr.BatchAction)
			admin.POST("/certs/batch", certCtr.BatchCreate)
			admin.POST("/certs/reissue", certCtr.Reissue)
			admin.GET("/certs/:id/download", certCtr.Download)
			admin.GET("/certs/default_settings", certCtr.GetDefaultSettings)
			admin.POST("/certs/default_settings", certCtr.UpdateDefaultSettings)

			dnsapiCtr := &controllers.DNSAPIController{}
			admin.GET("/dnsapi", dnsapiCtr.List)
			admin.POST("/dnsapi", dnsapiCtr.Create)
			admin.PUT("/dnsapi/:id", dnsapiCtr.Update)
			admin.DELETE("/dnsapi/:id", dnsapiCtr.Delete)
			admin.GET("/dnsapi/types", dnsapiCtr.Types)

			// Forward
			forwardCtr := &controllers.ForwardController{}
			admin.GET("/forwards", forwardCtr.AdminList)
			admin.POST("/forwards", forwardCtr.AdminCreate)
			admin.POST("/forwards/batch", forwardCtr.AdminBatchCreate)
			admin.POST("/forwards/batch_update", forwardCtr.AdminBatchUpdate)
			admin.POST("/forwards/batch_action", forwardCtr.AdminBatchAction)

			forwardGroupCtr := &controllers.ForwardGroupController{}
			admin.GET("/forward_groups", forwardGroupCtr.List)
			admin.POST("/forward_groups", forwardGroupCtr.Create)
			admin.PUT("/forward_groups", forwardGroupCtr.Update)
			admin.DELETE("/forward_groups", forwardGroupCtr.Delete)

			forwardDefaultCtr := &controllers.ForwardDefaultController{}
			admin.GET("/forward_defaults", forwardDefaultCtr.List)
			admin.POST("/forward_defaults", forwardDefaultCtr.Create)
			admin.DELETE("/forward_defaults", forwardDefaultCtr.Delete)

			// Task (Purge/Preheat)
			taskCtr := &controllers.TaskController{}
			admin.GET("/tasks", taskCtr.List)
			admin.POST("/tasks", taskCtr.Create)
			admin.GET("/tasks/usage", taskCtr.Usage)
			admin.POST("/tasks/:id/resubmit", taskCtr.Resubmit)

			// Rules Management (CC/ACL)
			ruleCtr := &controllers.RuleController{}
			admin.GET("/rules/cc/groups", ruleCtr.ListCCRuleGroups)
			admin.POST("/rules/cc/groups", ruleCtr.CreateCCRuleGroup)
			admin.PUT("/rules/cc/groups/:id", ruleCtr.UpdateCCRuleGroup)
			admin.GET("/rules/cc/groups/:id", ruleCtr.GetRuleGroup)
			admin.GET("/rules/cc/matchers", ruleCtr.ListMatchers)
			admin.GET("/rules/cc/matchers/:id", ruleCtr.GetMatcher)
			admin.POST("/rules/cc/matchers", ruleCtr.CreateMatcher)
			admin.PUT("/rules/cc/matchers/:id", ruleCtr.UpdateMatcher)
			admin.GET("/rules/cc/filters", ruleCtr.ListFilters)
			admin.GET("/rules/cc/filters/:id", ruleCtr.GetFilter)
			admin.POST("/rules/cc/filters", ruleCtr.CreateFilter)
			admin.PUT("/rules/cc/filters/:id", ruleCtr.UpdateFilter)
			admin.DELETE("/rules/cc/filters/:id", ruleCtr.DeleteFilter)

			aclCtr := &controllers.ACLController{}
			admin.GET("/rules/acl", aclCtr.List)
			admin.GET("/rules/acl/:id", aclCtr.Get)
			admin.POST("/rules/acl", aclCtr.Create)
			admin.PUT("/rules/acl/:id", aclCtr.Update)
			admin.DELETE("/rules/acl/:id", aclCtr.Delete)
		}

		// 2. User Routes (Require User Auth Middleware)
		user := v1.Group("/user")
		user.Use(middleware.AuthRequired("user"))
		user.Use(middleware.OperationLog())
		{
			// Domain Management
			user.GET("/domains", (&controllers.UserDomainController{}).ListDomains)
			user.POST("/domains", (&controllers.UserDomainController{}).CreateDomain)
			user.GET("/domains/:id/config", (&controllers.UserDomainController{}).GetConfig)
			configItemCtr := &controllers.ConfigItemController{}
			user.GET("/config_items", configItemCtr.ListUser)
			user.POST("/config_items", configItemCtr.UpsertUser)

			profileCtr := &controllers.UserProfileController{}
			user.GET("/profile", profileCtr.GetProfile)
			user.PUT("/profile", profileCtr.UpdateProfile)
			user.PUT("/password", profileCtr.UpdatePassword)
			user.POST("/recharge", (&controllers.FinanceController{}).UserRecharge)

			// Orders
			user.GET("/orders", (&controllers.FinanceController{}).ListUserOrders)
			user.GET("/logs/operation", (&controllers.LogController{}).ListOpLogsUser)
			user.GET("/messages", (&controllers.MessageController{}).UserList)
			user.POST("/messages/:id/read", (&controllers.MessageController{}).MarkRead)
			user.GET("/message_sub", (&controllers.MessageController{}).GetSubscriptions)
			user.PUT("/message_sub", (&controllers.MessageController{}).UpdateSubscriptions)

			apiKeyCtr := &controllers.APIKeyController{}
			user.GET("/api_key", apiKeyCtr.GetKey)
			user.PUT("/api_key", apiKeyCtr.UpdateKey)
			user.POST("/api_key/reset", apiKeyCtr.ResetSecret)

			// Site management
			siteController := new(controllers.SiteController)
			user.GET("/sites", siteController.List)
			user.POST("/sites", siteController.Create)

			// Cert management
			certController := new(controllers.CertController)
			user.GET("/certs", certController.List)
			user.POST("/certs", certController.Upload)
			user.PUT("/certs/:id", certController.Update)
			user.DELETE("/certs/:id", certController.Delete)
			user.POST("/certs/batch_action", certController.BatchAction)
			user.POST("/certs/batch", certController.BatchCreate)
			user.POST("/certs/reissue", certController.Reissue)
			user.GET("/certs/default_settings", certController.GetDefaultSettings)
			user.POST("/certs/default_settings", certController.UpdateDefaultSettings)

			userTaskCtr := &controllers.TaskController{}
			user.GET("/tasks", userTaskCtr.List)
			user.POST("/tasks", userTaskCtr.Create)
			user.GET("/tasks/usage", userTaskCtr.Usage)
			user.POST("/tasks/:id/resubmit", userTaskCtr.Resubmit)

			userPlanCtr := &controllers.PlanController{}
			user.GET("/plans", userPlanCtr.ListPlans)

			userPackageCtr := &controllers.UserPackageController{}
			user.GET("/user_packages", userPackageCtr.ListUserPackages)
			user.PUT("/user_packages/:id", userPackageCtr.UpdateUserPackage)
			user.POST("/user_packages/:id/renew", userPackageCtr.RenewUserPackage)
			user.POST("/user_packages/:id/switch", userPackageCtr.SwitchUserPackage)

			userSiteGroupCtr := &controllers.SiteGroupController{}
			user.GET("/site_groups", userSiteGroupCtr.List)
			user.POST("/site_groups", userSiteGroupCtr.Create)
			user.PUT("/site_groups/:id", userSiteGroupCtr.Update)
			user.DELETE("/site_groups/:id", userSiteGroupCtr.Delete)

			userSiteDefaultCtr := &controllers.SiteDefaultController{}
			user.GET("/site_defaults", userSiteDefaultCtr.List)
			user.POST("/site_defaults", userSiteDefaultCtr.Create)
			user.PUT("/site_defaults/:name", userSiteDefaultCtr.Update)
			user.DELETE("/site_defaults/:name", userSiteDefaultCtr.Delete)

			userDnsCtr := &controllers.DnsController{}
			user.GET("/dns/providers", userDnsCtr.ListProviders)
			user.GET("/dns/providers/types", userDnsCtr.GetProviderTypes)

			userDnsapiCtr := &controllers.DNSAPIController{}
			user.GET("/dnsapi", userDnsapiCtr.List)
			user.GET("/dnsapi/types", userDnsapiCtr.Types)

			userRuleCtr := &controllers.RuleController{}
			user.GET("/rules/cc/groups", userRuleCtr.ListCCRuleGroups)
			user.POST("/rules/cc/groups", userRuleCtr.CreateCCRuleGroup)
			user.PUT("/rules/cc/groups/:id", userRuleCtr.UpdateCCRuleGroup)
			user.GET("/rules/cc/groups/:id", userRuleCtr.GetRuleGroup)
			user.GET("/rules/cc/matchers", userRuleCtr.ListMatchers)
			user.GET("/rules/cc/matchers/:id", userRuleCtr.GetMatcher)
			user.POST("/rules/cc/matchers", userRuleCtr.CreateMatcher)
			user.PUT("/rules/cc/matchers/:id", userRuleCtr.UpdateMatcher)
			user.GET("/rules/cc/filters", userRuleCtr.ListFilters)
			user.GET("/rules/cc/filters/:id", userRuleCtr.GetFilter)
			user.POST("/rules/cc/filters", userRuleCtr.CreateFilter)
			user.PUT("/rules/cc/filters/:id", userRuleCtr.UpdateFilter)
			user.DELETE("/rules/cc/filters/:id", userRuleCtr.DeleteFilter)

			userAclCtr := &controllers.ACLController{}
			user.GET("/rules/acl", userAclCtr.List)
			user.GET("/rules/acl/:id", userAclCtr.Get)
			user.POST("/rules/acl", userAclCtr.Create)
			user.PUT("/rules/acl/:id", userAclCtr.Update)
			user.DELETE("/rules/acl/:id", userAclCtr.Delete)

			userLogCtr := &controllers.LogController{}
			user.GET("/logs/access", userLogCtr.ListAccessLogs)

			userBlockLogCtr := &controllers.BlockLogController{}
			user.GET("/logs/block/current", userBlockLogCtr.ListCurrent)
			user.GET("/logs/block/stats", userBlockLogCtr.ListStats)
			user.GET("/logs/block/history", userBlockLogCtr.ListHistory)

			userStatCtr := &controllers.StatController{}
			user.GET("/stats/basic", userStatCtr.ListBasic)
			user.GET("/stats/quality", userStatCtr.ListQuality)
			user.GET("/stats/origin", userStatCtr.ListOrigin)
			user.GET("/stats/ranking", userStatCtr.ListRanking)
			user.GET("/usage", userStatCtr.ListUsage)

			userForwardCtr := &controllers.ForwardController{}
			user.GET("/forwards", userForwardCtr.AdminList)
			user.POST("/forwards", userForwardCtr.AdminCreate)
			user.POST("/forwards/batch", userForwardCtr.AdminBatchCreate)
			user.POST("/forwards/batch_update", userForwardCtr.AdminBatchUpdate)
			user.POST("/forwards/batch_action", userForwardCtr.AdminBatchAction)

			userForwardGroupCtr := &controllers.ForwardGroupController{}
			user.GET("/forward_groups", userForwardGroupCtr.List)
			user.POST("/forward_groups", userForwardGroupCtr.Create)
			user.PUT("/forward_groups", userForwardGroupCtr.Update)
			user.DELETE("/forward_groups", userForwardGroupCtr.Delete)

			userForwardDefaultCtr := &controllers.ForwardDefaultController{}
			user.GET("/forward_defaults", userForwardDefaultCtr.List)
			user.POST("/forward_defaults", userForwardDefaultCtr.Create)
			user.DELETE("/forward_defaults", userForwardDefaultCtr.Delete)
		}

		// 3. Node Agent Routes (Require Node Token Middleware)
		agentGroup := v1.Group("/agent")
		agentGroup.Use(middleware.AgentDebug(), middleware.AuthAgent())
		{
			agentCtr := controllers.NewAgentController()
			agentGroup.POST("/heartbeat", agentCtr.Heartbeat)
			agentGroup.GET("/config", agentCtr.GetConfig)
			agentGroup.GET("/tasks", agentCtr.GetTasks)
			agentGroup.POST("/tasks/:id/finish", agentCtr.FinishTask)

			agentLogCtr := &controllers.AgentLogController{}
			agentGroup.POST("/logs/access", agentLogCtr.AccessLogs)
			agentGroup.POST("/logs/metrics", agentLogCtr.Metrics)
			agentGroup.POST("/logs/events", agentLogCtr.Events)
		}

	}
}
