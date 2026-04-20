using Cnn.Api.Services;

namespace Cnn.Api.DependencyInjection;

public static class HttpClientModuleExtensions
{
    public static IServiceCollection AddAppHttpClients(this IServiceCollection services)
    {
        services.AddCors(options =>
        {
            options.AddPolicy("AllowAll", policy =>
            {
                policy.AllowAnyHeader().AllowAnyMethod().SetIsOriginAllowed(_ => true).AllowCredentials();
            });
        });
        
        services.AddHttpClient<TaskApi>((sp, client) =>
        {
            client.BaseAddress = ResolveBaseUri(sp);
        });
        
        services.AddHttpClient<CacheApi>((sp, client) =>
        {
            client.BaseAddress = ResolveBaseUri(sp);
        });
        
        services.AddHttpClient<ApiClient>((sp, client) =>
        {
            client.BaseAddress = ResolveBaseUri(sp);
        });
        
        return services;
    }

    private static Uri ResolveBaseUri(IServiceProvider serviceProvider)
    {
        var httpContextAccessor = serviceProvider.GetRequiredService<IHttpContextAccessor>();
        var request = httpContextAccessor.HttpContext?.Request;
        if (request?.Host.HasValue == true)
        {
            return new Uri($"{request.Scheme}://{request.Host}{request.PathBase}".TrimEnd('/'));
        }

        var configuration = serviceProvider.GetRequiredService<IConfiguration>();
        var baseUrl = configuration["Api:BaseUrl"]?.TrimEnd('/') ?? "http://localhost:5035";
        return new Uri(baseUrl);
    }
}
