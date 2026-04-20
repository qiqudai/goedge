using System.Globalization;
using System.Text.Json;
using System.Text.Json.Serialization;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class NodeService : INodeService
{
    private readonly ISqlSugarClient _db;
    private readonly IConfiguration _configuration;
    private readonly IConfigVersionService _configVersionService;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly IAdminEventPublisher _eventPublisher;
    private readonly INodeStatusService _nodeStatusService;
    private readonly INodeConfigService _nodeConfigService;

    private const string InstallProgressType = "install_progress";
    private const string InstallProgressScopeName = "node";

    public NodeService(
        ISqlSugarClient db,
        IConfiguration configuration,
        IConfigVersionService configVersionService,
        IDnsSyncService dnsSyncService,
        IAdminEventPublisher eventPublisher,
        INodeStatusService nodeStatusService,
        INodeConfigService nodeConfigService
    )
    {
        _db = db;
        _configuration = configuration;
        _configVersionService = configVersionService;
        _dnsSyncService = dnsSyncService;
        _eventPublisher = eventPublisher;
        _nodeStatusService = nodeStatusService;
        _nodeConfigService = nodeConfigService;
    }

    public async Task<ServiceResult<NodeListResult>> ListAsync(NodeListQuery query, CancellationToken cancellationToken)
    {
        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.PageSize < 1 ? 20 : query.PageSize;

        var q = _db.Queryable<Node>().Where(n => n.Pid == 0);

        if (query.RegionId is > 0)
        {
            q = q.Where(n => n.RegionId == query.RegionId);
        }

        if (!string.IsNullOrWhiteSpace(query.Status))
        {
            if (string.Equals(query.Status, "enabled", StringComparison.OrdinalIgnoreCase))
            {
                q = q.Where(n => n.Enable == true);
            }
            else if (string.Equals(query.Status, "disabled", StringComparison.OrdinalIgnoreCase))
            {
                q = q.Where(n => n.Enable == false);
            }
        }

        if (query.NodeType is > 0)
        {
            q = q.Where(n => n.Level == query.NodeType);
        }

        if (!string.IsNullOrWhiteSpace(query.Keyword))
        {
            var keyword = query.Keyword!.Trim().ToLowerInvariant();
            if (long.TryParse(keyword, out var id) && id > 0)
            {
                q = q.Where(n => n.Id == id || SqlFunc.ToLower(n.Name)!.Contains(keyword) || n.Ip!.Contains(keyword));
            }
            else
            {
                q = q.Where(n => SqlFunc.ToLower(n.Name)!.Contains(keyword) || SqlFunc.ToLower(n.Ip)!.Contains(keyword));
            }
        }

        var total = await q.CountAsync();
        var nodes = await q.OrderBy(n => n.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var parentIds = nodes.Select(n => (long)n.Id).ToList();
        var subIpMap = await LoadSubIpMapAsync(parentIds);
        var lineCountMap = await LoadLineCountMapAsync(parentIds);
        var regionNameMap = await LoadRegionNameMapAsync(nodes);

        var list = nodes.Select(n =>
        {
            var id = (long)n.Id;
            return new NodeListItem
            {
                Id = id,
                Pid = n.Pid,
                RegionId = n.RegionId,
                RegionName = n.RegionId.HasValue && regionNameMap.TryGetValue(n.RegionId.Value, out var name) ? name : null,
                Name = n.Name,
                Remark = n.Des,
                Ip = n.Ip,
                Token = n.Token,
                Host = n.Host,
                Port = n.Port,
                HttpProxy = n.HttpProxy,
                IsMgmt = n.IsMgmt,
                Enable = n.Enable,
                ConfigTask = n.ConfigTask,
                CheckOn = n.CheckOn,
                CheckProtocol = n.CheckProtocol,
                CheckTimeout = n.CheckTimeout,
                CheckPort = n.CheckPort,
                CheckHost = n.CheckHost,
                CheckPath = n.CheckPath,
                CheckNodeGroup = n.CheckNodeGroup,
                CheckAction = n.CheckAction,
                BwLimit = n.BwLimit,
                Type = n.Level,
                SortOrder = n.Sort,
                CacheDir = n.CacheDir,
                CacheLimit = n.MaxCacheSize,
                LogDir = n.LogDir,
                SshHost = n.SshHost,
                SshPort = n.SshPort,
                SshUser = n.SshUser,
                SshAuthType = n.SshAuthType,
                SshPassword = n.SshPassword,
                SshKey = n.SshKey,
                WorkDir = n.WorkDir,
                AutoInstall = n.AutoInstall,
                InstallStatus = n.InstallStatus,
                InstallError = n.InstallError,
                InstallAt = n.InstallAt,
                SubIps = subIpMap.TryGetValue(id, out var subs) ? subs : Array.Empty<NodeSubIp>(),
                LineCount = lineCountMap.TryGetValue(id, out var cnt) ? cnt : 0,
                Online = false,
                InstallStage = null,
                InstallProgress = null,
            InstallProgressBytes = null,
            InstallProgressTotal = null,
            AntiBlocking = true,
            ReportedAntiBlocking = null,
            ConfigDrift = false,
            ConfigDriftFields = null
        };
        }).ToList();

        var progressMap = await LoadInstallProgressMapAsync(parentIds);
        if (progressMap.Count > 0)
        {
            foreach (var item in list)
            {
                if (!progressMap.TryGetValue(item.Id, out var progress))
                {
                    continue;
                }

                item.InstallStage = progress.Stage;
                item.InstallProgress = progress.Percent;
                item.InstallProgressBytes = progress.CurrentBytes;
                item.InstallProgressTotal = progress.TotalBytes;
            }
        }

        var antiBlockingMap = await _nodeConfigService.GetMapAsync("anti_blocking", cancellationToken);
        var reportedConfigMap = await _nodeConfigService.GetMapAsync("reported_config", cancellationToken);

        foreach (var item in list)
        {
            item.Online = _nodeStatusService.IsOnline(item.Id, TimeSpan.FromSeconds(30));
            item.AntiBlocking = true;
            if (antiBlockingMap.TryGetValue(item.Id, out var antiRaw) && !string.IsNullOrWhiteSpace(antiRaw))
            {
                item.AntiBlocking = ParseBoolFlag(antiRaw);
            }

            if (reportedConfigMap.TryGetValue(item.Id, out var reportedRaw))
            {
                var reported = ExtractReportedAntiBlocking(reportedRaw);
                if (reported.HasValue)
                {
                    item.ReportedAntiBlocking = reported.Value;
                    if (reported.Value != item.AntiBlocking)
                    {
                        item.ConfigDrift = true;
                        item.ConfigDriftFields = new[] { "anti_blocking" };
                    }
                }
            }
        }

        return ServiceResult<NodeListResult>.Ok(new NodeListResult(list, total));
    }

    public async Task<ServiceResult<NodeMonitorLogResult>> ListMonitorLogsAsync(long nodeId, NodeMonitorLogQuery query, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return ServiceResult<NodeMonitorLogResult>.Fail(ErrorCodes.InvalidParam);
        }

        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.PageSize < 1 ? 20 : query.PageSize;

        var baseQuery = _db.Queryable<NodeMonitorLog>().Where(l => l.NodeId == nodeId);
        if (!string.IsNullOrWhiteSpace(query.Type))
        {
            baseQuery = baseQuery.Where(l => l.Type == query.Type);
        }

        var (startAt, endAt) = ResolveTimeRange(query);
        if (startAt.HasValue && endAt.HasValue)
        {
            baseQuery = baseQuery.Where(l => l.CreateAt >= startAt && l.CreateAt <= endAt);
        }

        var total = await baseQuery
            .GroupBy(l => new { l.EventId, l.CreateAt })
            .CountAsync();

        var list = await baseQuery
            .GroupBy(l => new { l.EventId, l.CreateAt })
            .Select(l => new NodeMonitorLogItem(
                l.CreateAt,
                SqlFunc.AggregateSum(SqlFunc.IIF(l.Success == "1", 0, 1)),
                SqlFunc.AggregateCount(l.NodeId)
            ))
            .OrderBy(l => l.CheckedAt, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        return ServiceResult<NodeMonitorLogResult>.Ok(new NodeMonitorLogResult(list, total));
    }

    public async Task<ServiceResult<NodeListItem>> CreateAsync(NodeCreateRequest request, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.Name) || string.IsNullOrWhiteSpace(request.Ip))
        {
            return ServiceResult<NodeListItem>.Fail(ErrorCodes.MissingParam);
        }

        int? regionId = request.RegionId.GetValueOrDefault() == 0 ? null : (int?)request.RegionId;
        var enable = request.Enable ?? true;
        if (!enable)
        {
            enable = true;
        }

        var token = ResolveAgentToken();
        if (string.IsNullOrWhiteSpace(token))
        {
            token = DomainHelper.GenerateToken(32);
        }

        var now = DateTime.Now;
        var node = new Node
        {
            Pid = 0,
            RegionId = regionId,
            Name = request.Name?.Trim(),
            Des = request.Remark?.Trim(),
            Ip = request.Ip?.Trim(),
            Token = token,
            Host = request.Host,
            Port = request.Port,
            HttpProxy = request.HttpProxy,
            IsMgmt = request.IsMgmt,
            Enable = enable,
            CheckOn = request.CheckOn,
            CheckProtocol = request.CheckProtocol,
            CheckTimeout = request.CheckTimeout,
            CheckPort = request.CheckPort,
            CheckHost = request.CheckHost,
            CheckPath = request.CheckPath,
            CheckNodeGroup = request.CheckNodeGroup,
            CheckAction = request.CheckAction,
            BwLimit = request.BwLimit,
            Level = request.Type,
            Sort = request.SortOrder,
            CacheDir = request.CacheDir,
            MaxCacheSize = request.CacheLimit,
            LogDir = request.LogDir,
            SshHost = request.SshHost,
            SshPort = request.SshPort,
            SshUser = request.SshUser,
            SshAuthType = request.SshAuthType,
            SshPassword = request.SshPassword,
            SshKey = request.SshKey,
            WorkDir = "/www/node",
            AutoInstall = request.AutoInstall,
            InstallStatus = request.AutoInstall == true ? "running" : "idle",
            InstallError = string.Empty,
            InstallAt = request.AutoInstall == true ? now : null,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(node).ExecuteReturnIdentityAsync();
        node.Id = id;

        await ReplaceSubIpsAsync(id, node, request.SubIps);

        if (enable)
        {
            await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { (long)id }, "add");
        }

        var result = await BuildNodeListItemAsync(node);
        return ServiceResult<NodeListItem>.Ok(result);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long nodeId, NodeUpdateRequest request, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var existing = await _db.Queryable<Node>()
            .Where(n => n.Id == nodeId)
            .Select(n => new { n.Id, n.Enable, n.RegionId })
            .FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        int? regionId = request.RegionId.GetValueOrDefault() == 0 ? null : (int?)request.RegionId;
        if ((existing.RegionId ?? 0) != (regionId ?? 0))
        {
            var lineCount = await _db.Queryable<Line>()
                .Where(l => l.NodeId == nodeId || l.NodeIpId == nodeId)
                .CountAsync();
            if (lineCount > 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.StateConflict, "node.region_change_blocked");
            }
        }

        var syncTask = string.Empty;
        if (request.Enable.HasValue && request.Enable.Value != existing.Enable)
        {
            syncTask = request.Enable.Value ? "sync_enable" : "sync_disable";
        }

        var now = DateTime.Now;
        await _db.Ado.UseTranAsync(async () =>
        {
            var current = await _db.Queryable<Node>().Where(n => n.Id == nodeId).FirstAsync();
            if (current == null)
            {
                return;
            }

            var enable = request.Enable ?? current.Enable;
            var update = new Node
            {
                Name = request.Name?.Trim(),
                Des = request.Remark?.Trim(),
                Ip = request.Ip?.Trim(),
                RegionId = regionId,
                Host = request.Host,
                Port = request.Port,
                HttpProxy = request.HttpProxy,
                IsMgmt = request.IsMgmt,
                Enable = enable,
                CheckOn = request.CheckOn,
                CheckProtocol = request.CheckProtocol,
                CheckTimeout = request.CheckTimeout,
                CheckPort = request.CheckPort,
                CheckHost = request.CheckHost,
                CheckPath = request.CheckPath,
                CheckNodeGroup = request.CheckNodeGroup,
                CheckAction = request.CheckAction,
                BwLimit = request.BwLimit,
                Level = request.Type,
                Sort = request.SortOrder,
                CacheDir = request.CacheDir,
                MaxCacheSize = request.CacheLimit,
                LogDir = request.LogDir,
                SshHost = request.SshHost,
                SshPort = request.SshPort,
                SshUser = request.SshUser,
                SshAuthType = request.SshAuthType,
                SshPassword = string.IsNullOrWhiteSpace(request.SshPassword) ? current.SshPassword : request.SshPassword,
                SshKey = string.IsNullOrWhiteSpace(request.SshKey) ? current.SshKey : request.SshKey,
                WorkDir = "/www/node",
                AutoInstall = request.AutoInstall,
                UpdateAt = now,
                ConfigTask = string.IsNullOrWhiteSpace(syncTask) ? current.ConfigTask : syncTask
            };

            await _db.Updateable(update).Where(n => n.Id == nodeId).ExecuteCommandAsync();

            var parent = new Node
            {
                RegionId = regionId,
                Name = request.Name?.Trim(),
                Des = request.Remark?.Trim(),
                Ip = request.Ip?.Trim(),
                Host = request.Host,
                Port = request.Port,
                HttpProxy = request.HttpProxy,
                IsMgmt = request.IsMgmt,
                Enable = enable
            };
            await ReplaceSubIpsAsync(nodeId, parent, request.SubIps);
        });

        await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { nodeId }, "resync");
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateStatusAsync(long nodeId, NodeStatusRequest request, CancellationToken cancellationToken)
    {
        if (nodeId <= 0 || request.Enable == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var syncTask = request.Enable.Value ? "sync_enable" : "sync_disable";
        var now = DateTime.Now;

        await _db.Updateable<Node>()
            .SetColumns(n => new Node
            {
                Enable = request.Enable,
                ConfigTask = syncTask,
                UpdateAt = now
            })
            .Where(n => n.Id == nodeId)
            .ExecuteCommandAsync();

        await _db.Updateable<Node>()
            .SetColumns(n => new Node { Enable = request.Enable, UpdateAt = now })
            .Where(n => n.Pid == nodeId)
            .ExecuteCommandAsync();

        await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { nodeId }, request.Enable.Value ? "add" : "delete");
        await _eventPublisher.PublishToAdminsAsync("node.status.changed", new
        {
            node_id = nodeId,
            enable = request.Enable,
            online = false,
            checked_at = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
        });

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAntiBlockingAsync(long nodeId, NodeAntiBlockingRequest request, CancellationToken cancellationToken)
    {
        if (nodeId <= 0 || request.Enable == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var exists = await _db.Queryable<Node>().AnyAsync(n => n.Id == nodeId);
        if (!exists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "node_not_found");
        }

        var value = request.Enable.Value ? "1" : "0";
        await _nodeConfigService.UpsertAsync(nodeId, "anti_blocking", value, cancellationToken);
        await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { nodeId }, "resync");
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long nodeId, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { nodeId }, "delete");

        await _db.Ado.UseTranAsync(async () =>
        {
            var subIds = await _db.Queryable<Node>().Where(n => n.Pid == nodeId).Select(n => n.Id).ToListAsync();
            var ids = new List<int> { (int)nodeId };
            ids.AddRange(subIds);

            await _db.Deleteable<Line>()
                .Where(l => (l.NodeId.HasValue && ids.Contains(l.NodeId.Value)) || (l.NodeIpId.HasValue && ids.Contains(l.NodeIpId.Value)))
                .ExecuteCommandAsync();
            await _db.Deleteable<Node>().Where(n => n.Pid == nodeId).ExecuteCommandAsync();
            await _db.Deleteable<Node>().Where(n => n.Id == nodeId).ExecuteCommandAsync();
        });

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> BatchAsync(NodeBatchRequest request, CancellationToken cancellationToken)
    {
        if (request.Ids == null || request.Ids.Count == 0 || string.IsNullOrWhiteSpace(request.Action))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var action = request.Action.Trim().ToLowerInvariant();
        var ids = request.Ids.Select(id => (int)id).ToList();
        var now = DateTime.Now;

        switch (action)
        {
            case "start":
                await _db.Updateable<Node>()
                    .SetColumns(n => new Node { Enable = true, ConfigTask = "sync_enable", UpdateAt = now })
                    .Where(n => ids.Contains(n.Id))
                    .ExecuteCommandAsync();
                await _db.Updateable<Node>()
                    .SetColumns(n => new Node { Enable = true, UpdateAt = now })
                    .Where(n => ids.Contains(n.Pid))
                    .ExecuteCommandAsync();
                break;
            case "stop":
                await _db.Updateable<Node>()
                    .SetColumns(n => new Node { Enable = false, ConfigTask = "sync_disable", UpdateAt = now })
                    .Where(n => ids.Contains(n.Id))
                    .ExecuteCommandAsync();
                await _db.Updateable<Node>()
                    .SetColumns(n => new Node { Enable = false, UpdateAt = now })
                    .Where(n => ids.Contains(n.Pid))
                    .ExecuteCommandAsync();
                break;
            case "delete":
                await _db.Ado.UseTranAsync(async () =>
                {
                    await _db.Deleteable<Line>()
                        .Where(l => l.NodeId.HasValue && ids.Contains(l.NodeId.Value))
                        .ExecuteCommandAsync();
                    await _db.Deleteable<Node>().Where(n => ids.Contains(n.Pid)).ExecuteCommandAsync();
                    await _db.Deleteable<Node>().Where(n => ids.Contains(n.Id)).ExecuteCommandAsync();
                });
                break;
            default:
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "unknown_action");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<NodeInstallResult>> InstallAsync(long nodeId, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return ServiceResult<NodeInstallResult>.Fail(ErrorCodes.InvalidParam);
        }

        var node = await _db.Queryable<Node>().Where(n => n.Id == nodeId).FirstAsync();
        if (node == null)
        {
            return ServiceResult<NodeInstallResult>.Fail(ErrorCodes.NotFound);
        }

        var now = DateTime.Now;
        await _db.Updateable<Node>()
            .SetColumns(n => new Node
            {
                InstallStatus = "running",
                InstallError = string.Empty,
                InstallAt = now,
                UpdateAt = now
            })
            .Where(n => n.Id == nodeId)
            .ExecuteCommandAsync();

        await _eventPublisher.PublishToAdminsAsync("node.install.progress", new
        {
            node_id = nodeId,
            stage = "running",
            percent = 0,
            current_bytes = 0,
            total_bytes = 0,
            message = string.Empty
        });

        return ServiceResult<NodeInstallResult>.Ok(new NodeInstallResult("running"));
    }

    private async Task<Dictionary<long, IReadOnlyList<NodeSubIp>>> LoadSubIpMapAsync(IReadOnlyList<long> parentIds)
    {
        var map = new Dictionary<long, IReadOnlyList<NodeSubIp>>();
        if (parentIds.Count == 0)
        {
            return map;
        }

        var subs = await _db.Queryable<Node>()
            .Where(n => parentIds.Contains(n.Pid))
            .Select(n => new { n.Id, n.Pid, n.Ip })
            .ToListAsync();

        foreach (var sub in subs)
        {
            var pid = (long)sub.Pid;
            if (!map.TryGetValue(pid, out var list))
            {
                list = new List<NodeSubIp>();
                map[pid] = list;
            }
            ((List<NodeSubIp>)list).Add(new NodeSubIp(sub.Id, sub.Ip));
        }

        return map;
    }

    private async Task<Dictionary<long, long>> LoadLineCountMapAsync(IReadOnlyList<long> parentIds)
    {
        var map = new Dictionary<long, long>();
        if (parentIds.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<Line>()
            .Where(l => l.NodeId.HasValue && parentIds.Contains(l.NodeId.Value))
            .GroupBy(l => l.NodeId)
            .Select(l => new { NodeId = l.NodeId!.Value, Count = SqlFunc.AggregateCount(l.Id) })
            .ToListAsync();

        foreach (var row in rows)
        {
            map[row.NodeId] = row.Count;
        }

        return map;
    }

    private async Task<Dictionary<long, string>> LoadRegionNameMapAsync(IReadOnlyList<Node> nodes)
    {
        var ids = nodes.Where(n => n.RegionId.HasValue && n.RegionId.Value > 0)
            .Select(n => n.RegionId!.Value)
            .Distinct()
            .ToList();

        var map = new Dictionary<long, string>();
        if (ids.Count == 0)
        {
            return map;
        }

        var regions = await _db.Queryable<Region>().Where(r => ids.Contains(r.Id)).Select(r => new { r.Id, r.Name }).ToListAsync();
        foreach (var region in regions)
        {
            if (region.Name != null)
            {
                map[region.Id] = region.Name;
            }
        }

        return map;
    }

    private async Task ReplaceSubIpsAsync(long parentId, Node parent, IReadOnlyList<NodeSubIp>? subIps)
    {
        await _db.Deleteable<Node>().Where(n => n.Pid == parentId).ExecuteCommandAsync();

        if (subIps == null || subIps.Count == 0)
        {
            return;
        }

        var now = DateTime.Now;
        var nodes = new List<Node>();
        foreach (var sub in subIps)
        {
            if (string.IsNullOrWhiteSpace(sub.Ip))
            {
                continue;
            }

            nodes.Add(new Node
            {
                Pid = (int)parentId,
                RegionId = parent.RegionId,
                Name = parent.Name,
                Des = parent.Des,
                Ip = sub.Ip,
                Host = parent.Host,
                Port = parent.Port,
                HttpProxy = parent.HttpProxy,
                IsMgmt = parent.IsMgmt,
                Enable = parent.Enable,
                CreateAt = now,
                UpdateAt = now
            });
        }

        if (nodes.Count > 0)
        {
            await _db.Insertable(nodes).ExecuteCommandAsync();
        }
    }

    private async Task<NodeListItem> BuildNodeListItemAsync(Node node)
    {
        var regionNameMap = await LoadRegionNameMapAsync(new[] { node });
        var subIps = await LoadSubIpMapAsync(new[] { (long)node.Id });
        return new NodeListItem
        {
            Id = node.Id,
            Pid = node.Pid,
            RegionId = node.RegionId,
            RegionName = node.RegionId.HasValue && regionNameMap.TryGetValue(node.RegionId.Value, out var name) ? name : null,
            Name = node.Name,
            Remark = node.Des,
            Ip = node.Ip,
            Token = node.Token,
            Host = node.Host,
            Port = node.Port,
            HttpProxy = node.HttpProxy,
            IsMgmt = node.IsMgmt,
            Enable = node.Enable,
            ConfigTask = node.ConfigTask,
            CheckOn = node.CheckOn,
            CheckProtocol = node.CheckProtocol,
            CheckTimeout = node.CheckTimeout,
            CheckPort = node.CheckPort,
            CheckHost = node.CheckHost,
            CheckPath = node.CheckPath,
            CheckNodeGroup = node.CheckNodeGroup,
            CheckAction = node.CheckAction,
            BwLimit = node.BwLimit,
            Type = node.Level,
            SortOrder = node.Sort,
            CacheDir = node.CacheDir,
            CacheLimit = node.MaxCacheSize,
            LogDir = node.LogDir,
            SshHost = node.SshHost,
            SshPort = node.SshPort,
            SshUser = node.SshUser,
            SshAuthType = node.SshAuthType,
            SshPassword = node.SshPassword,
            SshKey = node.SshKey,
            WorkDir = node.WorkDir,
            AutoInstall = node.AutoInstall,
            InstallStatus = node.InstallStatus,
            InstallError = node.InstallError,
            InstallAt = node.InstallAt,
            SubIps = subIps.TryGetValue(node.Id, out var list) ? list : Array.Empty<NodeSubIp>(),
            LineCount = 0,
            Online = _nodeStatusService.IsOnline(node.Id, TimeSpan.FromSeconds(30)),
            AntiBlocking = true,
            ReportedAntiBlocking = null,
            ConfigDrift = false,
            ConfigDriftFields = null
        };
    }

    private async Task<Dictionary<long, InstallProgressPayload>> LoadInstallProgressMapAsync(IReadOnlyList<long> nodeIds)
    {
        var map = new Dictionary<long, InstallProgressPayload>();
        if (nodeIds.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == InstallProgressType && c.ScopeName == InstallProgressScopeName && c.ScopeId.HasValue && nodeIds.Contains(c.ScopeId.Value))
            .ToListAsync();

        if (rows.Count == 0)
        {
            return map;
        }

        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row.Value))
            {
                continue;
            }

            try
            {
                var payload = JsonSerializer.Deserialize<InstallProgressPayload>(row.Value);
                if (payload?.NodeId > 0)
                {
                    map[payload.NodeId] = payload;
                }
            }
            catch
            {
                continue;
            }
        }

        return map;
    }

    private static bool ParseBoolFlag(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static bool? ExtractReportedAntiBlocking(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (!doc.RootElement.TryGetProperty("anti_blocking", out var value))
            {
                return null;
            }

            return value.ValueKind switch
            {
                JsonValueKind.True => true,
                JsonValueKind.False => false,
                JsonValueKind.Number => value.GetDouble() != 0,
                JsonValueKind.String => ParseBoolFlag(value.GetString()),
                _ => null
            };
        }
        catch
        {
            return null;
        }
    }

    private sealed class InstallProgressPayload
    {
        [JsonPropertyName("node_id")]
        public long NodeId { get; set; }

        [JsonPropertyName("stage")]
        public string? Stage { get; set; }

        [JsonPropertyName("percent")]
        public int Percent { get; set; }

        [JsonPropertyName("current_bytes")]
        public long CurrentBytes { get; set; }

        [JsonPropertyName("total_bytes")]
        public long TotalBytes { get; set; }
    }

    private (DateTime? Start, DateTime? End) ResolveTimeRange(NodeMonitorLogQuery query)
    {
        var layout = "yyyy-MM-dd HH:mm:ss";
        if (query.TimeRange is { Length: >= 2 })
        {
            if (DateTime.TryParseExact(query.TimeRange[0], layout, CultureInfo.InvariantCulture, DateTimeStyles.None, out var start) &&
                DateTime.TryParseExact(query.TimeRange[1], layout, CultureInfo.InvariantCulture, DateTimeStyles.None, out var end))
            {
                return (start, end);
            }
        }

        if (!string.IsNullOrWhiteSpace(query.Start) && !string.IsNullOrWhiteSpace(query.End))
        {
            if (DateTime.TryParseExact(query.Start, layout, CultureInfo.InvariantCulture, DateTimeStyles.None, out var start) &&
                DateTime.TryParseExact(query.End, layout, CultureInfo.InvariantCulture, DateTimeStyles.None, out var end))
            {
                return (start, end);
            }
        }

        return (null, null);
    }

    private string? ResolveAgentToken()
    {
        var token = _configuration["Agent:Token"];
        if (!string.IsNullOrWhiteSpace(token))
        {
            return token;
        }

        return _configuration["AgentToken"];
    }
}
