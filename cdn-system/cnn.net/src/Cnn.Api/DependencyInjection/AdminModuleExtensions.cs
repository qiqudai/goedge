using Cnn.Api.Services.Admin;

namespace Cnn.Api.DependencyInjection;

public static class AdminModuleExtensions
{
    public static IServiceCollection AddAdminModule(this IServiceCollection services)
    {
        services.AddScoped<INodeService, NodeService>();
        services.AddScoped<INodeGroupService, NodeGroupService>();
        services.AddScoped<IRegionService, RegionService>();
        services.AddScoped<IDnsProviderService, DnsProviderService>();
        services.AddScoped<IDnsApiService, DnsApiService>();
        services.AddScoped<ICnameDomainService, CnameDomainService>();
        services.AddScoped<IMonitorConfigService, MonitorConfigService>();
        services.AddScoped<IGlobalConfigService, GlobalConfigService>();
        services.AddScoped<IAclService, AclService>();
        services.AddScoped<ICcRuleGroupService, CcRuleGroupService>();
        services.AddScoped<ICcMatcherService, CcMatcherService>();
        services.AddScoped<ICcFilterService, CcFilterService>();
        services.AddScoped<IForwardService, ForwardService>();
        services.AddScoped<IForwardGroupService, ForwardGroupService>();
        services.AddScoped<IForwardDefaultService, ForwardDefaultService>();
        services.AddScoped<IForwardMonitorService, ForwardMonitorService>();
        services.AddScoped<ISiteGroupService, SiteGroupService>();
        services.AddScoped<ISiteDefaultService, SiteDefaultService>();
        services.AddScoped<ISiteService, SiteService>();
        services.AddScoped<IPlanService, PlanService>();
        services.AddScoped<ICertService, CertService>();
        services.AddScoped<ICertIssueProcessor, CertIssueProcessor>();
        services.AddScoped<IAnnouncementService, AnnouncementService>();
        services.AddScoped<ILoginLogService, LoginLogService>();
        services.AddScoped<IOperationLogService, OperationLogService>();
        services.AddScoped<IBackupLogService, BackupLogService>();
        services.AddScoped<IMailLogService, MailLogService>();
        services.AddScoped<IAccessLogService, AccessLogService>();
        services.AddScoped<IAccessLogDownloadService, AccessLogDownloadService>();
        services.AddScoped<IEventLogService, EventLogService>();
        services.AddScoped<IBlockLogService, BlockLogService>();
        services.AddScoped<IUserService, UserService>();
        services.AddScoped<IUserPackageService, UserPackageService>();
        services.AddScoped<IUserPackageSyncService, UserPackageSyncService>();
        services.AddScoped<Cnn.Api.Services.Admin.ISiteSettingsStore, Cnn.Api.Services.Admin.SiteSettingsStore>();
        services.AddScoped<Cnn.Api.Services.Admin.ISiteCacheApplicationService, Cnn.Api.Services.Admin.SiteCacheApplicationService>();
        return services;
    }
}
