package routers

import (
	"cdn-api/controllers"
	"cdn-api/middleware"
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
		{
			nodeCtr := &controllers.NodeController{}
			admin.GET("/nodes", nodeCtr.ListNodes)
			admin.POST("/nodes", nodeCtr.CreateNode)
			admin.PUT("/nodes/:id", nodeCtr.UpdateNode)
			admin.POST("/nodes/batch", nodeCtr.BatchAction)
			ngCtr := &controllers.NodeGroupController{}
			admin.GET("/node-groups", ngCtr.ListNodeGroups)
			admin.POST("/node-groups", ngCtr.CreateNodeGroup)
			admin.PUT("/node-groups/:id", ngCtr.UpdateNodeGroup)
			admin.DELETE("/node-groups/:id", ngCtr.DeleteNodeGroup)

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

			// Packages
			admin.POST("/packages/grayscale", (&controllers.PackageController{}).UpdateGrayScale)

			// Plans (Packages)
			planCtr := &controllers.PlanController{}
			admin.GET("/plans", planCtr.ListPlans)
			admin.POST("/plans", planCtr.CreatePlan)
			admin.PUT("/plans/:id", planCtr.UpdatePlan)
			admin.DELETE("/plans/:id", planCtr.DeletePlan)

			// User Plans (Sold)
			admin.GET("/user_plans", planCtr.ListUserPlans)

			// Finance
			admin.GET("/orders", (&controllers.FinanceController{}).ListOrders)
			admin.POST("/recharge", (&controllers.FinanceController{}).Recharge)

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
			admin.GET("/users/:id/node-groups", userCtr.ListUserNodeGroups)
			admin.PUT("/users/:id/node-groups", userCtr.UpdateUserNodeGroups)

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

			// Task (Purge/Preheat)
			taskCtr := &controllers.TaskController{}
			admin.GET("/tasks", taskCtr.List)
			admin.POST("/tasks", taskCtr.Create)

			// Rules Management (CC/ACL)
			ruleCtr := &controllers.RuleController{}
			admin.GET("/rules/cc/groups", ruleCtr.ListCCRuleGroups)
			admin.GET("/rules/cc/groups/:id", ruleCtr.GetRuleGroup)
			admin.GET("/rules/cc/matchers", ruleCtr.ListMatchers)
			admin.GET("/rules/cc/filters", ruleCtr.ListFilters)
		}

		// 2. User Routes (Require User Auth Middleware)
		user := v1.Group("/user")
		user.Use(middleware.AuthRequired("user"))
		{
			// Domain Management
			user.GET("/domains", (&controllers.UserDomainController{}).ListDomains)
			user.POST("/domains", (&controllers.UserDomainController{}).CreateDomain)
			user.GET("/domains/:id/config", (&controllers.UserDomainController{}).GetConfig)

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
			user.POST("/certs/default_settings", certController.UpdateDefaultSettings)
		}

		// 3. Node Agent Routes (Require Node Token Middleware)
		agentGroup := v1.Group("/agent")
		agentGroup.Use(middleware.AuthAgent())
		{
			agentCtr := controllers.NewAgentController()
			agentGroup.POST("/heartbeat", agentCtr.Heartbeat)
			agentGroup.GET("/config", agentCtr.GetConfig)
		}

	}
}


