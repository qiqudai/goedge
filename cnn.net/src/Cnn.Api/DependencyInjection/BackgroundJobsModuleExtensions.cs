using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;

namespace Cnn.Api.DependencyInjection;

public static class BackgroundJobsModuleExtensions
{
    public static IServiceCollection AddBackgroundJobs(this IServiceCollection services)
    {
        services.AddHostedService<UserPackageExpirationWorker>();
        services.AddHostedService<UserPackageTrafficWorker>();
        services.AddHostedService<CertIssueWorker>();
        services.AddHostedService<CleanupAndBackupWorker>();
        services.AddHostedService<TaskDispatchWorker>();
        return services;
    }
}
