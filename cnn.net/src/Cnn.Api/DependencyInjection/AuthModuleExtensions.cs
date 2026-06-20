using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Authz;

namespace Cnn.Api.DependencyInjection;

public static class AuthModuleExtensions
{
    public static IServiceCollection AddAuthModule(this IServiceCollection services)
    {
        services.AddSingleton<IAuthTokenService, AuthTokenService>();
        services.AddSingleton<ICaptchaService, CaptchaService>();
        services.AddSingleton<IEmailService, EmailService>();
        services.AddSingleton<ISpiderIpAllowlistService, SpiderIpAllowlistService>();
        services.AddSingleton<IPermissionCatalog, InMemoryPermissionCatalog>();
        services.AddSingleton<IApiPermissionResolver, ApiPermissionResolver>();
        services.AddSingleton<IAuthorizationAuditSink, AuthorizationAuditLoggerSink>();
        services.AddSingleton<IAuthorizationService, AuthorizationService>();
        services.AddScoped<IAuthService, AuthService>();
        services.AddScoped<IUserPackagePermissionService, UserPackagePermissionService>();
        services.AddScoped<IApiKeyService, ApiKeyService>();
        return services;
    }
}
