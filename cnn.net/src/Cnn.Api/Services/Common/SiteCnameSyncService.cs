using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface ISiteCnameSyncService
{
    Task ResyncSitesForUserPackageAsync(long userPackageId, CancellationToken cancellationToken);
}

public sealed class SiteCnameSyncService : ISiteCnameSyncService
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;

    public SiteCnameSyncService(ISqlSugarClient db, IDnsSyncService dnsSyncService)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
    }

    public async Task ResyncSitesForUserPackageAsync(long userPackageId, CancellationToken cancellationToken)
    {
        if (userPackageId <= 0)
        {
            return;
        }

        var sites = await _db.Queryable<Site>()
            .Where(s => s.UserPackage == userPackageId)
            .ToListAsync();

        foreach (var site in sites)
        {
            await ResyncSiteAsync(site);
        }
    }

    private async Task ResyncSiteAsync(Site site)
    {
        if (!await ShouldSyncSiteCnameAsync(site))
        {
            return;
        }

        var groupId = await ResolveGroupIdFromSiteAsync(site);
        if (groupId > 0)
        {
            await ResyncGroupLineCnamesAsync(groupId);
        }

        var backupGroup = site.BackupNodeGroup ?? 0;
        var enableBackup = site.EnableBackupGroup ?? false;
        if (!enableBackup && site.UserPackage is > 0)
        {
            var pkg = await _db.Queryable<UserPackage>()
                .Where(p => p.Id == site.UserPackage)
                .Select(p => new { p.BackupNodeGroup, p.EnableBackupGroup })
                .FirstAsync();
            if (pkg != null)
            {
                if (backupGroup == 0)
                {
                    backupGroup = pkg.BackupNodeGroup ?? 0;
                }
                enableBackup = pkg.EnableBackupGroup ?? false;
            }
        }

        if (enableBackup && backupGroup > 0)
        {
            await ResyncGroupLineCnamesAsync(backupGroup);
        }
    }

    private async Task<bool> ShouldSyncSiteCnameAsync(Site site)
    {
        if (site.UserPackage is null or <= 0)
        {
            return false;
        }

        var mode = site.CnameMode?.Trim();
        if (!string.IsNullOrWhiteSpace(mode))
        {
            return !string.Equals(mode, "package", StringComparison.OrdinalIgnoreCase);
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == site.UserPackage)
            .Select(p => new { p.CnameMode })
            .FirstAsync();

        if (pkg == null)
        {
            return true;
        }

        return !string.Equals(pkg.CnameMode?.Trim(), "package", StringComparison.OrdinalIgnoreCase);
    }

    private async Task<long> ResolveGroupIdFromSiteAsync(Site site)
    {
        if (site.NodeGroupId is > 0)
        {
            return site.NodeGroupId.Value;
        }

        if (site.UserPackage is null or <= 0)
        {
            return 0;
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == site.UserPackage)
            .Select(p => new { p.NodeGroupId })
            .FirstAsync();

        return pkg?.NodeGroupId ?? 0;
    }

    private async Task ResyncGroupLineCnamesAsync(long groupId)
    {
        if (groupId <= 0)
        {
            return;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId)
            .Select(l => new { l.LineId, l.LineName })
            .ToListAsync();

        var lineMap = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var line in lines)
        {
            var lineId = line.LineId?.Trim();
            if (string.IsNullOrWhiteSpace(lineId))
            {
                lineId = "default";
            }

            var lineName = line.LineName?.Trim();
            if (string.IsNullOrWhiteSpace(lineName))
            {
                lineName = lineId;
            }

            if (!lineMap.ContainsKey(lineId))
            {
                lineMap[lineId] = lineName;
            }
        }

        foreach (var pair in lineMap)
        {
            await _dnsSyncService.SyncPackageCnameForLineChangeAsync(groupId, pair.Key, pair.Value, Array.Empty<long>(), "resync");
        }
    }
}
