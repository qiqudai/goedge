using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IUserPackageService
{
    Task<ServiceResult<UserPackageListResult>> ListAsync(UserPackageListQuery query, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, UserPackageUpdateRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<RenewUserPackageResult>> RenewAsync(long id, RenewUserPackageRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> SwitchAsync(long id, SwitchUserPackageRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);
}

public sealed class UserPackageService : IUserPackageService
{
    private const string ConfigType = "user_package_config";
    private const string ConfigScopeName = "user_package";

    private readonly ISqlSugarClient _db;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IUserPackageSyncService _syncService;
    private readonly ISiteCnameSyncService _siteCnameSyncService;

    public UserPackageService(
        ISqlSugarClient db,
        ISystemConfigService systemConfigService,
        IUserPackageSyncService syncService,
        ISiteCnameSyncService siteCnameSyncService)
    {
        _db = db;
        _systemConfigService = systemConfigService;
        _syncService = syncService;
        _siteCnameSyncService = siteCnameSyncService;
    }

    public async Task<ServiceResult<UserPackageListResult>> ListAsync(
        UserPackageListQuery query,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        var q = _db.Queryable<UserPackage>();
        if (isUserRequest)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<UserPackageListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            q = q.Where(p => p.Uid == userId.Value);
        }
        else if (query?.UserId is > 0)
        {
            q = q.Where(p => p.Uid == query.UserId.Value);
        }

        var packs = await q.OrderBy(p => p.Id, OrderByType.Desc).ToListAsync();
        var ipv6Map = await LoadUserPackageBoolConfigAsync(packs, "ipv6");
        var http3Map = await LoadUserPackageBoolConfigAsync(packs, "http3_enabled");

        var now = DateTime.Now;
        var list = new List<UserPackageItemDto>(packs.Count);
        foreach (var pack in packs)
        {
            var status = "active";
            if (pack.EndAt.HasValue && pack.EndAt.Value < now)
            {
                status = "expired";
            }

            var recordId = pack.RecordId?.Trim();
            if (string.IsNullOrWhiteSpace(recordId))
            {
                recordId = await GenerateUniqueRecordIdAsync();
                if (!string.IsNullOrWhiteSpace(recordId))
                {
                    pack.RecordId = recordId;
                    await _db.Updateable(pack).UpdateColumns(p => new { p.RecordId }).ExecuteCommandAsync();
                }
            }

            list.Add(new UserPackageItemDto
            {
                Id = pack.Id,
                Uid = pack.Uid ?? 0,
                Name = pack.Name,
                PackageId = pack.Package ?? 0,
                RegionId = pack.RegionId ?? 0,
                NodeGroupId = pack.NodeGroupId ?? 0,
                BackupNodeGroup = pack.BackupNodeGroup ?? 0,
                EnableBackupGroup = pack.EnableBackupGroup ?? false,
                CnameDomain = pack.CnameDomain?.Trim(),
                CnameHostname2 = pack.CnameHostname2?.Trim(),
                CnameHostname = pack.CnameHostname?.Trim(),
                CnameMode = pack.CnameMode,
                RecordId = recordId,
                Traffic = pack.Traffic ?? 0,
                Bandwidth = pack.Bandwidth,
                Connection = pack.Connection ?? 0,
                Domain = pack.Domain ?? 0,
                MainDomainLimit = pack.MainDomainLimit ?? 0,
                HttpPort = pack.HttpPort ?? 0,
                StreamPort = pack.StreamPort ?? 0,
                CustomCcRule = pack.CustomCcRule ?? false,
                Websocket = pack.Websocket ?? false,
                L2Origin = pack.L2Origin ?? false,
                MonthPrice = pack.MonthPrice ?? 0,
                QuarterPrice = pack.QuarterPrice ?? 0,
                YearPrice = pack.YearPrice ?? 0,
                CreateAt = pack.CreateAt,
                StartAt = pack.StartAt,
                EndAt = pack.EndAt,
                TaskId = pack.TaskId,
                Version = pack.Version ?? 0,
                IsExpired = pack.IsExpired ?? false,
                IPv6 = ipv6Map.TryGetValue(pack.Id, out var ipv6) && ipv6,
                Http3Enabled = http3Map.TryGetValue(pack.Id, out var http3) && http3,
                Status = status
            });
        }

        return ServiceResult<UserPackageListResult>.Ok(new UserPackageListResult(list));
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        long id,
        UserPackageUpdateRequest request,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var query = _db.Queryable<UserPackage>().Where(p => p.Id == id);
        if (isUserRequest)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            query = query.Where(p => p.Uid == userId.Value);
        }

        var pack = await query.FirstAsync();
        if (pack == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var changed = false;
        var name = request?.Name?.Trim();
        if (!string.IsNullOrWhiteSpace(name))
        {
            pack.Name = name;
            changed = true;
        }

        if (!string.IsNullOrWhiteSpace(request?.EndAt))
        {
            if (DateTime.TryParse(request.EndAt, out var endAt))
            {
                pack.EndAt = endAt;
                changed = true;
            }
        }

        if (request?.RegionId.HasValue == true)
        {
            if (request.RegionId > 0 || !isUserRequest)
            {
                pack.RegionId = (int)request.RegionId.Value;
                changed = true;
            }
        }

        if (request?.NodeGroupId.HasValue == true)
        {
            if (request.NodeGroupId > 0 || !isUserRequest)
            {
                pack.NodeGroupId = (int)request.NodeGroupId.Value;
                changed = true;
            }
        }

        if (request?.BackupGroupId.HasValue == true)
        {
            if (request.BackupGroupId > 0 || !isUserRequest)
            {
                pack.BackupNodeGroup = (int)request.BackupGroupId.Value;
                changed = true;
            }
        }

        if (request?.Traffic != null)
        {
            pack.Traffic = ParseIntValue(request.Traffic);
            changed = true;
        }

        if (request?.Bandwidth != null)
        {
            pack.Bandwidth = request.Bandwidth;
            changed = true;
        }

        if (request?.Connection != null)
        {
            pack.Connection = ParseIntValue(request.Connection);
            changed = true;
        }

        if (request?.Domain != null)
        {
            pack.Domain = ParseIntValue(request.Domain);
            changed = true;
        }

        if (request?.MainDomainLimit != null)
        {
            pack.MainDomainLimit = ParseIntValue(request.MainDomainLimit);
            changed = true;
        }

        if (request?.HttpPort != null)
        {
            pack.HttpPort = ParseIntValue(request.HttpPort);
            changed = true;
        }

        if (request?.StreamPort != null)
        {
            pack.StreamPort = ParseIntValue(request.StreamPort);
            changed = true;
        }

        if (request?.CustomCcRule.HasValue == true)
        {
            pack.CustomCcRule = request.CustomCcRule;
            changed = true;
        }

        if (request?.Websocket.HasValue == true)
        {
            pack.Websocket = request.Websocket;
            changed = true;
        }

        if (!isUserRequest)
        {
            if (request?.PriceMonthly.HasValue == true)
            {
                pack.MonthPrice = Convert.ToInt64(request.PriceMonthly.Value);
                changed = true;
            }
            if (request?.PriceQuarterly.HasValue == true)
            {
                pack.QuarterPrice = Convert.ToInt64(request.PriceQuarterly.Value);
                changed = true;
            }
            if (request?.PriceYearly.HasValue == true)
            {
                pack.YearPrice = Convert.ToInt64(request.PriceYearly.Value);
                changed = true;
            }
        }

        var cnameHostname = request?.CnameHostname?.Trim();
        var cnameDomain = request?.CnameDomain?.Trim();
        var cnameMode = request?.CnameMode?.Trim();
        if (!string.IsNullOrWhiteSpace(cnameHostname) || !isUserRequest)
        {
            pack.CnameHostname = cnameHostname;
            changed = true;
        }
        if (!string.IsNullOrWhiteSpace(cnameDomain) || !isUserRequest)
        {
            pack.CnameDomain = cnameDomain;
            changed = true;
        }
        if (!string.IsNullOrWhiteSpace(cnameMode) || !isUserRequest)
        {
            pack.CnameMode = cnameMode;
            changed = true;
        }

        if (changed)
        {
            var rows = await _db.Updateable(pack).ExecuteCommandAsync();
            if (rows <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
            }
        }

        if (request?.IPv6.HasValue == true)
        {
            if (!await SaveUserPackageBoolConfigAsync(id, "ipv6", request.IPv6.Value))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.ConfigError, "config_error");
            }
        }
        if (request?.Http3Enabled.HasValue == true)
        {
            if (!await SaveUserPackageBoolConfigAsync(id, "http3_enabled", request.Http3Enabled.Value))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.ConfigError, "config_error");
            }
        }

        await _syncService.SyncAsync(id, "update", cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<RenewUserPackageResult>> RenewAsync(
        long id,
        RenewUserPackageRequest request,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<RenewUserPackageResult>.Fail(ErrorCodes.InvalidParam);
        }

        var months = request?.Months ?? 0;
        if (months <= 0)
        {
            var period = request?.Period?.Trim().ToLowerInvariant();
            months = period switch
            {
                "month" => 1,
                "quarter" => 3,
                "year" => 12,
                _ => 0
            };
        }
        if (months <= 0)
        {
            return ServiceResult<RenewUserPackageResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var query = _db.Queryable<UserPackage>().Where(p => p.Id == id);
        if (isUserRequest)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<RenewUserPackageResult>.Fail(ErrorCodes.PermissionDenied);
            }
            query = query.Where(p => p.Uid == userId.Value);
        }

        var pack = await query.FirstAsync();
        if (pack == null)
        {
            return ServiceResult<RenewUserPackageResult>.Fail(ErrorCodes.NotFound);
        }

        var now = DateTime.Now;
        var baseTime = pack.EndAt.HasValue && pack.EndAt.Value > now ? pack.EndAt.Value : now;
        var newEnd = baseTime.AddMonths(months);

        pack.EndAt = newEnd;
        var rows = await _db.Updateable(pack).UpdateColumns(p => new { p.EndAt }).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<RenewUserPackageResult>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        await _syncService.SyncAsync(id, "renew", cancellationToken);
        return ServiceResult<RenewUserPackageResult>.Ok(new RenewUserPackageResult(newEnd));
    }

    public async Task<ServiceResult<bool>> SwitchAsync(
        long id,
        SwitchUserPackageRequest request,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }
        if (request?.PackageId is null or <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "package_id_required");
        }

        var query = _db.Queryable<UserPackage>().Where(p => p.Id == id);
        if (isUserRequest)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            query = query.Where(p => p.Uid == userId.Value);
        }

        var pack = await query.FirstAsync();
        if (pack == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var target = await _db.Queryable<Package>().Where(p => p.Id == request.PackageId).FirstAsync();
        if (target == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        Package? currentPkg = null;
        if (pack.Package is > 0)
        {
            currentPkg = await _db.Queryable<Package>().Where(p => p.Id == pack.Package).FirstAsync();
        }

        var changeType = ClassifyPackageChange(pack, currentPkg, target);
        var allowUpgrade = true;
        var allowDowngrade = true;
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (cfg.TryGetValue("package_allow_upgrade", out var allowUpVal))
        {
            allowUpgrade = _systemConfigService.ParseBoolFlag(allowUpVal);
        }
        if (cfg.TryGetValue("package_allow_downgrade", out var allowDownVal))
        {
            allowDowngrade = _systemConfigService.ParseBoolFlag(allowDownVal);
        }

        if (changeType == "upgrade" && !allowUpgrade)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied, "upgrade_disabled");
        }
        if (changeType == "downgrade" && !allowDowngrade)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied, "downgrade_disabled");
        }

        pack.Name = target.Name;
        pack.Package = target.Id;
        pack.RegionId = target.RegionId;
        pack.NodeGroupId = target.NodeGroupId;
        pack.BackupNodeGroup = target.BackupNodeGroup;
        pack.Traffic = target.Traffic;
        pack.Bandwidth = target.Bandwidth;
        pack.Connection = target.Connection;
        pack.Domain = target.Domain;
        pack.CustomCcRule = target.CustomCcRule;
        pack.Websocket = target.Websocket;
        pack.L2Origin = target.L2Origin;
        pack.MonthPrice = target.MonthPrice;
        pack.QuarterPrice = target.QuarterPrice;
        pack.YearPrice = target.YearPrice;

        var rows = await _db.Updateable(pack).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        await _syncService.SyncAsync(id, "upgrade", cancellationToken);
        await _siteCnameSyncService.ResyncSitesForUserPackageAsync(id, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task<Dictionary<long, bool>> LoadUserPackageBoolConfigAsync(IReadOnlyList<UserPackage> packs, string name)
    {
        var result = new Dictionary<long, bool>();
        if (packs.Count == 0)
        {
            return result;
        }

        var ids = packs.Select(p => p.Id).ToList();
        var configs = await _db.Queryable<Config>()
            .Where(c => c.Type == ConfigType && c.ScopeName == ConfigScopeName && c.Name == name && ids.Contains(c.ScopeId!.Value))
            .ToListAsync();

        foreach (var cfg in configs)
        {
            if (!cfg.ScopeId.HasValue)
            {
                continue;
            }
            result[cfg.ScopeId.Value] = ParseBoolString(cfg.Value);
        }

        return result;
    }

    private async Task<bool> SaveUserPackageBoolConfigAsync(long userPackageId, string name, bool value)
    {
        if (userPackageId <= 0)
        {
            return true;
        }

        var val = value ? "1" : "0";
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == name && c.Type == ConfigType && c.ScopeName == ConfigScopeName && c.ScopeId == userPackageId)
            .FirstAsync();

        var now = DateTime.Now;
        if (existing == null)
        {
            var cfg = new Config
            {
                Name = name,
                Value = val,
                Type = ConfigType,
                ScopeId = (int)userPackageId,
                ScopeName = ConfigScopeName,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            return await _db.Insertable(cfg).ExecuteCommandAsync() > 0;
        }

        existing.Value = val;
        existing.Enable = true;
        existing.UpdateAt = now;
        return await _db.Updateable(existing).ExecuteCommandAsync() > 0;
    }

    private static bool ParseBoolString(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }
        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static int ParseIntValue(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return 0;
        }
        return int.TryParse(raw.Trim(), out var parsed) ? parsed : 0;
    }

    private static string ClassifyPackageChange(UserPackage current, Package? currentPkg, Package target)
    {
        var currentScore = currentPkg != null ? PackageScore(currentPkg) : UserPackageScore(current);
        var targetScore = PackageScore(target);
        return ComparePackageScore(currentScore, targetScore);
    }

    private static string ComparePackageScore(double current, double target)
    {
        const double epsilon = 0.0001;
        if (target > current + epsilon)
        {
            return "upgrade";
        }
        if (target < current - epsilon)
        {
            return "downgrade";
        }
        return "same";
    }

    private static double PackageScore(Package pkg)
    {
        var price = NormalizedPrice(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice);
        if (price > 0)
        {
            return price;
        }
        return ResourceScore(pkg.Traffic, pkg.Bandwidth, pkg.Connection, pkg.Domain);
    }

    private static double UserPackageScore(UserPackage pkg)
    {
        var price = NormalizedPrice(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice);
        if (price > 0)
        {
            return price;
        }
        return ResourceScore(pkg.Traffic, pkg.Bandwidth, pkg.Connection, pkg.Domain);
    }

    private static double NormalizedPrice(long? month, long? quarter, long? year)
    {
        if (month.GetValueOrDefault() > 0)
        {
            return month!.Value;
        }
        if (quarter.GetValueOrDefault() > 0)
        {
            return quarter!.Value / 3d;
        }
        if (year.GetValueOrDefault() > 0)
        {
            return year!.Value / 12d;
        }
        return 0;
    }

    private static double ResourceScore(int? traffic, string? bandwidth, int? connection, int? domain)
    {
        var score = (traffic ?? 0) + (connection ?? 0) + (domain ?? 0);
        return score + ParseBandwidthMbps(bandwidth);
    }

    private static double ParseBandwidthMbps(string? raw)
    {
        var value = raw?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(value) || value is "0" or "unlimited" or "unlimit")
        {
            return 0;
        }

        var multiplier = 1d;
        if (value.EndsWith("g", StringComparison.Ordinal))
        {
            multiplier = 1024d;
            value = value[..^1];
        }
        else if (value.EndsWith("m", StringComparison.Ordinal))
        {
            value = value[..^1];
        }
        else if (value.EndsWith("k", StringComparison.Ordinal))
        {
            multiplier = 1d / 1024d;
            value = value[..^1];
        }

        return double.TryParse(value.Trim(), out var parsed) ? parsed * multiplier : 0d;
    }

    private async Task<string?> GenerateUniqueRecordIdAsync()
    {
        for (var i = 0; i < 5; i++)
        {
            var candidate = DomainHelper.GenerateToken(8);
            var count = await _db.Queryable<UserPackage>().CountAsync(p => p.RecordId == candidate);
            if (count == 0)
            {
                return candidate;
            }
        }
        return null;
    }
}
