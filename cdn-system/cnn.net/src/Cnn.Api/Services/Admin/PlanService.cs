using System.Security.Cryptography;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public interface IPlanService
{
    Task<ServiceResult<PlanListResult>> ListAsync(CancellationToken cancellationToken);
    Task<ServiceResult<PlanDetailDto>> GetAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(JsonElement payload, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, JsonElement payload, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<UserPlanListResult>> ListUserPlansAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> AssignUserPlanAsync(AssignUserPlanRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateUserPlanAsync(long id, JsonElement payload, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteUserPlansAsync(DeleteUserPlansRequest request, CancellationToken cancellationToken);
}

public sealed class PlanService : IPlanService
{
    private readonly ISqlSugarClient _db;
    private readonly IUserPackageSyncService _syncService;

    public PlanService(ISqlSugarClient db, IUserPackageSyncService syncService)
    {
        _db = db;
        _syncService = syncService;
    }

    public async Task<ServiceResult<PlanListResult>> ListAsync(CancellationToken cancellationToken)
    {
        var list = await _db.Queryable<Package>()
            .OrderBy(p => p.Sort, OrderByType.Asc)
            .OrderBy(p => p.Id, OrderByType.Desc)
            .ToListAsync();

        var items = list.Select(p => new PlanItemDto
        {
            Id = p.Id,
            Name = p.Name,
            Desc = p.Des,
            Group = "default",
            Region = p.RegionId ?? 0,
            LineGroup = p.NodeGroupId ?? 0,
            BackupGroup = p.BackupNodeGroup ?? 0,
            TrafficLimit = p.Traffic ?? 0,
            BandwidthLimit = p.Bandwidth,
            ConnectionLimit = p.Connection ?? 0,
            DomainLimit = p.Domain ?? 0,
            CustomCcRules = p.CustomCcRule ?? false,
            Websocket = p.Websocket ?? false,
            L2Origin = p.L2Origin ?? false,
            PriceMonthly = p.MonthPrice ?? 0,
            PriceQuarterly = p.QuarterPrice ?? 0,
            PriceYearly = p.YearPrice ?? 0,
            SortOrder = p.Sort ?? 0,
            Status = p.Enable ?? false
        }).ToList();

        return ServiceResult<PlanListResult>.Ok(new PlanListResult(items, items.Count));
    }

    public async Task<ServiceResult<PlanDetailDto>> GetAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<PlanDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var pkg = await _db.Queryable<Package>().Where(p => p.Id == id).FirstAsync();
        if (pkg == null)
        {
            return ServiceResult<PlanDetailDto>.Fail(ErrorCodes.NotFound, "plan_not_found");
        }

        var detail = new PlanDetailDto
        {
            Id = pkg.Id,
            Name = pkg.Name,
            Desc = pkg.Des,
            Group = "default",
            Region = pkg.RegionId ?? 0,
            LineGroup = pkg.NodeGroupId ?? 0,
            BackupGroup = pkg.BackupNodeGroup ?? 0,
            TrafficLimit = pkg.Traffic ?? 0,
            BandwidthLimit = pkg.Bandwidth,
            ConnectionLimit = pkg.Connection ?? 0,
            DomainLimit = pkg.Domain ?? 0,
            CustomCcRules = pkg.CustomCcRule ?? false,
            Websocket = pkg.Websocket ?? false,
            L2Origin = pkg.L2Origin ?? false,
            PriceMonthly = pkg.MonthPrice ?? 0,
            PriceQuarterly = pkg.QuarterPrice ?? 0,
            PriceYearly = pkg.YearPrice ?? 0,
            SortOrder = pkg.Sort ?? 0,
            Status = pkg.Enable ?? false,
            HttpPort = pkg.HttpPort ?? 0,
            StreamPort = pkg.StreamPort ?? 0,
            CnameDomain = pkg.CnameDomain,
            CnameHostname2 = pkg.CnameHostname2,
            CnameMode = pkg.CnameMode,
            BuyNumLimit = pkg.BuyNumLimit ?? 0,
            BackendIpLimit = pkg.BackendIpLimit,
            IdVerify = pkg.IdVerify ?? false,
            BeforeExpDaysRenew = pkg.BeforeExpDaysRenew ?? 0,
            Expire = pkg.Expire,
            Owner = pkg.Owner
        };

        return ServiceResult<PlanDetailDto>.Ok(detail);
    }

    public async Task<ServiceResult<bool>> CreateAsync(JsonElement payload, CancellationToken cancellationToken)
    {
        var name = GetString(payload, "name");
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "name_required");
        }

        var regionId = GetLong(payload, "region");
        var regionCheck = await EnsureRegionValid(regionId);
        if (regionCheck != null)
        {
            return regionCheck;
        }

        var nodeGroupId = GetLong(payload, "line_group");
        var groupCheck = await EnsureNodeGroupValid(nodeGroupId);
        if (groupCheck != null)
        {
            return groupCheck;
        }

        var backupGroupId = GetLong(payload, "backup_group");
        if (backupGroupId != 0)
        {
            if (backupGroupId == nodeGroupId)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
            }

            var backupCheck = await EnsureNodeGroupValid(backupGroupId);
            if (backupCheck != null)
            {
                return backupCheck;
            }
        }

        var now = DateTime.Now;
        var pkg = new Package
        {
            Name = name.Trim(),
            Des = GetString(payload, "desc"),
            RegionId = (int)regionId,
            NodeGroupId = (int)nodeGroupId,
            BackupNodeGroup = (int)backupGroupId,
            CnameDomain = GetString(payload, "cname_domain"),
            CnameHostname2 = GetString(payload, "cname_hostname2"),
            CnameMode = GetString(payload, "cname_mode"),
            MonthPrice = GetLong(payload, "price_monthly"),
            QuarterPrice = GetLong(payload, "price_quarterly"),
            YearPrice = GetLong(payload, "price_yearly"),
            Traffic = (int)GetLong(payload, "traffic_limit"),
            Bandwidth = GetString(payload, "bandwidth_limit"),
            Connection = (int)GetLong(payload, "connection_limit"),
            Domain = (int)GetLong(payload, "domain_limit"),
            HttpPort = (int)GetLong(payload, "http_port"),
            StreamPort = (int)GetLong(payload, "stream_port"),
            Expire = GetTimePtr(payload, "expire"),
            BuyNumLimit = (int)GetLong(payload, "buy_num_limit"),
            BackendIpLimit = GetString(payload, "backend_ip_limit"),
            IdVerify = GetBool(payload, "id_verify"),
            BeforeExpDaysRenew = (int)GetLong(payload, "before_exp_days_renew"),
            Websocket = GetBool(payload, "websocket"),
            CustomCcRule = GetBool(payload, "custom_cc_rules"),
            L2Origin = GetBool(payload, "l2_origin"),
            Sort = (int)GetLong(payload, "sort_order"),
            Owner = GetString(payload, "owner"),
            Enable = GetBool(payload, "status"),
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(pkg).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_create_error");
        }

        return ServiceResult<bool>.Ok(true);
    }
    public async Task<ServiceResult<bool>> UpdateAsync(long id, JsonElement payload, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var current = await _db.Queryable<Package>().Where(p => p.Id == id).FirstAsync();
        if (current == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "plan_not_found");
        }

        if (TryGetString(payload, "name", out var name))
        {
            current.Name = name;
        }
        if (TryGetString(payload, "desc", out var desc))
        {
            current.Des = desc;
        }
        if (HasProperty(payload, "region"))
        {
            var regionId = GetLong(payload, "region");
            var regionCheck = await EnsureRegionValid(regionId);
            if (regionCheck != null)
            {
                return regionCheck;
            }
            current.RegionId = (int)regionId;
        }
        if (HasProperty(payload, "line_group"))
        {
            var nodeGroupId = GetLong(payload, "line_group");
            var groupCheck = await EnsureNodeGroupValid(nodeGroupId);
            if (groupCheck != null)
            {
                return groupCheck;
            }

            var backupGroupId = (long)(current.BackupNodeGroup ?? 0);
            if (HasProperty(payload, "backup_group"))
            {
                backupGroupId = GetLong(payload, "backup_group");
            }
            if (backupGroupId != 0 && backupGroupId == nodeGroupId)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
            }
            current.NodeGroupId = (int)nodeGroupId;
        }
        if (HasProperty(payload, "backup_group"))
        {
            var backupGroupId = GetLong(payload, "backup_group");
            var nodeGroupId = current.NodeGroupId ?? 0;
            if (backupGroupId != 0)
            {
                if (backupGroupId == nodeGroupId)
                {
                    return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
                }

                var backupCheck = await EnsureNodeGroupValid(backupGroupId);
                if (backupCheck != null)
                {
                    return backupCheck;
                }
            }
            current.BackupNodeGroup = (int)backupGroupId;
        }
        if (HasProperty(payload, "cname_domain"))
        {
            current.CnameDomain = GetString(payload, "cname_domain");
        }
        if (HasProperty(payload, "cname_hostname2"))
        {
            current.CnameHostname2 = GetString(payload, "cname_hostname2");
        }
        if (HasProperty(payload, "cname_mode"))
        {
            current.CnameMode = GetString(payload, "cname_mode");
        }
        if (HasProperty(payload, "price_monthly"))
        {
            current.MonthPrice = GetLong(payload, "price_monthly");
        }
        if (HasProperty(payload, "price_quarterly"))
        {
            current.QuarterPrice = GetLong(payload, "price_quarterly");
        }
        if (HasProperty(payload, "price_yearly"))
        {
            current.YearPrice = GetLong(payload, "price_yearly");
        }
        if (HasProperty(payload, "traffic_limit"))
        {
            current.Traffic = (int)GetLong(payload, "traffic_limit");
        }
        if (HasProperty(payload, "bandwidth_limit"))
        {
            current.Bandwidth = GetString(payload, "bandwidth_limit");
        }
        if (HasProperty(payload, "connection_limit"))
        {
            current.Connection = (int)GetLong(payload, "connection_limit");
        }
        if (HasProperty(payload, "domain_limit"))
        {
            current.Domain = (int)GetLong(payload, "domain_limit");
        }
        if (HasProperty(payload, "http_port"))
        {
            current.HttpPort = (int)GetLong(payload, "http_port");
        }
        if (HasProperty(payload, "stream_port"))
        {
            current.StreamPort = (int)GetLong(payload, "stream_port");
        }
        if (HasProperty(payload, "expire"))
        {
            current.Expire = GetTimeUpdateValue(payload, "expire");
        }
        if (HasProperty(payload, "buy_num_limit"))
        {
            current.BuyNumLimit = (int)GetLong(payload, "buy_num_limit");
        }
        if (HasProperty(payload, "backend_ip_limit"))
        {
            current.BackendIpLimit = GetString(payload, "backend_ip_limit");
        }
        if (HasProperty(payload, "id_verify"))
        {
            current.IdVerify = GetBool(payload, "id_verify");
        }
        if (HasProperty(payload, "before_exp_days_renew"))
        {
            current.BeforeExpDaysRenew = (int)GetLong(payload, "before_exp_days_renew");
        }
        if (HasProperty(payload, "websocket"))
        {
            current.Websocket = GetBool(payload, "websocket");
        }
        if (HasProperty(payload, "custom_cc_rules"))
        {
            current.CustomCcRule = GetBool(payload, "custom_cc_rules");
        }
        if (HasProperty(payload, "l2_origin"))
        {
            current.L2Origin = GetBool(payload, "l2_origin");
        }
        if (HasProperty(payload, "sort_order"))
        {
            current.Sort = (int)GetLong(payload, "sort_order");
        }
        if (HasProperty(payload, "owner"))
        {
            current.Owner = GetString(payload, "owner");
        }
        if (HasProperty(payload, "status"))
        {
            current.Enable = GetBool(payload, "status");
        }

        current.UpdateAt = DateTime.Now;

        var rows = await _db.Updateable(current).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var rows = await _db.Deleteable<Package>().Where(p => p.Id == id).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<UserPlanListResult>> ListUserPlansAsync(CancellationToken cancellationToken)
    {
        var plans = await _db.Queryable<UserPackage>()
            .OrderBy(p => p.Id, OrderByType.Desc)
            .ToListAsync();

        var userIds = plans.Select(p => (long)(p.Uid ?? 0)).Where(id => id > 0).Distinct().ToList();
        var packageIds = plans.Select(p => (long)(p.Package ?? 0)).Where(id => id > 0).Distinct().ToList();

        var userNameMap = new Dictionary<long, string>();
        if (userIds.Count > 0)
        {
            var users = await _db.Queryable<User>().Where(u => userIds.Contains(u.Id)).ToListAsync();
            foreach (var user in users)
            {
                var display = user.Name?.Trim();
                if (string.IsNullOrWhiteSpace(display))
                {
                    display = user.Email?.Trim();
                }
                if (string.IsNullOrWhiteSpace(display))
                {
                    display = user.Phone?.Trim();
                }
                if (string.IsNullOrWhiteSpace(display))
                {
                    display = user.Qq?.Trim();
                }
                if (!string.IsNullOrWhiteSpace(display))
                {
                    userNameMap[user.Id] = display;
                }
            }
        }

        var packageNameMap = new Dictionary<long, string>();
        if (packageIds.Count > 0)
        {
            var packages = await _db.Queryable<Package>().Where(p => packageIds.Contains(p.Id)).ToListAsync();
            foreach (var pkg in packages)
            {
                if (!string.IsNullOrWhiteSpace(pkg.Name))
                {
                    packageNameMap[pkg.Id] = pkg.Name!;
                }
            }
        }

        var now = DateTime.Now;
        var list = new List<UserPlanItemDto>(plans.Count);
        foreach (var plan in plans)
        {
            var status = "active";
            if (plan.EndAt.HasValue && plan.EndAt.Value < now)
            {
                status = "expired";
            }

            var recordId = plan.RecordId?.Trim();
            if (string.IsNullOrWhiteSpace(recordId))
            {
                recordId = await GenerateUniqueRecordIdAsync();
                if (!string.IsNullOrWhiteSpace(recordId))
                {
                    plan.RecordId = recordId;
                    await _db.Updateable(plan).UpdateColumns(p => new { p.RecordId }).ExecuteCommandAsync();
                }
            }

            var packageId = plan.Package ?? 0;
            var packageName = packageId > 0 && packageNameMap.TryGetValue(packageId, out var nameValue)
                ? nameValue
                : plan.Name;

            var startAt = plan.StartAt ?? plan.CreateAt;

            list.Add(new UserPlanItemDto
            {
                Id = plan.Id,
                UserId = plan.Uid ?? 0,
                UserName = userNameMap.TryGetValue(plan.Uid ?? 0, out var userName) ? userName : null,
                PackageId = packageId,
                PackageName = packageName,
                PlanName = plan.Name,
                RecordId = recordId,
                RegionId = plan.RegionId ?? 0,
                NodeGroupId = plan.NodeGroupId ?? 0,
                BackupGroupId = plan.BackupNodeGroup ?? 0,
                EnableBackupGroup = plan.EnableBackupGroup ?? false,
                Traffic = plan.Traffic ?? 0,
                Bandwidth = plan.Bandwidth,
                Connection = plan.Connection ?? 0,
                Domain = plan.Domain ?? 0,
                MainDomainLimit = plan.MainDomainLimit ?? 0,
                HttpPort = plan.HttpPort ?? 0,
                StreamPort = plan.StreamPort ?? 0,
                CustomCcRule = plan.CustomCcRule ?? false,
                Websocket = plan.Websocket ?? false,
                CnameDomain = plan.CnameDomain,
                CnameHostname = plan.CnameHostname,
                CnameHostname2 = plan.CnameHostname2,
                CnameMode = plan.CnameMode,
                StartAt = startAt,
                EndAt = plan.EndAt,
                Status = status,
                CreatedAt = plan.CreateAt
            });
        }

        return ServiceResult<UserPlanListResult>.Ok(new UserPlanListResult(list));
    }
    public async Task<ServiceResult<bool>> AssignUserPlanAsync(AssignUserPlanRequest request, CancellationToken cancellationToken)
    {
        if (request == null || !request.PlanId.HasValue || !request.UserId.HasValue || request.PlanId <= 0 || request.UserId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "plan_user_required");
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == request.UserId).FirstAsync();
        if (user == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        var pkg = await _db.Queryable<Package>().Where(p => p.Id == request.PlanId).FirstAsync();
        if (pkg == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "plan_not_found");
        }

        var regionCheck = await EnsureRegionValid(pkg.RegionId ?? 0);
        if (regionCheck != null)
        {
            return regionCheck;
        }

        var groupCheck = await EnsureNodeGroupValid(pkg.NodeGroupId ?? 0);
        if (groupCheck != null)
        {
            return groupCheck;
        }

        if ((pkg.BackupNodeGroup ?? 0) != 0)
        {
            if (pkg.BackupNodeGroup == pkg.NodeGroupId)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
            }

            var backupCheck = await EnsureNodeGroupValid(pkg.BackupNodeGroup ?? 0);
            if (backupCheck != null)
            {
                return backupCheck;
            }
        }

        var now = DateTime.Now;
        var endAt = ResolveEndAt(now, request.DurationMonths, request.EndAt);
        if (!endAt.HasValue)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_end_at");
        }
        if (endAt.Value < now)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "end_at_future");
        }

        var userPackage = new UserPackage
        {
            Uid = (int)request.UserId.Value,
            Name = pkg.Name,
            Package = (int)pkg.Id,
            RegionId = pkg.RegionId,
            NodeGroupId = pkg.NodeGroupId,
            BackupNodeGroup = pkg.BackupNodeGroup,
            EnableBackupGroup = false,
            CnameDomain = pkg.CnameDomain,
            CnameHostname2 = pkg.CnameHostname2,
            CnameMode = pkg.CnameMode,
            Traffic = pkg.Traffic,
            Bandwidth = pkg.Bandwidth,
            Connection = pkg.Connection,
            Domain = pkg.Domain,
            HttpPort = pkg.HttpPort,
            StreamPort = pkg.StreamPort,
            CustomCcRule = pkg.CustomCcRule,
            Websocket = pkg.Websocket,
            L2Origin = pkg.L2Origin,
            MonthPrice = pkg.MonthPrice,
            QuarterPrice = pkg.QuarterPrice,
            YearPrice = pkg.YearPrice,
            CreateAt = now,
            StartAt = now,
            EndAt = endAt
        };

        if (string.IsNullOrWhiteSpace(userPackage.RecordId))
        {
            var recordId = await GenerateUniqueRecordIdAsync();
            if (string.IsNullOrWhiteSpace(recordId))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InternalError);
            }
            userPackage.RecordId = recordId;
        }

        var id = await _db.Insertable(userPackage).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_create_error");
        }

        await _syncService.SyncAsync(id, "update", cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateUserPlanAsync(long id, JsonElement payload, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var current = await _db.Queryable<UserPackage>().Where(p => p.Id == id).FirstAsync();
        if (current == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var updated = false;
        if (HasProperty(payload, "name"))
        {
            current.Name = GetString(payload, "name");
            updated = true;
        }
        if (HasProperty(payload, "end_at"))
        {
            var endAt = GetTimePtr(payload, "end_at");
            if (endAt.HasValue)
            {
                current.EndAt = endAt;
                updated = true;
            }
            else
            {
                if (TryGetString(payload, "end_at", out var raw) && !string.IsNullOrWhiteSpace(raw))
                {
                    return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_end_at");
                }
            }
        }
        if (HasProperty(payload, "region_id"))
        {
            var regionId = GetLong(payload, "region_id");
            var regionCheck = await EnsureRegionValid(regionId);
            if (regionCheck != null)
            {
                return regionCheck;
            }
            current.RegionId = (int)regionId;
            updated = true;
        }
        if (HasProperty(payload, "node_group_id"))
        {
            var nodeGroupId = GetLong(payload, "node_group_id");
            var groupCheck = await EnsureNodeGroupValid(nodeGroupId);
            if (groupCheck != null)
            {
                return groupCheck;
            }

            var backupGroupId = (long)(current.BackupNodeGroup ?? 0);
            if (HasProperty(payload, "backup_group_id"))
            {
                backupGroupId = GetLong(payload, "backup_group_id");
            }
            if (backupGroupId != 0 && backupGroupId == nodeGroupId)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
            }

            current.NodeGroupId = (int)nodeGroupId;
            updated = true;
        }
        if (HasProperty(payload, "backup_group_id"))
        {
            var backupGroupId = GetLong(payload, "backup_group_id");
            var nodeGroupId = current.NodeGroupId ?? 0;
            if (backupGroupId != 0)
            {
                if (backupGroupId == nodeGroupId)
                {
                    return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "backup_group_conflict");
                }

                var backupCheck = await EnsureNodeGroupValid(backupGroupId);
                if (backupCheck != null)
                {
                    return backupCheck;
                }
            }

            current.BackupNodeGroup = (int)backupGroupId;
            if (!HasProperty(payload, "enable_backup_group"))
            {
                current.EnableBackupGroup = backupGroupId > 0;
            }
            updated = true;
        }
        if (HasProperty(payload, "enable_backup_group"))
        {
            current.EnableBackupGroup = GetBool(payload, "enable_backup_group");
            updated = true;
        }
        if (HasProperty(payload, "main_domain_limit"))
        {
            current.MainDomainLimit = (int)GetLong(payload, "main_domain_limit");
            updated = true;
        }
        if (HasProperty(payload, "cname_domain"))
        {
            current.CnameDomain = GetString(payload, "cname_domain");
            updated = true;
        }
        if (HasProperty(payload, "cname_hostname"))
        {
            current.CnameHostname = GetString(payload, "cname_hostname");
            updated = true;
        }
        if (HasProperty(payload, "cname_mode"))
        {
            var previousMode = string.IsNullOrWhiteSpace(current.CnameMode) ? "domain" : current.CnameMode.Trim();
            var newMode = GetString(payload, "cname_mode");
            current.CnameMode = newMode;

            if (!string.IsNullOrWhiteSpace(newMode) && !string.Equals(newMode.Trim(), previousMode, StringComparison.OrdinalIgnoreCase))
            {
                await _db.Updateable<Site>()
                    .SetColumns(s => new Site { CnameMode = previousMode })
                    .Where(s => s.UserPackage == current.Id && (s.CnameMode == null || s.CnameMode == string.Empty))
                    .ExecuteCommandAsync();
            }
            updated = true;
        }

        if (!updated)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "no_updates");
        }

        var rows = await _db.Updateable(current).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        await _syncService.SyncAsync(id, "update", cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteUserPlansAsync(DeleteUserPlansRequest request, CancellationToken cancellationToken)
    {
        if (request?.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        var rows = await _db.Deleteable<UserPackage>()
            .Where(p => request.Ids.Contains(p.Id))
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private async Task<ServiceResult<bool>?> EnsureRegionValid(long regionId)
    {
        if (regionId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "region_required");
        }

        var exists = await _db.Queryable<Region>().AnyAsync(r => r.Id == regionId);
        if (!exists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "region_not_found");
        }

        return null;
    }

    private async Task<ServiceResult<bool>?> EnsureNodeGroupValid(long nodeGroupId)
    {
        if (nodeGroupId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "line_group_required");
        }

        var exists = await _db.Queryable<NodeGroup>().AnyAsync(n => n.Id == nodeGroupId);
        if (!exists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "line_group_not_found");
        }

        return null;
    }

    private static DateTime? ResolveEndAt(DateTime now, int? durationMonths, string? endAt)
    {
        if (!string.IsNullOrWhiteSpace(endAt))
        {
            return ParseTimeString(endAt);
        }

        var months = durationMonths.GetValueOrDefault();
        if (months <= 0)
        {
            months = 1;
        }
        return now.AddMonths(months);
    }

    private async Task<string?> GenerateUniqueRecordIdAsync()
    {
        for (var i = 0; i < 5; i++)
        {
            var candidate = RandomToken(8);
            if (string.IsNullOrWhiteSpace(candidate))
            {
                continue;
            }
            var count = await _db.Queryable<UserPackage>().CountAsync(p => p.RecordId == candidate);
            if (count == 0)
            {
                return candidate;
            }
        }
        return null;
    }

    private static string RandomToken(int length)
    {
        const string chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        var buffer = new byte[length];
        RandomNumberGenerator.Fill(buffer);
        var output = new char[length];
        for (var i = 0; i < length; i++)
        {
            output[i] = chars[buffer[i] % chars.Length];
        }
        return new string(output);
    }

    private static bool HasProperty(JsonElement payload, string key)
    {
        return payload.ValueKind == JsonValueKind.Object && payload.TryGetProperty(key, out _);
    }

    private static bool TryGetString(JsonElement payload, string key, out string? value)
    {
        value = null;
        if (!HasProperty(payload, key))
        {
            return false;
        }
        value = GetString(payload, key);
        return true;
    }

    private static string? GetString(JsonElement payload, string key)
    {
        if (!payload.TryGetProperty(key, out var value))
        {
            return null;
        }
        return GetString(value);
    }

    private static string? GetString(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString()?.Trim(),
            JsonValueKind.Number => value.TryGetInt64(out var number) ? number.ToString() : value.GetRawText(),
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            _ => value.GetRawText()?.Trim('"')
        };
    }

    private static long GetLong(JsonElement payload, string key)
    {
        if (!payload.TryGetProperty(key, out var value))
        {
            return 0;
        }
        return GetLong(value);
    }

    private static long GetLong(JsonElement value)
    {
        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var number))
        {
            return number;
        }
        if (value.ValueKind == JsonValueKind.String)
        {
            var raw = value.GetString();
            if (long.TryParse(raw?.Trim(), out var parsed))
            {
                return parsed;
            }
        }
        return 0;
    }

    private static bool GetBool(JsonElement payload, string key)
    {
        if (!payload.TryGetProperty(key, out var value))
        {
            return false;
        }
        return GetBool(value);
    }

    private static bool GetBool(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            JsonValueKind.Number => value.TryGetInt64(out var num) && num != 0,
            JsonValueKind.String => ParseBoolString(value.GetString()),
            _ => false
        };
    }

    private static bool ParseBoolString(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return false;
        }
        var normalized = input.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static DateTime? GetTimePtr(JsonElement payload, string key)
    {
        if (!payload.TryGetProperty(key, out var value))
        {
            return null;
        }
        if (value.ValueKind == JsonValueKind.String)
        {
            var raw = value.GetString()?.Trim();
            if (string.IsNullOrWhiteSpace(raw))
            {
                return null;
            }
            if (raw == "0000-00-00" || raw == "0000-00-00 00:00:00")
            {
                return null;
            }
            return ParseTimeString(raw);
        }
        return null;
    }

    private static DateTime? GetTimeUpdateValue(JsonElement payload, string key)
    {
        if (!payload.TryGetProperty(key, out var value))
        {
            return null;
        }
        if (value.ValueKind == JsonValueKind.String)
        {
            var raw = value.GetString()?.Trim();
            if (string.IsNullOrWhiteSpace(raw))
            {
                return null;
            }
            if (raw == "0000-00-00" || raw == "0000-00-00 00:00:00")
            {
                return null;
            }
            return ParseTimeString(raw);
        }
        return null;
    }

    private static DateTime? ParseTimeString(string input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return null;
        }
        if (DateTime.TryParse(input, out var parsed))
        {
            return parsed;
        }
        return null;
    }
}
