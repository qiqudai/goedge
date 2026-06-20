using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public sealed class UserPackageExpirationWorker : BackgroundService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<UserPackageExpirationWorker> _logger;

    public UserPackageExpirationWorker(IServiceScopeFactory scopeFactory, ILogger<UserPackageExpirationWorker> logger)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await CheckExpirationAsync(stoppingToken);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "User package expiration worker failed");
            }

            try
            {
                await Task.Delay(TimeSpan.FromHours(1), stoppingToken);
            }
            catch (TaskCanceledException)
            {
                return;
            }
        }
    }

    private async Task CheckExpirationAsync(CancellationToken cancellationToken)
    {
        using var scope = _scopeFactory.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<ISqlSugarClient>();
        var systemConfig = scope.ServiceProvider.GetRequiredService<ISystemConfigService>();
        var syncService = scope.ServiceProvider.GetRequiredService<IUserPackageSyncService>();

        var cfg = await systemConfig.LoadSystemConfigAsync(cancellationToken);
        if (cfg.TryGetValue("package_expire_close_site", out var enabledRaw) && !systemConfig.ParseBoolFlag(enabledRaw))
        {
            return;
        }

        var now = DateTime.Now;
        var expired = await db.Queryable<Cnn.Domain.Entities.UserPackage>()
            .Where(p => p.EndAt != null && p.EndAt < now && (p.IsExpired == null || p.IsExpired == false))
            .ToListAsync();

        foreach (var pkg in expired)
        {
            if (pkg.Id <= 0)
            {
                continue;
            }

            var rows = await db.Updateable<Cnn.Domain.Entities.UserPackage>()
                .SetColumns(p => new Cnn.Domain.Entities.UserPackage { IsExpired = true })
                .Where(p => p.Id == pkg.Id)
                .ExecuteCommandAsync();
            if (rows <= 0)
            {
                continue;
            }

            var userId = pkg.Uid ?? 0;
            if (userId > 0)
            {
                var title = "Package expired";
                var content = $"Package {pkg.Name} expired at {pkg.EndAt:yyyy-MM-dd HH:mm:ss}.";
                await NotificationHelper.CreateUserMessageAsync(db, systemConfig, userId, "package-expire", title, content, pkg.Id, 0, cancellationToken);
            }

            await syncService.SyncAsync(pkg.Id, "expire", cancellationToken);
        }
    }
}
