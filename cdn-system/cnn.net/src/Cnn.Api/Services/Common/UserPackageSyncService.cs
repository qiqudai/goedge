using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Localization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IUserPackageSyncService
{
    Task SyncAsync(long userPackageId, string trigger, CancellationToken cancellationToken);
}

public sealed class UserPackageSyncService : IUserPackageSyncService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IMessageLocalizer _localizer;

    public UserPackageSyncService(
        ISqlSugarClient db,
        ISystemConfigService systemConfigService,
        IMessageLocalizer localizer)
    {
        _db = db;
        _systemConfigService = systemConfigService;
        _localizer = localizer;
    }

    public async Task SyncAsync(long userPackageId, string trigger, CancellationToken cancellationToken)
    {
        if (userPackageId <= 0)
        {
            return;
        }

        var package = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == userPackageId)
            .FirstAsync();
        if (package == null)
        {
            return;
        }

        var version = (package.Version ?? 0) + 1;
        package.Version = version;
        await _db.Updateable<UserPackage>()
            .SetColumns(p => new UserPackage { Version = version })
            .Where(p => p.Id == userPackageId)
            .ExecuteCommandAsync();

        var config = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var expireCloseEnabled = true;
        if (config.TryGetValue("package_expire_close_site", out var expireRaw))
        {
            expireCloseEnabled = _systemConfigService.ParseBoolFlag(expireRaw);
        }

        var status = "active";
        var now = DateTime.Now;
        if (expireCloseEnabled)
        {
            if (string.Equals(trigger, "expire", StringComparison.OrdinalIgnoreCase) ||
                (package.EndAt.HasValue && package.EndAt.Value < now))
            {
                status = "expired";
            }
        }

        var agentConfig = new AgentPackageConfigDto
        {
            PackageId = package.Id,
            Uid = package.Uid ?? 0,
            Version = version,
            Status = status,
            RegionId = package.RegionId ?? 0,
            NodeGroupId = package.NodeGroupId ?? 0,
            BackupNodeGroup = package.BackupNodeGroup ?? 0,
            EnableBackup = package.EnableBackupGroup == true ? 1 : 0,
            Cname = new AgentPackageCnameDto
            {
                Domain = package.CnameDomain,
                Hostname = package.CnameHostname,
                Hostname2 = package.CnameHostname2,
                Mode = package.CnameMode,
                RecordId = package.RecordId
            },
            Limits = new AgentPackageLimitsDto
            {
                Traffic = package.Traffic ?? 0,
                Bandwidth = package.Bandwidth,
                Connection = package.Connection ?? 0,
                Domain = package.Domain ?? 0
            },
            Features = new AgentPackageFeaturesDto
            {
                HttpPort = package.HttpPort ?? 0,
                StreamPort = package.StreamPort ?? 0,
                Websocket = package.Websocket ?? false,
                CustomCcRule = package.CustomCcRule ?? false,
                L2Origin = package.L2Origin ?? false
            },
            Time = new AgentPackageTimeDto
            {
                StartAt = (package.StartAt ?? DateTime.MinValue).ToString("yyyy-MM-dd HH:mm:ss"),
                EndAt = (package.EndAt ?? DateTime.MinValue).ToString("yyyy-MM-dd HH:mm:ss")
            }
        };

        var payload = new AgentPackagePayloadDto
        {
            Packages = new List<AgentPackageItemDto>
            {
                new()
                {
                    PackageId = package.Id,
                    Version = version,
                    Config = agentConfig
                }
            }
        };

        var data = JsonSerializer.Serialize(payload, JsonOptions);
        var nodeIds = await ResolveNodeIdsAsync(package);
        var targets = TaskTargets.Create(nodeIds);

        var taskState = targets.Total == 0 ? "done" : "waiting";
        DateTime? endAt = targets.Total == 0 ? now : null;

        var task = new Cnn.Domain.Entities.Task
        {
            Type = _localizer.Translate("agent.task_sync_package", null),
            Name = _localizer.Translate("task.sync_package_prefix", null) + package.Id,
            Data = data,
            TargetsJson = targets.Marshal(),
            State = taskState,
            Enable = true,
            CreateAt = now,
            EndAt = endAt
        };

        await _db.Insertable(task).ExecuteCommandAsync();
    }

    private async Task<List<long>> ResolveNodeIdsAsync(UserPackage package)
    {
        var groupIds = new List<int>();
        if (package.NodeGroupId.HasValue && package.NodeGroupId.Value > 0)
        {
            groupIds.Add(package.NodeGroupId.Value);
        }
        if (package.EnableBackupGroup == true && package.BackupNodeGroup.HasValue && package.BackupNodeGroup.Value > 0)
        {
            groupIds.Add(package.BackupNodeGroup.Value);
        }

        if (groupIds.Count == 0)
        {
            return new List<long>();
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => groupIds.Contains(l.NodeGroupId ?? 0) && l.Enable == true)
            .Select(l => new { l.NodeId, l.NodeIpId })
            .ToListAsync();

        var nodeIdSet = new HashSet<int>();
        foreach (var line in lines)
        {
            if (line.NodeId.HasValue && line.NodeId.Value > 0)
            {
                nodeIdSet.Add(line.NodeId.Value);
            }
            if (line.NodeIpId.HasValue && line.NodeIpId.Value > 0)
            {
                nodeIdSet.Add(line.NodeIpId.Value);
            }
        }

        if (nodeIdSet.Count == 0)
        {
            return new List<long>();
        }

        var enabled = await _db.Queryable<Node>()
            .Where(n => nodeIdSet.Contains(n.Id) && n.Enable == true)
            .Select(n => n.Id)
            .ToListAsync();

        return enabled.Select(id => (long)id).ToList();
    }

}
