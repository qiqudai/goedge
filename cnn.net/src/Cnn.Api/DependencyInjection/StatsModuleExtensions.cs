using Cnn.Api.Services.Stats;

namespace Cnn.Api.DependencyInjection;

public static class StatsModuleExtensions
{
    public static IServiceCollection AddStatsModule(this IServiceCollection services)
    {
        services.AddScoped<IAccessStatsService, AccessStatsService>();
        services.AddScoped<IRankingService, RankingService>();
        services.AddScoped<ISiteHostIndexService, SiteHostIndexService>();
        services.AddScoped<IHostFilterResolver, HostFilterResolver>();
        services.AddScoped<IIpRegionService, IpRegionService>();
        services.AddScoped<IStatsService, StatsService>();
        services.AddScoped<IDashboardService, DashboardService>();
        return services;
    }
}
