using Cnn.Api.Services.Agent;
using Cnn.Common.Localization;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Api.Services.Tasks.Workflow.Handlers;
using Cnn.Api.Services.Users;

namespace Cnn.Api.DependencyInjection;

public static class CoreModuleExtensions
{
    public static IServiceCollection AddCoreModule(this IServiceCollection services)
    {
        services.AddSingleton<IMessageLocalizer, MessageLocalizer>();
        services.AddSingleton<IAdminEventPublisher, AdminEventPublisher>();
        services.AddSingleton<IAdminIdentityResolver, AdminIdentityResolver>();
        services.AddSingleton<ICryptoService, CryptoService>();
        services.AddSingleton<IAcmeTokenStore, AcmeTokenStore>();
        services.AddSingleton<ISystemConfigService, SystemConfigService>();
        services.AddSingleton<ISystemInfoService, SystemInfoService>();
        services.AddScoped<IConfigVersionService, ConfigVersionService>();
        services.AddScoped<IDnsSyncService, DnsSyncService>();
        services.AddScoped<IDnsMaintenanceService, DnsMaintenanceService>();
        services.AddScoped<IConfigItemService, ConfigItemService>();
        services.AddScoped<IDomainService, DomainService>();
        services.AddScoped<IFinanceService, FinanceService>();
        services.AddScoped<IUploadService, UploadService>();
        services.AddScoped<INodeConfigService, NodeConfigService>();
        services.AddScoped<IEdgeConfigService, EdgeConfigService>();
        services.AddScoped<IDebugControlService, DebugControlService>();
        services.AddScoped<ISiteCnameSyncService, SiteCnameSyncService>();
        services.AddScoped<IForwardCnameSyncService, ForwardCnameSyncService>();
        services.AddScoped<IDomainUsageService, DomainUsageService>();
        services.AddScoped<IUsageService, UsageService>();
        services.AddScoped<IMessageService, MessageService>();
        services.AddScoped<IDeletionGuard, NodeDeletionGuard>();
        services.AddScoped<IDeletionGuard, LineGroupDeletionGuard>();
        services.AddScoped<IDeletionGuard, CertificateDeletionGuard>();
        services.AddScoped<IDeletionGuard, SecurityRuleDeletionGuard>();
        services.AddScoped<IDeletionGuard, AclRuleDeletionGuard>();
        services.AddScoped<IDeletionGuard, ProductPlanDeletionGuard>();
        services.AddScoped<IDeletionGuard, SubscriptionDeletionGuard>();
        services.AddScoped<IDeletionGuard, SiteDeletionGuard>();
        services.AddScoped<IDeletionGuard, StreamDeletionGuard>();
        services.AddScoped<IDeletionGuard, StreamGroupDeletionGuard>();
        services.AddScoped<IDeletionGuard, SiteGroupDeletionGuard>();
        services.AddScoped<IDeletionGuard, UserAccountDeletionGuard>();
        services.AddScoped<IDeletionGuardRegistry, DeletionGuardRegistry>();
        services.AddScoped<IDeletionPreviewService, DeletionPreviewService>();
        services.AddScoped<ITaskCommandFactory, SqlTaskCommandFactory>();
        services.AddScoped<IResourceActionRequestService, ResourceActionRequestService>();
        services.AddScoped<ITaskHandler, NodeDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, NodeStatusTaskHandler>();
        services.AddScoped<ITaskHandler, ConfigSyncTaskHandler>();
        services.AddScoped<ITaskHandler, SiteStatusTaskHandler>();
        services.AddScoped<ITaskHandler, StreamStatusTaskHandler>();
        services.AddScoped<ITaskHandler, CertificateStatusTaskHandler>();
        services.AddScoped<ITaskHandler, CertificateDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, SubscriptionDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, LineGroupDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, ProductPlanDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, SiteDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, SiteBatchDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, StreamDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, StreamBatchDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, StreamGroupDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, SiteGroupDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, SecurityRuleDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, AclRuleDeleteTaskHandler>();
        services.AddScoped<ITaskHandler, UserPurgeTaskHandler>();
        services.AddScoped<ITaskHandlerRegistry, TaskHandlerRegistry>();
        services.AddScoped<ITaskExecutor, SqlTaskExecutor>();
        services.AddScoped<IResourceDeleteRequestService, ResourceDeleteRequestService>();
        services.AddScoped<IUserPurgePlanner, UserPurgePlanner>();
        services.AddScoped<IUserPurgeExecutor, UserPurgeExecutor>();
        services.AddScoped<Cnn.Api.Services.Common.Tasks.ITaskMetadataAccessor, Cnn.Api.Services.Common.Tasks.JsonTaskMetadataAccessor>();
        services.AddScoped<ITaskService, TaskService>();
        services.AddScoped<IUserProfileService, UserProfileService>();
        return services;
    }
}
