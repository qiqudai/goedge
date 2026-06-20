using Cnn.Api.Hubs;
using Cnn.Api.Services;

namespace Cnn.Api.DependencyInjection;

public static class UiClientModuleExtensions
{
    public static IServiceCollection AddUiClientModule(this IServiceCollection services)
    {
        services.AddScoped<TaskHubClient>();
        services.AddScoped<ClientSession>();
        services.AddScoped<LocalStorageService>();
        services.AddScoped<SystemInfoClient>();
        services.AddScoped<ConfigItemClient>();
        return services;
    }
}
