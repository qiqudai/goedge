using Cnn.Api.DependencyInjection;

namespace Cnn.Api.Extensions;

public static class DependencyInjectionExtensions
{
    public static IServiceCollection AddApplicationServices(this IServiceCollection services)
    {
        services.AddCoreModule();
        services.AddAuthModule();
        services.AddAdminModule();
        services.AddAgentModule();
        services.AddStatsModule();
        services.AddUiClientModule();
        services.AddBackgroundJobs();
        services.AddAppHttpClients();
        return services;
    }
}
