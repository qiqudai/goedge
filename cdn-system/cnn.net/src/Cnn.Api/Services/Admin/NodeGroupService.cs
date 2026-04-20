
using System.Text.Json;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class NodeGroupService : INodeGroupService
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly INodeStatusService _nodeStatusService;

    public NodeGroupService(
        ISqlSugarClient db,
        IConfigVersionService configVersionService,
        IDnsSyncService dnsSyncService,
        INodeStatusService nodeStatusService
    )
    {
        _db = db;
        _configVersionService = configVersionService;
        _dnsSyncService = dnsSyncService;
        _nodeStatusService = nodeStatusService;
    }

    public async Task<ServiceResult<NodeGroupListResult>> ListAsync(NodeGroupListQuery query, CancellationToken cancellationToken)
    {
        await EnsureCnameDomainColumnAsync();

        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.Limit < 1 ? 20 : query.Limit;

        var q = _db.Queryable<NodeGroup>();
        if (!string.IsNullOrWhiteSpace(query.Keyword))
        {
            var keyword = query.Keyword!.Trim();
            if (long.TryParse(keyword, out var id) && id > 0)
            {
                q = q.Where(g => g.Id == id || g.Name!.Contains(keyword) || g.CnameHostname!.Contains(keyword) || g.Des!.Contains(keyword));
            }
            else
            {
                q = q.Where(g => g.Name!.Contains(keyword) || g.CnameHostname!.Contains(keyword) || g.Des!.Contains(keyword));
            }
        }

        if (query.RegionId is > 0)
        {
            q = q.Where(g => g.RegionId == query.RegionId);
        }

        var total = await q.CountAsync();
        var groups = await q.OrderBy(g => g.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var counts = await LoadNodeGroupCountsAsync(groups);
        var siteCounts = await LoadSiteCountsAsync(groups);
        var forwardCounts = await LoadForwardCountsAsync(groups);

        var list = groups.Select(g =>
        {
            var policy = ParsePolicy(g.BackupSwitchPolicy);
            return new NodeGroupListItem
            {
                Id = g.Id,
                Name = g.Name,
                RegionId = g.RegionId,
                CnameHostname = g.CnameHostname,
                CnameDomain = g.CnameDomain,
                Remark = g.Des,
                SpareIpSwitch = g.BackupSwitchType,
                BackupSwitchPolicy = g.BackupSwitchPolicy,
                Ipv4Resolution = policy?.Ipv4Resolution,
                L2Config = policy?.L2Config,
                SortOrder = policy?.SortOrder,
                NodeCount = counts.TryGetValue(g.Id, out var c) ? c : 0,
                SiteCount = siteCounts.TryGetValue(g.Id, out var sc) ? sc : 0,
                ForwardCount = forwardCounts.TryGetValue(g.Id, out var fc) ? fc : 0
            };
        }).ToList();

        return ServiceResult<NodeGroupListResult>.Ok(new NodeGroupListResult(list, total));
    }

    public async Task<ServiceResult<bool>> CreateAsync(NodeGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        await EnsureCnameDomainColumnAsync();

        int? regionId = request.RegionId.GetValueOrDefault() == 0 ? null : (int?)request.RegionId;
        var domain = await ResolveCnameDomainAsync(request.CnameDomain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid cname domain");
        }

        var hostname = NormalizeGroupHostname(request.CnameHostname, domain);
        if (string.IsNullOrWhiteSpace(hostname))
        {
            hostname = await GenerateUniqueGroupHostnameAsync();
            if (string.IsNullOrWhiteSpace(hostname))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InternalError);
            }
        }

        var ipv4Resolution = string.IsNullOrWhiteSpace(request.Ipv4Resolution)
            ? DomainHelper.GenerateToken(8)
            : request.Ipv4Resolution.Trim();

        var policy = new NodeGroupPolicy
        {
            Ipv4Resolution = ipv4Resolution,
            L2Config = request.L2Config?.Trim(),
            SortOrder = request.SortOrder ?? 0
        };
        var policyJson = JsonSerializer.Serialize(policy);

        var now = DateTime.Now;
        var group = new NodeGroup
        {
            Name = request.Name?.Trim(),
            RegionId = regionId,
            CnameHostname = hostname,
            CnameDomain = domain,
            Des = request.Remark,
            BackupSwitchType = request.SpareIpSwitch,
            BackupSwitchPolicy = policyJson,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(group).ExecuteReturnIdentityAsync();
        group.Id = id;

        await _configVersionService.BumpAsync("node_group", new[] { (long)id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long groupId, NodeGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        await EnsureCnameDomainColumnAsync();

        var existing = await _db.Queryable<NodeGroup>().Where(g => g.Id == groupId).FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        int? regionId = request.RegionId.GetValueOrDefault() == 0 ? null : (int?)request.RegionId;
        var domainInput = string.IsNullOrWhiteSpace(request.CnameDomain) ? existing.CnameDomain : request.CnameDomain;
        var domain = await ResolveCnameDomainAsync(domainInput);
        if (string.IsNullOrWhiteSpace(domain))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid cname domain");
        }

        var hostname = NormalizeGroupHostname(request.CnameHostname, domain);
        if (string.IsNullOrWhiteSpace(hostname))
        {
            hostname = NormalizeGroupHostname(existing.CnameHostname, domain);
        }

        var policy = ParsePolicy(existing.BackupSwitchPolicy) ?? new NodeGroupPolicy();
        var ipv4Resolution = string.IsNullOrWhiteSpace(request.Ipv4Resolution) ? policy.Ipv4Resolution : request.Ipv4Resolution.Trim();
        var l2Config = string.IsNullOrWhiteSpace(request.L2Config) ? policy.L2Config : request.L2Config.Trim();
        var sortOrder = request.SortOrder ?? policy.SortOrder;

        var updatedPolicy = new NodeGroupPolicy
        {
            Ipv4Resolution = ipv4Resolution,
            L2Config = l2Config,
            SortOrder = sortOrder
        };
        var policyJson = JsonSerializer.Serialize(updatedPolicy);

        var now = DateTime.Now;
        var name = request.Name?.Trim();
        var remark = request.Remark?.Trim();
        await _db.Updateable<NodeGroup>()
            .SetColumns(g => new NodeGroup
            {
                Name = name,
                RegionId = regionId,
                CnameHostname = hostname,
                CnameDomain = domain,
                Des = remark,
                BackupSwitchType = request.SpareIpSwitch,
                BackupSwitchPolicy = policyJson,
                UpdateAt = now
            })
            .Where(g => g.Id == groupId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("node_group", new[] { groupId }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long groupId, CancellationToken cancellationToken)
    {
        if (groupId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var lineCount = await _db.Queryable<Line>().Where(l => l.NodeGroupId == groupId).CountAsync();
        if (lineCount > 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "node_group.has_nodes");
        }

        var pkgCount = await _db.Queryable<Package>()
            .Where(p => p.NodeGroupId == groupId || p.BackupNodeGroup == groupId)
            .CountAsync();
        if (pkgCount > 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "node_group.has_packages");
        }

        var userPkgCount = await _db.Queryable<UserPackage>()
            .Where(p => p.NodeGroupId == groupId || p.BackupNodeGroup == groupId)
            .CountAsync();
        if (userPkgCount > 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "node_group.has_packages");
        }

        await _db.Deleteable<NodeGroup>().Where(g => g.Id == groupId).ExecuteCommandAsync();
        await _configVersionService.BumpAsync("node_group", new[] { groupId }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }
    public async Task<ServiceResult<NodeGroupResolutionResult>> GetResolutionAsync(long groupId, NodeGroupResolutionQuery query, CancellationToken cancellationToken)
    {
        await EnsureCnameDomainColumnAsync();

        var group = await _db.Queryable<NodeGroup>().Where(g => g.Id == groupId).FirstAsync();
        if (group == null)
        {
            return ServiceResult<NodeGroupResolutionResult>.Fail(ErrorCodes.NotFound);
        }

        var rawLineId = query.LineId?.Trim() ?? string.Empty;
        var lineId = rawLineId;
        if (string.Equals(lineId, "all", StringComparison.OrdinalIgnoreCase))
        {
            lineId = string.Empty;
        }
        if (string.IsNullOrWhiteSpace(lineId) && string.IsNullOrWhiteSpace(rawLineId))
        {
            lineId = "default";
        }

        var regionName = string.Empty;
        if (group.RegionId.HasValue && group.RegionId.Value > 0)
        {
            var region = await _db.Queryable<Region>()
                .Where(r => r.Id == group.RegionId.Value)
                .Select(r => r.Name)
                .FirstAsync();
            regionName = region ?? string.Empty;
        }

        var lineQuery = _db.Queryable<Line>().Where(l => l.NodeGroupId == groupId);
        if (!string.IsNullOrWhiteSpace(lineId))
        {
            lineQuery = lineQuery.Where(l => l.LineId == lineId);
        }

        var lines = await lineQuery.ToListAsync();
        var (assigned, assignedIpIds) = await BuildAssignedLineItemsAsync(lines);
        var available = await BuildAvailableLineItemsAsync(group, assignedIpIds);

        var result = new NodeGroupResolutionResult(
            new NodeGroupResolutionMeta(group.Id, group.Name, regionName),
            new NodeGroupResolutionLine(lineId, lineId),
            available,
            assigned
        );

        return ServiceResult<NodeGroupResolutionResult>.Ok(result);
    }

    public async Task<ServiceResult<bool>> AssignResolutionAsync(long groupId, NodeGroupAssignRequest request, CancellationToken cancellationToken)
    {
        if (groupId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var group = await _db.Queryable<NodeGroup>()
            .Where(g => g.Id == groupId)
            .Select(g => new { g.Id, g.RegionId })
            .FirstAsync();
        if (group == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        if (request.Items == null || request.Items.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var lineId = string.IsNullOrWhiteSpace(request.LineId) ? "default" : request.LineId.Trim();
        var lineName = string.IsNullOrWhiteSpace(request.LineName) ? lineId : request.LineName.Trim();

        var nodeIds = request.Items
            .Select(item => item.NodeId == 0 ? item.NodeIpId : item.NodeId)
            .Where(id => id > 0)
            .Select(id => (int)id)
            .Distinct()
            .ToList();
        if (nodeIds.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "no valid items");
        }

        var conflicts = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId != groupId && ((l.NodeId.HasValue && nodeIds.Contains(l.NodeId.Value)) || (l.NodeIpId.HasValue && nodeIds.Contains(l.NodeIpId.Value))))
            .AnyAsync();
        if (conflicts)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.StateConflict, "node already assigned to another group");
        }

        var regionIds = await _db.Queryable<Region>().Select(r => r.Id).ToListAsync();
        if (regionIds.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PreconditionFailed, "node.region_required");
        }

        var regionSet = regionIds.ToHashSet();
        var nodes = await _db.Queryable<Node>()
            .Where(n => nodeIds.Contains(n.Id))
            .Select(n => new { n.Id, n.Enable, n.RegionId })
            .ToListAsync();

        var nodeRegionMap = nodes.ToDictionary(n => n.Id, n => n.RegionId);
        var nodeEnabled = nodes.Where(n => n.Enable == true).Select(n => n.Id).ToHashSet();

        foreach (var id in nodeIds)
        {
            if (!nodeEnabled.Contains(id))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.StateConflict, "node disabled");
            }

            if (!nodeRegionMap.TryGetValue(id, out var rid) || rid == null || rid <= 0 || !regionSet.Contains(rid.Value))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PreconditionFailed, "node.region_required");
            }

            if (group.RegionId is > 0 && rid.Value != group.RegionId)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PreconditionFailed, "node.region_mismatch");
            }
        }

        var now = DateTime.Now;
        var createItems = new List<Line>();
        var assignedIpIds = new List<long>();
        foreach (var item in request.Items)
        {
            var nodeId = item.NodeId == 0 ? item.NodeIpId : item.NodeId;
            var nodeIpId = item.NodeIpId == 0 ? item.NodeId : item.NodeIpId;
            if (nodeId == 0 || nodeIpId == 0)
            {
                continue;
            }
            assignedIpIds.Add(nodeIpId);
            createItems.Add(new Line
            {
                NodeGroupId = (int)groupId,
                NodeId = (int)nodeId,
                NodeIpId = (int)nodeIpId,
                LineId = lineId,
                LineName = lineName,
                Weight = "1",
                Enable = true,
                IsBackup = false,
                EnableBackup = false,
                IsBackupDefaultLine = false,
                EnableBackupDefaultLine = false,
                CreateAt = now,
                UpdateAt = now
            });
        }

        if (createItems.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "no valid items");
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            foreach (var item in createItems)
            {
                var exists = await _db.Queryable<Line>()
                    .Where(l => l.NodeGroupId == item.NodeGroupId && l.LineId == item.LineId && l.NodeIpId == item.NodeIpId)
                    .AnyAsync();
                if (exists)
                {
                    continue;
                }

                await _db.Insertable(item).ExecuteCommandAsync();
            }
        });

        await _configVersionService.BumpAsync("line", new[] { groupId }, cancellationToken);
        var ids = assignedIpIds.Distinct().ToList();
        if (!await _dnsSyncService.SyncLineRecordsAsync(groupId, lineId, lineName, "add", ids))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.ExternalProviderError, "dns_sync_failed");
        }
        if (!await _dnsSyncService.SyncPackageCnameForLineChangeAsync(groupId, lineId, lineName, ids, "add"))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.ExternalProviderError, "dns_sync_failed");
        }

        return ServiceResult<bool>.Ok(true);
    }
    public async Task<ServiceResult<bool>> ResolutionActionAsync(long groupId, NodeGroupActionRequest request, CancellationToken cancellationToken)
    {
        if (groupId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var action = request.Action?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(action))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var allowed = new HashSet<string>
        {
            "enable", "disable", "delete", "set_backup", "unset_backup",
            "set_backup_default", "unset_backup_default", "set_weight", "set_sort"
        };
        if (!allowed.Contains(action))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "unknown action");
        }

        var value = request.Value?.Trim() ?? string.Empty;
        if ((action == "set_weight" || action == "set_sort") && string.IsNullOrWhiteSpace(value))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var targetLines = new List<Line>();
        if (action is "enable" or "disable" or "delete" or "set_weight" or "set_sort")
        {
            targetLines = await _db.Queryable<Line>().Where(l => request.Ids.Contains(l.Id)).ToListAsync();
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            switch (action)
            {
                case "enable":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { Enable = true, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "disable":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { Enable = false, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "delete":
                    await _db.Deleteable<Line>().Where(l => request.Ids.Contains(l.Id)).ExecuteCommandAsync();
                    break;
                case "set_backup":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { IsBackup = true, EnableBackup = true, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "unset_backup":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { IsBackup = false, EnableBackup = false, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "set_backup_default":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { IsBackupDefaultLine = true, EnableBackupDefaultLine = true, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "unset_backup_default":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { IsBackupDefaultLine = false, EnableBackupDefaultLine = false, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "set_weight":
                    await _db.Updateable<Line>()
                        .SetColumns(l => new Line { Weight = value, UpdateAt = DateTime.Now })
                        .Where(l => request.Ids.Contains(l.Id))
                        .ExecuteCommandAsync();
                    break;
                case "set_sort":
                    if (int.TryParse(value, out var sortOrder))
                    {
                        var nodeIds = await _db.Queryable<Line>()
                            .Where(l => request.Ids.Contains(l.Id) && l.NodeId.HasValue)
                            .Select(l => l.NodeId!.Value)
                            .Distinct()
                            .ToListAsync();

                        if (nodeIds.Count > 0)
                        {
                            await _db.Updateable<Node>()
                                .SetColumns(n => new Node { Sort = sortOrder, UpdateAt = DateTime.Now })
                                .Where(n => nodeIds.Contains(n.Id))
                                .ExecuteCommandAsync();
                        }
                    }
                    break;
            }
        });

        await _configVersionService.BumpAsync("line", new[] { groupId }, cancellationToken);

        if (targetLines.Count > 0 && action is "enable" or "disable" or "delete" or "set_weight" or "set_sort")
        {
            var groupLineNodes = new Dictionary<long, Dictionary<(string, string), List<long>>>();
            foreach (var line in targetLines)
            {
                var gid = line.NodeGroupId ?? 0;
                var key = (line.LineId ?? string.Empty, line.LineName ?? string.Empty);
                if (!groupLineNodes.TryGetValue(gid, out var lineMap))
                {
                    lineMap = new Dictionary<(string, string), List<long>>();
                    groupLineNodes[gid] = lineMap;
                }

                var nodeId = line.NodeIpId ?? line.NodeId ?? 0;
                if (!lineMap.TryGetValue(key, out var list))
                {
                    list = new List<long>();
                    lineMap[key] = list;
                }
                if (nodeId > 0)
                {
                    list.Add(nodeId);
                }
            }

            var dnsAction = action switch
            {
                "enable" => "add",
                "disable" => "delete",
                "delete" => "delete",
                _ => "resync"
            };

            var dnsOk = true;
            foreach (var (gid, lineMap) in groupLineNodes)
            {
                foreach (var (key, nodes) in lineMap)
                {
                    var ids = nodes.Distinct().ToList();
                    if (dnsAction == "resync")
                    {
                        ids = await LoadLineNodeIdsAsync(gid, key.Item1);
                    }
                    if (ids.Count == 0 && dnsAction != "resync")
                    {
                        continue;
                    }

                    if (!await _dnsSyncService.SyncLineRecordsAsync(gid, key.Item1, key.Item2, dnsAction, ids))
                    {
                        dnsOk = false;
                        break;
                    }
                    if (!await _dnsSyncService.SyncPackageCnameForLineChangeAsync(gid, key.Item1, key.Item2, ids, dnsAction))
                    {
                        dnsOk = false;
                        break;
                    }
                }

                if (!dnsOk)
                {
                    break;
                }
            }

            if (!dnsOk)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.ExternalProviderError, "dns_sync_failed");
            }
        }

        return ServiceResult<bool>.Ok(true);
    }

    private async Task EnsureCnameDomainColumnAsync()
    {
        if (_db.DbMaintenance.IsAnyColumn("node_group", "cname_domain"))
        {
            return;
        }

        await Task.Run(() =>
        {
            _db.DbMaintenance.AddColumn("node_group", new DbColumnInfo
            {
                DbColumnName = "cname_domain",
                DataType = "varchar(255)",
                IsNullable = true
            });
        });
    }

    private async Task<string?> ResolveCnameDomainAsync(string? input)
    {
        var normalized = DomainHelper.NormalizeDomainInput(input);
        if (string.IsNullOrWhiteSpace(normalized))
        {
            var first = await _db.Queryable<CnameDomains>()
                .OrderBy(c => c.Id, OrderByType.Asc)
                .FirstAsync();
            if (first == null)
            {
                return null;
            }
            normalized = DomainHelper.NormalizeDomainInput(first.Domain);
        }

        if (string.IsNullOrWhiteSpace(normalized) || !DomainHelper.IsValidDomain(normalized))
        {
            return null;
        }

        var exists = await _db.Queryable<CnameDomains>()
            .Where(c => c.Domain == normalized)
            .AnyAsync();
        return exists ? normalized : null;
    }

    private static string NormalizeGroupHostname(string? host, string domain)
    {
        var normalized = DomainHelper.NormalizeDomainInput(host);
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return string.Empty;
        }

        var normalizedDomain = DomainHelper.NormalizeDomainInput(domain);
        if (!string.IsNullOrWhiteSpace(normalizedDomain))
        {
            if (string.Equals(normalized, normalizedDomain, StringComparison.OrdinalIgnoreCase))
            {
                return "@";
            }

            var suffix = "." + normalizedDomain;
            if (normalized.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
            {
                normalized = normalized[..^suffix.Length];
            }
        }

        return normalized.TrimEnd('.');
    }

    private async Task<string?> GenerateUniqueGroupHostnameAsync()
    {
        for (var i = 0; i < 5; i++)
        {
            var token = DomainHelper.GenerateToken(8);
            var count = await _db.Queryable<NodeGroup>().Where(g => g.CnameHostname == token).CountAsync();
            if (count == 0)
            {
                return token;
            }
        }

        return null;
    }

    private async Task<(IReadOnlyList<NodeGroupResolutionAssigned> Assigned, Dictionary<long, bool> AssignedIpIds)> BuildAssignedLineItemsAsync(IReadOnlyList<Line> lines)
    {
        var items = new List<NodeGroupResolutionAssigned>();
        var ipIds = new Dictionary<long, bool>();
        if (lines.Count == 0)
        {
            return (items, ipIds);
        }

        var nodeIds = new HashSet<int>();
        foreach (var line in lines)
        {
            if (line.NodeId.HasValue)
            {
                nodeIds.Add(line.NodeId.Value);
            }
            if (line.NodeIpId.HasValue)
            {
                nodeIds.Add(line.NodeIpId.Value);
                ipIds[line.NodeIpId.Value] = true;
            }
        }

        var nodes = await _db.Queryable<Node>().Where(n => nodeIds.Contains(n.Id)).ToListAsync();
        var nodeMap = nodes.ToDictionary(n => n.Id, n => n);

        foreach (var line in lines)
        {
            nodeMap.TryGetValue(line.NodeId ?? 0, out var node);
            nodeMap.TryGetValue(line.NodeIpId ?? 0, out var nodeIp);
            nodeIp ??= node;
            items.Add(new NodeGroupResolutionAssigned(
                line.Id,
                line.NodeId ?? 0,
                line.NodeIpId ?? 0,
                line.LineId,
                line.LineName,
                node?.Name ?? string.Empty,
                nodeIp?.Ip ?? string.Empty,
                _nodeStatusService.IsOnline(line.NodeId ?? 0, TimeSpan.FromSeconds(90)),
                line.Enable ?? false,
                node?.Enable ?? false,
                line.IsBackup ?? false,
                line.IsBackupDefaultLine ?? false,
                line.Weight,
                node?.Sort
            ));
        }

        return (items, ipIds);
    }

    private async Task<IReadOnlyList<NodeGroupResolutionItem>> BuildAvailableLineItemsAsync(NodeGroup group, Dictionary<long, bool> assignedIpIds)
    {
        var regionIds = await _db.Queryable<Region>().Select(r => r.Id).ToListAsync();
        if (regionIds.Count == 0)
        {
            return Array.Empty<NodeGroupResolutionItem>();
        }

        var q = _db.Queryable<Node>()
            .Where(n => n.Enable == true)
            .Where(n => n.RegionId.HasValue && regionIds.Contains(n.RegionId.Value));

        if (group.RegionId.HasValue && group.RegionId.Value > 0)
        {
            q = q.Where(n => n.RegionId == group.RegionId);
        }

        var nodes = await q.ToListAsync();
        if (nodes.Count == 0)
        {
            return Array.Empty<NodeGroupResolutionItem>();
        }

        var otherLines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId != group.Id)
            .Select(l => new { l.NodeId, l.NodeIpId })
            .ToListAsync();
        var otherAssigned = new HashSet<int>();
        foreach (var line in otherLines)
        {
            if (line.NodeId.HasValue)
            {
                otherAssigned.Add(line.NodeId.Value);
            }
            if (line.NodeIpId.HasValue)
            {
                otherAssigned.Add(line.NodeIpId.Value);
            }
        }

        var nameMap = nodes.ToDictionary(n => n.Id, n => n.Name ?? string.Empty);
        var result = new List<NodeGroupResolutionItem>();
        foreach (var node in nodes)
        {
            var nodeId = node.Id;
            var parentId = node.Pid > 0 ? node.Pid : nodeId;
            var name = node.Pid > 0 && nameMap.TryGetValue(parentId, out var parentName) ? parentName : node.Name;

            if (assignedIpIds.ContainsKey(nodeId))
            {
                continue;
            }

            if (otherAssigned.Contains(nodeId))
            {
                continue;
            }

            result.Add(new NodeGroupResolutionItem(parentId, nodeId, name, node.Ip, _nodeStatusService.IsOnline(parentId, TimeSpan.FromSeconds(90))));
        }

        return result;
    }

    private async Task<Dictionary<long, long>> LoadNodeGroupCountsAsync(IReadOnlyList<NodeGroup> groups)
    {
        var map = new Dictionary<long, long>();
        var ids = groups.Select(g => g.Id).ToList();
        if (ids.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId.HasValue && ids.Contains(l.NodeGroupId.Value))
            .GroupBy(l => l.NodeGroupId)
            .Select(l => new { NodeGroupId = l.NodeGroupId!.Value, Count = SqlFunc.AggregateDistinctCount(l.NodeId) })
            .ToListAsync();
        foreach (var row in rows)
        {
            map[row.NodeGroupId] = row.Count;
        }
        return map;
    }

    private async Task<Dictionary<long, long>> LoadSiteCountsAsync(IReadOnlyList<NodeGroup> groups)
    {
        var map = new Dictionary<long, long>();
        var ids = groups.Select(g => g.Id).ToList();
        if (ids.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<Site>()
            .Where(s => s.NodeGroupId.HasValue && ids.Contains(s.NodeGroupId.Value))
            .GroupBy(s => s.NodeGroupId)
            .Select(s => new { NodeGroupId = s.NodeGroupId!.Value, Count = SqlFunc.AggregateCount(s.Id) })
            .ToListAsync();
        foreach (var row in rows)
        {
            map[row.NodeGroupId] = row.Count;
        }
        return map;
    }

    private async Task<Dictionary<long, long>> LoadForwardCountsAsync(IReadOnlyList<NodeGroup> groups)
    {
        var map = new Dictionary<long, long>();
        var ids = groups.Select(g => g.Id).ToList();
        if (ids.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<StreamEntity>()
            .Where(s => s.NodeGroupId.HasValue && ids.Contains(s.NodeGroupId.Value))
            .GroupBy(s => s.NodeGroupId)
            .Select(s => new { NodeGroupId = s.NodeGroupId!.Value, Count = SqlFunc.AggregateCount(s.Id) })
            .ToListAsync();
        foreach (var row in rows)
        {
            map[row.NodeGroupId] = row.Count;
        }
        return map;
    }

    private async Task<List<long>> LoadLineNodeIdsAsync(long groupId, string lineId)
    {
        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId && l.LineId == lineId && l.Enable == true)
            .Select(l => new { l.NodeId, l.NodeIpId })
            .ToListAsync();

        var ids = new HashSet<long>();
        foreach (var line in lines)
        {
            var nodeId = line.NodeIpId ?? line.NodeId ?? 0;
            if (nodeId > 0)
            {
                ids.Add(nodeId);
            }
        }

        return ids.ToList();
    }

    private static NodeGroupPolicy? ParsePolicy(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<NodeGroupPolicy>(raw);
        }
        catch
        {
            return null;
        }
    }

    private sealed class NodeGroupPolicy
    {
        public string? Ipv4Resolution { get; set; }

        public string? L2Config { get; set; }

        public int SortOrder { get; set; }
    }
}
